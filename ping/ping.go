package ping

import (
	"net/http"

	"github.com/ngamux/ngamux"
)

var configDefault = Config{
	Path: "/ping",
}

func New(config ...Config) func(next ngamux.Handler) ngamux.Handler {
	cfg := configDefault

	if len(config) > 0 {
		cfg = config[0]
	}

	return func(next ngamux.Handler) ngamux.Handler {
		return func(rw http.ResponseWriter, r *http.Request) {
			if r.URL.Path == cfg.Path {
				ngamux.Res(rw).Text("pong")
				return
			}

			next(rw, r)
		}
	}
}
