package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/nikita-simankov/upstore/services/accounts/internal/queries"
	"github.com/nikita-simankov/upstore/services/accounts/internal/tokens"
)

// ErrInvalidRefreshToken is returned for an unknown, expired, revoked, or reused refresh token.
// The same error is used for all of them, so callers cannot tell the cases apart.
var ErrInvalidRefreshToken = errors.New("invalid or expired refresh token")

// refreshTokenTTL is how long a session lasts without a refresh.
const refreshTokenTTL = 30 * 24 * time.Hour

// Tokens is what a client receives after signing in or refreshing.
type Tokens struct {
	AccessToken  string
	RefreshToken string
	// ExpiresIn is the access token lifetime in seconds.
	ExpiresIn int64
}

// Sessions issues, rotates, and revokes the tokens for a sign-in.
type Sessions struct {
	pool   *pgxpool.Pool
	issuer *tokens.Issuer
	now    func() time.Time
}

// NewSessions returns a Sessions that uses the system clock.
func NewSessions(pool *pgxpool.Pool, issuer *tokens.Issuer) *Sessions {
	return &Sessions{pool: pool, issuer: issuer, now: time.Now}
}

// Issue starts a new session for an account and returns its tokens.
func (s *Sessions) Issue(ctx context.Context, account queries.Account) (Tokens, error) {
	return s.issue(ctx, queries.New(s.pool), account)
}

// issue creates the session row and signs its access token. q may be bound to a transaction.
func (s *Sessions) issue(ctx context.Context, q *queries.Queries, account queries.Account) (Tokens, error) {
	refresh, hash, err := tokens.NewRefreshToken()
	if err != nil {
		return Tokens{}, err
	}

	session, err := q.CreateSession(ctx, queries.CreateSessionParams{
		AccountID:        account.ID,
		RefreshTokenHash: hash,
		ExpiresAt:        pgtype.Timestamptz{Time: s.now().Add(refreshTokenTTL), Valid: true},
	})
	if err != nil {
		return Tokens{}, fmt.Errorf("create session: %w", err)
	}

	access, err := s.issuer.IssueAccess(formatUUID(account.ID), formatUUID(session.ID))
	if err != nil {
		return Tokens{}, err
	}
	return Tokens{
		AccessToken:  access,
		RefreshToken: refresh,
		ExpiresIn:    int64(tokens.AccessTokenTTL.Seconds()),
	}, nil
}

// Refresh exchanges a refresh token for a new pair. The old refresh token stops working.
//
// If a token that was already used is presented again, every session of that account is
// revoked. That is the usual response to a stolen token: whoever holds the old token is
// stopped, and the real user has to sign in again.
//
// The account is checked again here, so a ban or a deactivation takes effect at the next refresh.
func (s *Sessions) Refresh(ctx context.Context, refreshToken string) (Tokens, error) {
	q := queries.New(s.pool)

	session, err := q.GetSessionByRefreshHash(ctx, tokens.HashRefreshToken(refreshToken))
	if errors.Is(err, pgx.ErrNoRows) {
		return Tokens{}, ErrInvalidRefreshToken
	}
	if err != nil {
		return Tokens{}, fmt.Errorf("look up session: %w", err)
	}

	if session.RevokedAt.Valid {
		if err := q.RevokeAccountSessions(ctx, session.AccountID); err != nil {
			return Tokens{}, fmt.Errorf("revoke sessions after reuse: %w", err)
		}
		return Tokens{}, ErrInvalidRefreshToken
	}

	now := s.now()
	if !session.ExpiresAt.Valid || !session.ExpiresAt.Time.After(now) {
		return Tokens{}, ErrInvalidRefreshToken
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Tokens{}, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx) // no-op after a successful commit

	qtx := q.WithTx(tx)

	account, err := qtx.GetAccountByID(ctx, session.AccountID)
	if err != nil {
		return Tokens{}, fmt.Errorf("load account: %w", err)
	}
	if isActiveUntil(account.BannedUntil, now) {
		return Tokens{}, ErrAccountBanned
	}
	if account.Status != queries.AccountStatusActive {
		return Tokens{}, ErrAccountNotActive
	}

	// Revoking is conditional on the row still being active. If another request revoked it
	// first, this one lost a race with the same token, which counts as reuse.
	changed, err := qtx.RevokeSession(ctx, session.ID)
	if err != nil {
		return Tokens{}, fmt.Errorf("revoke old session: %w", err)
	}
	if changed == 0 {
		if err := q.RevokeAccountSessions(ctx, session.AccountID); err != nil {
			return Tokens{}, fmt.Errorf("revoke sessions after reuse: %w", err)
		}
		return Tokens{}, ErrInvalidRefreshToken
	}

	issued, err := s.issue(ctx, qtx, account)
	if err != nil {
		return Tokens{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Tokens{}, fmt.Errorf("commit: %w", err)
	}
	return issued, nil
}

// Logout revokes the session behind a refresh token. It returns nil whether or not the token
// was valid, so the response does not reveal anything about the token.
func (s *Sessions) Logout(ctx context.Context, refreshToken string) error {
	q := queries.New(s.pool)

	session, err := q.GetSessionByRefreshHash(ctx, tokens.HashRefreshToken(refreshToken))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("look up session: %w", err)
	}
	if _, err := q.RevokeSession(ctx, session.ID); err != nil {
		return fmt.Errorf("revoke session: %w", err)
	}
	return nil
}
