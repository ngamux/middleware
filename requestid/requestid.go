package requestid

import (
	"context"
	"fmt"
	"net/http"

	"github.com/ngamux/ngamux"
)

func New(opts ...func(*config)) ngamux.MiddlewareFunc {
	c := &config{
		KeyHeader: cKeyHeader,
		ID:        cID,
		OnError:   func(err error) {},
	}
	for _, o := range opts {
		o(c)
	}

	return func(next http.HandlerFunc) http.HandlerFunc {
		fmt.Println(123)
		return func(w http.ResponseWriter, r *http.Request) {
			key := c.KeyHeader()
			id := r.Header.Get(key)
			if id == "" {
				id = c.ID()
			}

			ctx := context.WithValue(r.Context(), c.KeyHeader(), id)

			r.Header.Set(key, id)
			w.Header().Set(key, id)
			next(w, r.WithContext(ctx))
		}
	}
}
