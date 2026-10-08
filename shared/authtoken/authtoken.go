// Package authtoken verifies access tokens. The accounts service signs them, and every other
// service verifies them with the public key only, so no service besides accounts can mint a token.
package authtoken

import (
	"crypto/ed25519"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	// IssuerName is the iss claim. Verifiers reject tokens from any other issuer.
	IssuerName = "upstore-accounts"
	// Algorithm is the only signing method accepted. It is pinned so that "none" and
	// algorithm-confusion tokens are rejected.
	Algorithm = "EdDSA"
)

// ErrInvalidToken is returned for any token that fails verification. It does not say which check failed.
var ErrInvalidToken = errors.New("invalid or expired access token")

// Claims are the verified contents of an access token.
type Claims struct {
	AccountID string
	SessionID string
}

type accessClaims struct {
	SessionID string `json:"sid"`
	jwt.RegisteredClaims
}

// Verify checks the signature, algorithm, issuer, and expiry of an access token against publicKey
// at the given time.
func Verify(publicKey ed25519.PublicKey, token string, now time.Time) (Claims, error) {
	parsed, err := jwt.ParseWithClaims(token, &accessClaims{}, func(*jwt.Token) (any, error) {
		return publicKey, nil
	},
		jwt.WithValidMethods([]string{Algorithm}),
		jwt.WithIssuer(IssuerName),
		jwt.WithExpirationRequired(),
		jwt.WithTimeFunc(func() time.Time { return now }),
	)
	if err != nil {
		return Claims{}, ErrInvalidToken
	}

	claims, ok := parsed.Claims.(*accessClaims)
	if !ok || !parsed.Valid || claims.Subject == "" || claims.SessionID == "" {
		return Claims{}, ErrInvalidToken
	}
	return Claims{AccountID: claims.Subject, SessionID: claims.SessionID}, nil
}

// ParsePublicKey decodes an Ed25519 public key from base64.
func ParsePublicKey(encoded string) (ed25519.PublicKey, error) {
	raw, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("public key is not base64: %w", err)
	}
	if len(raw) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("public key must be %d bytes, got %d", ed25519.PublicKeySize, len(raw))
	}
	return ed25519.PublicKey(raw), nil
}
