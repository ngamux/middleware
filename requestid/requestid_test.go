package requestid

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang-must/must"
	"github.com/ngamux/ngamux"
)

func TestNew(t *testing.T) {
	cases := []struct {
		name   string
		config *config
		output ngamux.MiddlewareFunc
		check  func(*testing.T, *config, http.ResponseWriter, *http.Request)
	}{
		{
			"positive",
			&config{},
			New(),
			func(t *testing.T, c *config, w http.ResponseWriter, r *http.Request) {
				must.NotEqual(t, "", w.Header().Get(cKeyHeader()))
				must.NotEqual(t, "", r.Header.Get(cKeyHeader()))
				must.NotNil(t, r.Context().Value(keyHeader(cKeyHeader())))
			},
		},
		{
			"positive with key header",
			&config{KeyHeader: func() string { return "X-Reqid" }},
			New(WithKeyHeader(func() string { return "X-Reqid" })),
			func(t *testing.T, c *config, w http.ResponseWriter, r *http.Request) {
				must.NotEqual(t, "", w.Header().Get(c.KeyHeader()))
				must.NotEqual(t, "", r.Header.Get(c.KeyHeader()))
				must.NotNil(t, r.Context().Value(keyHeader(c.KeyHeader())))
			},
		},
		{
			"positive with id",
			&config{ID: func() string { return "123" }},
			New(WithID(func() string { return "123" })),
			func(t *testing.T, c *config, w http.ResponseWriter, r *http.Request) {
				must.Equal(t, c.ID(), w.Header().Get(cKeyHeader()))
				must.Equal(t, c.ID(), r.Header.Get(cKeyHeader()))
				must.Equal(t, c.ID(), r.Context().Value(keyHeader(cKeyHeader())))
			},
		},
		{
			"negative with empty key header",
			&config{},
			New(WithKeyHeader(nil)),
			func(t *testing.T, c *config, w http.ResponseWriter, r *http.Request) {
				must.NotEqual(t, "", w.Header().Get(cKeyHeader()))
				must.NotEqual(t, "", r.Header.Get(cKeyHeader()))
				must.NotNil(t, r.Context().Value(keyHeader(cKeyHeader())))
			},
		},
		{
			"negative with empty key header",
			&config{},
			New(WithID(nil)),
			func(t *testing.T, c *config, w http.ResponseWriter, r *http.Request) {
				must.NotEqual(t, "", w.Header().Get(cKeyHeader()))
				must.NotEqual(t, "", r.Header.Get(cKeyHeader()))
				must.NotNil(t, r.Context().Value(keyHeader(cKeyHeader())))
			},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			r, _ := http.NewRequest("GET", "", nil)
			c.output(func(_ http.ResponseWriter, rr *http.Request) {
				r = rr
			})(w, r)
			c.check(t, c.config, w, r)
		})
	}
}
