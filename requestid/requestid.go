package requestid

import (
	"context"
	"net/http"
)

func New(opts ...func(*config)) func(http.HandlerFunc) http.HandlerFunc {
	c := &config{
		KeyHeader: cKeyHeader,
		ID:        cID,
		OnError:   func(err error) {},
	}
	for _, o := range opts {
		o(c)
	}

	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			key := c.KeyHeader()
			id := r.Header.Get(key)
			if id == "" {
				id = c.ID()
			}

			ctx := context.WithValue(r.Context(), keyHeader(c.KeyHeader()), id)

			r.Header.Set(key, id)
			w.Header().Set(key, id)
			next(w, r.WithContext(ctx))
		}
	}
}
