// Package testdb provides a database pool for profiles integration tests.
//
// Tests that use it run only when PROFILES_TEST_DATABASE_URL is set, and skip otherwise.
// The database must already have the migrations in services/profiles/migrations applied.
// Pool empties the profiles table, so never point this variable at a real database.
package testdb

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Pool returns a pool connected to the test database with the profiles table emptied.
func Pool(t testing.TB) *pgxpool.Pool {
	t.Helper()

	url := os.Getenv("PROFILES_TEST_DATABASE_URL")
	if url == "" {
		t.Skip("PROFILES_TEST_DATABASE_URL is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(pool.Close)

	if _, err := pool.Exec(ctx, "TRUNCATE profiles"); err != nil {
		t.Fatalf("truncate: %v", err)
	}
	return pool
}
