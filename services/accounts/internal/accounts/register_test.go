package accounts

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nikita-simankov/upstore/services/accounts/internal/testdb"
	"github.com/nikita-simankov/upstore/shared/events"
)

// countOutboxEvents returns the number of rows in outbox_events.
func countOutboxEvents(t *testing.T, pool *pgxpool.Pool) int {
	t.Helper()
	var n int
	if err := pool.QueryRow(context.Background(), "SELECT count(*) FROM outbox_events").Scan(&n); err != nil {
		t.Fatalf("count outbox events: %v", err)
	}
	return n
}

// TestRegisterWithPasswordWritesEvent tests that registration records an account.created
// event with the account ID and type, and no personal data.
func TestRegisterWithPasswordWritesEvent(t *testing.T) {
	pool := testdb.Pool(t)
	ctx := context.Background()

	account, err := RegisterWithPassword(ctx, pool, "ivan@mail.by", "hash", events.AccountTypeSeller)
	if err != nil {
		t.Fatalf("RegisterWithPassword: %v", err)
	}

	var eventType string
	var payload []byte
	if err := pool.QueryRow(ctx,
		"SELECT event_type, payload FROM outbox_events").Scan(&eventType, &payload); err != nil {
		t.Fatalf("read outbox event: %v", err)
	}
	if eventType != events.RoutingKeyAccountCreated {
		t.Errorf("event_type = %q, want %q", eventType, events.RoutingKeyAccountCreated)
	}

	var got events.AccountCreated
	if err := json.Unmarshal(payload, &got); err != nil {
		t.Fatalf("payload is not JSON: %v", err)
	}
	if got.AccountID != uuidString(account.ID) {
		t.Errorf("payload account_id = %s, want %s", got.AccountID, uuidString(account.ID))
	}
	if got.AccountType != events.AccountTypeSeller {
		t.Errorf("payload account_type = %q, want seller", got.AccountType)
	}

	var raw map[string]any
	if err := json.Unmarshal(payload, &raw); err != nil {
		t.Fatalf("payload is not a JSON object: %v", err)
	}
	if _, hasEmail := raw["email"]; hasEmail {
		t.Error("payload contains email, which must not be published")
	}
}

// TestRegisterWithPasswordRejectsInvalidType tests that an unknown account type is refused
// before anything is written.
func TestRegisterWithPasswordRejectsInvalidType(t *testing.T) {
	pool := testdb.Pool(t)
	ctx := context.Background()

	if _, err := RegisterWithPassword(ctx, pool, "ivan@mail.by", "hash", "admin"); err == nil {
		t.Fatal("RegisterWithPassword accepted an invalid account type")
	}
	if n := countOutboxEvents(t, pool); n != 0 {
		t.Errorf("outbox events = %d, want 0", n)
	}
}

// TestRegisterWithPasswordRollsBackEventOnFailure tests that a failed account insert
// leaves no event behind, so the outbox never describes an account that does not exist.
func TestRegisterWithPasswordRollsBackEventOnFailure(t *testing.T) {
	pool := testdb.Pool(t)
	ctx := context.Background()

	if _, err := RegisterWithPassword(ctx, pool, "ivan@mail.by", "hash", events.AccountTypeShopper); err != nil {
		t.Fatalf("first RegisterWithPassword: %v", err)
	}

	// Same email in another case: the unique index rejects it, so the transaction must roll back.
	if _, err := RegisterWithPassword(ctx, pool, "Ivan@Mail.by", "hash", events.AccountTypeShopper); err == nil {
		t.Fatal("duplicate registration succeeded, want error")
	}

	if n := countOutboxEvents(t, pool); n != 1 {
		t.Errorf("outbox events = %d, want 1 (failed registration must not add an event)", n)
	}
}
