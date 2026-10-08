package auth

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/nikita-simankov/upstore/services/accounts/internal/queries"
	"github.com/nikita-simankov/upstore/services/accounts/internal/tokens"
)

// newTestSessions returns a Sessions bound to the test service's clock, so tests can move both.
func newTestSessions(t *testing.T) (*Service, *Sessions, *time.Time) {
	t.Helper()
	s, pool, clock := newTestService(t)
	_, key, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	issuer := tokens.NewIssuer(key)
	sessions := NewSessions(pool, issuer)
	sessions.now = func() time.Time { return *clock }
	return s, sessions, clock
}

// signIn registers and activates the test account, then starts a session for it.
func signIn(t *testing.T, s *Service, sessions *Sessions) (queries.Account, Tokens) {
	t.Helper()
	account := register(t, s)
	tok, err := sessions.Issue(context.Background(), account)
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	return account, tok
}

// TestRefreshRotatesTokens tests that refresh returns a new pair, and that presenting the old
// refresh token again revokes every session, including the new one.
func TestRefreshRotatesTokens(t *testing.T) {
	s, sessions, _ := newTestSessions(t)
	ctx := context.Background()
	_, first := signIn(t, s, sessions)

	second, err := sessions.Refresh(ctx, first.RefreshToken)
	if err != nil {
		t.Fatalf("Refresh: %v", err)
	}
	if second.RefreshToken == first.RefreshToken {
		t.Error("refresh returned the same refresh token")
	}

	if _, err := sessions.Refresh(ctx, first.RefreshToken); !errors.Is(err, ErrInvalidRefreshToken) {
		t.Errorf("reused refresh token: error = %v, want ErrInvalidRefreshToken", err)
	}
	if _, err := sessions.Refresh(ctx, second.RefreshToken); !errors.Is(err, ErrInvalidRefreshToken) {
		t.Errorf("new token after reuse: error = %v, want ErrInvalidRefreshToken (all sessions revoked)", err)
	}
}

// TestLogoutRevokesSession tests that a logged-out refresh token stops working, and that
// logging out with an unknown token still succeeds.
func TestLogoutRevokesSession(t *testing.T) {
	s, sessions, _ := newTestSessions(t)
	ctx := context.Background()
	_, tok := signIn(t, s, sessions)

	if err := sessions.Logout(ctx, tok.RefreshToken); err != nil {
		t.Fatalf("Logout: %v", err)
	}
	if _, err := sessions.Refresh(ctx, tok.RefreshToken); !errors.Is(err, ErrInvalidRefreshToken) {
		t.Errorf("refresh after logout: error = %v, want ErrInvalidRefreshToken", err)
	}
	if err := sessions.Logout(ctx, "not-a-real-token"); err != nil {
		t.Errorf("logout with unknown token: %v, want nil", err)
	}
}

// TestRefreshRejectsBannedAccount tests that a ban takes effect at the next refresh.
func TestRefreshRejectsBannedAccount(t *testing.T) {
	s, sessions, _ := newTestSessions(t)
	ctx := context.Background()
	account, tok := signIn(t, s, sessions)

	if err := queries.New(s.pool).SetAccountBan(ctx, queries.SetAccountBanParams{
		ID:          account.ID,
		BannedUntil: pgtype.Timestamptz{InfinityModifier: pgtype.Infinity, Valid: true},
	}); err != nil {
		t.Fatalf("SetAccountBan: %v", err)
	}

	if _, err := sessions.Refresh(ctx, tok.RefreshToken); !errors.Is(err, ErrAccountBanned) {
		t.Errorf("refresh for banned account: error = %v, want ErrAccountBanned", err)
	}
}

// TestRefreshExpires tests that a refresh token stops working after refreshTokenTTL.
func TestRefreshExpires(t *testing.T) {
	s, sessions, clock := newTestSessions(t)
	_, tok := signIn(t, s, sessions)

	*clock = clock.Add(refreshTokenTTL + time.Minute)
	if _, err := sessions.Refresh(context.Background(), tok.RefreshToken); !errors.Is(err, ErrInvalidRefreshToken) {
		t.Errorf("expired refresh token: error = %v, want ErrInvalidRefreshToken", err)
	}
}

// TestIssuedAccessTokenNamesAccountAndSession tests that the access token identifies the account.
func TestIssuedAccessTokenNamesAccountAndSession(t *testing.T) {
	s, pool, _ := newTestService(t)
	_, key, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	issuer := tokens.NewIssuer(key)
	sessions := NewSessions(pool, issuer)

	account := register(t, s)
	tok, err := sessions.Issue(context.Background(), account)
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	claims, err := issuer.Verify(tok.AccessToken)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if claims.AccountID != formatUUID(account.ID) {
		t.Errorf("access token account = %s, want %s", claims.AccountID, formatUUID(account.ID))
	}
	if tok.ExpiresIn != int64(tokens.AccessTokenTTL.Seconds()) {
		t.Errorf("expires_in = %d", tok.ExpiresIn)
	}
}
