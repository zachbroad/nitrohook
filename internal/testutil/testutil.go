package testutil

import (
	"context"
	"encoding/json"
	"os"
	"reflect"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/zachbroad/nitrohook/internal/database"
	"github.com/zachbroad/nitrohook/internal/store"
)

// JSONEqual reports whether two JSON documents are semantically equal, ignoring
// key order and insignificant whitespace. Use this instead of comparing raw
// bytes: payloads round-tripped through Postgres JSONB come back reordered and
// re-spaced (`{"a":1,"b":2}` → `{"b": 2, "a": 1}`), so byte-exact assertions are
// wrong. Fails the test if either argument is not valid JSON.
func JSONEqual(t *testing.T, a, b []byte) bool {
	t.Helper()
	var av, bv any
	if err := json.Unmarshal(a, &av); err != nil {
		t.Fatalf("JSONEqual: first argument is not valid JSON (%q): %v", a, err)
	}
	if err := json.Unmarshal(b, &bv); err != nil {
		t.Fatalf("JSONEqual: second argument is not valid JSON (%q): %v", b, err)
	}
	return reflect.DeepEqual(av, bv)
}

const (
	defaultTestDatabaseURL = "postgres://nitrohook:nitrohook@localhost:5432/nitrohook?sslmode=disable"
	defaultTestRedisURL    = "redis://localhost:6379"
)

// SetupTestDB connects to Postgres, runs migrations, and returns a Store.
// It registers a cleanup function that truncates all tables.
func SetupTestDB(t *testing.T) (*store.Store, *pgxpool.Pool) {
	t.Helper()

	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		dbURL = defaultTestDatabaseURL
	}

	ctx := context.Background()

	// Run migrations
	if err := database.Migrate(dbURL); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}

	pool, err := database.Connect(ctx, dbURL)
	if err != nil {
		t.Fatalf("failed to connect to test database: %v", err)
	}

	s := store.New(pool)

	t.Cleanup(func() {
		// Truncate tables in dependency order
		_, _ = pool.Exec(ctx, "TRUNCATE delivery_attempts, deliveries, actions, sources CASCADE")
		pool.Close()
	})

	// Truncate before test to ensure clean state
	_, err = pool.Exec(ctx, "TRUNCATE delivery_attempts, deliveries, actions, sources CASCADE")
	if err != nil {
		t.Fatalf("failed to truncate tables: %v", err)
	}

	return s, pool
}

// SetupTestRedis connects to Redis and returns a client.
// It registers a cleanup function that flushes the database.
func SetupTestRedis(t *testing.T) *redis.Client {
	t.Helper()

	redisURL := os.Getenv("TEST_REDIS_URL")
	if redisURL == "" {
		redisURL = defaultTestRedisURL
	}

	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		t.Fatalf("failed to parse redis URL: %v", err)
	}

	rdb := redis.NewClient(opts)
	ctx := context.Background()

	if err := rdb.Ping(ctx).Err(); err != nil {
		t.Fatalf("failed to connect to test redis: %v", err)
	}

	// Flush before test
	rdb.FlushDB(ctx)

	t.Cleanup(func() {
		rdb.FlushDB(ctx)
		rdb.Close()
	})

	return rdb
}
