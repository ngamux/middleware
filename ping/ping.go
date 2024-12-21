package ping

import (
	"net/http"

	"github.com/ngamux/ngamux"
)

var configDefault = Config{
	Path: "/ping",
}

func New(config ...Config) func(next http.HandlerFunc) http.HandlerFunc {
	cfg := configDefault

	if len(config) > 0 {
		cfg = config[0]
	}

	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(rw http.ResponseWriter, r *http.Request) {
			if r.URL.Path == cfg.Path {
				ngamux.Res(rw).Text("pong")
				return
			}

			next(rw, r)
		}
	}
}
