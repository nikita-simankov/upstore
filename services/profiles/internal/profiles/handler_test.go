package profiles

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/nikita-simankov/upstore/services/profiles/internal/queries"
	"github.com/nikita-simankov/upstore/services/profiles/internal/testdb"
	"github.com/nikita-simankov/upstore/shared/events"
)

const testAccountID = "11111111-2222-3333-4444-555555555555"

// TestHandleAccountCreatedCreatesProfile tests that the event creates a profile with the
// account type and defaults for the rest.
func TestHandleAccountCreatedCreatesProfile(t *testing.T) {
	pool := testdb.Pool(t)
	ctx := context.Background()
	q := queries.New(pool)

	err := HandleAccountCreated(ctx, q, events.AccountCreated{
		AccountID:   testAccountID,
		AccountType: events.AccountTypeSeller,
	})
	if err != nil {
		t.Fatalf("HandleAccountCreated: %v", err)
	}

	var id pgtype.UUID
	if err := id.Scan(testAccountID); err != nil {
		t.Fatalf("scan id: %v", err)
	}
	got, err := q.GetProfileByAccountID(ctx, id)
	if err != nil {
		t.Fatalf("GetProfileByAccountID: %v", err)
	}
	if got.AccountType != queries.AccountTypeSeller {
		t.Errorf("account_type = %q, want seller", got.AccountType)
	}
	if got.Locale != "ru" {
		t.Errorf("locale = %q, want ru", got.Locale)
	}
	if got.FullName.Valid {
		t.Errorf("full_name = %q, want empty until the user sets it", got.FullName.String)
	}
}

// TestHandleAccountCreatedIsIdempotent tests that a redelivered event succeeds and keeps
// the original profile unchanged.
func TestHandleAccountCreatedIsIdempotent(t *testing.T) {
	pool := testdb.Pool(t)
	ctx := context.Background()
	q := queries.New(pool)

	first := events.AccountCreated{AccountID: testAccountID, AccountType: events.AccountTypeShopper}
	if err := HandleAccountCreated(ctx, q, first); err != nil {
		t.Fatalf("first delivery: %v", err)
	}

	// Redelivery, even with a different type, must not change the stored profile.
	redelivered := events.AccountCreated{AccountID: testAccountID, AccountType: events.AccountTypeSeller}
	if err := HandleAccountCreated(ctx, q, redelivered); err != nil {
		t.Fatalf("redelivery: %v", err)
	}

	var count int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM profiles").Scan(&count); err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 1 {
		t.Errorf("profiles = %d, want 1", count)
	}

	var id pgtype.UUID
	if err := id.Scan(testAccountID); err != nil {
		t.Fatalf("scan id: %v", err)
	}
	got, err := q.GetProfileByAccountID(ctx, id)
	if err != nil {
		t.Fatalf("GetProfileByAccountID: %v", err)
	}
	if got.AccountType != queries.AccountTypeShopper {
		t.Errorf("account_type = %q, want the original shopper", got.AccountType)
	}
}

// TestHandleAccountCreatedRejectsBadEvents tests that malformed events return an error
// and write nothing.
func TestHandleAccountCreatedRejectsBadEvents(t *testing.T) {
	pool := testdb.Pool(t)
	ctx := context.Background()
	q := queries.New(pool)

	tests := []struct {
		name  string
		event events.AccountCreated
	}{
		{"unknown account type", events.AccountCreated{AccountID: testAccountID, AccountType: "admin"}},
		{"malformed account id", events.AccountCreated{AccountID: "not-a-uuid", AccountType: events.AccountTypeSeller}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := HandleAccountCreated(ctx, q, tt.event); err == nil {
				t.Error("HandleAccountCreated returned nil error, want error")
			}
		})
	}

	var count int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM profiles").Scan(&count); err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 0 {
		t.Errorf("profiles = %d, want 0", count)
	}
}
