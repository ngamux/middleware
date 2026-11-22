package idempotency

import (
	"bytes"
	"errors"
	"net/http"
	"strings"

	"github.com/ngamux/ngamux"
)

var (
	ErrorKeyRequired = errors.New("idempotency key required")
	ErrorKeyInvalid  = errors.New("invalid idempotency key")
	ErrorInProgress  = errors.New("request with this idempotency key is in progress")
)

type responseRecorder struct {
	http.ResponseWriter
	status int
	body   bytes.Buffer
}

func newResponseRecorder(rw http.ResponseWriter) *responseRecorder {
	return &responseRecorder{ResponseWriter: rw}
}

func (r *responseRecorder) WriteHeader(status int) {
	if r.status != 0 {
		return
	}

	r.ResponseWriter.WriteHeader(status)
	if status >= 100 && status < 200 {
		return
	}

	r.status = status
}

func (r *responseRecorder) Write(p []byte) (int, error) {
	if r.status == 0 {
		r.WriteHeader(http.StatusOK)
	}

	written, err := r.ResponseWriter.Write(p)
	if written > 0 {
		_, _ = r.body.Write(p[:written])
	}

	return written, err
}

func (r *responseRecorder) Flush() {
	if flusher, ok := r.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (r *responseRecorder) Unwrap() http.ResponseWriter {
	return r.ResponseWriter
}

func (r *responseRecorder) snapshot() *StoredResponse {
	status := r.status
	if status == 0 {
		status = http.StatusOK
	}

	header := make(http.Header, len(r.Header()))
	for key, values := range r.Header() {
		header[key] = append([]string(nil), values...)
	}

	body := make([]byte, r.body.Len())
	copy(body, r.body.Bytes())

	return &StoredResponse{
		Status: status,
		Header: header,
		Body:   body,
	}
}

func writeStored(rw http.ResponseWriter, response *StoredResponse) {
	for key, values := range response.Header {
		rw.Header()[key] = append([]string(nil), values...)
	}

	rw.WriteHeader(response.Status)
	_, _ = rw.Write(response.Body)
}

func cacheKey(r *http.Request, key string) string {
	return r.Host + "\x00" + r.Method + "\x00" + r.URL.RequestURI() + "\x00" + key
}

func defaultErrorHandler(rw http.ResponseWriter, _ *http.Request, err error) {
	status := http.StatusBadRequest
	if errors.Is(err, ErrorInProgress) {
		status = http.StatusConflict
		rw.Header().Set("Retry-After", "1")
	}

	ngamux.Res(rw).Status(status).JSON(ngamux.Map{
		"error": err.Error(),
	})
}

func New(configs ...Config) ngamux.MiddlewareFunc {
	var config Config
	if len(configs) > 0 {
		config = configs[0]
	}
	config = buildConfig(config)

	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(rw http.ResponseWriter, r *http.Request) {
			if !config.allowsMethod(r.Method) {
				next(rw, r)
				return
			}

			key := r.Header.Get(config.Header)
			if key == "" {
				if config.Required {
					config.ErrorHandler(rw, r, ErrorKeyRequired)
					return
				}

				next(rw, r)
				return
			}

			if len(key) > maxKeyLength || strings.TrimSpace(key) == "" {
				config.ErrorHandler(rw, r, ErrorKeyInvalid)
				return
			}

			storageKey := cacheKey(r, key)
			stored, state := config.Store.Begin(storageKey)
			switch state {
			case StateFound:
				writeStored(rw, stored)
				return
			case StateInProgress:
				config.ErrorHandler(rw, r, ErrorInProgress)
				return
			}

			completed := false
			defer func() {
				if completed {
					return
				}

				config.Store.Abort(storageKey)
				if recovered := recover(); recovered != nil {
					panic(recovered)
				}
			}()

			recorder := newResponseRecorder(rw)
			next(recorder, r)

			if recorder.status >= http.StatusInternalServerError {
				return
			}

			config.Store.Complete(storageKey, recorder.snapshot())
			completed = true
		}
	}
}
