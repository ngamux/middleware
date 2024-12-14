package redirect

import (
	"fmt"
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

	return func(next ngamux.Handler) ngamux.Handler {
		return func(rw http.ResponseWriter, r *http.Request) error {
			redirectTo, ok := config.Rewrite[r.URL.Path]
			if !ok {
				return next(rw, r)
			}

			req, err := http.NewRequest(r.Method, redirectTo, nil)
			if err != nil {
				return ngamux.Res(rw).Status(http.StatusInternalServerError).Text(err.Error())
			}

			client := &http.Client{}
			res, err := client.Do(req)
			if err != nil {
				fmt.Println(err)
				return ngamux.Res(rw).Status(res.StatusCode).Text(err.Error())
			}

			rw.Header().Set("content-type", res.Header.Get("content-type"))
			rw.WriteHeader(res.StatusCode)
			_, _ = io.Copy(rw, res.Body)
			defer res.Body.Close()
			return nil
		}
	}
}
