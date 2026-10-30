package pprof

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewDoesNotHijackSimilarPaths(t *testing.T) {
	called := false
	handler := func(rw http.ResponseWriter, r *http.Request) {
		called = true
		rw.WriteHeader(http.StatusNoContent)
	}

	middleware := New()
	result := middleware(handler)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/debug/pprofile", nil)
	result(rec, req)

	if !called {
		t.Fatalf("next handler should be called for /debug/pprofile, got status %d", rec.Code)
	}
}
