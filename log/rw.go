package log

import "net/http"

type logResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func newLogResponseWriter(rw http.ResponseWriter) *logResponseWriter {
	return &logResponseWriter{rw, http.StatusOK}
}

func (l *logResponseWriter) WriteHeader(status int) {
	l.statusCode = status
	l.ResponseWriter.WriteHeader(status)
}
