package bodylimit

import (
	"errors"
	"net/http"

	"github.com/ngamux/ngamux"
)

var ErrorBodyTooLarge = errors.New("request body too large")

func New(configs ...Config) ngamux.MiddlewareFunc {
	var config Config
	if len(configs) > 0 {
		config = configs[0]
	}
	config = buildConfig(config)

	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(rw http.ResponseWriter, r *http.Request) {
			if r.ContentLength > config.Limit {
				config.ErrorHandler(rw, r, ErrorBodyTooLarge)
				return
			}

			if r.Body != nil {
				r.Body = http.MaxBytesReader(rw, r.Body, config.Limit)
			}

			next(rw, r)
		}
	}
}
