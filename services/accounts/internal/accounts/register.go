// Package accounts holds the account operations that must write to the outbox
// in the same transaction as the change they describe.
package accounts

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nikita-simankov/upstore/services/accounts/internal/queries"
	"github.com/nikita-simankov/upstore/shared/events"
)

// RegisterWithPassword creates an account with an email and password hash, and records
// an account.created event in the outbox within the same transaction.
// If either write fails, neither is committed.
//
// The account type is not stored here. It is carried in the event for the profiles service.
func RegisterWithPassword(
	ctx context.Context,
	pool *pgxpool.Pool,
	email, passwordHash string,
	accountType events.AccountType,
) (queries.Account, error) {
	if !accountType.Valid() {
		return queries.Account{}, fmt.Errorf("invalid account type %q", accountType)
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return queries.Account{}, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx) // no-op after a successful commit

	q := queries.New(tx)

	account, err := q.CreateAccount(ctx, queries.CreateAccountParams{
		Email:        pgtype.Text{String: email, Valid: true},
		PasswordHash: pgtype.Text{String: passwordHash, Valid: true},
	})
	if err != nil {
		return queries.Account{}, fmt.Errorf("create account: %w", err)
	}

	if err := enqueueAccountCreated(ctx, q, account.ID, accountType); err != nil {
		return queries.Account{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return queries.Account{}, fmt.Errorf("commit: %w", err)
	}
	return account, nil
}

// enqueueAccountCreated writes the account.created event through q, which must be
// bound to the transaction that created the account.
func enqueueAccountCreated(ctx context.Context, q *queries.Queries, accountID pgtype.UUID, accountType events.AccountType) error {
	payload, err := json.Marshal(events.AccountCreated{
		AccountID:   uuidString(accountID),
		AccountType: accountType,
	})
	if err != nil {
		return fmt.Errorf("marshal event payload: %w", err)
	}

	if _, err := q.InsertOutboxEvent(ctx, queries.InsertOutboxEventParams{
		AggregateID: accountID,
		EventType:   events.RoutingKeyAccountCreated,
		Payload:     payload,
	}); err != nil {
		return fmt.Errorf("insert outbox event: %w", err)
	}
	return nil
}

// uuidString formats a pgtype.UUID in the canonical 8-4-4-4-12 form.
func uuidString(id pgtype.UUID) string {
	b := id.Bytes
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
