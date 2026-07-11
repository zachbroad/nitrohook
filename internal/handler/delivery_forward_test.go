package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// TestForward_BadID checks the pre-store validation path: a non-UUID id is
// rejected with 400 without needing Redis or the DB.
func TestForward_BadID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := &DeliveryHandler{}
	r := gin.New()
	r.POST("/api/deliveries/:id/forward", h.Forward)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/deliveries/not-a-uuid/forward", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}
