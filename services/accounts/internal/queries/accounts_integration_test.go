package queries

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// testQueries connects to the database named by ACCOUNTS_TEST_DATABASE_URL.
// The test is skipped when the variable is unset, so it runs only where a database is available.
// The migrations in services/accounts/migrations must already be applied to that database.
func testQueries(t *testing.T) *Queries {
	t.Helper()

	url := os.Getenv("ACCOUNTS_TEST_DATABASE_URL")
	if url == "" {
		t.Skip("ACCOUNTS_TEST_DATABASE_URL is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(pool.Close)

	// Start each test from an empty accounts table.
	if _, err := pool.Exec(ctx, "TRUNCATE accounts, auth_identities, sessions"); err != nil {
		t.Fatalf("truncate: %v", err)
	}

	return New(pool)
}

// TestCreateAndGetAccount tests creating an account and reading it back by ID and by email.
func TestCreateAndGetAccount(t *testing.T) {
	q := testQueries(t)
	ctx := context.Background()

	created, err := q.CreateAccount(ctx, CreateAccountParams{Email: text("ivan@mail.by"), PasswordHash: text("hash")})
	if err != nil {
		t.Fatalf("CreateAccount: %v", err)
	}
	if created.Status != AccountStatusPending {
		t.Errorf("new account status = %q, want pending", created.Status)
	}

	byID, err := q.GetAccountByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetAccountByID: %v", err)
	}
	if byID.Email.String != "ivan@mail.by" {
		t.Errorf("GetAccountByID email = %q", byID.Email.String)
	}

	byEmail, err := q.GetAccountByEmail(ctx, "IVAN@Mail.BY")
	if err != nil {
		t.Fatalf("GetAccountByEmail with different case: %v", err)
	}
	if byEmail.ID != created.ID {
		t.Errorf("GetAccountByEmail returned a different account")
	}
}

// TestCreateAccountDuplicateEmail tests that emails differing only in case are rejected.
func TestCreateAccountDuplicateEmail(t *testing.T) {
	q := testQueries(t)
	ctx := context.Background()

	if _, err := q.CreateAccount(ctx, CreateAccountParams{Email: text("ivan@mail.by"), PasswordHash: text("hash")}); err != nil {
		t.Fatalf("first CreateAccount: %v", err)
	}
	if _, err := q.CreateAccount(ctx, CreateAccountParams{Email: text("Ivan@Mail.by"), PasswordHash: text("hash")}); err == nil {
		t.Error("second CreateAccount with case-variant email succeeded, want error")
	}
}

// TestMarkEmailVerified tests that verifying the email activates the account.
func TestMarkEmailVerified(t *testing.T) {
	q := testQueries(t)
	ctx := context.Background()

	created, err := q.CreateAccount(ctx, CreateAccountParams{Email: text("ivan@mail.by"), PasswordHash: text("hash")})
	if err != nil {
		t.Fatalf("CreateAccount: %v", err)
	}
	if err := q.MarkEmailVerified(ctx, created.ID); err != nil {
		t.Fatalf("MarkEmailVerified: %v", err)
	}

	got, err := q.GetAccountByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetAccountByID: %v", err)
	}
	if got.Status != AccountStatusActive {
		t.Errorf("status = %q, want active", got.Status)
	}
	if !got.EmailVerifiedAt.Valid {
		t.Error("email_verified_at is not set")
	}
}

// TestLoginCounters tests failed-login counting and the reset on successful login.
func TestLoginCounters(t *testing.T) {
	q := testQueries(t)
	ctx := context.Background()

	created, err := q.CreateAccount(ctx, CreateAccountParams{Email: text("ivan@mail.by"), PasswordHash: text("hash")})
	if err != nil {
		t.Fatalf("CreateAccount: %v", err)
	}

	for i := 0; i < 3; i++ {
		if _, err := q.RecordFailedLogin(ctx, RecordFailedLoginParams{
			ID:          created.ID,
			MaxAttempts: 5,
			LockUntil:   pgtype.Timestamptz{Time: time.Now().Add(time.Minute), Valid: true},
		}); err != nil {
			t.Fatalf("RecordFailedLogin: %v", err)
		}
	}
	got, err := q.GetAccountByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetAccountByID: %v", err)
	}
	if got.FailedLoginAttempts != 3 {
		t.Errorf("failed_login_attempts = %d, want 3", got.FailedLoginAttempts)
	}

	if err := q.RecordSuccessfulLogin(ctx, created.ID); err != nil {
		t.Fatalf("RecordSuccessfulLogin: %v", err)
	}
	got, err = q.GetAccountByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetAccountByID: %v", err)
	}
	if got.FailedLoginAttempts != 0 {
		t.Errorf("failed_login_attempts after success = %d, want 0", got.FailedLoginAttempts)
	}
	if !got.LastLoginAt.Valid {
		t.Error("last_login_at is not set after successful login")
	}
}

// text returns a non-null pgtype.Text for the test fixtures.
func text(s string) pgtype.Text {
	return pgtype.Text{String: s, Valid: true}
}

// TestSocialAccountWithoutEmail tests an account created through a provider, with no email or password.
func TestSocialAccountWithoutEmail(t *testing.T) {
	q := testQueries(t)
	ctx := context.Background()

	account, err := q.CreateAccount(ctx, CreateAccountParams{})
	if err != nil {
		t.Fatalf("CreateAccount without email or password: %v", err)
	}
	if _, err := q.CreateAuthIdentity(ctx, CreateAuthIdentityParams{
		AccountID:      account.ID,
		Provider:       AuthProviderTelegram,
		ProviderUserID: "12345",
	}); err != nil {
		t.Fatalf("CreateAuthIdentity: %v", err)
	}

	found, err := q.GetAccountByAuthIdentity(ctx, GetAccountByAuthIdentityParams{
		Provider:       AuthProviderTelegram,
		ProviderUserID: "12345",
	})
	if err != nil {
		t.Fatalf("GetAccountByAuthIdentity: %v", err)
	}
	if found.ID != account.ID {
		t.Errorf("identity resolved to a different account")
	}
}

// TestAuthIdentityUniqueness tests that one external identity cannot belong to two accounts,
// and that an account cannot hold two identities for the same provider.
func TestAuthIdentityUniqueness(t *testing.T) {
	q := testQueries(t)
	ctx := context.Background()

	first, err := q.CreateAccount(ctx, CreateAccountParams{})
	if err != nil {
		t.Fatalf("CreateAccount: %v", err)
	}
	second, err := q.CreateAccount(ctx, CreateAccountParams{})
	if err != nil {
		t.Fatalf("CreateAccount: %v", err)
	}

	if _, err := q.CreateAuthIdentity(ctx, CreateAuthIdentityParams{
		AccountID: first.ID, Provider: AuthProviderGoogle, ProviderUserID: "g-1",
	}); err != nil {
		t.Fatalf("first CreateAuthIdentity: %v", err)
	}

	if _, err := q.CreateAuthIdentity(ctx, CreateAuthIdentityParams{
		AccountID: second.ID, Provider: AuthProviderGoogle, ProviderUserID: "g-1",
	}); err == nil {
		t.Error("same Google identity linked to a second account, want error")
	}

	if _, err := q.CreateAuthIdentity(ctx, CreateAuthIdentityParams{
		AccountID: first.ID, Provider: AuthProviderGoogle, ProviderUserID: "g-2",
	}); err == nil {
		t.Error("second Google identity for the same account, want error")
	}
}

// TestSetAccountBan tests that a ban is stored with its end time and reason, and that status is unchanged.
func TestSetAccountBan(t *testing.T) {
	q := testQueries(t)
	ctx := context.Background()

	account, err := q.CreateAccount(ctx, CreateAccountParams{Email: text("ivan@mail.by"), PasswordHash: text("hash")})
	if err != nil {
		t.Fatalf("CreateAccount: %v", err)
	}

	until := time.Now().Add(24 * time.Hour).Truncate(time.Microsecond)
	if err := q.SetAccountBan(ctx, SetAccountBanParams{
		ID:          account.ID,
		BannedUntil: pgtype.Timestamptz{Time: until, Valid: true},
		BanReason:   text("spam"),
	}); err != nil {
		t.Fatalf("SetAccountBan: %v", err)
	}

	got, err := q.GetAccountByID(ctx, account.ID)
	if err != nil {
		t.Fatalf("GetAccountByID: %v", err)
	}
	if !got.BannedUntil.Valid {
		t.Fatal("banned_until is not set")
	}
	if d := got.BannedUntil.Time.Sub(until); d < -time.Second || d > time.Second {
		t.Errorf("banned_until = %v, want about %v", got.BannedUntil.Time, until)
	}
	if got.BanReason.String != "spam" {
		t.Errorf("ban_reason = %q, want spam", got.BanReason.String)
	}
	if got.Status != AccountStatusPending {
		t.Errorf("status changed to %q, want pending", got.Status)
	}
}
