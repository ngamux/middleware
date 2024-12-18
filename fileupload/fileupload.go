package fileupload

import (
	"io"
	"net/http"
	"os"

	"github.com/ngamux/ngamux"
)

func New(config Config) ngamux.MiddlewareFunc {
	return func(next ngamux.Handler) ngamux.Handler {
		return func(rw http.ResponseWriter, r *http.Request) {
			res := ngamux.Res(rw)
			err := r.ParseMultipartForm(config.MaxMemoryLimit)

			if err != nil {
				res.Status(http.StatusBadRequest).Text(err.Error())
				return
			}

			file, _, err := r.FormFile(config.FormKey)

			if err != nil {
				res.Status(http.StatusBadRequest).Text(err.Error())
				return
			}

			defer file.Close()

			filename, err := config.FilenameFunc(r)
			if err != nil {
				res.Status(http.StatusBadRequest).Text(err.Error())
				return
			}

			err = os.MkdirAll(config.Destination, 0700)
			if err != nil {
				res.Status(http.StatusBadRequest).Text(err.Error())
				return
			}

			destination, err := os.Create(config.Destination + string(os.PathSeparator) + filename)
			if err != nil {
				res.Status(http.StatusBadRequest).Text(err.Error())
				return
			}

			defer destination.Close()
			if err != nil {
				res.Status(http.StatusBadRequest).Text(err.Error())
				return
			}

			_, err = io.Copy(destination, file)
			if err != nil {
				res.Status(http.StatusBadRequest).Text(err.Error())
				return
			}

			next(rw, r)
		}
	}
}

type Config struct {
	Destination    string
	FormKey        string
	FilenameFunc   func(request *http.Request) (string, error)
	MaxMemoryLimit int64
}
