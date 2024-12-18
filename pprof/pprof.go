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
	mux.HandlerFunc(http.MethodGet, prefix+"/pprof", func(w http.ResponseWriter, r *http.Request) {
		pprof.Index(w, r)
	})
	mux.HandlerFunc(http.MethodGet, prefix+"/allocs", func(w http.ResponseWriter, r *http.Request) {
		pprof.Handler("allocs").ServeHTTP(w, r)
	})
	mux.HandlerFunc(http.MethodGet, prefix+"/block", func(w http.ResponseWriter, r *http.Request) {
		pprof.Handler("block").ServeHTTP(w, r)
	})
	mux.HandlerFunc(http.MethodGet, prefix+"/cmdline", func(w http.ResponseWriter, r *http.Request) {
		pprof.Cmdline(w, r)
	})
	mux.HandlerFunc(http.MethodGet, prefix+"/goroutine", func(w http.ResponseWriter, r *http.Request) {
		pprof.Handler("goroutine").ServeHTTP(w, r)
	})
	mux.HandlerFunc(http.MethodGet, prefix+"/heap", func(w http.ResponseWriter, r *http.Request) {
		pprof.Handler("heap").ServeHTTP(w, r)
	})
	mux.HandlerFunc(http.MethodGet, prefix+"/mutex", func(w http.ResponseWriter, r *http.Request) {
		pprof.Handler("mutex").ServeHTTP(w, r)
	})
	mux.HandlerFunc(http.MethodGet, prefix+"/profile", func(w http.ResponseWriter, r *http.Request) {
		pprof.Profile(w, r)
	})
	mux.HandlerFunc(http.MethodGet, prefix+"/threadcreate", func(w http.ResponseWriter, r *http.Request) {
		pprof.Handler("threadcreate").ServeHTTP(w, r)
	})
	mux.HandlerFunc(http.MethodGet, prefix+"/trace", func(w http.ResponseWriter, r *http.Request) {
		pprof.Trace(w, r)
	})

	return func(next ngamux.Handler) ngamux.Handler {
		return func(rw http.ResponseWriter, r *http.Request) {
			next(rw, r)
		}
	}
}
