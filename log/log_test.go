package log

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewWithEmptyConfig(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("New(Config{}) panicked: %v", r)
		}
	}()

	middleware := New(Config{})

	called := false
	handler := func(rw http.ResponseWriter, r *http.Request) {
		called = true
		rw.WriteHeader(http.StatusNoContent)
	}
	result := middleware(handler)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	result(rec, req)

	if !called {
		t.Error("expected next handler to be called")
	}
	if rec.Code != http.StatusNoContent {
		t.Errorf("expected status %d, got %d", http.StatusNoContent, rec.Code)
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
