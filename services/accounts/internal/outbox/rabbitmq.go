package outbox

import (
	"context"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

// RabbitMQPublisher publishes events to a topic exchange. It uses publisher confirms,
// so Publish returns only after RabbitMQ has accepted the message.
type RabbitMQPublisher struct {
	channel  *amqp.Channel
	exchange string
	confirms chan amqp.Confirmation
}

// NewRabbitMQPublisher declares a durable topic exchange and enables publisher confirms on a new channel.
func NewRabbitMQPublisher(conn *amqp.Connection, exchange string) (*RabbitMQPublisher, error) {
	channel, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("open channel: %w", err)
	}

	if err := channel.ExchangeDeclare(exchange, "topic", true, false, false, false, nil); err != nil {
		channel.Close()
		return nil, fmt.Errorf("declare exchange %q: %w", exchange, err)
	}

	if err := channel.Confirm(false); err != nil {
		channel.Close()
		return nil, fmt.Errorf("enable publisher confirms: %w", err)
	}

	return &RabbitMQPublisher{
		channel:  channel,
		exchange: exchange,
		confirms: channel.NotifyPublish(make(chan amqp.Confirmation, 1)),
	}, nil
}

// Publish sends body as a persistent message and waits for the broker to confirm it.
// Publish must not be called concurrently, because confirms are matched by order.
func (p *RabbitMQPublisher) Publish(ctx context.Context, routingKey string, body []byte) error {
	err := p.channel.PublishWithContext(ctx, p.exchange, routingKey, false, false, amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		Body:         body,
	})
	if err != nil {
		return fmt.Errorf("publish to %q: %w", routingKey, err)
	}

	select {
	case confirm := <-p.confirms:
		if !confirm.Ack {
			return fmt.Errorf("broker rejected message for %q", routingKey)
		}
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Close closes the channel. It does not close the connection, which the caller owns.
func (p *RabbitMQPublisher) Close() error {
	return p.channel.Close()
}
