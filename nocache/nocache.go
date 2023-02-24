package nocache

import (
	"net/http"
	"time"

	"github.com/ngamux/ngamux"
)

var epoch = time.Unix(0, 0).UTC().Format(http.TimeFormat)

func New() func(next ngamux.Handler) ngamux.Handler {
	return func(next ngamux.Handler) ngamux.Handler {
		return func(rw http.ResponseWriter, r *http.Request) error {
			rw.Header().Add("Cache-Control", "no-cache, private, max-age=0")
			rw.Header().Add("Expires", epoch)
			rw.Header().Add("Pragma", "no-cache")
			rw.Header().Add("X-Accel-Expires", "0")

			return next(rw, r)
		}
	}
}
