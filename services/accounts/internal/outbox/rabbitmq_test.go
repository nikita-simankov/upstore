package outbox

import (
	"context"
	"os"
	"testing"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

// TestRabbitMQPublisherDeliversMessage tests that a published message is confirmed and
// reaches a queue bound to its routing key.
// It runs only when ACCOUNTS_TEST_AMQP_URL is set, for example amqp://guest:guest@127.0.0.1:5672/.
func TestRabbitMQPublisherDeliversMessage(t *testing.T) {
	url := os.Getenv("ACCOUNTS_TEST_AMQP_URL")
	if url == "" {
		t.Skip("ACCOUNTS_TEST_AMQP_URL is not set")
	}

	conn, err := amqp.Dial(url)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	const exchange = "upstore.test.events"
	publisher, err := NewRabbitMQPublisher(conn, exchange)
	if err != nil {
		t.Fatalf("NewRabbitMQPublisher: %v", err)
	}
	defer publisher.Close()

	consumer, err := conn.Channel()
	if err != nil {
		t.Fatalf("consumer channel: %v", err)
	}
	defer consumer.Close()
	defer consumer.ExchangeDelete(exchange, false, false)

	queue, err := consumer.QueueDeclare("", false, true, true, false, nil)
	if err != nil {
		t.Fatalf("declare queue: %v", err)
	}
	if err := consumer.QueueBind(queue.Name, "account.created", exchange, false, nil); err != nil {
		t.Fatalf("bind queue: %v", err)
	}
	deliveries, err := consumer.Consume(queue.Name, "", true, true, false, false, nil)
	if err != nil {
		t.Fatalf("consume: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	body := []byte(`{"account_id":"test"}`)
	if err := publisher.Publish(ctx, "account.created", body); err != nil {
		t.Fatalf("Publish: %v", err)
	}

	select {
	case delivery := <-deliveries:
		if string(delivery.Body) != string(body) {
			t.Errorf("body = %s, want %s", delivery.Body, body)
		}
		if delivery.RoutingKey != "account.created" {
			t.Errorf("routing key = %q, want account.created", delivery.RoutingKey)
		}
	case <-ctx.Done():
		t.Fatal("message did not arrive in the bound queue")
	}
}
