package pprof

import (
	"net/http"
	"net/http/pprof"
	"strings"
)

func New() func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if !strings.HasPrefix(r.URL.Path, "/debug/pprof") {
				next(w, r)
				return
			}

			switch r.URL.Path {
			case "/debug/pprof":
				http.Redirect(w, r, "/debug/pprof/", http.StatusMovedPermanently)
			case "/debug/pprof/cmdline":
				pprof.Cmdline(w, r)
			case "/debug/pprof/profile":
				pprof.Profile(w, r)
			case "/debug/pprof/trace":
				pprof.Trace(w, r)
			case "/debug/pprof/symbol":
				pprof.Symbol(w, r)
			default:
				pprof.Index(w, r)
			}
		}
	}
}
