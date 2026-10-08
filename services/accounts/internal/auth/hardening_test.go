package auth

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/nikita-simankov/upstore/shared/events"
)

// TestVerifyPasswordRejectsForeignParameters tests that a stored hash with parameters this
// service does not write is refused, even when the password is correct.
func TestVerifyPasswordRejectsForeignParameters(t *testing.T) {
	hash, err := HashPassword("correct horse")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}

	tampered := []string{
		strings.Replace(hash, "m=65536", "m=4194304", 1),
		strings.Replace(hash, "t=3", "t=100", 1),
		strings.Replace(hash, "p=4", "p=255", 1),
	}
	for _, h := range tampered {
		if h == hash {
			t.Fatalf("test setup did not change the hash: %q", hash)
		}
		ok, err := VerifyPassword("correct horse", h)
		if !errors.Is(err, ErrMalformedHash) || ok {
			t.Errorf("VerifyPassword(%q) = %v, %v; want false, ErrMalformedHash", h, ok, err)
		}
	}
}

// TestLoginRefusesOverlongPassword tests that a password longer than the maximum is refused
// with the generic error, without a database lookup or a hash.
func TestLoginRefusesOverlongPassword(t *testing.T) {
	s, _, _ := newTestService(t)
	register(t, s)

	long := strings.Repeat("a", maxPasswordLength+1)
	if _, err := s.Login(context.Background(), testEmail, long); !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("overlong password: error = %v, want ErrInvalidCredentials", err)
	}
}

// TestLockedAccountLooksLikeWrongPassword tests that a locked account returns the same error
// for the right and the wrong password, so the lock state is not revealed.
func TestLockedAccountLooksLikeWrongPassword(t *testing.T) {
	s, _, _ := newTestService(t)
	register(t, s)
	ctx := context.Background()

	for i := 0; i < maxFailedAttempts; i++ {
		_, _ = s.Login(ctx, testEmail, "wrong password")
	}

	_, rightErr := s.Login(ctx, testEmail, testPassword)
	_, wrongErr := s.Login(ctx, testEmail, "wrong password")
	if !errors.Is(rightErr, ErrInvalidCredentials) || !errors.Is(wrongErr, ErrInvalidCredentials) {
		t.Errorf("locked account: right password = %v, wrong password = %v; want both ErrInvalidCredentials",
			rightErr, wrongErr)
	}
}

// TestFailedSignInIsLoggedWithoutSecrets tests that a failed sign-in writes a log line with the
// account ID, and that the log never contains the email or either password.
func TestFailedSignInIsLoggedWithoutSecrets(t *testing.T) {
	s, _, _ := newTestService(t)
	account := register(t, s)

	var buf bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))
	t.Cleanup(func() { slog.SetDefault(previous) })

	_, _ = s.Login(context.Background(), testEmail, "wrong password")

	out := buf.String()
	if !strings.Contains(out, "sign-in failed") {
		t.Fatalf("no failure log line, got %q", out)
	}
	if !strings.Contains(out, "account_id="+formatUUID(account.ID)) {
		t.Errorf("log line lacks the account ID: %q", out)
	}
	for _, secret := range []string{testEmail, "wrong password", testPassword} {
		if strings.Contains(out, secret) {
			t.Errorf("log line contains %q: %q", secret, out)
		}
	}
}

// TestPendingAccountWithCorrectPasswordAfterLockWindow tests that the lock window expires
// and a pending account still gets ErrAccountNotActive, not a lock error.
func TestPendingAccountWithCorrectPasswordAfterLockWindow(t *testing.T) {
	s, _, clock := newTestService(t)
	if _, err := s.Register(context.Background(), testEmail, testPassword, events.AccountTypeSeller); err != nil {
		t.Fatalf("Register: %v", err)
	}
	*clock = clock.Add(time.Hour)

	if _, err := s.Login(context.Background(), testEmail, testPassword); !errors.Is(err, ErrAccountNotActive) {
		t.Errorf("error = %v, want ErrAccountNotActive", err)
	}
}
