//go:build integration

package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/zachbroad/nitrohook/internal/handler"
	"github.com/zachbroad/nitrohook/internal/model"
	"github.com/zachbroad/nitrohook/internal/testutil"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func setupRouter(t *testing.T) (*gin.Engine, func()) {
	t.Helper()
	s, _ := testutil.SetupTestDB(t)
	rdb := testutil.SetupTestRedis(t)

	webhookH := handler.NewWebhookHandler(s, rdb)
	sourceH := handler.NewSourceHandler(s)
	actionH := handler.NewActionHandler(s)
	deliveryH := handler.NewDeliveryHandler(s)

	r := gin.New()
	r.POST("/webhooks/:sourceSlug", webhookH.Ingest)

	api := r.Group("/api")
	sources := api.Group("/sources")
	sources.GET("", sourceH.List)
	sources.POST("", sourceH.Create)
	srcGroup := sources.Group("/:sourceSlug")
	srcGroup.GET("", sourceH.Get)
	srcGroup.PATCH("", sourceH.Update)
	srcGroup.DELETE("", sourceH.Delete)
	actions := srcGroup.Group("/actions")
	actions.POST("", actionH.Create)
	actions.GET("", actionH.List)
	actions.GET("/:id", actionH.Get)
	actions.PATCH("/:id", actionH.Update)
	actions.DELETE("/:id", actionH.Delete)
	deliveries := api.Group("/deliveries")
	deliveries.GET("", deliveryH.List)
	deliveries.GET("/:id", deliveryH.Get)
	deliveries.GET("/:id/attempts", deliveryH.ListAttempts)

	return r, func() {}
}

func createSource(t *testing.T, r *gin.Engine, name, slug string) {
	t.Helper()
	body, _ := json.Marshal(map[string]string{"name": name, "slug": slug, "mode": "active"})
	req := httptest.NewRequest(http.MethodPost, "/api/sources", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create source: expected 201, got %d: %s", w.Code, w.Body.String())
	}
}

func TestWebhookIngest(t *testing.T) {
	r, cleanup := setupRouter(t)
	defer cleanup()

	createSource(t, r, "Ingest Test", "ingest-test")

	payload := `{"event":"push","ref":"main"}`
	req := httptest.NewRequest(http.MethodPost, "/webhooks/ingest-test", bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Idempotency-Key", "test-key-1")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["delivery_id"] == nil {
		t.Fatal("expected delivery_id in response")
	}
}

func TestWebhookIngestIdempotency(t *testing.T) {
	r, cleanup := setupRouter(t)
	defer cleanup()

	createSource(t, r, "Idem Test", "idem-test")

	payload := `{"event":"push"}`

	// First request
	req1 := httptest.NewRequest(http.MethodPost, "/webhooks/idem-test", bytes.NewBufferString(payload))
	req1.Header.Set("Content-Type", "application/json")
	req1.Header.Set("X-Idempotency-Key", "idem-key-123")
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req1)

	if w1.Code != http.StatusAccepted {
		t.Fatalf("first request: expected 202, got %d", w1.Code)
	}

	// Second request with same key should fail (unique violation)
	req2 := httptest.NewRequest(http.MethodPost, "/webhooks/idem-test", bytes.NewBufferString(payload))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("X-Idempotency-Key", "idem-key-123")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	if w2.Code != http.StatusInternalServerError {
		t.Fatalf("second request: expected 500 (duplicate key), got %d: %s", w2.Code, w2.Body.String())
	}
}

func TestWebhookIngestInvalidJSON(t *testing.T) {
	r, cleanup := setupRouter(t)
	defer cleanup()

	createSource(t, r, "Invalid JSON", "invalid-json")

	req := httptest.NewRequest(http.MethodPost, "/webhooks/invalid-json", bytes.NewBufferString("not json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestWebhookIngestUnknownSource(t *testing.T) {
	r, cleanup := setupRouter(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodPost, "/webhooks/nonexistent", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestSourceCRUDEndpoints(t *testing.T) {
	r, cleanup := setupRouter(t)
	defer cleanup()

	// Create
	createSource(t, r, "CRUD Source", "crud-source")

	// Get
	req := httptest.NewRequest(http.MethodGet, "/api/sources/crud-source", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("get: expected 200, got %d", w.Code)
	}

	// List
	req = httptest.NewRequest(http.MethodGet, "/api/sources", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("list: expected 200, got %d", w.Code)
	}

	// Update
	updateBody, _ := json.Marshal(map[string]string{"name": "Updated"})
	req = httptest.NewRequest(http.MethodPatch, "/api/sources/crud-source", bytes.NewReader(updateBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("update: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Delete
	req = httptest.NewRequest(http.MethodDelete, "/api/sources/crud-source", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNoContent {
		t.Fatalf("delete: expected 204, got %d", w.Code)
	}
}

func TestActionCRUDEndpoints(t *testing.T) {
	r, cleanup := setupRouter(t)
	defer cleanup()

	// Need dispatchers registered for validation
	// Register webhook dispatcher for tests
	registerTestDispatchers()

	createSource(t, r, "Action CRUD", "action-crud")

	// Create action
	actionBody, _ := json.Marshal(map[string]string{
		"type":       "webhook",
		"target_url": "https://example.com/hook",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/sources/action-crud/actions", bytes.NewReader(actionBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create action: expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var action model.Action
	json.Unmarshal(w.Body.Bytes(), &action)

	// List actions
	req = httptest.NewRequest(http.MethodGet, "/api/sources/action-crud/actions", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("list actions: expected 200, got %d", w.Code)
	}

	// Get action
	req = httptest.NewRequest(http.MethodGet, "/api/sources/action-crud/actions/"+action.ID.String(), nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("get action: expected 200, got %d", w.Code)
	}

	// Update action
	isActive := false
	updateBody, _ := json.Marshal(map[string]any{"is_active": isActive})
	req = httptest.NewRequest(http.MethodPatch, "/api/sources/action-crud/actions/"+action.ID.String(), bytes.NewReader(updateBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("update action: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Delete action
	req = httptest.NewRequest(http.MethodDelete, "/api/sources/action-crud/actions/"+action.ID.String(), nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNoContent {
		t.Fatalf("delete action: expected 204, got %d", w.Code)
	}
}

func TestDeliveryListEndpoint(t *testing.T) {
	r, cleanup := setupRouter(t)
	defer cleanup()

	createSource(t, r, "Del List", "del-list")

	// Ingest a delivery
	req := httptest.NewRequest(http.MethodPost, "/webhooks/del-list", bytes.NewBufferString(`{"test":true}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var ingestResp map[string]any
	json.Unmarshal(w.Body.Bytes(), &ingestResp)
	deliveryID := ingestResp["delivery_id"].(string)

	// List deliveries
	req = httptest.NewRequest(http.MethodGet, "/api/deliveries", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("list deliveries: expected 200, got %d", w.Code)
	}

	// Get delivery
	req = httptest.NewRequest(http.MethodGet, "/api/deliveries/"+deliveryID, nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("get delivery: expected 200, got %d", w.Code)
	}

	// List attempts (should be empty)
	req = httptest.NewRequest(http.MethodGet, "/api/deliveries/"+deliveryID+"/attempts", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("list attempts: expected 200, got %d", w.Code)
	}
}
