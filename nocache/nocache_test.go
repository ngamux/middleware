package nocache

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewReplacesExistingCacheHeaders(t *testing.T) {
	middleware := New()
	handler := func(rw http.ResponseWriter, r *http.Request) {}
	result := middleware(handler)

	rec := httptest.NewRecorder()
	rec.Header().Set("Cache-Control", "public, max-age=3600")
	req, _ := http.NewRequest(http.MethodGet, "/", nil)

	result(rec, req)

	values := rec.Header().Values("Cache-Control")
	if len(values) != 1 {
		t.Fatalf("expected exactly 1 Cache-Control value, got %d: %q", len(values), values)
	}
	if values[0] != "no-cache, private, max-age=0" {
		t.Errorf("expected Cache-Control to be replaced, got %q", values[0])
	}
}

func BenchmarkLog(b *testing.B) {
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
