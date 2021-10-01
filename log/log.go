package log

import (
	"fmt"
	"net/http"

	"github.com/ngamux/ngamux"
)

const (
	TagMethod string = "method"
	TagPath   string = "path"
	TagStatus string = "status"
)

func New(config ...Config) func(next ngamux.Handler) ngamux.Handler {
	cfg := configDefault

	if len(config) > 0 {
		cfg = config[0]
	}

	return func(next ngamux.Handler) ngamux.Handler {
		return func(rw http.ResponseWriter, r *http.Request) error {
			lrw := newLogResponseWriter(rw)
			handle := next(lrw, r)
			logString := buildLog(cfg, lrw, r)
			fmt.Println(logString)
			return handle
		}
	}
}
