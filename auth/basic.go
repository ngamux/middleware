package auth

import (
	"crypto/sha256"
	"crypto/subtle"
	"errors"
	"net/http"

	"github.com/ngamux/ngamux"
)

var (
	ErrorUnauthorized = errors.New("unauthorized")
)

// BasicConfig is a configuration used in Basic function for basic authentication middleware.
// If credentials, authorizer, or errorhandler is not provided, it uses the default one when passed as an argument to Basic function.
// For default crendetials, there is default root:root credential stored inside a map with this format `username=password`.
// You can also provide a realm name for the challenge authentication method.
type BasicConfig struct {
	Authorizer   func(username string, password string) bool
	ErrorHandler func(rw http.ResponseWriter, err error) error
	Realm        string
	Creds        map[string]string
}

func Basic(configs ...BasicConfig) ngamux.MiddlewareFunc {
	var config BasicConfig
	if len(configs) > 0 {
		config = configs[0]
	}
	config = makeBasicConfig(config)

	return func(next ngamux.Handler) ngamux.Handler {
		return func(rw http.ResponseWriter, r *http.Request) error {
			username, password, ok := r.BasicAuth()
			if !ok {
				return config.ErrorHandler(rw, ErrorUnauthorized)
			}

			if ok := config.Authorizer(username, password); !ok {
				return config.ErrorHandler(rw, ErrorUnauthorized)
			}

			return next(rw, r)
		}
	}
}

func makeBasicConfig(config BasicConfig) BasicConfig {
	if config.Creds == nil {
		config.Creds = map[string]string{
			"root": "root",
		}
	}

	if config.Authorizer == nil {
		config.Authorizer = config.defaultAuthorizer
	}

	if config.ErrorHandler == nil {
		config.ErrorHandler = config.defaultBasicErrorHandler
	}

	return config
}

func (c *BasicConfig) defaultAuthorizer(username string, password string) bool {
	credPass, ok := c.Creds[username]
	if !ok {
		return false
	}

	// hash the passwords so they have the same bytes length
	passwordHash := sha256.Sum256([]byte(credPass))
	expectedPasswordHash := sha256.Sum256([]byte(password))

	// avoid risk of timing attack using ConstantTimeCompare
	return subtle.ConstantTimeCompare(passwordHash[:], expectedPasswordHash[:]) == 1
}

func (c *BasicConfig) defaultBasicErrorHandler(rw http.ResponseWriter, err error) error {
	if c.Realm != "" {
		rw.Header().Add("WWW-Authenticate", "Basic realm="+c.Realm)
	}

	return ngamux.JSONWithStatus(rw, http.StatusUnauthorized, ngamux.Map{
		"error": err.Error(),
	})
}
