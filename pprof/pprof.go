package pprof

import (
	"net/http"
	"net/http/pprof"

	"github.com/ngamux/ngamux"
)

type inputNew interface {
	HandlerFunc(string, string, ngamux.Handler)
}

func New(mux inputNew, cfgs ...Config) func(ngamux.Handler) ngamux.Handler {
	cfg := Config{}
	if len(cfgs) > 0 {
		cfg = cfgs[0]
	}

	prefix := cfg.Prefix + "/debug"
	mux.HandlerFunc(http.MethodGet, prefix+"/pprof", func(w http.ResponseWriter, r *http.Request) error {
		pprof.Index(w, r)
		return nil
	})
	mux.HandlerFunc(http.MethodGet, prefix+"/allocs", func(w http.ResponseWriter, r *http.Request) error {
		pprof.Handler("allocs").ServeHTTP(w, r)
		return nil
	})
	mux.HandlerFunc(http.MethodGet, prefix+"/block", func(w http.ResponseWriter, r *http.Request) error {
		pprof.Handler("block").ServeHTTP(w, r)
		return nil
	})
	mux.HandlerFunc(http.MethodGet, prefix+"/cmdline", func(w http.ResponseWriter, r *http.Request) error {
		pprof.Cmdline(w, r)
		return nil
	})
	mux.HandlerFunc(http.MethodGet, prefix+"/goroutine", func(w http.ResponseWriter, r *http.Request) error {
		pprof.Handler("goroutine").ServeHTTP(w, r)
		return nil
	})
	mux.HandlerFunc(http.MethodGet, prefix+"/heap", func(w http.ResponseWriter, r *http.Request) error {
		pprof.Handler("heap").ServeHTTP(w, r)
		return nil
	})
	mux.HandlerFunc(http.MethodGet, prefix+"/mutex", func(w http.ResponseWriter, r *http.Request) error {
		pprof.Handler("mutex").ServeHTTP(w, r)
		return nil
	})
	mux.HandlerFunc(http.MethodGet, prefix+"/profile", func(w http.ResponseWriter, r *http.Request) error {
		pprof.Profile(w, r)
		return nil
	})
	mux.HandlerFunc(http.MethodGet, prefix+"/threadcreate", func(w http.ResponseWriter, r *http.Request) error {
		pprof.Handler("threadcreate").ServeHTTP(w, r)
		return nil
	})
	mux.HandlerFunc(http.MethodGet, prefix+"/trace", func(w http.ResponseWriter, r *http.Request) error {
		pprof.Trace(w, r)
		return nil
	})

	return func(next ngamux.Handler) ngamux.Handler {
		return func(rw http.ResponseWriter, r *http.Request) error {
			return next(rw, r)
		}

	}
}
