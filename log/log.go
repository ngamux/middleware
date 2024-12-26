package log

import (
	"log/slog"
	"net/http"
	"os"
)

const (
	TagMethod string = "method"
	TagPath   string = "path"
	TagStatus string = "status"
)

func New(config ...Config) func(next http.HandlerFunc) http.HandlerFunc {
	cfg := configDefault()
	cfg.Handler = slog.NewJSONHandler(os.Stdout, nil)

	if len(config) > 0 {
		cfg = config[0]
	}

	logger := slog.New(cfg.Handler)
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(rw http.ResponseWriter, r *http.Request) {
			lrw := newLogResponseWriter(rw)
			next(lrw, r)
			logger.Info("", "method", r.Method, "path", r.URL.Path, "status", lrw.statusCode)
		}
	}
}
