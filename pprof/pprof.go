package pprof

import (
	"net/http"
	"net/http/pprof"
)

func New(cfgs ...Config) func(http.HandlerFunc) http.HandlerFunc {
	cfg := Config{}
	if len(cfgs) > 0 {
		cfg = cfgs[0]
	}

	prefix := cfg.Prefix + "/debug"
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case prefix + "/pprof":
				pprof.Index(w, r)
			case prefix + "/allocs":
				pprof.Handler("allocs").ServeHTTP(w, r)
			case prefix + "/block":
				pprof.Handler("block").ServeHTTP(w, r)
			case prefix + "/cmdline":
				pprof.Cmdline(w, r)
			case prefix + "/goroutine":
				pprof.Handler("goroutine").ServeHTTP(w, r)
			case prefix + "/heap":
				pprof.Handler("heap").ServeHTTP(w, r)
			case prefix + "/mutex":
				pprof.Handler("mutex").ServeHTTP(w, r)
			case prefix + "/profile":
				pprof.Profile(w, r)
			case prefix + "/threadcreate":
				pprof.Handler("threadcreate").ServeHTTP(w, r)
			case prefix + "/trace":
				pprof.Trace(w, r)
			default:
				next(w, r)
			}
		}
	}
}
