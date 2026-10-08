package profiles

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/nikita-simankov/upstore/services/profiles/internal/queries"
	"github.com/nikita-simankov/upstore/shared/events"
)

// QueueAccountCreated is the durable queue that receives account.created events.
const QueueAccountCreated = "profiles.account-created"

// prefetchCount limits how many unacknowledged messages the broker sends at once.
const prefetchCount = 10

// Consume reads account.created events from QueueAccountCreated and creates profiles
// until ctx is cancelled. Each message is acknowledged only after it has been handled:
//   - a message that cannot be decoded or is invalid is rejected without requeue, so it
//     is dropped instead of being retried forever;
//   - a message that fails for another reason, such as a database error, is requeued.
func Consume(ctx context.Context, conn *amqp.Connection, pool *pgxpool.Pool) error {
	channel, err := conn.Channel()
	if err != nil {
		return fmt.Errorf("open channel: %w", err)
	}
	defer channel.Close()

	if err := channel.ExchangeDeclare(events.Exchange, "topic", true, false, false, false, nil); err != nil {
		return fmt.Errorf("declare exchange: %w", err)
	}
	queue, err := channel.QueueDeclare(QueueAccountCreated, true, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("declare queue: %w", err)
	}
	if err := channel.QueueBind(queue.Name, events.RoutingKeyAccountCreated, events.Exchange, false, nil); err != nil {
		return fmt.Errorf("bind queue: %w", err)
	}
	if err := channel.Qos(prefetchCount, 0, false); err != nil {
		return fmt.Errorf("set prefetch: %w", err)
	}

	deliveries, err := channel.Consume(queue.Name, "profiles", false, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("consume: %w", err)
	}

	q := queries.New(pool)
	for {
		select {
		case <-ctx.Done():
			return nil
		case delivery, ok := <-deliveries:
			if !ok {
				return errors.New("delivery channel closed")
			}
			handleDelivery(ctx, q, delivery)
		}
	}
}

// handleDelivery processes one message and acknowledges, rejects, or requeues it.
func handleDelivery(ctx context.Context, q *queries.Queries, delivery amqp.Delivery) {
	var event events.AccountCreated
	if err := json.Unmarshal(delivery.Body, &event); err != nil {
		log.Printf("profiles: dropping undecodable message: %v", err)
		settle(delivery.Reject(false))
		return
	}

	if err := HandleAccountCreated(ctx, q, event); err != nil {
		if errors.Is(err, ErrInvalidEvent) {
			log.Printf("profiles: dropping invalid message: %v", err)
			settle(delivery.Reject(false))
			return
		}
		log.Printf("profiles: requeueing message after error: %v", err)
		settle(delivery.Nack(false, true))
		return
	}

	settle(delivery.Ack(false))
}

// settle logs a failure to acknowledge. The broker redelivers unacknowledged messages
// when the channel closes, so there is nothing more to do here.
func settle(err error) {
	if err != nil {
		log.Printf("profiles: acknowledging message: %v", err)
	}
}
