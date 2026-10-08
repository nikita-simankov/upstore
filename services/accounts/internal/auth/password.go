// Package auth implements registration and sign-in with email and password.
package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

// Argon2id parameters. They follow the OWASP recommendation for Argon2id:
// 64 MiB of memory, 3 iterations, 4 lanes.
const (
	argonTime    uint32 = 3
	argonMemory  uint32 = 64 * 1024
	argonThreads uint8  = 4
	argonKeyLen  uint32 = 32
	argonSaltLen        = 16
)

const (
	minPasswordLength = 8
	maxPasswordLength = 128
)

// maxConcurrentHashes bounds how many Argon2 operations run at once. Each one allocates
// argonMemory, so without a bound a burst of sign-ins can exhaust the process memory.
const maxConcurrentHashes = 8

var hashSlots = make(chan struct{}, maxConcurrentHashes)

// deriveKey runs Argon2id while holding one of the hashSlots.
func deriveKey(password string, salt []byte, time uint32, memory uint32, threads uint8, keyLen uint32) []byte {
	hashSlots <- struct{}{}
	defer func() { <-hashSlots }()
	return argon2.IDKey([]byte(password), salt, time, memory, threads, keyLen)
}

// ErrMalformedHash means a stored password hash cannot be parsed.
var ErrMalformedHash = errors.New("malformed password hash")

// ValidatePassword checks the password length rules.
func ValidatePassword(password string) error {
	switch {
	case len(password) < minPasswordLength:
		return fmt.Errorf("password must be at least %d characters", minPasswordLength)
	case len(password) > maxPasswordLength:
		return fmt.Errorf("password must be at most %d characters", maxPasswordLength)
	}
	return nil
}

// HashPassword returns an Argon2id hash in the PHC string format, with a random salt.
// The parameters are stored in the string, so hashes stay verifiable after they change.
func HashPassword(password string) (string, error) {
	salt := make([]byte, argonSaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("generate salt: %w", err)
	}

	key := deriveKey(password, salt, argonTime, argonMemory, argonThreads, argonKeyLen)

	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, argonMemory, argonTime, argonThreads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key),
	), nil
}

// VerifyPassword reports whether password matches the encoded hash.
// The comparison runs in constant time. An error means the hash itself is malformed.
func VerifyPassword(password, encoded string) (bool, error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return false, ErrMalformedHash
	}

	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil || version != argon2.Version {
		return false, ErrMalformedHash
	}

	var memory, iterations uint32
	var threads uint8
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &iterations, &threads); err != nil {
		return false, ErrMalformedHash
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false, ErrMalformedHash
	}
	want, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil || len(want) == 0 {
		return false, ErrMalformedHash
	}

	// Only the parameters this service writes are accepted. Anything else in storage is
	// refused before the hash runs, so a tampered row cannot make each login expensive.
	// When the parameters change, add a version check here before accepting the old values.
	if memory != argonMemory || iterations != argonTime || threads != argonThreads || uint32(len(want)) != argonKeyLen {
		return false, ErrMalformedHash
	}

	got := deriveKey(password, salt, iterations, memory, threads, argonKeyLen)
	return subtle.ConstantTimeCompare(got, want) == 1, nil
}
