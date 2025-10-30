package pprof

import (
	"cmp"
	"net/http"
	"net/http/pprof"
	"path"
)

func New(cfgs ...Config) func(http.HandlerFunc) http.HandlerFunc {
	cfg := Config{}
	if len(cfgs) > 0 {
		cfg = cfgs[0]
	}
	cfg.Prefix = cmp.Or(cfg.Prefix, "/debug")

	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case path.Join(cfg.Prefix, "pprof"):
				pprof.Index(w, r)
			case path.Join(cfg.Prefix, "allocs"):
				pprof.Handler("allocs").ServeHTTP(w, r)
			case path.Join(cfg.Prefix, "block"):
				pprof.Handler("block").ServeHTTP(w, r)
			case path.Join(cfg.Prefix, "cmdline"):
				pprof.Cmdline(w, r)
			case path.Join(cfg.Prefix, "goroutine"):
				pprof.Handler("goroutine").ServeHTTP(w, r)
			case path.Join(cfg.Prefix, "heap"):
				pprof.Handler("heap").ServeHTTP(w, r)
			case path.Join(cfg.Prefix, "mutex"):
				pprof.Handler("mutex").ServeHTTP(w, r)
			case path.Join(cfg.Prefix, "profile"):
				pprof.Profile(w, r)
			case path.Join(cfg.Prefix, "threadcreate"):
				pprof.Handler("threadcreate").ServeHTTP(w, r)
			case path.Join(cfg.Prefix, "trace"):
				pprof.Trace(w, r)
			case path.Join(cfg.Prefix, "symbol"):
				pprof.Symbol(w, r)
			default:
				next(w, r)
			}
		}
	}
}
