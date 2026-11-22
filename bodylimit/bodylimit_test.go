package bodylimit

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestBuildConfigDefaults(t *testing.T) {
	config := buildConfig(Config{})

	if config.Limit != defaultLimit {
		t.Errorf("Limit = %d, want %d", config.Limit, defaultLimit)
	}
	if config.ErrorHandler == nil {
		t.Error("ErrorHandler is nil")
	}
}

func TestBuildConfigKeepsValues(t *testing.T) {
	errorHandler := func(http.ResponseWriter, *http.Request, error) {}
	config := buildConfig(Config{
		Limit:        42,
		ErrorHandler: errorHandler,
	})

	if config.Limit != 42 {
		t.Errorf("Limit = %d, want 42", config.Limit)
	}
	if config.ErrorHandler == nil {
		t.Error("ErrorHandler was replaced")
	}
}

func TestNewAllowsBodyWithinLimit(t *testing.T) {
	handler := New(Config{Limit: 10})(func(rw http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		if string(body) != "hello" {
			t.Errorf("body = %q, want %q", body, "hello")
		}
		rw.WriteHeader(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("hello"))
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
}

func TestNewRejectsContentLengthAboveLimit(t *testing.T) {
	nextCalled := false
	handler := New(Config{Limit: 4})(func(http.ResponseWriter, *http.Request) {
		nextCalled = true
	})

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("hello"))
	rec := httptest.NewRecorder()
	handler(rec, req)

	if nextCalled {
		t.Fatal("next was called for an oversized body")
	}
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusRequestEntityTooLarge)
	}

	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["error"] != ErrorBodyTooLarge.Error() {
		t.Errorf("error = %q, want %q", body["error"], ErrorBodyTooLarge.Error())
	}
}

func TestNewLimitsUnknownLengthBody(t *testing.T) {
	handler := New(Config{Limit: 4})(func(rw http.ResponseWriter, r *http.Request) {
		_, err := io.ReadAll(r.Body)
		var maxErr *http.MaxBytesError
		if !errors.As(err, &maxErr) {
			t.Errorf("read error = %v, want *http.MaxBytesError", err)
		}
		rw.WriteHeader(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("hello"))
	req.ContentLength = -1
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
}

func TestNewUsesCustomErrorHandler(t *testing.T) {
	nextCalled := false
	handler := New(Config{
		Limit: 1,
		ErrorHandler: func(rw http.ResponseWriter, r *http.Request, err error) {
			if !errors.Is(err, ErrorBodyTooLarge) {
				t.Errorf("error = %v, want %v", err, ErrorBodyTooLarge)
			}
			rw.WriteHeader(http.StatusTeapot)
		},
	})(func(http.ResponseWriter, *http.Request) {
		nextCalled = true
	})

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("ab"))
	rec := httptest.NewRecorder()
	handler(rec, req)

	if nextCalled {
		t.Fatal("next was called for an oversized body")
	}
	if rec.Code != http.StatusTeapot {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusTeapot)
	}
}

func TestNewDefaultLimit(t *testing.T) {
	handler := New()(func(http.ResponseWriter, *http.Request) {})
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(strings.Repeat("a", int(defaultLimit)+1)))
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusRequestEntityTooLarge)
	}
}
