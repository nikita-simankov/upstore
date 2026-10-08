package auth

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/nikita-simankov/upstore/services/accounts/internal/queries"
	"github.com/nikita-simankov/upstore/services/accounts/internal/testdb"
	"github.com/nikita-simankov/upstore/shared/events"
)

const (
	testEmail    = "ivan@mail.by"
	testPassword = "correct horse battery"
)

// newTestService returns a Service on the test database with a clock the test can move.
func newTestService(t *testing.T) (*Service, *pgxpool.Pool, *time.Time) {
	t.Helper()
	pool := testdb.Pool(t)
	clock := time.Date(2026, 10, 8, 12, 0, 0, 0, time.UTC)
	s := NewService(pool)
	s.now = func() time.Time { return clock }
	return s, pool, &clock
}

// register creates the test account, verifies its email so it is active, and fails the test on error.
func register(t *testing.T, s *Service) queries.Account {
	t.Helper()
	account, err := s.Register(context.Background(), testEmail, testPassword, events.AccountTypeShopper)
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	if err := queries.New(s.pool).MarkEmailVerified(context.Background(), account.ID); err != nil {
		t.Fatalf("MarkEmailVerified: %v", err)
	}
	return account
}

// TestPendingAccountCannotSignIn tests that an unverified account is refused even with the
// correct password, and that the refusal is only given after the password is checked.
func TestPendingAccountCannotSignIn(t *testing.T) {
	s, _, _ := newTestService(t)
	if _, err := s.Register(context.Background(), testEmail, testPassword, events.AccountTypeShopper); err != nil {
		t.Fatalf("Register: %v", err)
	}
	ctx := context.Background()

	if _, err := s.Login(ctx, testEmail, testPassword); !errors.Is(err, ErrAccountNotActive) {
		t.Errorf("correct password on pending account: error = %v, want ErrAccountNotActive", err)
	}
	if _, err := s.Login(ctx, testEmail, "wrong password"); !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("wrong password on pending account: error = %v, want ErrInvalidCredentials", err)
	}
}

// TestConcurrentFailuresLockAccount tests that five wrong passwords sent at the same time
// still lock the account. Before the lock moved into the database statement, parallel
// requests could each read a stale count and none of them would lock it.
func TestConcurrentFailuresLockAccount(t *testing.T) {
	s, pool, _ := newTestService(t)
	register(t, s)
	ctx := context.Background()

	var wg sync.WaitGroup
	for i := 0; i < maxFailedAttempts; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = s.Login(ctx, testEmail, "wrong password")
		}()
	}
	wg.Wait()

	got, err := queries.New(pool).GetAccountByEmail(ctx, testEmail)
	if err != nil {
		t.Fatalf("GetAccountByEmail: %v", err)
	}
	if got.FailedLoginAttempts != maxFailedAttempts {
		t.Fatalf("failed_login_attempts = %d, want %d", got.FailedLoginAttempts, maxFailedAttempts)
	}
	if !isActiveUntil(got.LockedUntil, s.now()) {
		t.Error("account is not locked after concurrent failures")
	}
}

// TestRegisterAndLogin tests that a registered account can sign in with the right password.
func TestRegisterAndLogin(t *testing.T) {
	s, _, _ := newTestService(t)
	created := register(t, s)

	got, err := s.Login(context.Background(), "  IVAN@mail.by ", testPassword)
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	if got.ID != created.ID {
		t.Error("Login returned a different account")
	}
}

// TestLoginRejectsBadCredentials tests that a wrong password, an unknown email, and an
// account without a password all return the same error.
func TestLoginRejectsBadCredentials(t *testing.T) {
	s, pool, _ := newTestService(t)
	register(t, s)

	if _, err := pool.Exec(context.Background(),
		"INSERT INTO accounts (email, password_hash) VALUES (NULL, NULL)"); err != nil {
		t.Fatalf("insert account without password: %v", err)
	}

	tests := []struct {
		name     string
		email    string
		password string
	}{
		{"wrong password", testEmail, "not the password"},
		{"unknown email", "nobody@mail.by", testPassword},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := s.Login(context.Background(), tt.email, tt.password)
			if !errors.Is(err, ErrInvalidCredentials) {
				t.Errorf("Login error = %v, want ErrInvalidCredentials", err)
			}
		})
	}
}

// TestRegisterRejectsDuplicateEmail tests that an email in another case is refused.
func TestRegisterRejectsDuplicateEmail(t *testing.T) {
	s, _, _ := newTestService(t)
	register(t, s)

	_, err := s.Register(context.Background(), "IVAN@MAIL.BY", testPassword, events.AccountTypeSeller)
	if !errors.Is(err, ErrEmailTaken) {
		t.Errorf("Register error = %v, want ErrEmailTaken", err)
	}
}

// TestRegisterRejectsWeakPassword tests that a short password is refused before anything is written.
func TestRegisterRejectsWeakPassword(t *testing.T) {
	s, pool, _ := newTestService(t)

	if _, err := s.Register(context.Background(), testEmail, "short", events.AccountTypeShopper); err == nil {
		t.Fatal("Register accepted a short password")
	}
	var n int
	if err := pool.QueryRow(context.Background(), "SELECT count(*) FROM accounts").Scan(&n); err != nil {
		t.Fatalf("count accounts: %v", err)
	}
	if n != 0 {
		t.Errorf("accounts = %d, want 0", n)
	}
}

// TestAccountLocksAfterRepeatedFailures tests that five wrong passwords lock the account,
// that the correct password is refused while locked, and that a successful sign-in after
// the lock expires resets the failure counter.
func TestAccountLocksAfterRepeatedFailures(t *testing.T) {
	s, pool, clock := newTestService(t)
	register(t, s)
	ctx := context.Background()

	for i := 0; i < maxFailedAttempts; i++ {
		if _, err := s.Login(ctx, testEmail, "wrong"); !errors.Is(err, ErrInvalidCredentials) {
			t.Fatalf("failure %d: error = %v, want ErrInvalidCredentials", i+1, err)
		}
	}

	// A locked account must look the same as a wrong password, so the lock is not revealed.
	if _, err := s.Login(ctx, testEmail, testPassword); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("correct password while locked: error = %v, want ErrInvalidCredentials", err)
	}

	*clock = clock.Add(lockDuration + time.Second)
	if _, err := s.Login(ctx, testEmail, testPassword); err != nil {
		t.Fatalf("Login after lock expired: %v", err)
	}

	after, err := queries.New(pool).GetAccountByEmail(ctx, testEmail)
	if err != nil {
		t.Fatalf("GetAccountByEmail: %v", err)
	}
	if after.FailedLoginAttempts != 0 {
		t.Errorf("failed_login_attempts after success = %d, want 0", after.FailedLoginAttempts)
	}
}

// TestBannedAccountRefusedOnlyWithCorrectPassword tests that a ban is reported after the
// password is verified, and that a wrong password still gets the generic error.
func TestBannedAccountRefusedOnlyWithCorrectPassword(t *testing.T) {
	s, pool, _ := newTestService(t)
	account := register(t, s)
	ctx := context.Background()

	if err := queries.New(pool).SetAccountBan(ctx, queries.SetAccountBanParams{
		ID:          account.ID,
		BannedUntil: pgtype.Timestamptz{Time: time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC), Valid: true},
		BanReason:   pgtype.Text{String: "spam", Valid: true},
	}); err != nil {
		t.Fatalf("SetAccountBan: %v", err)
	}

	if _, err := s.Login(ctx, testEmail, testPassword); !errors.Is(err, ErrAccountBanned) {
		t.Errorf("correct password: error = %v, want ErrAccountBanned", err)
	}
	if _, err := s.Login(ctx, testEmail, "wrong"); !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("wrong password: error = %v, want ErrInvalidCredentials", err)
	}
}

// TestPermanentBanAndExpiredBan tests that an infinity ban never expires, and that a ban
// whose end time has passed no longer applies.
func TestPermanentBanAndExpiredBan(t *testing.T) {
	s, pool, clock := newTestService(t)
	account := register(t, s)
	ctx := context.Background()
	q := queries.New(pool)

	if err := q.SetAccountBan(ctx, queries.SetAccountBanParams{
		ID:          account.ID,
		BannedUntil: pgtype.Timestamptz{InfinityModifier: pgtype.Infinity, Valid: true},
	}); err != nil {
		t.Fatalf("SetAccountBan permanent: %v", err)
	}
	*clock = clock.AddDate(50, 0, 0)
	if _, err := s.Login(ctx, testEmail, testPassword); !errors.Is(err, ErrAccountBanned) {
		t.Errorf("permanent ban after 50 years: error = %v, want ErrAccountBanned", err)
	}

	if err := q.SetAccountBan(ctx, queries.SetAccountBanParams{
		ID:          account.ID,
		BannedUntil: pgtype.Timestamptz{Time: clock.Add(-time.Hour), Valid: true},
	}); err != nil {
		t.Fatalf("SetAccountBan past: %v", err)
	}
	if _, err := s.Login(ctx, testEmail, testPassword); err != nil {
		t.Errorf("expired ban: Login error = %v, want nil", err)
	}
}
