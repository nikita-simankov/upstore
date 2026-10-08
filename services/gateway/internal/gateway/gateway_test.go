package gateway

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/nikita-simankov/upstore/shared/authtoken"
)

// upstreamSeen is what the fake upstream received.
type upstreamSeen struct {
	path          string
	authorization string
	accountID     string
}

// fakeUpstream records the last request it received and answers 200.
func fakeUpstream(t *testing.T) (*httptest.Server, *upstreamSeen) {
	t.Helper()
	seen := &upstreamSeen{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen.path = r.URL.Path
		seen.authorization = r.Header.Get("Authorization")
		seen.accountID = r.Header.Get(accountIDHeader)
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)
	return server, seen
}

// newTestGateway returns a gateway with an accounts upstream and a protected "profiles" upstream.
// The signing key is returned so tests can mint tokens.
func newTestGateway(t *testing.T) (*Gateway, *upstreamSeen, *upstreamSeen, ed25519.PrivateKey, *time.Time) {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}

	accounts, accountsSeen := fakeUpstream(t)
	profiles, profilesSeen := fakeUpstream(t)
	accountsURL, _ := url.Parse(accounts.URL)
	profilesURL, _ := url.Parse(profiles.URL)

	g := New(pub, accountsURL)
	g.Protect("/v1/profiles/", profilesURL)
	clock := time.Date(2026, 10, 8, 12, 0, 0, 0, time.UTC)
	g.now = func() time.Time { return clock }
	return g, accountsSeen, profilesSeen, priv, &clock
}

// mint signs an access token the way the accounts service does.
func mint(t *testing.T, key ed25519.PrivateKey, subject string, expires time.Time) string {
	t.Helper()
	claims := struct {
		SessionID string `json:"sid"`
		jwt.RegisteredClaims
	}{
		SessionID: "session-1",
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   subject,
			Issuer:    authtoken.IssuerName,
			IssuedAt:  jwt.NewNumericDate(expires.Add(-15 * time.Minute)),
			ExpiresAt: jwt.NewNumericDate(expires),
		},
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims).SignedString(key)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	return token
}

// send issues a request through the gateway and returns the status code.
func send(g *Gateway, method, path string, headers map[string]string) int {
	req := httptest.NewRequest(method, path, strings.NewReader(""))
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	g.Handler().ServeHTTP(rec, req)
	return rec.Code
}

// TestAuthRoutesPassThroughWithoutToken tests that auth requests reach accounts with no token,
// and that a client-supplied account header is removed.
func TestAuthRoutesPassThroughWithoutToken(t *testing.T) {
	g, accounts, _, _, _ := newTestGateway(t)

	code := send(g, http.MethodPost, "/v1/auth/login", map[string]string{
		accountIDHeader: "spoofed-account",
	})
	if code != http.StatusOK {
		t.Fatalf("login through gateway: status %d", code)
	}
	if accounts.path != "/v1/auth/login" {
		t.Errorf("accounts received path %q", accounts.path)
	}
	if accounts.accountID != "" {
		t.Errorf("accounts received account header %q, want none on an auth route", accounts.accountID)
	}
}

// TestProtectedRouteNeedsValidToken tests the status codes for missing, malformed, expired,
// and forged tokens, and that none of them reach the upstream.
func TestProtectedRouteNeedsValidToken(t *testing.T) {
	g, _, profiles, key, clock := newTestGateway(t)
	valid := mint(t, key, "account-1", clock.Add(time.Hour))
	expired := mint(t, key, "account-1", clock.Add(-time.Minute))

	_, otherKey, _ := ed25519.GenerateKey(rand.Reader)
	forged := mint(t, otherKey, "account-1", clock.Add(time.Hour))

	cases := map[string]string{
		"no header":     "",
		"not bearer":    "Basic abc",
		"empty bearer":  "Bearer ",
		"expired":       "Bearer " + expired,
		"forged by key": "Bearer " + forged,
		"garbage":       "Bearer not.a.token",
	}
	for name, header := range cases {
		headers := map[string]string{}
		if header != "" {
			headers["Authorization"] = header
		}
		if code := send(g, http.MethodGet, "/v1/profiles/me", headers); code != http.StatusUnauthorized {
			t.Errorf("%s: status %d, want 401", name, code)
		}
	}
	if profiles.path != "" {
		t.Errorf("a rejected request reached the upstream: %q", profiles.path)
	}

	if code := send(g, http.MethodGet, "/v1/profiles/me", map[string]string{"Authorization": "Bearer " + valid}); code != http.StatusOK {
		t.Errorf("valid token: status %d, want 200", code)
	}
}

// TestProtectedRouteSetsVerifiedAccountHeader tests that the upstream gets the verified account,
// never the client's copy, and never the bearer token.
func TestProtectedRouteSetsVerifiedAccountHeader(t *testing.T) {
	g, _, profiles, key, clock := newTestGateway(t)
	valid := mint(t, key, "account-42", clock.Add(time.Hour))

	code := send(g, http.MethodGet, "/v1/profiles/me", map[string]string{
		"Authorization": "Bearer " + valid,
		accountIDHeader: "someone-else",
	})
	if code != http.StatusOK {
		t.Fatalf("status %d", code)
	}
	if profiles.accountID != "account-42" {
		t.Errorf("upstream account = %q, want account-42", profiles.accountID)
	}
	if profiles.authorization != "" {
		t.Errorf("upstream received the bearer token")
	}
}

// TestUnknownRoutesAreNotFound tests that paths without a route are not forwarded.
func TestUnknownRoutesAreNotFound(t *testing.T) {
	g, accounts, profiles, _, _ := newTestGateway(t)

	if code := send(g, http.MethodGet, "/v1/admin/secrets", nil); code != http.StatusNotFound {
		t.Errorf("unknown route: status %d, want 404", code)
	}
	if accounts.path != "" || profiles.path != "" {
		t.Error("an unknown route reached an upstream")
	}
}

// TestHealth tests the gateway's own health endpoint.
func TestHealth(t *testing.T) {
	g, _, _, _, _ := newTestGateway(t)
	if code := send(g, http.MethodGet, "/health", nil); code != http.StatusOK {
		t.Errorf("health: status %d", code)
	}
}

// TestErrorBodyIsJSON tests that 401 responses are JSON.
func TestErrorBodyIsJSON(t *testing.T) {
	g, _, _, _, _ := newTestGateway(t)
	req := httptest.NewRequest(http.MethodGet, "/v1/profiles/me", nil)
	rec := httptest.NewRecorder()
	g.Handler().ServeHTTP(rec, req)

	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil || body["error"] == "" {
		t.Errorf("401 body = %q", rec.Body.String())
	}
}
