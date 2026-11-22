# BodyLimit
Middleware to limit the size of request bodies.

# Usage
## Import
```go
import "github.com/ngamux/middleware/bodylimit"
```

## Instance With Default Config
```go
bodyLimitMiddleware := bodylimit.New()
```

The default limit is 1 MiB.

## Instance With Custom Config
```go
bodyLimitMiddleware := bodylimit.New(bodylimit.Config{
	Limit: 5 << 20,
	ErrorHandler: func(rw http.ResponseWriter, r *http.Request, err error) {
		http.Error(rw, err.Error(), http.StatusRequestEntityTooLarge)
	},
})
```

`Limit` is the maximum number of bytes accepted in a request body. Zero or negative values fall back to `1 << 20` (1 MiB).

The default error handler responds with `413 Request Entity Too Large` and a JSON body:

```json
{"error":"request body too large"}
```

# Behavior
- If `Content-Length` is already above the limit, the request is rejected before the handler runs.
- Otherwise, the body is wrapped with `http.MaxBytesReader`. Handlers reading past the limit receive an `*http.MaxBytesError`.
- Requests with unknown content length are enforced while reading.

# Mount Instance to Ngamux
### Global Middleware
```go
mux := ngamux.New()
mux.Use(bodyLimitMiddleware)
```

### Route Middleware
```go
mux := ngamux.New()
mux.Post("/upload", bodyLimitMiddleware(handler))
```
