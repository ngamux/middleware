# Timeout
Middleware to limit how long a request handler can take.

When the deadline is reached, the request context is canceled and the handler's response is discarded.
A `503 Service Unavailable` response is written instead.

# Usage
## Import
```go
import "github.com/ngamux/middleware/timeout"
```

## Instance With Default Config
```go
timeoutMiddleware := timeout.New()
```

The default timeout is 30 seconds.

## Instance With Custom Config
```go
timeoutMiddleware := timeout.New(timeout.Config{
	Timeout: 30 * time.Second,
	ErrorHandler: func(rw http.ResponseWriter, r *http.Request) {
		ngamux.Res(rw).Status(http.StatusServiceUnavailable).JSON(ngamux.Map{
			"error": "request timeout",
		})
	},
})
```

These are the built-in defaults; any field left at its zero value falls back to them.
With every field at its default, this is equivalent to `timeout.New()`.

# Mount Instance to Ngamux
### Global Middleware
```go
mux := ngamux.New()
mux.Use(timeoutMiddleware)
```

### Route Middleware
```go
mux := ngamux.New()
mux.Get("/", timeoutMiddleware(handler))
```
