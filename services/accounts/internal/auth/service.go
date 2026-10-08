package auth

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/nikita-simankov/upstore/services/accounts/internal/accounts"
	"github.com/nikita-simankov/upstore/services/accounts/internal/queries"
	"github.com/nikita-simankov/upstore/shared/events"
)

var (
	// ErrInvalidCredentials is returned for an unknown email, a wrong password, or an account
	// without a password. The same error is used for all three, so callers cannot tell them apart.
	ErrInvalidCredentials = errors.New("invalid email or password")
	// ErrAccountBanned is returned for a banned account, only after the password was verified.
	ErrAccountBanned = errors.New("account is banned")
	// ErrEmailTaken is returned when registering an email that already has an account.
	ErrEmailTaken = errors.New("email is already registered")
	// ErrAccountNotActive is returned for a correct password on an account whose email has
	// not been verified yet. It is only returned after the password is verified.
	ErrAccountNotActive = errors.New("account is not active; verify your email first")
)

const (
	// maxFailedAttempts is the number of consecutive wrong passwords that locks an account.
	maxFailedAttempts = 5
	// lockDuration is how long an account stays locked after too many failures.
	lockDuration = 15 * time.Minute
	// uniqueViolation is the PostgreSQL error code for a unique constraint violation.
	uniqueViolation = "23505"
)

// dummyHash is verified when an email is unknown, so that response time does not reveal
// whether the email has an account.
var dummyHash = mustHash("timing-equalizer-password")

func mustHash(password string) string {
	hash, err := HashPassword(password)
	if err != nil {
		panic(fmt.Sprintf("hash timing equalizer: %v", err))
	}
	return hash
}

// Service registers accounts and checks sign-in attempts.
type Service struct {
	pool *pgxpool.Pool
	now  func() time.Time
}

// NewService returns a Service that uses the system clock.
func NewService(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool, now: time.Now}
}

// Register creates an account with an email and password, and records account.created
// in the outbox. It returns ErrEmailTaken if the email is already registered.
func (s *Service) Register(ctx context.Context, email, password string, accountType events.AccountType) (queries.Account, error) {
	if err := ValidatePassword(password); err != nil {
		return queries.Account{}, err
	}

	hash, err := HashPassword(password)
	if err != nil {
		return queries.Account{}, err
	}

	account, err := accounts.RegisterWithPassword(ctx, s.pool, normalizeEmail(email), hash, accountType)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == uniqueViolation {
			return queries.Account{}, ErrEmailTaken
		}
		return queries.Account{}, err
	}
	return account, nil
}

// Login checks an email and password and returns the account on success.
//
// Every refusal that is not a ban or an inactive account returns ErrInvalidCredentials. That
// covers an unknown email, a missing password, a wrong password, and a locked account. Each
// of those does the same password work, so neither the error nor the response time shows
// whether an account exists or is locked.
//
// Failed attempts are counted. After maxFailedAttempts in a row, the account is locked for
// lockDuration. A banned account is refused only after the password is correct, so the ban is
// not revealed to someone who does not know the password.
func (s *Service) Login(ctx context.Context, email, password string) (queries.Account, error) {
	// Longer passwords cannot have been set through Register, so refuse them before any hashing.
	if len(password) > maxPasswordLength {
		return queries.Account{}, ErrInvalidCredentials
	}

	q := queries.New(s.pool)

	account, err := q.GetAccountByEmail(ctx, normalizeEmail(email))
	if errors.Is(err, pgx.ErrNoRows) {
		_, _ = VerifyPassword(password, dummyHash)
		return queries.Account{}, ErrInvalidCredentials
	}
	if err != nil {
		return queries.Account{}, fmt.Errorf("look up account: %w", err)
	}

	now := s.now()
	if !account.PasswordHash.Valid || isActiveUntil(account.LockedUntil, now) {
		_, _ = VerifyPassword(password, dummyHash)
		return queries.Account{}, ErrInvalidCredentials
	}

	ok, err := VerifyPassword(password, account.PasswordHash.String)
	if err != nil {
		return queries.Account{}, fmt.Errorf("verify password: %w", err)
	}
	if !ok {
		return queries.Account{}, s.recordFailure(ctx, q, account, now)
	}

	if isActiveUntil(account.BannedUntil, now) {
		return queries.Account{}, ErrAccountBanned
	}

	if account.Status != queries.AccountStatusActive {
		return queries.Account{}, ErrAccountNotActive
	}

	if err := q.RecordSuccessfulLogin(ctx, account.ID); err != nil {
		return queries.Account{}, fmt.Errorf("record successful login: %w", err)
	}
	return account, nil
}

// recordFailure counts a failed sign-in and returns ErrInvalidCredentials.
// The lock is applied by the database in the same statement as the increment, so the
// decision never uses a count that another request may already have changed.
// The log line carries the account ID only. It never includes the email or the password.
func (s *Service) recordFailure(ctx context.Context, q *queries.Queries, account queries.Account, now time.Time) error {
	updated, err := q.RecordFailedLogin(ctx, queries.RecordFailedLoginParams{
		ID:          account.ID,
		MaxAttempts: maxFailedAttempts,
		LockUntil:   pgtype.Timestamptz{Time: now.Add(lockDuration), Valid: true},
	})
	if err != nil {
		return fmt.Errorf("record failed login: %w", err)
	}

	slog.WarnContext(ctx, "sign-in failed",
		"account_id", formatUUID(account.ID),
		"failed_attempts", updated.FailedLoginAttempts,
		"locked", isActiveUntil(updated.LockedUntil, now),
	)
	return ErrInvalidCredentials
}

// formatUUID returns the canonical 8-4-4-4-12 form of id, for log lines.
func formatUUID(id pgtype.UUID) string {
	b := id.Bytes
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// isActiveUntil reports whether a timestamp is set and still in the future.
// A timestamp with the infinity modifier never expires.
func isActiveUntil(ts pgtype.Timestamptz, now time.Time) bool {
	if !ts.Valid {
		return false
	}
	if ts.InfinityModifier == pgtype.Infinity {
		return true
	}
	return ts.Time.After(now)
}

// normalizeEmail trims surrounding spaces. Comparisons in the database ignore case.
func normalizeEmail(email string) string {
	return strings.TrimSpace(email)
}
