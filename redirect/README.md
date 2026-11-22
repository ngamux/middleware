# Redirect
Middleware to proxy requests to upstream targets.

Despite the name, this middleware rewrites and proxies requests. It does not send HTTP redirects.

# Usage
## Import
```go
import "github.com/ngamux/middleware/redirect"
```

## Instance With Default Config
```go
redirectMiddleware := redirect.New()
```

With no rewrites configured, every request is passed to the next handler.

## Instance With Custom Config
```go
redirectMiddleware := redirect.New(redirect.Config{
	Rewrite: redirect.Rewrite{
		"/old": "http://localhost:8081/new",
	},
	Timeout:       5 * time.Second,
	PreserveHost:  false,
	FlushInterval: 0,
	ErrorHandler: func(rw http.ResponseWriter, r *http.Request, err error) {
		http.Error(rw, err.Error(), http.StatusBadGateway)
	},
})
```

## Configuration

- `Rewrite` maps exact request paths to upstream URLs. Invalid targets panic when the middleware is created.
- `Transport` overrides the HTTP transport. When nil, `http.DefaultTransport` is used.
- `Timeout` limits the whole upstream request when greater than zero.
- `PreserveHost` forwards the incoming `Host` header instead of using the target host.
- `FlushInterval` controls periodic flushing of streamed responses. `-1` flushes after every write.
- `ErrorHandler` replaces the default failure response. The default returns `502 Bad Gateway` for transport errors and `504 Gateway Timeout` for deadlines.

# Mount Instance to Ngamux
### Global Middleware
```go
mux := ngamux.New()
mux.Use(redirectMiddleware)
```

### Route Middleware
```go
mux := ngamux.New()
mux.Get("/", redirectMiddleware(handler))
```
