package nocache

import (
	"net/http"
	"time"
)

var epoch = time.Unix(0, 0).UTC().Format(http.TimeFormat)

func New() func(next http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(rw http.ResponseWriter, r *http.Request) {
			rw.Header().Add("Cache-Control", "no-cache, private, max-age=0")
			rw.Header().Add("Expires", epoch)
			rw.Header().Add("Pragma", "no-cache")
			rw.Header().Add("X-Accel-Expires", "0")

			next(rw, r)
		}
	}
}
