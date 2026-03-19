//go:build integration

package store_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/zachbroad/nitrohook/internal/model"
	"github.com/zachbroad/nitrohook/internal/testutil"
)

func TestSourceCRUD(t *testing.T) {
	s, _ := testutil.SetupTestDB(t)
	ctx := context.Background()

	// Create
	src, err := s.Sources.Create(ctx, "Test Source", "test-source", "active", nil)
	if err != nil {
		t.Fatalf("create source: %v", err)
	}
	if src.Name != "Test Source" {
		t.Fatalf("expected name 'Test Source', got %q", src.Name)
	}
	if src.Slug != "test-source" {
		t.Fatalf("expected slug 'test-source', got %q", src.Slug)
	}
	if src.Mode != "active" {
		t.Fatalf("expected mode 'active', got %q", src.Mode)
	}

	// Get by ID
	got, err := s.Sources.GetByID(ctx, src.ID)
	if err != nil {
		t.Fatalf("get by ID: %v", err)
	}
	if got.ID != src.ID {
		t.Fatalf("expected ID %s, got %s", src.ID, got.ID)
	}

	// Get by slug
	got, err = s.Sources.GetBySlug(ctx, "test-source")
	if err != nil {
		t.Fatalf("get by slug: %v", err)
	}
	if got.ID != src.ID {
		t.Fatalf("expected ID %s, got %s", src.ID, got.ID)
	}

	// List
	sources, err := s.Sources.List(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(sources) != 1 {
		t.Fatalf("expected 1 source, got %d", len(sources))
	}

	// Update
	newName := "Updated Source"
	updated, err := s.Sources.Update(ctx, "test-source", &newName, nil, nil, false)
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updated.Name != "Updated Source" {
		t.Fatalf("expected updated name, got %q", updated.Name)
	}

	// Delete
	if err := s.Sources.Delete(ctx, "test-source"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	_, err = s.Sources.GetBySlug(ctx, "test-source")
	if err == nil {
		t.Fatal("expected error after delete")
	}
}

func TestActionCRUD(t *testing.T) {
	s, _ := testutil.SetupTestDB(t)
	ctx := context.Background()

	src, err := s.Sources.Create(ctx, "Action Test", "action-test", "active", nil)
	if err != nil {
		t.Fatalf("create source: %v", err)
	}

	targetURL := "https://example.com/hook"
	secret := "s3cret"
	config := json.RawMessage(`{"key":"value"}`)

	// Create
	action, err := s.Actions.Create(ctx, src.ID, model.ActionTypeWebhook, &targetURL, &secret, nil, config)
	if err != nil {
		t.Fatalf("create action: %v", err)
	}
	if *action.TargetURL != targetURL {
		t.Fatalf("expected target_url %q, got %q", targetURL, *action.TargetURL)
	}
	if !action.IsActive {
		t.Fatal("expected action to be active by default")
	}

	// Get by ID
	got, err := s.Actions.GetByID(ctx, action.ID)
	if err != nil {
		t.Fatalf("get by ID: %v", err)
	}
	if got.ID != action.ID {
		t.Fatalf("expected ID %s, got %s", action.ID, got.ID)
	}

	// List
	actions, err := s.Actions.List(ctx, src.ID)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(actions) != 1 {
		t.Fatalf("expected 1 action, got %d", len(actions))
	}

	// Toggle active
	isActive := false
	updated, err := s.Actions.Update(ctx, action.ID, nil, nil, &isActive, nil, nil)
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updated.IsActive {
		t.Fatal("expected action to be inactive after toggle")
	}

	// List active should be empty
	activeActions, err := s.Actions.ListActiveBySource(ctx, src.ID)
	if err != nil {
		t.Fatalf("list active: %v", err)
	}
	if len(activeActions) != 0 {
		t.Fatalf("expected 0 active actions, got %d", len(activeActions))
	}

	// Delete
	if err := s.Actions.Delete(ctx, action.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	_, err = s.Actions.GetByID(ctx, action.ID)
	if err == nil {
		t.Fatal("expected error after delete")
	}
}

func TestDeliveryLifecycle(t *testing.T) {
	s, _ := testutil.SetupTestDB(t)
	ctx := context.Background()

	src, _ := s.Sources.Create(ctx, "Delivery Test", "delivery-test", "active", nil)

	headers := json.RawMessage(`{"Content-Type":"application/json"}`)
	payload := json.RawMessage(`{"event":"push"}`)

	// Create delivery
	del, err := s.Deliveries.Create(ctx, src.ID, "idem-key-1", headers, payload)
	if err != nil {
		t.Fatalf("create delivery: %v", err)
	}
	if del.Status != model.DeliveryPending {
		t.Fatalf("expected status pending, got %s", del.Status)
	}
	if del.IdempotencyKey != "idem-key-1" {
		t.Fatalf("expected idempotency_key 'idem-key-1', got %q", del.IdempotencyKey)
	}

	// Get by ID
	got, err := s.Deliveries.GetByID(ctx, del.ID)
	if err != nil {
		t.Fatalf("get by ID: %v", err)
	}
	if string(got.Payload) != `{"event":"push"}` {
		t.Fatalf("expected payload preserved, got %q", string(got.Payload))
	}

	// Update status
	if err := s.Deliveries.UpdateStatus(ctx, del.ID, model.DeliveryProcessing); err != nil {
		t.Fatalf("update status: %v", err)
	}
	got, _ = s.Deliveries.GetByID(ctx, del.ID)
	if got.Status != model.DeliveryProcessing {
		t.Fatalf("expected status processing, got %s", got.Status)
	}

	// List pending (should be empty since status is processing)
	pending, err := s.Deliveries.ListPending(ctx, 100)
	if err != nil {
		t.Fatalf("list pending: %v", err)
	}
	if len(pending) != 0 {
		t.Fatalf("expected 0 pending, got %d", len(pending))
	}

	// Idempotency: same key should fail with unique violation
	_, err = s.Deliveries.Create(ctx, src.ID, "idem-key-1", headers, payload)
	if err == nil {
		t.Fatal("expected error for duplicate idempotency key")
	}
}

func TestAttemptTracking(t *testing.T) {
	s, _ := testutil.SetupTestDB(t)
	ctx := context.Background()

	src, _ := s.Sources.Create(ctx, "Attempt Test", "attempt-test", "active", nil)
	targetURL := "https://example.com"
	action, _ := s.Actions.Create(ctx, src.ID, model.ActionTypeWebhook, &targetURL, nil, nil, nil)
	del, _ := s.Deliveries.Create(ctx, src.ID, "attempt-idem", json.RawMessage(`{}`), json.RawMessage(`{}`))

	// Create attempt
	attempt, err := s.Deliveries.CreateAttempt(ctx, del.ID, action.ID, 1)
	if err != nil {
		t.Fatalf("create attempt: %v", err)
	}
	if attempt.AttemptNumber != 1 {
		t.Fatalf("expected attempt_number 1, got %d", attempt.AttemptNumber)
	}
	if attempt.Status != model.AttemptPending {
		t.Fatalf("expected status pending, got %s", attempt.Status)
	}

	// Update attempt
	status := 500
	errMsg := "server error"
	retryAt := time.Now().Add(10 * time.Second)
	if err := s.Deliveries.UpdateAttempt(ctx, attempt.ID, model.AttemptFailed, &status, nil, &errMsg, &retryAt); err != nil {
		t.Fatalf("update attempt: %v", err)
	}

	// List retryable
	retryable, err := s.Deliveries.ListRetryableAttempts(ctx, 100)
	if err != nil {
		t.Fatalf("list retryable: %v", err)
	}
	// Should be empty since next_retry_at is in the future
	if len(retryable) != 0 {
		t.Fatalf("expected 0 retryable (future retry time), got %d", len(retryable))
	}

	// Update to past retry time
	pastRetry := time.Now().Add(-1 * time.Second)
	s.Deliveries.UpdateAttempt(ctx, attempt.ID, model.AttemptFailed, &status, nil, &errMsg, &pastRetry)

	retryable, _ = s.Deliveries.ListRetryableAttempts(ctx, 100)
	if len(retryable) != 1 {
		t.Fatalf("expected 1 retryable, got %d", len(retryable))
	}

	// List by delivery
	attempts, err := s.Deliveries.ListAttemptsByDelivery(ctx, del.ID)
	if err != nil {
		t.Fatalf("list by delivery: %v", err)
	}
	if len(attempts) != 1 {
		t.Fatalf("expected 1 attempt, got %d", len(attempts))
	}

	// Get max attempt number
	maxNum, err := s.Deliveries.GetMaxAttemptNumber(ctx, del.ID, action.ID)
	if err != nil {
		t.Fatalf("get max attempt number: %v", err)
	}
	if maxNum != 1 {
		t.Fatalf("expected max attempt 1, got %d", maxNum)
	}

	// No attempts for random action
	maxNum, err = s.Deliveries.GetMaxAttemptNumber(ctx, del.ID, uuid.New())
	if err != nil {
		t.Fatalf("get max attempt number: %v", err)
	}
	if maxNum != 0 {
		t.Fatalf("expected max attempt 0 for unknown action, got %d", maxNum)
	}
}
