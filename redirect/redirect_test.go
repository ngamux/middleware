package redirect

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestUpstreamRequestFailure(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		rw.WriteHeader(http.StatusOK)
	}))
	target := upstream.URL + "/new"
	upstream.Close()

	nextCalled := false
	middleware := New(Config{Rewrite: Rewrite{"/old": target}})
	handler := middleware(func(rw http.ResponseWriter, r *http.Request) {
		nextCalled = true
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/old", nil)

	handler(rec, req)

	if nextCalled {
		t.Error("next handler should not be called when the upstream request fails")
	}

	if rec.Code != http.StatusBadGateway {
		t.Errorf("expected status %d, got %d", http.StatusBadGateway, rec.Code)
	}

	if rec.Body.Len() == 0 {
		t.Error("expected the upstream error to be written to the response body")
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
