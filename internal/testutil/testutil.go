package testutil

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/zachbroad/nitrohook/internal/database"
	"github.com/zachbroad/nitrohook/internal/store"
)

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
