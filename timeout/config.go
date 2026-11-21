package timeout

import (
	"net/http"
	"time"
)

type Config struct {
	Timeout      time.Duration
	ErrorHandler func(rw http.ResponseWriter, r *http.Request)
}

const defaultTimeout = 30 * time.Second

func buildConfig(config Config) Config {
	if config.Timeout <= 0 {
		config.Timeout = defaultTimeout
	}

	if config.ErrorHandler == nil {
		config.ErrorHandler = defaultErrorHandler
	}

	return config
}
