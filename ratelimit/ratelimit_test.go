package ratelimit

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"
	"time"
)

type fakeClock struct {
	t time.Time
}

func (c *fakeClock) now() time.Time {
	return c.t
}

func (c *fakeClock) advance(d time.Duration) {
	c.t = c.t.Add(d)
}

func newTestLimiter(config Config) (*limiter, *fakeClock) {
	config = buildConfig(config)
	clock := &fakeClock{t: time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)}
	l := newLimiter(config)
	l.now = clock.now
	l.lastGC = clock.t

	return l, clock
}

func TestBuildConfigDefaults(t *testing.T) {
	config := buildConfig(Config{})

	if config.Requests != defaultRequests {
		t.Errorf("Requests = %d, want %d", config.Requests, defaultRequests)
	}
	if config.Window != defaultWindow {
		t.Errorf("Window = %s, want %s", config.Window, defaultWindow)
	}
	if config.Burst != defaultRequests {
		t.Errorf("Burst = %d, want %d", config.Burst, defaultRequests)
	}
	if config.KeyFunc == nil {
		t.Error("KeyFunc is nil")
	}
	if config.ErrorHandler == nil {
		t.Error("ErrorHandler is nil")
	}
}

func TestBuildConfigKeepsValues(t *testing.T) {
	keyFunc := func(*http.Request) string { return "key" }
	errorHandler := func(http.ResponseWriter, *http.Request, time.Duration) {}

	config := buildConfig(Config{
		Requests:     10,
		Window:       time.Second,
		Burst:        5,
		KeyFunc:      keyFunc,
		ErrorHandler: errorHandler,
	})

	if config.Requests != 10 || config.Window != time.Second || config.Burst != 5 {
		t.Errorf("unexpected config: %+v", config)
	}
	if config.KeyFunc(nil) != "key" {
		t.Error("KeyFunc was replaced")
	}
	if config.ErrorHandler == nil {
		t.Error("ErrorHandler was replaced")
	}
}

func TestAllowUpToBurst(t *testing.T) {
	l, _ := newTestLimiter(Config{
		Requests: 2,
		Window:   time.Second,
		Burst:    2,
	})

	for i := 0; i < 2; i++ {
		d := l.allow("client")
		if !d.allowed {
			t.Fatalf("request %d denied, want allowed", i+1)
		}
	}

	d := l.allow("client")
	if d.allowed {
		t.Fatal("request above burst allowed, want denied")
	}
	if d.retryAfter != 500*time.Millisecond {
		t.Errorf("retryAfter = %s, want %s", d.retryAfter, 500*time.Millisecond)
	}
}

func TestAllowRefillsOverTime(t *testing.T) {
	l, clock := newTestLimiter(Config{
		Requests: 60,
		Window:   time.Minute,
		Burst:    1,
	})

	if d := l.allow("client"); !d.allowed || d.remaining != 0 {
		t.Fatalf("first request = %+v, want allowed with 0 remaining", d)
	}

	if d := l.allow("client"); d.allowed || d.retryAfter != time.Second {
		t.Fatalf("immediate request = %+v, want denied with 1s retry", d)
	}

	clock.advance(500 * time.Millisecond)
	if d := l.allow("client"); d.allowed || d.retryAfter != 500*time.Millisecond {
		t.Fatalf("half-refill request = %+v, want denied with 500ms retry", d)
	}

	clock.advance(500 * time.Millisecond)
	if d := l.allow("client"); !d.allowed {
		t.Fatalf("request after refill = %+v, want allowed", d)
	}
}

func TestAllowSeparatesKeys(t *testing.T) {
	l, _ := newTestLimiter(Config{
		Requests: 1,
		Window:   time.Second,
		Burst:    1,
	})

	if d := l.allow("alice"); !d.allowed {
		t.Fatal("alice's first request denied")
	}
	if d := l.allow("alice"); d.allowed {
		t.Fatal("alice's second request allowed")
	}
	if d := l.allow("bob"); !d.allowed {
		t.Fatal("bob's first request denied")
	}
}

func TestCleanupRemovesIdleBuckets(t *testing.T) {
	l, clock := newTestLimiter(Config{
		Requests: 1,
		Window:   time.Second,
		Burst:    1,
	})

	l.allow("idle")

	clock.advance(time.Second)
	l.allow("active")

	if _, ok := l.buckets["idle"]; ok {
		t.Error("idle bucket was not removed")
	}
	if _, ok := l.buckets["active"]; !ok {
		t.Error("active bucket was removed")
	}
}

func TestDefaultKeyFuncUsesClientIP(t *testing.T) {
	cases := []struct {
		remoteAddr string
		want       string
	}{
		{"192.0.2.1:1234", "192.0.2.1"},
		{"192.0.2.1", "192.0.2.1"},
		{"[::1]:8080", "::1"},
		{"", ""},
	}

	for _, c := range cases {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.RemoteAddr = c.remoteAddr
		if got := defaultKeyFunc(r); got != c.want {
			t.Errorf("defaultKeyFunc(%q) = %q, want %q", c.remoteAddr, got, c.want)
		}
	}
}

func TestNewAllowsRequestAndSetsHeaders(t *testing.T) {
	called := false
	handler := New(Config{
		Requests: 5,
		Window:   time.Minute,
		Burst:    5,
	})(func(rw http.ResponseWriter, _ *http.Request) {
		called = true
		rw.WriteHeader(http.StatusNoContent)
	})

	rw := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = "192.0.2.1:1234"
	handler(rw, r)

	if !called {
		t.Fatal("next handler was not called")
	}
	if rw.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d", rw.Code, http.StatusNoContent)
	}
	if got := rw.Header().Get(HeaderLimit); got != "5" {
		t.Errorf("%s = %q, want %q", HeaderLimit, got, "5")
	}
	if got := rw.Header().Get(HeaderRemaining); got != "4" {
		t.Errorf("%s = %q, want %q", HeaderRemaining, got, "4")
	}
	if got := rw.Header().Get(HeaderReset); got == "" {
		t.Errorf("%s is empty", HeaderReset)
	}
}

func TestNewRejectsRequestsAboveLimit(t *testing.T) {
	called := false
	handler := New(Config{
		Requests: 1,
		Window:   time.Hour,
		Burst:    1,
	})(func(http.ResponseWriter, *http.Request) {
		called = true
	})

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = "192.0.2.1:1234"

	first := httptest.NewRecorder()
	handler(first, r)
	if !called {
		t.Fatal("next handler was not called for first request")
	}

	second := httptest.NewRecorder()
	handler(second, r)

	if second.Code != http.StatusTooManyRequests {
		t.Errorf("status = %d, want %d", second.Code, http.StatusTooManyRequests)
	}
	if got := second.Header().Get(HeaderRemaining); got != "0" {
		t.Errorf("%s = %q, want %q", HeaderRemaining, got, "0")
	}
	retryAfter, err := strconv.Atoi(second.Header().Get(HeaderRetryAfter))
	if err != nil || retryAfter < 1 {
		t.Errorf("%s = %q, want a positive integer", HeaderRetryAfter, second.Header().Get(HeaderRetryAfter))
	}

	var body map[string]string
	if err := json.NewDecoder(second.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["error"] != "too many requests" {
		t.Errorf("error = %q, want %q", body["error"], "too many requests")
	}
}

func TestNewSeparatesClients(t *testing.T) {
	handler := New(Config{
		Requests: 1,
		Window:   time.Hour,
		Burst:    1,
	})(func(rw http.ResponseWriter, _ *http.Request) {
		rw.WriteHeader(http.StatusNoContent)
	})

	alice := httptest.NewRequest(http.MethodGet, "/", nil)
	alice.RemoteAddr = "192.0.2.1:1234"
	bob := httptest.NewRequest(http.MethodGet, "/", nil)
	bob.RemoteAddr = "192.0.2.2:1234"

	rw := httptest.NewRecorder()
	handler(rw, alice)
	if rw.Code != http.StatusNoContent {
		t.Fatalf("alice's first status = %d, want %d", rw.Code, http.StatusNoContent)
	}

	rw = httptest.NewRecorder()
	handler(rw, alice)
	if rw.Code != http.StatusTooManyRequests {
		t.Fatalf("alice's second status = %d, want %d", rw.Code, http.StatusTooManyRequests)
	}

	rw = httptest.NewRecorder()
	handler(rw, bob)
	if rw.Code != http.StatusNoContent {
		t.Fatalf("bob's first status = %d, want %d", rw.Code, http.StatusNoContent)
	}
}

func TestNewUsesCustomKeyFunc(t *testing.T) {
	handler := New(Config{
		Requests: 1,
		Window:   time.Hour,
		Burst:    1,
		KeyFunc: func(r *http.Request) string {
			return r.Header.Get("X-API-Key")
		},
	})(func(rw http.ResponseWriter, _ *http.Request) {
		rw.WriteHeader(http.StatusNoContent)
	})

	request := func(key string) int {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.Header.Set("X-API-Key", key)
		rw := httptest.NewRecorder()
		handler(rw, r)
		return rw.Code
	}

	if got := request("one"); got != http.StatusNoContent {
		t.Errorf("first key request status = %d, want %d", got, http.StatusNoContent)
	}
	if got := request("one"); got != http.StatusTooManyRequests {
		t.Errorf("repeated key request status = %d, want %d", got, http.StatusTooManyRequests)
	}
	if got := request("two"); got != http.StatusNoContent {
		t.Errorf("second key request status = %d, want %d", got, http.StatusNoContent)
	}
}

func TestNewUsesCustomErrorHandler(t *testing.T) {
	var gotRetryAfter time.Duration
	handler := New(Config{
		Requests: 1,
		Window:   time.Hour,
		Burst:    1,
		ErrorHandler: func(rw http.ResponseWriter, _ *http.Request, retryAfter time.Duration) {
			gotRetryAfter = retryAfter
			rw.WriteHeader(http.StatusTeapot)
		},
	})(func(http.ResponseWriter, *http.Request) {})

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = "192.0.2.1:1234"

	handler(httptest.NewRecorder(), r)

	rw := httptest.NewRecorder()
	handler(rw, r)

	if rw.Code != http.StatusTeapot {
		t.Errorf("status = %d, want %d", rw.Code, http.StatusTeapot)
	}
	if gotRetryAfter <= 0 {
		t.Errorf("retryAfter = %s, want a positive duration", gotRetryAfter)
	}
}

func TestNewIsConcurrencySafe(t *testing.T) {
	const requests = 100
	handler := New(Config{
		Requests: requests,
		Window:   time.Minute,
		Burst:    requests,
	})(func(http.ResponseWriter, *http.Request) {})

	var wg sync.WaitGroup
	for i := 0; i < requests; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r.RemoteAddr = "192.0.2.1:1234"
			handler(httptest.NewRecorder(), r)
		}()
	}
	wg.Wait()
}

func BenchmarkNew(b *testing.B) {
	handler := New()(func(http.ResponseWriter, *http.Request) {})
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = "192.0.2.1:1234"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		handler(httptest.NewRecorder(), r)
	}
}
