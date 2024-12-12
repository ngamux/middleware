package log

import (
	"log/slog"
	"net/http"
	"os"

	"github.com/ngamux/ngamux"
)

const (
	TagMethod string = "method"
	TagPath   string = "path"
	TagStatus string = "status"
)

func New(config ...Config) func(next ngamux.Handler) ngamux.Handler {
	cfg := configDefault()
	cfg.Handler = slog.NewJSONHandler(os.Stdout, nil)

	if len(config) > 0 {
		cfg = config[0]
	}

	lrw := new(logResponseWriter)
	logger := slog.New(cfg.Handler)
	return func(next ngamux.Handler) ngamux.Handler {
		return func(rw http.ResponseWriter, r *http.Request) error {
			newLogResponseWriter(lrw, rw)
			handle := next(lrw, r)
			logger.Info("", "method", r.Method, "path", r.URL.Path, "status", lrw.statusCode)
			return handle
		}
	}
}
