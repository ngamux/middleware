package pprof

import (
	"net/http"
	"net/http/pprof"
	"strings"

	"github.com/ngamux/ngamux"
)

func withPath(r *http.Request, path string) *http.Request {
	rewritten := r.Clone(r.Context())
	rewritten.URL.Path = path
	rewritten.URL.RawPath = ""

	return rewritten
}

// New returns a middleware that serves the standard net/http/pprof endpoints.
func New(configs ...Config) ngamux.MiddlewareFunc {
	var config Config
	if len(configs) > 0 {
		config = configs[0]
	}
	config = buildConfig(config)
	prefix := config.Prefix

	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(rw http.ResponseWriter, r *http.Request) {
			if prefix != "/" && r.URL.Path == prefix {
				target := prefix + "/"
				if r.URL.RawQuery != "" {
					target += "?" + r.URL.RawQuery
				}
				http.Redirect(rw, r, target, http.StatusMovedPermanently)
				return
			}

			var rest string
			var ok bool
			if prefix == "/" {
				rest = strings.TrimPrefix(r.URL.Path, "/")
				ok = true
			} else {
				rest, ok = strings.CutPrefix(r.URL.Path, prefix+"/")
			}
			if !ok {
				next(rw, r)
				return
			}

			if prefix != defaultPrefix {
				r = withPath(r, defaultPrefix+"/"+rest)
			}

			switch r.URL.Path {
			case "/debug/pprof/cmdline":
				pprof.Cmdline(rw, r)
			case "/debug/pprof/profile":
				pprof.Profile(rw, r)
			case "/debug/pprof/trace":
				pprof.Trace(rw, r)
			case "/debug/pprof/symbol":
				pprof.Symbol(rw, r)
			default:
				pprof.Index(rw, r)
			}
		}
	}
}
