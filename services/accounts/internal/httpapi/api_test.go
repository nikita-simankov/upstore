package httpapi

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/time/rate"

	"github.com/nikita-simankov/upstore/services/accounts/internal/auth"
	"github.com/nikita-simankov/upstore/services/accounts/internal/queries"
	"github.com/nikita-simankov/upstore/services/accounts/internal/testdb"
	"github.com/nikita-simankov/upstore/services/accounts/internal/tokens"
	"github.com/nikita-simankov/upstore/services/accounts/internal/verification"
)

const (
	testEmail    = "ivan@mail.by"
	testPassword = "correct horse battery"
)

// recordingMailer keeps every verification link it is asked to send.
type recordingMailer struct {
	links []string
}

func (m *recordingMailer) SendVerification(_ context.Context, _, link string) error {
	m.links = append(m.links, link)
	return nil
}

// newTestAPIFull returns a handler on the test database, the query helper, and the mailer that
// receives verification links. Limits are generous unless a test sets its own.
func newTestAPIFull(t *testing.T, cfg Config) (http.Handler, *queries.Queries, *recordingMailer) {
	t.Helper()
	pool := testdb.Pool(t)
	_, key, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	sessions := auth.NewSessions(pool, tokens.NewIssuer(key))
	mailer := &recordingMailer{}
	verify := verification.NewService(pool, mailer, "https://app.test/verify-email?token=")
	return New(auth.NewService(pool), sessions, verify, cfg).Routes(), queries.New(pool), mailer
}

// newTestAPI returns a handler on the test database. Limits are generous unless a test sets its own.
func newTestAPI(t *testing.T, cfg Config) (http.Handler, *queries.Queries) {
	t.Helper()
	h, q, _ := newTestAPIFull(t, cfg)
	return h, q
}

// TestRegisterSendsLinkAndVerifyActivatesAccount tests the full verification path over HTTP:
// registration emails a link, the link activates the account, and sign-in then works.
func TestRegisterSendsLinkAndVerifyActivatesAccount(t *testing.T) {
	h, _, mailer := newTestAPIFull(t, generousConfig())

	if rec := register(h, testEmail, testPassword, "shopper"); rec.Code != http.StatusCreated {
		t.Fatalf("register: status %d, body %s", rec.Code, rec.Body)
	}
	if len(mailer.links) != 1 {
		t.Fatalf("links sent = %d, want 1", len(mailer.links))
	}
	token := strings.TrimPrefix(mailer.links[0], "https://app.test/verify-email?token=")

	if rec := do(h, "/v1/auth/login", `{"email":"`+testEmail+`","password":"`+testPassword+`"}`, "192.0.2.30:1"); rec.Code != http.StatusForbidden {
		t.Fatalf("login before verification: status %d, want 403", rec.Code)
	}
	if rec := do(h, "/v1/auth/verify-email", `{"token":"`+token+`"}`, "192.0.2.30:1"); rec.Code != http.StatusOK {
		t.Fatalf("verify-email: status %d, body %s", rec.Code, rec.Body)
	}
	if rec := do(h, "/v1/auth/verify-email", `{"token":"`+token+`"}`, "192.0.2.30:1"); rec.Code != http.StatusBadRequest {
		t.Errorf("second verify with the same link: status %d, want 400", rec.Code)
	}
	if rec := do(h, "/v1/auth/login", `{"email":"`+testEmail+`","password":"`+testPassword+`"}`, "192.0.2.30:1"); rec.Code != http.StatusOK {
		t.Errorf("login after verification: status %d, want 200", rec.Code)
	}
}

// TestResendVerificationAlwaysAccepts tests that resend answers 202 for every email, and sends
// a link only when the account is pending.
func TestResendVerificationAlwaysAccepts(t *testing.T) {
	h, _, mailer := newTestAPIFull(t, generousConfig())
	register(h, testEmail, testPassword, "shopper")
	before := len(mailer.links)

	for _, email := range []string{testEmail, "nobody@mail.by"} {
		rec := do(h, "/v1/auth/resend-verification", `{"email":"`+email+`"}`, "192.0.2.31:1")
		if rec.Code != http.StatusAccepted {
			t.Errorf("resend for %s: status %d, want 202", email, rec.Code)
		}
	}
	if len(mailer.links) != before+1 {
		t.Errorf("links sent by resend = %d, want exactly 1 (for the pending account)", len(mailer.links)-before)
	}
}

// generousConfig is high enough that no test hits the limiter by accident.
func generousConfig() Config {
	return Config{
		LoginPerIP:    rate.Inf,
		LoginBurst:    1000,
		LoginPerEmail: rate.Inf,
		EmailBurst:    1000,
		RegisterPerIP: rate.Inf,
		RegisterBurst: 1000,
	}
}

// do sends a request to h from the given peer address and returns the response.
func do(h http.Handler, path, body, peer string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = peer
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

// errorOf returns the "error" field of a JSON error response.
func errorOf(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("response is not JSON: %q", rec.Body.String())
	}
	return body["error"]
}

// register posts a registration and returns the response.
func register(h http.Handler, email, password, accountType string) *httptest.ResponseRecorder {
	body := `{"email":"` + email + `","password":"` + password + `","account_type":"` + accountType + `"}`
	return do(h, "/v1/auth/register", body, "192.0.2.1:1000")
}

// TestRegisterAndLoginFlow tests the happy path: registration returns 201 with a pending account,
// and sign-in after email verification returns 200.
func TestRegisterAndLoginFlow(t *testing.T) {
	h, q := newTestAPI(t, generousConfig())

	rec := register(h, testEmail, testPassword, "shopper")
	if rec.Code != http.StatusCreated {
		t.Fatalf("register: status %d, body %s", rec.Code, rec.Body)
	}
	var created accountResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("register response: %v", err)
	}
	if created.Status != "pending" || len(created.ID) != 36 {
		t.Errorf("register response = %+v, want pending account with a UUID", created)
	}

	var id pgtype.UUID
	if err := id.Scan(created.ID); err != nil {
		t.Fatalf("scan id: %v", err)
	}
	if err := q.MarkEmailVerified(t.Context(), id); err != nil {
		t.Fatalf("MarkEmailVerified: %v", err)
	}

	login := do(h, "/v1/auth/login", `{"email":"`+testEmail+`","password":"`+testPassword+`"}`, "192.0.2.1:1000")
	if login.Code != http.StatusOK {
		t.Fatalf("login: status %d, body %s", login.Code, login.Body)
	}
}

// TestRegisterErrors tests the status code for each rejected registration.
func TestRegisterErrors(t *testing.T) {
	h, _ := newTestAPI(t, generousConfig())
	register(h, testEmail, testPassword, "shopper")

	tests := []struct {
		name        string
		email       string
		password    string
		accountType string
		want        int
	}{
		{"duplicate email in other case", "IVAN@MAIL.BY", testPassword, "shopper", http.StatusConflict},
		{"invalid email", "not-an-email", testPassword, "shopper", http.StatusBadRequest},
		{"weak password", "new@mail.by", "short", "shopper", http.StatusBadRequest},
		{"unknown account type", "new@mail.by", testPassword, "admin", http.StatusBadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := register(h, tt.email, tt.password, tt.accountType)
			if rec.Code != tt.want {
				t.Errorf("status = %d, want %d, body %s", rec.Code, tt.want, rec.Body)
			}
			if rec.Code >= 500 {
				t.Errorf("internal error leaked to the client: %s", rec.Body)
			}
		})
	}
}

// TestLoginErrors tests the status code for each refused sign-in, and that the message never
// reveals whether the email exists.
func TestLoginErrors(t *testing.T) {
	h, q := newTestAPI(t, generousConfig())

	// Pending account: correct password, not verified.
	register(h, testEmail, testPassword, "seller")

	wrongPassword := do(h, "/v1/auth/login", `{"email":"`+testEmail+`","password":"nope nope nope"}`, "192.0.2.2:1")
	unknown := do(h, "/v1/auth/login", `{"email":"nobody@mail.by","password":"nope nope nope"}`, "192.0.2.2:1")
	if wrongPassword.Code != http.StatusUnauthorized || unknown.Code != http.StatusUnauthorized {
		t.Fatalf("wrong password %d, unknown email %d; want 401 for both", wrongPassword.Code, unknown.Code)
	}
	if errorOf(t, wrongPassword) != errorOf(t, unknown) {
		t.Errorf("messages differ: %q vs %q", errorOf(t, wrongPassword), errorOf(t, unknown))
	}

	pending := do(h, "/v1/auth/login", `{"email":"`+testEmail+`","password":"`+testPassword+`"}`, "192.0.2.2:1")
	if pending.Code != http.StatusForbidden {
		t.Errorf("pending account: status %d, want 403", pending.Code)
	}

	acc, err := q.GetAccountByEmail(t.Context(), testEmail)
	if err != nil {
		t.Fatalf("GetAccountByEmail: %v", err)
	}
	if err := q.MarkEmailVerified(t.Context(), acc.ID); err != nil {
		t.Fatalf("MarkEmailVerified: %v", err)
	}
	if err := q.SetAccountBan(t.Context(), queries.SetAccountBanParams{
		ID:          acc.ID,
		BannedUntil: pgtype.Timestamptz{InfinityModifier: pgtype.Infinity, Valid: true},
		BanReason:   pgtype.Text{String: "spam", Valid: true},
	}); err != nil {
		t.Fatalf("SetAccountBan: %v", err)
	}
	banned := do(h, "/v1/auth/login", `{"email":"`+testEmail+`","password":"`+testPassword+`"}`, "192.0.2.2:1")
	if banned.Code != http.StatusForbidden {
		t.Errorf("banned account: status %d, want 403", banned.Code)
	}
}

// TestRequestBodyChecks tests the size cap and the unknown-field rejection.
func TestRequestBodyChecks(t *testing.T) {
	h, _ := newTestAPI(t, generousConfig())

	big := `{"email":"` + strings.Repeat("a", maxBodyBytes) + `"}`
	if rec := do(h, "/v1/auth/login", big, "192.0.2.3:1"); rec.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("oversized body: status %d, want 413", rec.Code)
	}

	unknown := `{"email":"` + testEmail + `","password":"x","role":"admin"}`
	if rec := do(h, "/v1/auth/login", unknown, "192.0.2.3:1"); rec.Code != http.StatusBadRequest {
		t.Errorf("unknown field: status %d, want 400", rec.Code)
	}
}

// TestLoginRateLimitPerIP tests that repeated sign-in attempts from one address are limited.
func TestLoginRateLimitPerIP(t *testing.T) {
	cfg := generousConfig()
	cfg.LoginPerIP = rate.Every(time.Hour)
	cfg.LoginBurst = 3
	h, _ := newTestAPI(t, cfg)

	body := `{"email":"nobody@mail.by","password":"nope nope nope"}`
	for i := 0; i < 3; i++ {
		if rec := do(h, "/v1/auth/login", body, "192.0.2.4:1"); rec.Code == http.StatusTooManyRequests {
			t.Fatalf("attempt %d limited too early", i+1)
		}
	}
	if rec := do(h, "/v1/auth/login", body, "192.0.2.4:1"); rec.Code != http.StatusTooManyRequests {
		t.Errorf("fourth attempt: status %d, want 429", rec.Code)
	}
	// A different address has its own bucket.
	if rec := do(h, "/v1/auth/login", body, "192.0.2.5:1"); rec.Code == http.StatusTooManyRequests {
		t.Errorf("other address limited: status %d", rec.Code)
	}
}

// TestLoginRateLimitPerEmail tests that attempts against one email are limited even when they
// come from many addresses.
func TestLoginRateLimitPerEmail(t *testing.T) {
	cfg := generousConfig()
	cfg.LoginPerEmail = rate.Every(time.Hour)
	cfg.EmailBurst = 2
	h, _ := newTestAPI(t, cfg)

	body := `{"email":"Victim@Mail.BY","password":"nope nope nope"}`
	do(h, "/v1/auth/login", body, "192.0.2.10:1")
	do(h, "/v1/auth/login", body, "192.0.2.11:1")
	if rec := do(h, "/v1/auth/login", body, "192.0.2.12:1"); rec.Code != http.StatusTooManyRequests {
		t.Errorf("third attempt for the same email: status %d, want 429", rec.Code)
	}
}

// TestKeyedLimiterBoundsMemory tests that the key table does not grow past its cap.
func TestKeyedLimiterBoundsMemory(t *testing.T) {
	now := time.Now()
	l := newKeyedLimiter(rate.Every(time.Hour), 1, func() time.Time { return now })
	for i := 0; i < maxLimiterKeys+10; i++ {
		l.Allow("key-" + strconv.Itoa(i))
	}
	if len(l.entries) > maxLimiterKeys {
		t.Errorf("tracked keys = %d, want at most %d", len(l.entries), maxLimiterKeys)
	}
}

// signInTokens registers, activates, and signs in the test account, and returns the sign-in body.
func signInTokens(t *testing.T, h http.Handler, q *queries.Queries) sessionResponse {
	t.Helper()
	register(h, testEmail, testPassword, "shopper")
	acc, err := q.GetAccountByEmail(t.Context(), testEmail)
	if err != nil {
		t.Fatalf("GetAccountByEmail: %v", err)
	}
	if err := q.MarkEmailVerified(t.Context(), acc.ID); err != nil {
		t.Fatalf("MarkEmailVerified: %v", err)
	}
	rec := do(h, "/v1/auth/login", `{"email":"`+testEmail+`","password":"`+testPassword+`"}`, "192.0.2.20:1")
	if rec.Code != http.StatusOK {
		t.Fatalf("login: status %d, body %s", rec.Code, rec.Body)
	}
	var body sessionResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("login response: %v", err)
	}
	return body
}

// TestLoginReturnsTokensAndRefreshRotates tests that sign-in returns tokens, that refresh
// returns a new pair, and that the old refresh token is then refused.
func TestLoginReturnsTokensAndRefreshRotates(t *testing.T) {
	h, q := newTestAPI(t, generousConfig())
	session := signInTokens(t, h, q)
	if session.AccessToken == "" || session.RefreshToken == "" || session.ExpiresIn != 900 {
		t.Fatalf("login tokens = %+v", session)
	}

	refreshed := do(h, "/v1/auth/refresh", `{"refresh_token":"`+session.RefreshToken+`"}`, "192.0.2.20:1")
	if refreshed.Code != http.StatusOK {
		t.Fatalf("refresh: status %d, body %s", refreshed.Code, refreshed.Body)
	}
	var next tokenResponse
	if err := json.Unmarshal(refreshed.Body.Bytes(), &next); err != nil {
		t.Fatalf("refresh response: %v", err)
	}
	if next.RefreshToken == session.RefreshToken {
		t.Error("refresh returned the same refresh token")
	}

	reused := do(h, "/v1/auth/refresh", `{"refresh_token":"`+session.RefreshToken+`"}`, "192.0.2.20:1")
	if reused.Code != http.StatusUnauthorized {
		t.Errorf("reused refresh token: status %d, want 401", reused.Code)
	}
}

// TestLogoutEndsSession tests that logout returns 204 and the refresh token stops working.
// Logout with an unknown token also returns 204.
func TestLogoutEndsSession(t *testing.T) {
	h, q := newTestAPI(t, generousConfig())
	session := signInTokens(t, h, q)

	if rec := do(h, "/v1/auth/logout", `{"refresh_token":"`+session.RefreshToken+`"}`, "192.0.2.21:1"); rec.Code != http.StatusNoContent {
		t.Fatalf("logout: status %d, want 204", rec.Code)
	}
	if rec := do(h, "/v1/auth/refresh", `{"refresh_token":"`+session.RefreshToken+`"}`, "192.0.2.21:1"); rec.Code != http.StatusUnauthorized {
		t.Errorf("refresh after logout: status %d, want 401", rec.Code)
	}
	if rec := do(h, "/v1/auth/logout", `{"refresh_token":"unknown"}`, "192.0.2.21:1"); rec.Code != http.StatusNoContent {
		t.Errorf("logout with unknown token: status %d, want 204", rec.Code)
	}
}
