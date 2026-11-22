package pprof

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func serve(middleware func(http.HandlerFunc) http.HandlerFunc, next http.HandlerFunc, method, target string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	middleware(next)(rec, httptest.NewRequest(method, target, nil))
	return rec
}

func TestNewDoesNotHijackSimilarPaths(t *testing.T) {
	called := false
	handler := func(rw http.ResponseWriter, r *http.Request) {
		called = true
		rw.WriteHeader(http.StatusNoContent)
	}

	rec := serve(New(), handler, http.MethodGet, "/debug/pprofile")

	if !called {
		t.Fatalf("next handler should be called for /debug/pprofile, got status %d", rec.Code)
	}
}

func TestNewRedirectsToTrailingSlash(t *testing.T) {
	rec := serve(New(), func(http.ResponseWriter, *http.Request) {}, http.MethodGet, "/debug/pprof?x=1")

	if rec.Code != http.StatusMovedPermanently {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusMovedPermanently)
	}
	if got := rec.Header().Get("Location"); got != "/debug/pprof/?x=1" {
		t.Errorf("Location = %q, want %q", got, "/debug/pprof/?x=1")
	}
}

func TestNewServesIndex(t *testing.T) {
	rec := serve(New(), func(http.ResponseWriter, *http.Request) {}, http.MethodGet, "/debug/pprof/")

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if got := rec.Header().Get("Content-Type"); !strings.HasPrefix(got, "text/html") {
		t.Errorf("Content-Type = %q, want text/html", got)
	}
	if got := rec.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Errorf("X-Content-Type-Options = %q, want %q", got, "nosniff")
	}
}

func TestNewServesCmdline(t *testing.T) {
	rec := serve(New(), func(http.ResponseWriter, *http.Request) {}, http.MethodGet, "/debug/pprof/cmdline")

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if got := rec.Header().Get("Content-Type"); !strings.HasPrefix(got, "text/plain") {
		t.Errorf("Content-Type = %q, want text/plain", got)
	}
}

func TestNewServesNamedProfile(t *testing.T) {
	rec := serve(New(), func(http.ResponseWriter, *http.Request) {}, http.MethodGet, "/debug/pprof/goroutine?debug=1")

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if got := rec.Header().Get("Content-Type"); !strings.HasPrefix(got, "text/plain") {
		t.Errorf("Content-Type = %q, want text/plain", got)
	}
}

func TestNewRejectsUnknownProfile(t *testing.T) {
	rec := serve(New(), func(http.ResponseWriter, *http.Request) {}, http.MethodGet, "/debug/pprof/unknown")

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestNewUsesCustomPrefix(t *testing.T) {
	middleware := New(Config{Prefix: "/admin/pprof/"})

	t.Run("redirect", func(t *testing.T) {
		rec := serve(middleware, func(http.ResponseWriter, *http.Request) {}, http.MethodGet, "/admin/pprof")
		if rec.Code != http.StatusMovedPermanently {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusMovedPermanently)
		}
		if got := rec.Header().Get("Location"); got != "/admin/pprof/" {
			t.Errorf("Location = %q, want %q", got, "/admin/pprof/")
		}
	})

	t.Run("index", func(t *testing.T) {
		rec := serve(middleware, func(http.ResponseWriter, *http.Request) {}, http.MethodGet, "/admin/pprof/")
		if rec.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
		}
		if got := rec.Header().Get("Content-Type"); !strings.HasPrefix(got, "text/html") {
			t.Errorf("Content-Type = %q, want text/html", got)
		}
	})

	t.Run("named profile", func(t *testing.T) {
		rec := serve(middleware, func(http.ResponseWriter, *http.Request) {}, http.MethodGet, "/admin/pprof/goroutine?debug=1")
		if rec.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
		}
		if got := rec.Header().Get("Content-Type"); !strings.HasPrefix(got, "text/plain") {
			t.Errorf("Content-Type = %q, want text/plain", got)
		}
	})

	t.Run("explicit handler", func(t *testing.T) {
		rec := serve(middleware, func(http.ResponseWriter, *http.Request) {}, http.MethodGet, "/admin/pprof/cmdline")
		if rec.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
		}
		if got := rec.Header().Get("Content-Type"); !strings.HasPrefix(got, "text/plain") {
			t.Errorf("Content-Type = %q, want text/plain", got)
		}
	})

	t.Run("default prefix falls through", func(t *testing.T) {
		called := false
		rec := serve(middleware, func(rw http.ResponseWriter, r *http.Request) {
			called = true
			rw.WriteHeader(http.StatusNoContent)
		}, http.MethodGet, "/debug/pprof/")

		if !called {
			t.Error("next handler was not called for the default prefix")
		}
		if rec.Code != http.StatusNoContent {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusNoContent)
		}
	})
}

func TestBuildConfigPrefix(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"", defaultPrefix},
		{"/admin/pprof", "/admin/pprof"},
		{"/admin/pprof/", "/admin/pprof"},
		{"admin/pprof", "/admin/pprof"},
		{" /admin/pprof/ ", "/admin/pprof"},
		{"/", "/"},
	}

	for _, c := range cases {
		if got := buildConfig(Config{Prefix: c.in}).Prefix; got != c.want {
			t.Errorf("buildConfig(%q).Prefix = %q, want %q", c.in, got, c.want)
		}
	}
}
