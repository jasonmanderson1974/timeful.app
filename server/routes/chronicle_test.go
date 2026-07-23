package routes

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"sirtom/server/middleware"
)

// The Chronicle is members-only: an unauthenticated caller is rejected before
// the handler runs (no DB needed). Broader allowlist gating is covered by
// access_control_test.go's AuthRequired suite.
func TestGetChronicle_Unauthorized(t *testing.T) {
	r := newTestRouter()
	r.GET("/chronicle", middleware.AuthRequired(), getChronicle)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/chronicle", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated chronicle: got %d, want 401 (%s)", w.Code, w.Body.String())
	}
}
