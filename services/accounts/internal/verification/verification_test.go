package verification

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/nikita-simankov/upstore/services/accounts/internal/queries"
	"github.com/nikita-simankov/upstore/services/accounts/internal/testdb"
)

const linkBase = "https://app.test/verify-email?token="

// fakeMailer records every link it is asked to send.
type fakeMailer struct {
	to    string
	links []string
}

func (m *fakeMailer) SendVerification(_ context.Context, to, link string) error {
	m.to = to
	m.links = append(m.links, link)
	return nil
}

// tokenFrom returns the token part of the most recent link.
func (m *fakeMailer) tokenFrom(t *testing.T) string {
	t.Helper()
	if len(m.links) == 0 {
		t.Fatal("no link was sent")
	}
	return strings.TrimPrefix(m.links[len(m.links)-1], linkBase)
}

// newTest returns a Service on the test database, a fake mailer, and a clock the test can move.
func newTest(t *testing.T) (*Service, *pgxpool.Pool, *fakeMailer, *time.Time) {
	t.Helper()
	pool := testdb.Pool(t)
	mailer := &fakeMailer{}
	clock := time.Date(2026, 10, 8, 12, 0, 0, 0, time.UTC)
	s := NewService(pool, mailer, linkBase)
	s.now = func() time.Time { return clock }
	return s, pool, mailer, &clock
}

// pendingAccount creates an unverified account with the given email.
func pendingAccount(t *testing.T, pool *pgxpool.Pool, email string) queries.Account {
	t.Helper()
	account, err := queries.New(pool).CreateAccount(context.Background(), queries.CreateAccountParams{
		Email:        pgtype.Text{String: email, Valid: true},
		PasswordHash: pgtype.Text{String: "hash", Valid: true},
	})
	if err != nil {
		t.Fatalf("CreateAccount: %v", err)
	}
	return account
}

// TestIssueAndVerify tests the happy path: the emailed link activates the account, and the
// same link cannot be used twice.
func TestIssueAndVerify(t *testing.T) {
	s, pool, mailer, _ := newTest(t)
	ctx := context.Background()
	account := pendingAccount(t, pool, "ivan@mail.by")

	if err := s.Issue(ctx, account, "ivan@mail.by"); err != nil {
		t.Fatalf("Issue: %v", err)
	}
	if mailer.to != "ivan@mail.by" {
		t.Errorf("mail sent to %q", mailer.to)
	}
	token := mailer.tokenFrom(t)

	if err := s.Verify(ctx, token); err != nil {
		t.Fatalf("Verify: %v", err)
	}
	got, err := queries.New(pool).GetAccountByID(ctx, account.ID)
	if err != nil {
		t.Fatalf("GetAccountByID: %v", err)
	}
	if got.Status != queries.AccountStatusActive || !got.EmailVerifiedAt.Valid {
		t.Errorf("after verify: status = %q, email_verified_at valid = %v", got.Status, got.EmailVerifiedAt.Valid)
	}

	if err := s.Verify(ctx, token); !errors.Is(err, ErrInvalidToken) {
		t.Errorf("second use of the same link: error = %v, want ErrInvalidToken", err)
	}
}

// TestNewLinkReplacesOldLink tests that issuing a new link makes the earlier one useless.
func TestNewLinkReplacesOldLink(t *testing.T) {
	s, pool, mailer, _ := newTest(t)
	ctx := context.Background()
	account := pendingAccount(t, pool, "ivan@mail.by")

	if err := s.Issue(ctx, account, "ivan@mail.by"); err != nil {
		t.Fatalf("first Issue: %v", err)
	}
	first := mailer.tokenFrom(t)
	if err := s.Issue(ctx, account, "ivan@mail.by"); err != nil {
		t.Fatalf("second Issue: %v", err)
	}
	second := mailer.tokenFrom(t)

	if err := s.Verify(ctx, first); !errors.Is(err, ErrInvalidToken) {
		t.Errorf("old link: error = %v, want ErrInvalidToken", err)
	}
	if err := s.Verify(ctx, second); err != nil {
		t.Errorf("new link: %v", err)
	}
}

// TestExpiredLinkRejected tests that a link stops working after linkTTL.
func TestExpiredLinkRejected(t *testing.T) {
	s, pool, mailer, clock := newTest(t)
	ctx := context.Background()
	account := pendingAccount(t, pool, "ivan@mail.by")

	if err := s.Issue(ctx, account, "ivan@mail.by"); err != nil {
		t.Fatalf("Issue: %v", err)
	}
	token := mailer.tokenFrom(t)

	*clock = clock.Add(linkTTL + time.Minute)
	if err := s.Verify(ctx, token); !errors.Is(err, ErrInvalidToken) {
		t.Errorf("expired link: error = %v, want ErrInvalidToken", err)
	}
}

// TestUnknownLinkRejected tests that a token that was never issued is refused.
func TestUnknownLinkRejected(t *testing.T) {
	s, _, _, _ := newTest(t)
	if err := s.Verify(context.Background(), "made-up-token"); !errors.Is(err, ErrInvalidToken) {
		t.Errorf("unknown link: error = %v, want ErrInvalidToken", err)
	}
}

// TestResendOnlyForPendingAccounts tests that resend sends a link only to a pending account,
// and that it reports success for every address so callers cannot tell the cases apart.
func TestResendOnlyForPendingAccounts(t *testing.T) {
	s, pool, mailer, _ := newTest(t)
	ctx := context.Background()

	if err := s.Resend(ctx, "nobody@mail.by"); err != nil {
		t.Errorf("resend for unknown email: %v", err)
	}
	if len(mailer.links) != 0 {
		t.Error("a link was sent for an unknown email")
	}

	active := pendingAccount(t, pool, "active@mail.by")
	if err := queries.New(pool).MarkEmailVerified(ctx, active.ID); err != nil {
		t.Fatalf("MarkEmailVerified: %v", err)
	}
	if err := s.Resend(ctx, "active@mail.by"); err != nil {
		t.Errorf("resend for active account: %v", err)
	}
	if len(mailer.links) != 0 {
		t.Error("a link was sent to an active account")
	}

	pendingAccount(t, pool, "pending@mail.by")
	if err := s.Resend(ctx, "pending@mail.by"); err != nil {
		t.Errorf("resend for pending account: %v", err)
	}
	if len(mailer.links) != 1 {
		t.Errorf("links sent = %d, want 1 for the pending account", len(mailer.links))
	}
}
