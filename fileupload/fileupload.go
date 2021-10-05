package fileupload

import (
	"github.com/ngamux/ngamux"
	"io"
	"net/http"
	"os"
)

func New(config Config) ngamux.MiddlewareFunc {
	return func(next ngamux.Handler) ngamux.Handler {
		return func(rw http.ResponseWriter, r *http.Request) error {
			err := r.ParseMultipartForm(config.MaxMemoryLimit)

			if err != nil {
				return err
			}

			file, _, err := r.FormFile(config.FormKey)

			if err != nil {
				return err
			}

			defer file.Close()

			filename, err := config.FilenameFunc(r)

			if err != nil {
				return err
			}

			_ = os.MkdirAll(config.Destination, 0700)

			destination, err := os.Create(config.Destination + string(os.PathSeparator) + filename)

			defer destination.Close()
			if err != nil {
				return err
			}

			_, err = io.Copy(destination, file)

			if err != nil {
				return err
			}

			return next(rw, r)
		}
	}
}

type Config struct {
	Destination    string
	FormKey        string
	FilenameFunc   func(request *http.Request) (string, error)
	MaxMemoryLimit int64
}
