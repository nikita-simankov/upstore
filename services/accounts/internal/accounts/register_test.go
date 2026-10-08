package accounts

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nikita-simankov/upstore/services/accounts/internal/testdb"
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
// event that carries the account ID and no personal data.
func TestRegisterWithPasswordWritesEvent(t *testing.T) {
	pool := testdb.Pool(t)
	ctx := context.Background()

	account, err := RegisterWithPassword(ctx, pool, "ivan@mail.by", "hash")
	if err != nil {
		t.Fatalf("RegisterWithPassword: %v", err)
	}

	var eventType string
	var payload []byte
	if err := pool.QueryRow(ctx,
		"SELECT event_type, payload FROM outbox_events").Scan(&eventType, &payload); err != nil {
		t.Fatalf("read outbox event: %v", err)
	}
	if eventType != EventAccountCreated {
		t.Errorf("event_type = %q, want %q", eventType, EventAccountCreated)
	}

	var got map[string]any
	if err := json.Unmarshal(payload, &got); err != nil {
		t.Fatalf("payload is not JSON: %v", err)
	}
	if got["account_id"] != uuidString(account.ID) {
		t.Errorf("payload account_id = %v, want %s", got["account_id"], uuidString(account.ID))
	}
	if _, hasEmail := got["email"]; hasEmail {
		t.Error("payload contains email, which must not be published")
	}
}

// TestRegisterWithPasswordRollsBackEventOnFailure tests that a failed account insert
// leaves no event behind, so the outbox never describes an account that does not exist.
func TestRegisterWithPasswordRollsBackEventOnFailure(t *testing.T) {
	pool := testdb.Pool(t)
	ctx := context.Background()

	if _, err := RegisterWithPassword(ctx, pool, "ivan@mail.by", "hash"); err != nil {
		t.Fatalf("first RegisterWithPassword: %v", err)
	}

	// Same email in another case: the unique index rejects it, so the transaction must roll back.
	if _, err := RegisterWithPassword(ctx, pool, "Ivan@Mail.by", "hash"); err == nil {
		t.Fatal("duplicate registration succeeded, want error")
	}

	if n := countOutboxEvents(t, pool); n != 1 {
		t.Errorf("outbox events = %d, want 1 (failed registration must not add an event)", n)
	}
}
