package ratelimit

import (
	"net"
	"net/http"
	"time"
)

type Config struct {
	Requests     int
	Window       time.Duration
	Burst        int
	KeyFunc      func(*http.Request) string
	ErrorHandler func(rw http.ResponseWriter, r *http.Request, retryAfter time.Duration)
}

const (
	defaultRequests = 60
	defaultWindow   = time.Minute
)

func defaultKeyFunc(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}

	return host
}

func buildConfig(config Config) Config {
	if config.Requests <= 0 {
		config.Requests = defaultRequests
	}

	if config.Window <= 0 {
		config.Window = defaultWindow
	}

	if config.Burst <= 0 {
		config.Burst = config.Requests
	}

	if config.KeyFunc == nil {
		config.KeyFunc = defaultKeyFunc
	}

	if config.ErrorHandler == nil {
		config.ErrorHandler = defaultErrorHandler
	}

	return config
}
