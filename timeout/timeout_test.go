package timeout

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func TestBuildConfigDefaults(t *testing.T) {
	config := buildConfig(Config{})

	if config.Timeout != defaultTimeout {
		t.Errorf("Timeout = %s, want %s", config.Timeout, defaultTimeout)
	}
	if config.ErrorHandler == nil {
		t.Error("ErrorHandler is nil")
	}
}

func TestBuildConfigKeepsValues(t *testing.T) {
	errorHandler := func(http.ResponseWriter, *http.Request) {}
	config := buildConfig(Config{
		Timeout:      time.Second,
		ErrorHandler: errorHandler,
	})

	if config.Timeout != time.Second {
		t.Errorf("Timeout = %s, want %s", config.Timeout, time.Second)
	}
	if config.ErrorHandler == nil {
		t.Error("ErrorHandler was replaced")
	}
}

func TestNewWritesHandlerResponse(t *testing.T) {
	handler := New(Config{Timeout: time.Second})(func(rw http.ResponseWriter, r *http.Request) {
		rw.Header().Set("X-Test", "ok")
		rw.WriteHeader(http.StatusCreated)
		_, _ = rw.Write([]byte("hello"))
	})

	rw := httptest.NewRecorder()
	handler(rw, httptest.NewRequest(http.MethodGet, "/", nil))

	if rw.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d", rw.Code, http.StatusCreated)
	}
	if got := rw.Header().Get("X-Test"); got != "ok" {
		t.Errorf("X-Test = %q, want %q", got, "ok")
	}
	if got := rw.Body.String(); got != "hello" {
		t.Errorf("body = %q, want %q", got, "hello")
	}
}

func TestNewDefaultsToOKStatus(t *testing.T) {
	handler := New(Config{Timeout: time.Second})(func(rw http.ResponseWriter, r *http.Request) {
		_, _ = rw.Write([]byte("hello"))
	})

	rw := httptest.NewRecorder()
	handler(rw, httptest.NewRequest(http.MethodGet, "/", nil))

	if rw.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rw.Code, http.StatusOK)
	}
	if got := rw.Body.String(); got != "hello" {
		t.Errorf("body = %q, want %q", got, "hello")
	}
}

func TestNewTimesOut(t *testing.T) {
	release := make(chan struct{})
	handlerDone := make(chan struct{})

	handler := New(Config{Timeout: 10 * time.Millisecond})(func(rw http.ResponseWriter, r *http.Request) {
		defer close(handlerDone)
		<-release
		rw.WriteHeader(http.StatusCreated)
		_, _ = rw.Write([]byte("late"))
	})

	rw := httptest.NewRecorder()
	handler(rw, httptest.NewRequest(http.MethodGet, "/", nil))

	if rw.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", rw.Code, http.StatusServiceUnavailable)
	}

	var body map[string]string
	if err := json.NewDecoder(rw.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["error"] != "request timeout" {
		t.Errorf("error = %q, want %q", body["error"], "request timeout")
	}

	close(release)
	<-handlerDone

	if got := rw.Body.String(); got == "late" {
		t.Error("late handler write reached the response")
	}
}

func TestNewCancelsRequestContext(t *testing.T) {
	ctxDone := make(chan struct{})
	release := make(chan struct{})

	handler := New(Config{Timeout: 10 * time.Millisecond})(func(rw http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
		close(ctxDone)
		<-release
	})

	handler(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))

	select {
	case <-ctxDone:
	case <-time.After(time.Second):
		t.Fatal("request context was not canceled")
	}

	close(release)
}

func TestNewUsesCustomErrorHandler(t *testing.T) {
	release := make(chan struct{})
	handlerDone := make(chan struct{})
	called := false

	handler := New(Config{
		Timeout: 10 * time.Millisecond,
		ErrorHandler: func(rw http.ResponseWriter, r *http.Request) {
			called = true
			rw.WriteHeader(http.StatusGatewayTimeout)
		},
	})(func(http.ResponseWriter, *http.Request) {
		defer close(handlerDone)
		<-release
	})

	rw := httptest.NewRecorder()
	handler(rw, httptest.NewRequest(http.MethodGet, "/", nil))

	close(release)
	<-handlerDone

	if !called {
		t.Error("custom ErrorHandler was not called")
	}
	if rw.Code != http.StatusGatewayTimeout {
		t.Errorf("status = %d, want %d", rw.Code, http.StatusGatewayTimeout)
	}
}

func TestNewPropagatesPanic(t *testing.T) {
	defer func() {
		if p := recover(); p != "boom" {
			t.Errorf("recovered %v, want %q", p, "boom")
		}
	}()

	handler := New(Config{Timeout: time.Second})(func(http.ResponseWriter, *http.Request) {
		panic("boom")
	})

	handler(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
}

func TestNewIsConcurrencySafe(t *testing.T) {
	const requests = 50
	handler := New(Config{Timeout: time.Second})(func(rw http.ResponseWriter, r *http.Request) {
		rw.WriteHeader(http.StatusNoContent)
	})

	var wg sync.WaitGroup
	for i := 0; i < requests; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			handler(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
		}()
	}
	wg.Wait()
}

func BenchmarkNew(b *testing.B) {
	handler := New()(func(rw http.ResponseWriter, r *http.Request) {
		rw.WriteHeader(http.StatusNoContent)
	})
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		handler(httptest.NewRecorder(), req)
	}
}
