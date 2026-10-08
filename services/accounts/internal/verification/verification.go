// Package verification issues and consumes the one-time links that prove an account's email.
package verification

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/nikita-simankov/upstore/services/accounts/internal/mail"
	"github.com/nikita-simankov/upstore/services/accounts/internal/queries"
	"github.com/nikita-simankov/upstore/services/accounts/internal/tokens"
)

// ErrInvalidToken is returned for an unknown, used, or expired link. It does not say which.
var ErrInvalidToken = errors.New("invalid or expired verification link")

// linkTTL is how long a verification link stays valid.
const linkTTL = 24 * time.Hour

// Service issues links, sends them, and consumes them.
type Service struct {
	pool     *pgxpool.Pool
	mailer   mail.Mailer
	linkBase string
	now      func() time.Time
}

// NewService returns a Service. linkBase is the page that handles the link, for example
// "https://app.example/verify-email?token=". The token is appended to it.
func NewService(pool *pgxpool.Pool, mailer mail.Mailer, linkBase string) *Service {
	return &Service{pool: pool, mailer: mailer, linkBase: linkBase, now: time.Now}
}

// Issue creates a new link for a pending account and emails it to the given address.
// Earlier links for the account stop working, so only the newest one is valid.
//
// The link is emailed only after the transaction commits, so the recipient never receives
// a link that was not stored.
func (s *Service) Issue(ctx context.Context, account queries.Account, email string) error {
	token, hash, err := tokens.NewRefreshToken()
	if err != nil {
		return err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx) // no-op after a successful commit

	q := queries.New(tx)
	if err := q.InvalidateEmailVerifications(ctx, account.ID); err != nil {
		return fmt.Errorf("invalidate old links: %w", err)
	}
	if _, err := q.CreateEmailVerification(ctx, queries.CreateEmailVerificationParams{
		AccountID: account.ID,
		TokenHash: hash,
		ExpiresAt: pgtypeTime(s.now().Add(linkTTL)),
	}); err != nil {
		return fmt.Errorf("create link: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit: %w", err)
	}

	return s.mailer.SendVerification(ctx, email, s.linkBase+url.QueryEscape(token))
}

// Verify consumes a link and marks its account's email as verified. The link and the account
// update happen in one transaction, so a link is never used without the account being activated.
func (s *Service) Verify(ctx context.Context, token string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx) // no-op after a successful commit

	q := queries.New(tx)
	accountID, err := q.ConsumeEmailVerification(ctx, queries.ConsumeEmailVerificationParams{
		NowAt:     pgtypeTime(s.now()),
		TokenHash: tokens.HashRefreshToken(token),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrInvalidToken
	}
	if err != nil {
		return fmt.Errorf("consume link: %w", err)
	}

	if err := q.MarkEmailVerified(ctx, accountID); err != nil {
		return fmt.Errorf("mark verified: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

// Resend issues a new link for the account with this email, if it is still pending.
// It returns nil whether or not an account exists or needs a link, so callers can
// answer the same way for every email.
func (s *Service) Resend(ctx context.Context, email string) error {
	account, err := queries.New(s.pool).GetAccountByEmail(ctx, email)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("look up account: %w", err)
	}
	if account.Status != queries.AccountStatusPending {
		return nil
	}
	return s.Issue(ctx, account, email)
}

func pgtypeTime(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: true}
}
