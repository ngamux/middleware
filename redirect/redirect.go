package redirect

import (
	"io"
	"net/http"

	"github.com/ngamux/ngamux"
)

type Rewrite map[string]string

func New(configs ...Config) ngamux.MiddlewareFunc {
	var config Config
	if len(configs) > 0 {
		config = configs[0]
	}
	config = buildConfig(config)

	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(rw http.ResponseWriter, r *http.Request) {
			redirectTo, ok := config.Rewrite[r.URL.Path]
			if !ok {
				next(rw, r)
				return
			}

			res_ := ngamux.Res(rw)
			req, err := http.NewRequest(r.Method, redirectTo, nil)
			if err != nil {
				res_.Status(http.StatusInternalServerError).Text(err.Error())
				return
			}

			client := &http.Client{}
			res, err := client.Do(req)
			if err != nil {
				res_.Status(res.StatusCode).Text(err.Error())
				return
			}

			rw.Header().Set("content-type", res.Header.Get("content-type"))
			rw.WriteHeader(res.StatusCode)
			_, err = io.Copy(rw, res.Body)
			if err != nil {
				res_.Status(http.StatusInternalServerError).Text(err.Error())
				return
			}
			defer res.Body.Close()
		}
	}
}
