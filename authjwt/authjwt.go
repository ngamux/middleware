package authjwt

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

func defaultErrorHandler(rw http.ResponseWriter, err error) error {
	return ngamux.Res(rw).Status(http.StatusForbidden).JSON(ngamux.Map{
		"error": err.Error(),
	})
}

func New(configs ...Config) func(next ngamux.Handler) ngamux.Handler {
	var config Config
	if len(configs) > 0 {
		config = configs[0]
	}
	config = makeConfig(config)

	return func(next ngamux.Handler) ngamux.Handler {
		return func(rw http.ResponseWriter, r *http.Request) error {
			authorizationHeader := r.Header.Get(config.Header)
			if authorizationHeader == "" {
				return config.ErrorHandler(rw, ErrorForbidden)
			}

			tokenString := strings.ReplaceAll(authorizationHeader, "Bearer ", "")
			token, err := jwt.Parse(tokenString, config.keyFunc)
			if err == nil && token.Valid {
				tmpR := ngamux.Req(r)
				tmpR.Locals(config.ContextKey, token)
				return next(rw, tmpR.Request)
			}

			return config.ErrorHandler(rw, err)
		}
	}
}
