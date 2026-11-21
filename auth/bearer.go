package auth

import (
	"errors"
	"net/http"
	"strings"

	"github.com/ngamux/ngamux"
)

var (
	ErrorForbidden = errors.New("forbidden")
)

// BearerConfig is a configuration used in Bearer function for bearer token authentication middleware.
// ContextKey is the key used to store the validated value in the request locals and defaults to "token".
// Header is the request header containing the token and defaults to "authorization".
// Authorizer validates the token and returns the value stored in the request locals. It receives
// the token with the "Bearer " prefix removed. When nil, every request is rejected.
// When ErrorHandler is nil, the middleware responds with 403 Forbidden.
type BearerConfig struct {
	ContextKey   string
	Header       string
	Authorizer   func(token string) (any, error)
	ErrorHandler func(rw http.ResponseWriter, err error)
}

func makeBearerConfig(config BearerConfig) BearerConfig {
	if config.ContextKey == "" {
		config.ContextKey = "token"
	}

	if config.Header == "" {
		config.Header = "authorization"
	}

	if config.Authorizer == nil {
		config.Authorizer = rejectAuthorizer
	}

	if config.ErrorHandler == nil {
		config.ErrorHandler = defaultBearerErrorHandler
	}

	return config
}

func rejectAuthorizer(string) (any, error) {
	return nil, ErrorForbidden
}

func defaultBearerErrorHandler(rw http.ResponseWriter, err error) {
	ngamux.Res(rw).Status(http.StatusForbidden).JSON(ngamux.Map{
		"error": err.Error(),
	})
}

// Bearer returns a middleware that authenticates requests using a bearer token.
// The token is read from the configured header, with an optional "Bearer "
// prefix, and validated by Authorizer. The validated value is stored in the
// request locals under ContextKey.
func Bearer(configs ...BearerConfig) ngamux.MiddlewareFunc {
	var config BearerConfig
	if len(configs) > 0 {
		config = configs[0]
	}
	config = makeBearerConfig(config)

	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(rw http.ResponseWriter, r *http.Request) {
			authorizationHeader := r.Header.Get(config.Header)
			if authorizationHeader == "" {
				config.ErrorHandler(rw, ErrorForbidden)
				return
			}

			const bearerPrefix = "Bearer "
			tokenString := authorizationHeader
			if len(authorizationHeader) > len(bearerPrefix) && strings.EqualFold(authorizationHeader[:len(bearerPrefix)], bearerPrefix) {
				tokenString = authorizationHeader[len(bearerPrefix):]
			}

			value, err := config.Authorizer(tokenString)
			if err == nil {
				tmpR := ngamux.Req(r)
				tmpR.Locals(config.ContextKey, value)
				next(rw, tmpR.Request)
				return
			}

			config.ErrorHandler(rw, err)
		}
	}
}
