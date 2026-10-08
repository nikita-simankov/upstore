// Package tokens issues and verifies access tokens (signed JWTs) and creates refresh tokens.
package tokens

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/nikita-simankov/upstore/shared/authtoken"
)

const (
	// IssuerName is the iss claim. It is defined in shared/authtoken, where verifiers check it.
	IssuerName = authtoken.IssuerName
	// AccessTokenTTL is how long an access token is valid. It is short, so a stolen token
	// expires quickly, and the refresh flow handles the rest.
	AccessTokenTTL = 15 * time.Minute
	// refreshTokenBytes is the size of a refresh token before encoding: 256 random bits.
	refreshTokenBytes = 32
)

// ErrInvalidToken is returned for any access token that fails verification.
var ErrInvalidToken = authtoken.ErrInvalidToken

// Claims are the verified contents of an access token.
type Claims = authtoken.Claims

type accessClaims struct {
	SessionID string `json:"sid"`
	jwt.RegisteredClaims
}

// Issuer signs and verifies access tokens with an Ed25519 key.
type Issuer struct {
	key ed25519.PrivateKey
	now func() time.Time
}

// NewIssuer returns an Issuer that uses the system clock.
func NewIssuer(key ed25519.PrivateKey) *Issuer {
	return &Issuer{key: key, now: time.Now}
}

// IssueAccess signs an access token for an account and session.
func (i *Issuer) IssueAccess(accountID, sessionID string) (string, error) {
	now := i.now()
	claims := accessClaims{
		SessionID: sessionID,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   accountID,
			Issuer:    IssuerName,
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(AccessTokenTTL)),
		},
	}
	signed, err := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims).SignedString(i.key)
	if err != nil {
		return "", fmt.Errorf("sign access token: %w", err)
	}
	return signed, nil
}

// Verify checks the signature, algorithm, issuer, and expiry of an access token.
// It returns ErrInvalidToken for any failure, without saying which check failed.
func (i *Issuer) Verify(token string) (Claims, error) {
	claims, err := authtoken.Verify(i.key.Public().(ed25519.PublicKey), token, i.now())
	if err != nil {
		return Claims{}, ErrInvalidToken
	}
	return claims, nil
}

// NewRefreshToken returns a new random refresh token and the hash to store in place of it.
// The token itself is only given to the client.
func NewRefreshToken() (token string, hash []byte, err error) {
	raw := make([]byte, refreshTokenBytes)
	if _, err := rand.Read(raw); err != nil {
		return "", nil, fmt.Errorf("generate refresh token: %w", err)
	}
	token = base64.RawURLEncoding.EncodeToString(raw)
	return token, HashRefreshToken(token), nil
}

// HashRefreshToken returns the SHA-256 hash of a refresh token, which is what the database stores.
func HashRefreshToken(token string) []byte {
	sum := sha256.Sum256([]byte(token))
	return sum[:]
}

// ParseSigningKey decodes an Ed25519 private key from its 32-byte seed in base64.
func ParseSigningKey(encoded string) (ed25519.PrivateKey, error) {
	seed, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("signing key is not base64: %w", err)
	}
	if len(seed) != ed25519.SeedSize {
		return nil, fmt.Errorf("signing key must be %d bytes, got %d", ed25519.SeedSize, len(seed))
	}
	return ed25519.NewKeyFromSeed(seed), nil
}
