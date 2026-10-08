package outbox

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nikita-simankov/upstore/services/accounts/internal/queries"
	"github.com/nikita-simankov/upstore/services/accounts/internal/testdb"
)

// fakePublisher records the routing keys it accepts and fails on the call numbered failOn.
type fakePublisher struct {
	calls  int
	failOn int
	sent   []string
}

func (p *fakePublisher) Publish(_ context.Context, routingKey string, _ []byte) error {
	p.calls++
	if p.calls == p.failOn {
		return errors.New("broker unavailable")
	}
	p.sent = append(p.sent, routingKey)
	return nil
}

// TestRelayRunOnceStopsAtFirstFailure tests that a failed publish leaves the failed event
// and everything after it unpublished, and that a later pass publishes them in order.
func TestRelayRunOnceStopsAtFirstFailure(t *testing.T) {
	pool := testdb.Pool(t)
	ctx := context.Background()
	q := queries.New(pool)

	for _, eventType := range []string{"first", "second", "third"} {
		if _, err := q.InsertOutboxEvent(ctx, queries.InsertOutboxEventParams{
			AggregateID: pgtype.UUID{Valid: true},
			EventType:   eventType,
			Payload:     []byte(`{}`),
		}); err != nil {
			t.Fatalf("InsertOutboxEvent: %v", err)
		}
	}

	publisher := &fakePublisher{failOn: 2}
	relay := NewRelay(pool, publisher, 10)

	published, err := relay.RunOnce(ctx)
	if err == nil {
		t.Fatal("RunOnce returned nil error, want the publish failure")
	}
	if published != 1 {
		t.Errorf("published = %d, want 1", published)
	}

	pending := countPending(t, pool)
	if pending != 2 {
		t.Errorf("unpublished events = %d, want 2", pending)
	}

	publisher.failOn = 0
	published, err = relay.RunOnce(ctx)
	if err != nil {
		t.Fatalf("second RunOnce: %v", err)
	}
	if published != 2 {
		t.Errorf("second pass published = %d, want 2", published)
	}

	want := []string{"first", "second", "third"}
	if len(publisher.sent) != len(want) {
		t.Fatalf("sent = %v, want %v", publisher.sent, want)
	}
	for i := range want {
		if publisher.sent[i] != want[i] {
			t.Errorf("sent[%d] = %q, want %q", i, publisher.sent[i], want[i])
		}
	}

	if n := countPending(t, pool); n != 0 {
		t.Errorf("unpublished events after full drain = %d, want 0", n)
	}
}

// countPending returns the number of outbox rows that are not yet published.
func countPending(t *testing.T, pool *pgxpool.Pool) int {
	t.Helper()
	var n int
	if err := pool.QueryRow(context.Background(),
		"SELECT count(*) FROM outbox_events WHERE published_at IS NULL").Scan(&n); err != nil {
		t.Fatalf("count pending events: %v", err)
	}
	return n
}
