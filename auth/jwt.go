package auth

import (
	"errors"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v4"
	"github.com/ngamux/ngamux"
)

var (
	ErrorForbidden = errors.New("forbidden")
)

type JWTConfig struct {
	SigningKey   []byte
	ContextKey   string
	ErrorHandler func(rw http.ResponseWriter, err error) error
}

func defaultErrorHandler(rw http.ResponseWriter, err error) error {
	return ngamux.JSONWithStatus(rw, http.StatusForbidden, ngamux.Map{
		"error": err.Error(),
	})
}

func makeConfig(config JWTConfig) JWTConfig {
	if config.ContextKey == "" {
		config.ContextKey = "token"
	}

	if config.ErrorHandler == nil {
		config.ErrorHandler = defaultErrorHandler
	}

	return config
}

func (config JWTConfig) keyFunc(t *jwt.Token) (interface{}, error) {
	return config.SigningKey, nil
}

func JWT(configs ...JWTConfig) ngamux.MiddlewareFunc {
	var config JWTConfig
	if len(configs) > 0 {
		config = configs[0]
	}
	config = makeConfig(config)

	return func(next ngamux.Handler) ngamux.Handler {
		return func(rw http.ResponseWriter, r *http.Request) error {
			authorizationHeader := r.Header.Get("authorization")
			if authorizationHeader == "" {
				return config.ErrorHandler(rw, ErrorForbidden)
			}

			tokenString := strings.ReplaceAll(authorizationHeader, "Bearer ", "")
			token, err := jwt.Parse(tokenString, config.keyFunc)
			if err == nil && token.Valid {
				r = ngamux.SetContextValue(r, config.ContextKey, token)
				return next(rw, r)
			}

			return config.ErrorHandler(rw, err)
		}
	}
}
