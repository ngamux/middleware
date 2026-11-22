package idempotency

import (
	"net/http"
	"strings"
	"time"
)

const (
	defaultHeader = "Idempotency-Key"
	defaultTTL    = 24 * time.Hour
	maxKeyLength  = 255
)

// Config is a configuration used in New for idempotency middleware.
//
// Header is the request header containing the idempotency key and defaults to
// "Idempotency-Key".
//
// TTL is how long completed responses are kept for replay and defaults to 24
// hours. It is only used by the default in-memory store.
//
// Methods is the list of HTTP methods handled by the middleware. When empty,
// POST and PATCH are used.
//
// Required rejects requests without a key. When false, requests without a key
// pass through without idempotency handling.
//
// Store is the storage backend. When nil, an in-memory store using TTL is
// created.
//
// ErrorHandler writes rejection responses. When nil, the middleware responds
// with 400 for invalid keys and 409 for requests whose key is still in progress.
type Config struct {
	Header       string
	TTL          time.Duration
	Methods      []string
	Required     bool
	Store        Store
	ErrorHandler func(rw http.ResponseWriter, r *http.Request, err error)
}

func buildConfig(config Config) Config {
	if config.Header == "" {
		config.Header = defaultHeader
	}

	if config.TTL <= 0 {
		config.TTL = defaultTTL
	}

	if config.Store == nil {
		config.Store = NewMemoryStore(config.TTL)
	}

	if len(config.Methods) == 0 {
		config.Methods = []string{http.MethodPost, http.MethodPatch}
	}

	if config.ErrorHandler == nil {
		config.ErrorHandler = defaultErrorHandler
	}

	return config
}

func (c Config) allowsMethod(method string) bool {
	for _, allowed := range c.Methods {
		if strings.EqualFold(strings.TrimSpace(allowed), method) {
			return true
		}
	}

	return false
}
