package idempotency

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func perform(middleware func(http.HandlerFunc) http.HandlerFunc, handler http.HandlerFunc, method, target, key string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, target, strings.NewReader(`{"value":1}`))
	if key != "" {
		req.Header.Set("Idempotency-Key", key)
	}

	rec := httptest.NewRecorder()
	middleware(handler)(rec, req)

	return rec
}

func TestBuildConfigDefaults(t *testing.T) {
	config := buildConfig(Config{})

	if config.Header != defaultHeader {
		t.Errorf("Header = %q, want %q", config.Header, defaultHeader)
	}
	if config.TTL != defaultTTL {
		t.Errorf("TTL = %s, want %s", config.TTL, defaultTTL)
	}
	if len(config.Methods) != 2 || config.Methods[0] != http.MethodPost || config.Methods[1] != http.MethodPatch {
		t.Errorf("Methods = %v, want [POST PATCH]", config.Methods)
	}
	if config.Required {
		t.Error("Required = true, want false")
	}
	if config.ErrorHandler == nil {
		t.Error("ErrorHandler is nil")
	}
	if config.Store == nil {
		t.Error("Store is nil")
	}
	if !config.allowsMethod(http.MethodPost) {
		t.Error("POST should be allowed")
	}
	if config.allowsMethod(http.MethodGet) {
		t.Error("GET should not be allowed")
	}
}

func TestBuildConfigKeepsValues(t *testing.T) {
	errorHandler := func(http.ResponseWriter, *http.Request, error) {}
	config := buildConfig(Config{
		Header:       "X-Idempotency-Key",
		TTL:          time.Hour,
		Methods:      []string{"put"},
		Required:     true,
		ErrorHandler: errorHandler,
	})

	if config.Header != "X-Idempotency-Key" {
		t.Errorf("Header = %q", config.Header)
	}
	if config.TTL != time.Hour {
		t.Errorf("TTL = %s, want %s", config.TTL, time.Hour)
	}
	if !config.allowsMethod(http.MethodPut) {
		t.Error("PUT should be allowed case-insensitively")
	}
	if !config.Required {
		t.Error("Required = false, want true")
	}
	if config.ErrorHandler == nil {
		t.Error("ErrorHandler was replaced")
	}
}

func TestNewReplaysStoredResponse(t *testing.T) {
	calls := 0
	middleware := New()
	handler := func(rw http.ResponseWriter, r *http.Request) {
		calls++
		rw.Header().Set("X-Custom", "value")
		rw.WriteHeader(http.StatusCreated)
		_, _ = rw.Write([]byte("created"))
	}

	first := perform(middleware, handler, http.MethodPost, "/orders", "key-1")
	second := perform(middleware, handler, http.MethodPost, "/orders", "key-1")

	if calls != 1 {
		t.Errorf("handler calls = %d, want 1", calls)
	}
	for _, rec := range []*httptest.ResponseRecorder{first, second} {
		if rec.Code != http.StatusCreated {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusCreated)
		}
		if got := rec.Header().Get("X-Custom"); got != "value" {
			t.Errorf("X-Custom = %q, want %q", got, "value")
		}
		if got := rec.Body.String(); got != "created" {
			t.Errorf("body = %q, want %q", got, "created")
		}
	}
}

type trackingStore struct {
	*MemoryStore
	begins int
}

func (s *trackingStore) Begin(key string) (*StoredResponse, State) {
	s.begins++
	return s.MemoryStore.Begin(key)
}

func TestNewUsesCustomStore(t *testing.T) {
	store := &trackingStore{MemoryStore: NewMemoryStore(time.Minute)}
	calls := 0
	middleware := New(Config{Store: store})
	handler := func(rw http.ResponseWriter, r *http.Request) {
		calls++
		rw.WriteHeader(http.StatusCreated)
	}

	perform(middleware, handler, http.MethodPost, "/", "key")
	perform(middleware, handler, http.MethodPost, "/", "key")

	if calls != 1 {
		t.Errorf("handler calls = %d, want 1", calls)
	}
	if store.begins != 2 {
		t.Errorf("store Begin calls = %d, want 2", store.begins)
	}
}

func TestNewScopesByMethodAndPath(t *testing.T) {
	calls := 0
	middleware := New()
	handler := func(rw http.ResponseWriter, r *http.Request) {
		calls++
		_, _ = rw.Write([]byte("ok"))
	}

	perform(middleware, handler, http.MethodPost, "/a", "key")
	perform(middleware, handler, http.MethodPost, "/b", "key")
	perform(middleware, handler, http.MethodPatch, "/a", "key")
	perform(middleware, handler, http.MethodGet, "/a", "key")

	if calls != 4 {
		t.Errorf("handler calls = %d, want 4", calls)
	}
}

func TestNewUsesCustomMethods(t *testing.T) {
	calls := 0
	middleware := New(Config{Methods: []string{http.MethodPut}})
	handler := func(rw http.ResponseWriter, r *http.Request) {
		calls++
		_, _ = rw.Write([]byte("ok"))
	}

	perform(middleware, handler, http.MethodPut, "/", "key")
	perform(middleware, handler, http.MethodPut, "/", "key")

	if calls != 1 {
		t.Errorf("handler calls = %d, want 1", calls)
	}
}

func TestNewRequiresKey(t *testing.T) {
	calls := 0
	middleware := New(Config{Required: true})
	handler := func(rw http.ResponseWriter, r *http.Request) {
		calls++
	}

	rec := perform(middleware, handler, http.MethodPost, "/", "")

	if calls != 0 {
		t.Errorf("handler calls = %d, want 0", calls)
	}
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}

	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["error"] != ErrorKeyRequired.Error() {
		t.Errorf("error = %q, want %q", body["error"], ErrorKeyRequired.Error())
	}
}

func TestNewRejectsInvalidKeys(t *testing.T) {
	cases := []string{
		strings.Repeat("a", maxKeyLength+1),
		"   ",
	}

	for _, key := range cases {
		calls := 0
		middleware := New()
		handler := func(rw http.ResponseWriter, r *http.Request) {
			calls++
		}

		rec := perform(middleware, handler, http.MethodPost, "/", key)

		if calls != 0 {
			t.Errorf("key %q: handler calls = %d, want 0", key, calls)
		}
		if rec.Code != http.StatusBadRequest {
			t.Errorf("key %q: status = %d, want %d", key, rec.Code, http.StatusBadRequest)
		}
	}
}

func TestNewAcceptsMaximumKeyLength(t *testing.T) {
	calls := 0
	middleware := New()
	handler := func(rw http.ResponseWriter, r *http.Request) {
		calls++
	}
	key := strings.Repeat("a", maxKeyLength)

	perform(middleware, handler, http.MethodPost, "/", key)
	perform(middleware, handler, http.MethodPost, "/", key)

	if calls != 1 {
		t.Errorf("handler calls = %d, want 1", calls)
	}
}

func TestNewUsesCustomHeader(t *testing.T) {
	calls := 0
	middleware := New(Config{Header: "X-Request-Key"})
	handler := func(rw http.ResponseWriter, r *http.Request) {
		calls++
	}

	request := func() {
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		req.Header.Set("X-Request-Key", "key")
		middleware(handler)(httptest.NewRecorder(), req)
	}

	request()
	request()

	if calls != 1 {
		t.Errorf("handler calls = %d, want 1", calls)
	}
}

func TestNewRejectsInProgressKey(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	firstDone := make(chan struct{})
	calls := 0

	middleware := New()
	handler := func(rw http.ResponseWriter, r *http.Request) {
		calls++
		close(started)
		<-release
		rw.WriteHeader(http.StatusCreated)
		_, _ = rw.Write([]byte("created"))
	}

	go func() {
		defer close(firstDone)
		perform(middleware, handler, http.MethodPost, "/", "key")
	}()

	<-started

	rec := perform(middleware, handler, http.MethodPost, "/", "key")
	if rec.Code != http.StatusConflict {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusConflict)
	}
	if got := rec.Header().Get("Retry-After"); got != "1" {
		t.Errorf("Retry-After = %q, want %q", got, "1")
	}

	close(release)
	<-firstDone

	rec = perform(middleware, handler, http.MethodPost, "/", "key")
	if rec.Code != http.StatusCreated {
		t.Errorf("replayed status = %d, want %d", rec.Code, http.StatusCreated)
	}
	if calls != 1 {
		t.Errorf("handler calls = %d, want 1", calls)
	}
}

func TestNewDoesNotStoreServerErrors(t *testing.T) {
	calls := 0
	middleware := New()
	handler := func(rw http.ResponseWriter, r *http.Request) {
		calls++
		rw.WriteHeader(http.StatusInternalServerError)
		_, _ = rw.Write([]byte("boom"))
	}

	perform(middleware, handler, http.MethodPost, "/", "key")
	perform(middleware, handler, http.MethodPost, "/", "key")

	if calls != 2 {
		t.Errorf("handler calls = %d, want 2", calls)
	}
}

func TestNewClearsEntryOnPanic(t *testing.T) {
	calls := 0
	middleware := New()
	handler := func(rw http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			panic("boom")
		}
		rw.WriteHeader(http.StatusCreated)
	}

	func() {
		defer func() {
			if recovered := recover(); recovered == nil {
				t.Error("expected panic")
			}
		}()
		perform(middleware, handler, http.MethodPost, "/", "key")
	}()

	rec := perform(middleware, handler, http.MethodPost, "/", "key")
	if rec.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusCreated)
	}
	if calls != 2 {
		t.Errorf("handler calls = %d, want 2", calls)
	}
}

func TestNewUsesCustomErrorHandler(t *testing.T) {
	middleware := New(Config{
		Required: true,
		ErrorHandler: func(rw http.ResponseWriter, r *http.Request, err error) {
			if !errors.Is(err, ErrorKeyRequired) {
				t.Errorf("error = %v, want %v", err, ErrorKeyRequired)
			}
			rw.WriteHeader(http.StatusTeapot)
		},
	})
	handler := func(http.ResponseWriter, *http.Request) {}

	rec := perform(middleware, handler, http.MethodPost, "/", "")
	if rec.Code != http.StatusTeapot {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusTeapot)
	}
}

type fakeClock struct {
	t time.Time
}

func (c *fakeClock) now() time.Time {
	return c.t
}

func (c *fakeClock) advance(d time.Duration) {
	c.t = c.t.Add(d)
}

func TestStoreExpiresEntries(t *testing.T) {
	clock := &fakeClock{t: time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)}
	store := NewMemoryStore(time.Minute)
	store.now = clock.now
	store.lastCleanup = clock.t

	if _, state := store.Begin("key"); state != StateMissing {
		t.Fatalf("state = %v, want StateMissing", state)
	}
	store.Complete("key", &StoredResponse{Status: http.StatusOK})

	stored, state := store.Begin("key")
	if state != StateFound || stored == nil {
		t.Fatalf("state = %v, stored = %v, want StateFound", state, stored)
	}

	clock.advance(2 * time.Minute)
	if _, state := store.Begin("other"); state != StateMissing {
		t.Fatalf("state = %v, want StateMissing", state)
	}
	if _, ok := store.entries["key"]; ok {
		t.Error("expired entry was not removed")
	}
}

func TestStoreExpiredEntryIsNotReplayedBeforeCleanup(t *testing.T) {
	clock := &fakeClock{t: time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)}
	store := NewMemoryStore(time.Hour)
	store.now = clock.now
	store.lastCleanup = clock.t

	store.Begin("key")
	store.Complete("key", &StoredResponse{Status: http.StatusOK})

	clock.advance(59*time.Minute + 30*time.Second)
	store.lastCleanup = clock.t
	clock.advance(40 * time.Second)

	stored, state := store.Begin("key")
	if state != StateMissing || stored != nil {
		t.Fatalf("state = %v, stored = %v, want StateMissing", state, stored)
	}
}

func TestStoreAbortsReservation(t *testing.T) {
	store := NewMemoryStore(time.Minute)

	if _, state := store.Begin("key"); state != StateMissing {
		t.Fatalf("state = %v, want StateMissing", state)
	}

	if _, state := store.Begin("key"); state != StateInProgress {
		t.Fatalf("state = %v, want StateInProgress", state)
	}

	store.Abort("key")

	if _, state := store.Begin("key"); state != StateMissing {
		t.Fatalf("state = %v, want StateMissing", state)
	}
}
