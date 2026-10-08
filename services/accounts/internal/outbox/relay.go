// Package outbox moves events from the outbox_events table to the message broker.
//
// Delivery is at least once: an event can be published twice if the process stops
// after the broker accepts it but before the row is marked published. Consumers
// must handle duplicates.
package outbox

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nikita-simankov/upstore/services/accounts/internal/queries"
)

// Publisher sends one event to the message broker.
// It must return nil only after the broker has accepted the message.
type Publisher interface {
	Publish(ctx context.Context, routingKey string, body []byte) error
}

// Relay reads unpublished events from the outbox table and publishes them in insertion order.
type Relay struct {
	pool      *pgxpool.Pool
	publisher Publisher
	batchSize int32
}

// NewRelay creates a Relay that publishes up to batchSize events per pass.
func NewRelay(pool *pgxpool.Pool, publisher Publisher, batchSize int32) *Relay {
	return &Relay{pool: pool, publisher: publisher, batchSize: batchSize}
}

// RunOnce publishes one batch of unpublished events and returns how many were published.
//
// Each event is marked published only after the publisher succeeds. On the first
// publish error, RunOnce stops, commits the events already marked, and returns the
// error. The remaining events stay unpublished and are retried on the next call.
func (r *Relay) RunOnce(ctx context.Context) (int, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx) // no-op after a successful commit

	q := queries.New(tx)
	events, err := q.ListUnpublishedOutboxEvents(ctx, r.batchSize)
	if err != nil {
		return 0, fmt.Errorf("list unpublished events: %w", err)
	}

	published := 0
	var publishErr error
	for _, event := range events {
		if err := r.publisher.Publish(ctx, event.EventType, event.Payload); err != nil {
			publishErr = fmt.Errorf("publish event %d: %w", event.ID, err)
			break
		}
		if err := q.MarkOutboxEventPublished(ctx, event.ID); err != nil {
			return published, fmt.Errorf("mark event %d published: %w", event.ID, err)
		}
		published++
	}

	if err := tx.Commit(ctx); err != nil {
		return published, fmt.Errorf("commit: %w", err)
	}
	return published, publishErr
}

// Run calls RunOnce every interval until ctx is cancelled. When a pass is full,
// it runs again immediately so a backlog drains quickly.
func (r *Relay) Run(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			for {
				n, err := r.RunOnce(ctx)
				if err != nil {
					log.Printf("outbox relay: %v", err)
					break
				}
				if int32(n) < r.batchSize {
					break
				}
			}
		}
	}
}
