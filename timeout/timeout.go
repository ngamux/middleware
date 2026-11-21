package timeout

import (
	"bytes"
	"context"
	"net/http"
	"sync"

	"github.com/ngamux/ngamux"
)

type timeoutWriter struct {
	mu       sync.Mutex
	header   http.Header
	body     bytes.Buffer
	status   int
	wrote    bool
	timedOut bool
}

func newTimeoutWriter() *timeoutWriter {
	return &timeoutWriter{header: make(http.Header)}
}

func (tw *timeoutWriter) Header() http.Header {
	return tw.header
}

func (tw *timeoutWriter) WriteHeader(status int) {
	tw.mu.Lock()
	defer tw.mu.Unlock()

	if tw.timedOut || tw.wrote {
		return
	}

	tw.status = status
	tw.wrote = true
}

func (tw *timeoutWriter) Write(p []byte) (int, error) {
	tw.mu.Lock()
	defer tw.mu.Unlock()

	if tw.timedOut {
		return 0, http.ErrHandlerTimeout
	}

	if !tw.wrote {
		tw.status = http.StatusOK
		tw.wrote = true
	}

	return tw.body.Write(p)
}

func (tw *timeoutWriter) timeout() {
	tw.mu.Lock()
	defer tw.mu.Unlock()

	tw.timedOut = true
}

func (tw *timeoutWriter) commit(rw http.ResponseWriter) {
	tw.mu.Lock()
	defer tw.mu.Unlock()

	for key, values := range tw.header {
		rw.Header()[key] = values
	}

	if tw.wrote {
		rw.WriteHeader(tw.status)
	} else {
		rw.WriteHeader(http.StatusOK)
	}

	_, _ = rw.Write(tw.body.Bytes())
}

func defaultErrorHandler(rw http.ResponseWriter, _ *http.Request) {
	ngamux.Res(rw).Status(http.StatusServiceUnavailable).JSON(ngamux.Map{
		"error": "request timeout",
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
			ctx, cancel := context.WithTimeout(r.Context(), config.Timeout)
			defer cancel()

			tw := newTimeoutWriter()
			done := make(chan struct{})
			panicChan := make(chan any, 1)

			go func() {
				defer func() {
					if p := recover(); p != nil {
						panicChan <- p
					}
				}()

				next(tw, r.WithContext(ctx))
				close(done)
			}()

			select {
			case p := <-panicChan:
				panic(p)
			case <-done:
				tw.commit(rw)
			case <-ctx.Done():
				select {
				case <-done:
					tw.commit(rw)
				default:
					tw.timeout()
					config.ErrorHandler(rw, r)
				}
			}
		}
	}
}
