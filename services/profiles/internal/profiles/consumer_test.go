package profiles

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/nikita-simankov/upstore/services/profiles/internal/testdb"
	"github.com/nikita-simankov/upstore/shared/events"
)

// profileExists reports whether a profile row exists for accountID.
func profileExists(t *testing.T, pool *pgxpool.Pool, accountID string) bool {
	t.Helper()
	var n int
	if err := pool.QueryRow(context.Background(),
		"SELECT count(*) FROM profiles WHERE account_id = $1::uuid", accountID).Scan(&n); err != nil {
		t.Fatalf("count profiles: %v", err)
	}
	return n > 0
}

// publish sends body to the events exchange with the account.created routing key.
func publish(t *testing.T, channel *amqp.Channel, body []byte) {
	t.Helper()
	if err := channel.PublishWithContext(context.Background(), events.Exchange,
		events.RoutingKeyAccountCreated, false, false, amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		}); err != nil {
		t.Fatalf("publish: %v", err)
	}
}

// TestConsumeCreatesProfilesAndSurvivesBadMessages runs the consumer against a real broker
// and database. It checks that a valid event creates a profile, that an invalid event is
// dropped without stopping the consumer, and that a later valid event is still handled.
//
// It needs PROFILES_TEST_DATABASE_URL and PROFILES_TEST_AMQP_URL, and skips without them.
func TestConsumeCreatesProfilesAndSurvivesBadMessages(t *testing.T) {
	amqpURL := os.Getenv("PROFILES_TEST_AMQP_URL")
	if amqpURL == "" {
		t.Skip("PROFILES_TEST_AMQP_URL is not set")
	}
	pool := testdb.Pool(t)

	conn, err := amqp.Dial(amqpURL)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	channel, err := conn.Channel()
	if err != nil {
		t.Fatalf("channel: %v", err)
	}
	defer channel.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	consumerDone := make(chan error, 1)
	go func() {
		consumerDone <- Consume(ctx, conn, pool)
	}()

	const readyID = "aaaaaaaa-0000-0000-0000-000000000001"
	const afterBadID = "aaaaaaaa-0000-0000-0000-000000000002"
	validBody := func(id string) []byte {
		b, _ := json.Marshal(events.AccountCreated{AccountID: id, AccountType: events.AccountTypeShopper})
		return b
	}

	// The queue is declared by the consumer, so publish until a profile appears. This shows
	// the consumer is ready before the messages that matter are sent.
	deadline := time.Now().Add(15 * time.Second)
	for !profileExists(t, pool, readyID) {
		if time.Now().After(deadline) {
			t.Fatal("consumer did not create a profile in time")
		}
		publish(t, channel, validBody(readyID))
		time.Sleep(200 * time.Millisecond)
	}

	publish(t, channel, []byte(`{"account_id":"aaaaaaaa-0000-0000-0000-000000000003","account_type":"admin"}`))
	publish(t, channel, validBody(afterBadID))

	deadline = time.Now().Add(15 * time.Second)
	for !profileExists(t, pool, afterBadID) {
		if time.Now().After(deadline) {
			t.Fatal("consumer stopped handling messages after an invalid one")
		}
		time.Sleep(100 * time.Millisecond)
	}

	select {
	case err := <-consumerDone:
		t.Fatalf("consumer exited early: %v", err)
	default:
	}
}
