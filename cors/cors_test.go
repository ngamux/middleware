package cors

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewDerivesOriginFromReferer(t *testing.T) {
	middleware := New(Config{AllowOrigins: "https://app.example.com"})
	handler := middleware(func(rw http.ResponseWriter, r *http.Request) {})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/data", nil)
	req.Header.Set("Referer", "https://app.example.com/dashboard?tab=1")

	handler(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://app.example.com" {
		t.Errorf("Access-Control-Allow-Origin = %q, want %q", got, "https://app.example.com")
	}
}

func BenchmarkCors(b *testing.B) {
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
