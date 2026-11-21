# RateLimit
Middleware to limit the number of requests per client.

Requests are tracked in memory using a [token bucket](https://en.wikipedia.org/wiki/Token_bucket) per key.
A client can burst up to `Burst` requests, then keeps receiving tokens at a rate of `Requests` per `Window`.

# Usage
## Import
```go
import "github.com/ngamux/middleware/ratelimit"
```

## Instance With Default Config
```go
ratelimitMiddleware := ratelimit.New()
```

The default is 60 requests per minute per client IP, with a burst of 60.

## Instance With Custom Config
```go
ratelimitMiddleware := ratelimit.New(ratelimit.Config{
	Requests: 60,
	Window:   time.Minute,
	Burst:    60,
	KeyFunc: func(r *http.Request) string {
		host, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			return r.RemoteAddr
		}
		return host
	},
	ErrorHandler: func(rw http.ResponseWriter, _ *http.Request, retryAfter time.Duration) {
		seconds := int(math.Ceil(retryAfter.Seconds()))
		if seconds < 1 {
			seconds = 1
		}
		rw.Header().Set("Retry-After", strconv.Itoa(seconds))
		ngamux.Res(rw).Status(http.StatusTooManyRequests).JSON(ngamux.Map{
			"error": "too many requests",
		})
	},
})
```

These are the built-in defaults; any field left at its zero value falls back to them. With every field at its default, this is equivalent to `ratelimit.New()`.

# Response Headers
Every response includes:

| Header | Description |
| --- | --- |
| `X-RateLimit-Limit` | Requests allowed per window |
| `X-RateLimit-Remaining` | Tokens remaining for the client |
| `X-RateLimit-Reset` | Unix time when the client's bucket is full |
| `Retry-After` | Only for rejected requests, in seconds |

# Mount Instance to Ngamux
### Global Middleware
```go
mux := ngamux.New()
mux.Use(ratelimitMiddleware)
```

### Route Middleware
```go
mux := ngamux.New()
mux.Get("/", ratelimitMiddleware(handler))
```
