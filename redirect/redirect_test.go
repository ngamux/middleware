package redirect

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewProxiesRequest(t *testing.T) {
	type captured struct {
		method string
		path   string
		query  string
		body   string
		header string
		xff    string
	}

	requests := make(chan captured, 1)
	upstream := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		requests <- captured{
			method: r.Method,
			path:   r.URL.Path,
			query:  r.URL.RawQuery,
			body:   string(body),
			header: r.Header.Get("X-Test"),
			xff:    r.Header.Get("X-Forwarded-For"),
		}

		rw.Header().Set("X-Upstream", "yes")
		rw.WriteHeader(http.StatusCreated)
		_, _ = rw.Write([]byte("proxied"))
	}))
	defer upstream.Close()

	middleware := New(Config{
		Rewrite: Rewrite{"/old": upstream.URL + "/new?target=1"},
	})
	handler := middleware(func(http.ResponseWriter, *http.Request) {
		t.Error("next handler should not be called")
	})

	req := httptest.NewRequest(http.MethodPost, "/old?incoming=2", strings.NewReader("hello"))
	req.Header.Set("X-Test", "value")
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusCreated)
	}
	if got := rec.Header().Get("X-Upstream"); got != "yes" {
		t.Errorf("X-Upstream = %q, want %q", got, "yes")
	}
	if got := rec.Body.String(); got != "proxied" {
		t.Errorf("body = %q, want %q", got, "proxied")
	}

	c := <-requests
	if c.method != http.MethodPost {
		t.Errorf("method = %q, want %q", c.method, http.MethodPost)
	}
	if c.path != "/new" {
		t.Errorf("path = %q, want %q", c.path, "/new")
	}
	if c.query != "target=1&incoming=2" {
		t.Errorf("query = %q, want %q", c.query, "target=1&incoming=2")
	}
	if c.body != "hello" {
		t.Errorf("body = %q, want %q", c.body, "hello")
	}
	if c.header != "value" {
		t.Errorf("X-Test = %q, want %q", c.header, "value")
	}
	if c.xff != "192.0.2.1" {
		t.Errorf("X-Forwarded-For = %q, want %q", c.xff, "192.0.2.1")
	}
}

func TestNewPassesThroughUnmatchedPath(t *testing.T) {
	called := false
	handler := New()(func(rw http.ResponseWriter, r *http.Request) {
		called = true
		rw.WriteHeader(http.StatusNoContent)
	})

	rec := httptest.NewRecorder()
	handler(rec, httptest.NewRequest(http.MethodGet, "/other", nil))

	if !called {
		t.Error("next handler was not called")
	}
	if rec.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
}

func TestNewHandlesUpstreamFailure(t *testing.T) {
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
	handler(rec, httptest.NewRequest(http.MethodGet, "/old", nil))

	if nextCalled {
		t.Error("next handler should not be called when the upstream request fails")
	}
	if rec.Code != http.StatusBadGateway {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadGateway)
	}
	if rec.Body.Len() == 0 {
		t.Error("expected the upstream error to be written to the response body")
	}
}

func TestNewTimesOut(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		rw.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	middleware := New(Config{
		Rewrite: Rewrite{"/slow": upstream.URL},
		Timeout: 10 * time.Millisecond,
	})

	rec := httptest.NewRecorder()
	middleware(func(http.ResponseWriter, *http.Request) {})(rec, httptest.NewRequest(http.MethodGet, "/slow", nil))

	if rec.Code != http.StatusGatewayTimeout {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusGatewayTimeout)
	}
}

func TestNewUsesCustomErrorHandler(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {}))
	target := upstream.URL
	upstream.Close()

	called := false
	middleware := New(Config{
		Rewrite: Rewrite{"/old": target},
		ErrorHandler: func(rw http.ResponseWriter, r *http.Request, err error) {
			called = true
			if err == nil {
				t.Error("error is nil")
			}
			rw.WriteHeader(http.StatusTeapot)
		},
	})

	rec := httptest.NewRecorder()
	middleware(func(http.ResponseWriter, *http.Request) {})(rec, httptest.NewRequest(http.MethodGet, "/old", nil))

	if !called {
		t.Error("custom ErrorHandler was not called")
	}
	if rec.Code != http.StatusTeapot {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusTeapot)
	}
}

func TestNewPreservesHost(t *testing.T) {
	hosts := make(chan string, 2)
	upstream := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		hosts <- r.Host
	}))
	defer upstream.Close()

	request := func(preserve bool) string {
		middleware := New(Config{
			Rewrite:      Rewrite{"/old": upstream.URL},
			PreserveHost: preserve,
		})

		req := httptest.NewRequest(http.MethodGet, "/old", nil)
		req.Host = "example.com"
		middleware(func(http.ResponseWriter, *http.Request) {})(httptest.NewRecorder(), req)

		return <-hosts
	}

	if got := request(true); got != "example.com" {
		t.Errorf("preserved host = %q, want %q", got, "example.com")
	}
	if got := request(false); got == "example.com" {
		t.Errorf("host = %q, want the upstream host", got)
	}
}

func TestNewPassesRedirectResponses(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		rw.Header().Set("Location", "/other")
		rw.WriteHeader(http.StatusFound)
	}))
	defer upstream.Close()

	middleware := New(Config{Rewrite: Rewrite{"/old": upstream.URL}})
	rec := httptest.NewRecorder()
	middleware(func(http.ResponseWriter, *http.Request) {})(rec, httptest.NewRequest(http.MethodGet, "/old", nil))

	if rec.Code != http.StatusFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusFound)
	}
	if got := rec.Header().Get("Location"); got != "/other" {
		t.Errorf("Location = %q, want %q", got, "/other")
	}
}

func TestNewPanicsOnInvalidTarget(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("expected panic for invalid target")
		}
	}()

	New(Config{Rewrite: Rewrite{"/old": "not-a-url"}})
}

func BenchmarkNew(b *testing.B) {
	middleware := New()
	handler := func(rw http.ResponseWriter, r *http.Request) {}
	result := middleware(handler)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result(rec, req)
	}
}
