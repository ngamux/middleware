package bodylimit

import (
	"net/http"

	"github.com/ngamux/ngamux"
)

type Config struct {
	Limit        int64
	ErrorHandler func(rw http.ResponseWriter, r *http.Request, err error)
}

const defaultLimit int64 = 1 << 20

func buildConfig(config Config) Config {
	if config.Limit <= 0 {
		config.Limit = defaultLimit
	}

	if config.ErrorHandler == nil {
		config.ErrorHandler = defaultErrorHandler
	}

	return config
}

func defaultErrorHandler(rw http.ResponseWriter, _ *http.Request, err error) {
	ngamux.Res(rw).Status(http.StatusRequestEntityTooLarge).JSON(ngamux.Map{
		"error": err.Error(),
	})
}
