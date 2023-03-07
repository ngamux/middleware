package log

import (
	"bytes"
	"net/http"

	"github.com/ngamux/ngamux"
)

const (
	TagMethod string = "method"
	TagPath   string = "path"
	TagStatus string = "status"
)

func New(config ...Config) func(next ngamux.Handler) ngamux.Handler {
	cfg := &configDefault

	if len(config) > 0 {
		*cfg = config[0]
	}

	logString := new(bytes.Buffer)
	lrw := new(logResponseWriter)
	return func(next ngamux.Handler) ngamux.Handler {
		return func(rw http.ResponseWriter, r *http.Request) error {
			newLogResponseWriter(lrw, rw)
			handle := next(lrw, r)
			buildLog(logString, cfg.Format, lrw, r)
			println(logString.String())
			logString.Reset()
			return handle
		}
	}
}
