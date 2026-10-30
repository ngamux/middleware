package ping

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewWithEmptyConfigUsesDefaultPath(t *testing.T) {
	middleware := New(Config{})
	handler := func(rw http.ResponseWriter, r *http.Request) {
		t.Error("next handler should not be called for the default ping path")
	}
	result := middleware(handler)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)

	result(rec, req)

	if got := rec.Body.String(); got != "pong" {
		t.Errorf("expected body %q, got %q", "pong", got)
	}
}

func BenchmarkPing(b *testing.B) {
	b.Run("new", func(b *testing.B) {
		middleware := New()
		handler := func(rw http.ResponseWriter, r *http.Request) {}
		result := middleware(handler)

		rec := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/", nil)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			result(rec, req)
		}
	})
}
