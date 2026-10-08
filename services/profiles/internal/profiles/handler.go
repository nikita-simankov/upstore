// Package profiles creates and reads user profiles from events published by other services.
package profiles

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/nikita-simankov/upstore/services/profiles/internal/queries"
	"github.com/nikita-simankov/upstore/shared/events"
)

// HandleAccountCreated creates the profile for a newly registered account.
//
// Events are delivered at least once, so the same event can arrive twice. Creating the
// profile is idempotent, and a redelivered event leaves the existing profile unchanged.
// An event with an unknown account type or a malformed account ID is rejected with an error.
func HandleAccountCreated(ctx context.Context, q *queries.Queries, event events.AccountCreated) error {
	if !event.AccountType.Valid() {
		return fmt.Errorf("account.created: invalid account type %q", event.AccountType)
	}

	var accountID pgtype.UUID
	if err := accountID.Scan(event.AccountID); err != nil {
		return fmt.Errorf("account.created: invalid account_id %q: %w", event.AccountID, err)
	}

	if err := q.CreateProfile(ctx, queries.CreateProfileParams{
		AccountID:   accountID,
		AccountType: queries.AccountType(event.AccountType),
	}); err != nil {
		return fmt.Errorf("account.created: create profile: %w", err)
	}
	return nil
}
