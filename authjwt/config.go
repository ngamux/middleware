package authjwt

import (
	"net/http"

	"github.com/golang-jwt/jwt/v4"
)

type Config struct {
	SigningKey   []byte
	ContextKey   string
	Header       string
	ErrorHandler func(rw http.ResponseWriter, err error) error
}

func makeConfig(config Config) Config {
	if config.ContextKey == "" {
		config.ContextKey = "token"
	}

	if config.Header == "" {
		config.Header = "authorization"
	}

	if config.ErrorHandler == nil {
		config.ErrorHandler = defaultErrorHandler
	}

	return config
}

func (config Config) keyFunc(t *jwt.Token) (interface{}, error) {
	return config.SigningKey, nil
}
