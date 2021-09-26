package static

import (
	"io/ioutil"
	"net/http"
	"path"
	"strings"

	"github.com/ngamux/ngamux"
)

func New(configs ...Config) func(next ngamux.Handler) ngamux.Handler {
	config := Config{}
	if len(configs) > 0 {
		config = configs[0]
	}

	cfg := makeConfig(config)

	_, err := ioutil.ReadDir(cfg.Root)
	if err != nil {
		panic("static middleware: " + err.Error())
	}
	return func(next ngamux.Handler) ngamux.Handler {
		return func(rw http.ResponseWriter, r *http.Request) error {
			isGetStatic := strings.HasPrefix(r.URL.Path, cfg.Prefix)
			if isGetStatic {
				filePath := path.Join(cfg.Root, strings.TrimPrefix(r.URL.Path, cfg.Prefix))
				http.ServeFile(rw, r, filePath)
				return nil
			}
			return next(rw, r)
		}
	}
}
