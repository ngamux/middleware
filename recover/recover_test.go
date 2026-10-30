package recover

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNewRecoversNonStringPanic(t *testing.T) {
	middleware := New()
	handler := func(rw http.ResponseWriter, r *http.Request) {
		panic(errors.New("upstream failure"))
	}
	result := middleware(handler)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	var escaped any
	func() {
		defer func() { escaped = recover() }()
		result(rec, req)
	}()

	if escaped != nil {
		t.Fatalf("panic escaped the middleware: %v", escaped)
	}
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
	if got := rec.Body.String(); !strings.Contains(got, "upstream failure") {
		t.Errorf("expected body to contain the panic message, got %q", got)
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
