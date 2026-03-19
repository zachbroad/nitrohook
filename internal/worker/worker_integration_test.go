//go:build integration

package worker_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/zachbroad/nitrohook/internal/model"
	"github.com/zachbroad/nitrohook/internal/testutil"
	"github.com/zachbroad/nitrohook/internal/worker"
)

func TestWorkerEndToEnd(t *testing.T) {
	s, _ := testutil.SetupTestDB(t)
	rdb := testutil.SetupTestRedis(t)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Set up httptest server to receive webhook
	var received atomic.Int32
	var receivedPayload []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPayload, _ = io.ReadAll(r.Body)
		received.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Seed source and action
	src, err := s.Sources.Create(ctx, "E2E Test", "e2e-test", "active", nil)
	if err != nil {
		t.Fatalf("create source: %v", err)
	}

	targetURL := server.URL
	_, err = s.Actions.Create(ctx, src.ID, model.ActionTypeWebhook, &targetURL, nil, nil, nil)
	if err != nil {
		t.Fatalf("create action: %v", err)
	}

	// Create delivery
	payload := json.RawMessage(`{"event":"push","ref":"main"}`)
	headers := json.RawMessage(`{"Content-Type":"application/json"}`)
	del, err := s.Deliveries.Create(ctx, src.ID, "e2e-idem-1", headers, payload)
	if err != nil {
		t.Fatalf("create delivery: %v", err)
	}

	// XADD to Redis
	err = rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: "deliveries",
		Values: map[string]any{"delivery_id": del.ID.String()},
	}).Err()
	if err != nil {
		t.Fatalf("xadd: %v", err)
	}

	// Start worker
	w := worker.New(s, rdb, 1, 5, 5*time.Second, 10*time.Second, 30*time.Second)
	if err := w.Start(ctx); err != nil {
		t.Fatalf("start worker: %v", err)
	}

	// Wait for delivery
	deadline := time.After(10 * time.Second)
	for received.Load() == 0 {
		select {
		case <-deadline:
			t.Fatal("timed out waiting for webhook delivery")
		default:
			time.Sleep(100 * time.Millisecond)
		}
	}

	if string(receivedPayload) != `{"event":"push","ref":"main"}` {
		t.Fatalf("unexpected payload: %q", string(receivedPayload))
	}

	// Verify delivery completed
	time.Sleep(500 * time.Millisecond)
	got, _ := s.Deliveries.GetByID(ctx, del.ID)
	if got.Status != model.DeliveryCompleted {
		t.Fatalf("expected delivery completed, got %s", got.Status)
	}
}

func TestWorkerRetryFlow(t *testing.T) {
	s, _ := testutil.SetupTestDB(t)
	rdb := testutil.SetupTestRedis(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var callCount atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := callCount.Add(1)
		if count <= 1 {
			// First attempt fails
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("error"))
		} else {
			// Retry succeeds
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	src, _ := s.Sources.Create(ctx, "Retry Test", "retry-test", "active", nil)
	targetURL := server.URL
	_, _ = s.Actions.Create(ctx, src.ID, model.ActionTypeWebhook, &targetURL, nil, nil, nil)

	del, _ := s.Deliveries.Create(ctx, src.ID, "retry-idem", json.RawMessage(`{}`), json.RawMessage(`{}`))

	rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: "deliveries",
		Values: map[string]any{"delivery_id": del.ID.String()},
	})

	// Start worker with short retry delay and poll interval for faster testing
	w := worker.New(s, rdb, 1, 5, 1*time.Second, 10*time.Second, 2*time.Second)
	w.Start(ctx)

	// Wait for initial delivery (will fail)
	deadline := time.After(10 * time.Second)
	for callCount.Load() == 0 {
		select {
		case <-deadline:
			t.Fatal("timed out waiting for initial delivery attempt")
		default:
			time.Sleep(100 * time.Millisecond)
		}
	}

	// Wait for retry
	deadline = time.After(20 * time.Second)
	for callCount.Load() < 2 {
		select {
		case <-deadline:
			t.Fatal("timed out waiting for retry attempt")
		default:
			time.Sleep(200 * time.Millisecond)
		}
	}

	// Verify attempt was created
	attempts, _ := s.Deliveries.ListAttemptsByDelivery(ctx, del.ID)
	if len(attempts) < 2 {
		t.Fatalf("expected at least 2 attempts, got %d", len(attempts))
	}
}

func TestWorkerRecordMode(t *testing.T) {
	s, _ := testutil.SetupTestDB(t)
	rdb := testutil.SetupTestRedis(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var received atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create source in record mode
	src, _ := s.Sources.Create(ctx, "Record Test", "record-test", "record", nil)
	targetURL := server.URL
	s.Actions.Create(ctx, src.ID, model.ActionTypeWebhook, &targetURL, nil, nil, nil)

	del, _ := s.Deliveries.Create(ctx, src.ID, "record-idem", json.RawMessage(`{}`), json.RawMessage(`{}`))

	rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: "deliveries",
		Values: map[string]any{"delivery_id": del.ID.String()},
	})

	w := worker.New(s, rdb, 1, 5, 5*time.Second, 10*time.Second, 30*time.Second)
	w.Start(ctx)

	// Wait a bit, webhook should NOT be called
	time.Sleep(3 * time.Second)

	if received.Load() != 0 {
		t.Fatal("expected no webhook call in record mode")
	}

	got, _ := s.Deliveries.GetByID(ctx, del.ID)
	if got.Status != model.DeliveryRecorded {
		t.Fatalf("expected delivery status 'recorded', got %s", got.Status)
	}
}
