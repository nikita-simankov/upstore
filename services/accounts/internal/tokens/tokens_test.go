package tokens

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// newTestIssuer returns an Issuer with a fresh key and a clock the test can move.
func newTestIssuer(t *testing.T) (*Issuer, *time.Time) {
	t.Helper()
	_, key, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	clock := time.Date(2026, 10, 8, 12, 0, 0, 0, time.UTC)
	i := NewIssuer(key)
	i.now = func() time.Time { return clock }
	return i, &clock
}

// TestIssueAndVerifyAccessToken tests that a fresh token verifies and returns its claims.
func TestIssueAndVerifyAccessToken(t *testing.T) {
	i, _ := newTestIssuer(t)

	token, err := i.IssueAccess("11111111-1111-1111-1111-111111111111", "22222222-2222-2222-2222-222222222222")
	if err != nil {
		t.Fatalf("IssueAccess: %v", err)
	}
	claims, err := i.Verify(token)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if claims.AccountID != "11111111-1111-1111-1111-111111111111" || claims.SessionID != "22222222-2222-2222-2222-222222222222" {
		t.Errorf("claims = %+v", claims)
	}
}

// TestVerifyRejectsBadTokens tests that each invalid token fails with ErrInvalidToken.
func TestVerifyRejectsBadTokens(t *testing.T) {
	i, clock := newTestIssuer(t)
	valid, err := i.IssueAccess("acc", "sess")
	if err != nil {
		t.Fatalf("IssueAccess: %v", err)
	}

	otherIssuer, _ := newTestIssuer(t)
	foreign, _ := otherIssuer.IssueAccess("acc", "sess")

	noneClaims := accessClaims{SessionID: "sess", RegisteredClaims: jwt.RegisteredClaims{
		Subject: "acc", Issuer: IssuerName, ExpiresAt: jwt.NewNumericDate(clock.Add(time.Hour)),
	}}
	noneToken, err := jwt.NewWithClaims(jwt.SigningMethodNone, noneClaims).SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("sign none token: %v", err)
	}

	wrongIssuer := accessClaims{SessionID: "sess", RegisteredClaims: jwt.RegisteredClaims{
		Subject: "acc", Issuer: "someone-else", ExpiresAt: jwt.NewNumericDate(clock.Add(time.Hour)),
	}}
	wrongIssuerToken, err := jwt.NewWithClaims(jwt.SigningMethodEdDSA, wrongIssuer).SignedString(i.key)
	if err != nil {
		t.Fatalf("sign wrong-issuer token: %v", err)
	}

	parts := strings.Split(valid, ".")
	tampered := parts[0] + "." + base64.RawURLEncoding.EncodeToString([]byte(`{"sub":"admin","sid":"sess","iss":"upstore-accounts","exp":9999999999}`)) + "." + parts[2]

	noSession := accessClaims{RegisteredClaims: jwt.RegisteredClaims{
		Subject: "acc", Issuer: IssuerName, ExpiresAt: jwt.NewNumericDate(clock.Add(time.Hour)),
	}}
	noSessionToken, _ := jwt.NewWithClaims(jwt.SigningMethodEdDSA, noSession).SignedString(i.key)

	cases := map[string]string{
		"none algorithm":        noneToken,
		"signed by another key": foreign,
		"wrong issuer":          wrongIssuerToken,
		"tampered payload":      tampered,
		"missing session claim": noSessionToken,
		"garbage":               "not.a.token",
		"empty string":          "",
	}
	for name, token := range cases {
		if _, err := i.Verify(token); !errors.Is(err, ErrInvalidToken) {
			t.Errorf("%s: error = %v, want ErrInvalidToken", name, err)
		}
	}
}

// TestVerifyRejectsExpiredToken tests that a token stops working after AccessTokenTTL.
func TestVerifyRejectsExpiredToken(t *testing.T) {
	i, clock := newTestIssuer(t)
	token, err := i.IssueAccess("acc", "sess")
	if err != nil {
		t.Fatalf("IssueAccess: %v", err)
	}

	*clock = clock.Add(AccessTokenTTL - time.Second)
	if _, err := i.Verify(token); err != nil {
		t.Errorf("token just before expiry: %v", err)
	}

	*clock = clock.Add(2 * time.Second)
	if _, err := i.Verify(token); !errors.Is(err, ErrInvalidToken) {
		t.Errorf("token after expiry: error = %v, want ErrInvalidToken", err)
	}
}

// TestNewRefreshTokenIsRandomAndHashed tests that refresh tokens differ, and that the stored
// value is the hash of the token, not the token itself.
func TestNewRefreshTokenIsRandomAndHashed(t *testing.T) {
	a, hashA, err := NewRefreshToken()
	if err != nil {
		t.Fatalf("NewRefreshToken: %v", err)
	}
	b, _, _ := NewRefreshToken()
	if a == b {
		t.Error("two refresh tokens are equal")
	}
	if len(a) < 40 {
		t.Errorf("refresh token is %d characters, want at least 40", len(a))
	}
	if string(hashA) != string(HashRefreshToken(a)) {
		t.Error("stored hash does not match HashRefreshToken(token)")
	}
	if strings.Contains(string(hashA), a) {
		t.Error("stored value contains the token")
	}
}

// TestParseSigningKey tests the key length check and round trip.
func TestParseSigningKey(t *testing.T) {
	seed := make([]byte, ed25519.SeedSize)
	if _, err := rand.Read(seed); err != nil {
		t.Fatalf("rand: %v", err)
	}
	key, err := ParseSigningKey(base64.StdEncoding.EncodeToString(seed))
	if err != nil {
		t.Fatalf("ParseSigningKey: %v", err)
	}
	if len(key) != ed25519.PrivateKeySize {
		t.Errorf("key length = %d", len(key))
	}

	for _, bad := range []string{"", "!!!", base64.StdEncoding.EncodeToString(make([]byte, 16))} {
		if _, err := ParseSigningKey(bad); err == nil {
			t.Errorf("ParseSigningKey(%q) accepted a bad key", bad)
		}
	}
}
