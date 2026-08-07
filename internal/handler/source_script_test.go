package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// TestScript_BadBody exercises validation without a DB: an empty JSON body
// must be rejected with 400 before any store access.
func TestScript_BadBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := &SourceHandler{} // store unused on the validation path
	r := gin.New()
	r.POST("/api/sources/:sourceSlug/script/test", h.TestScript)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/sources/demo/script/test", strings.NewReader(`not json`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

// scriptTestResponse documents the success envelope shape for consumers.
type scriptTestResponse struct {
	Result json.RawMessage `json:"result"`
	Error  *string         `json:"error"`
}
