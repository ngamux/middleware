package log

import "net/http"

type logResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func newLogResponseWriter(result *logResponseWriter, rw http.ResponseWriter) {
	*result = logResponseWriter{rw, http.StatusOK}
}

func (l *logResponseWriter) WriteHeader(status int) {
	l.statusCode = status
	l.ResponseWriter.WriteHeader(status)
}
