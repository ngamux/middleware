# RequestID
Middleware to generates id for every incoming request.

# Usage
## Import
```go
import "github.com/ngamux/middleware/requestid"
```

## Instance With Default Config
```go
reqidMiddleware := requestid.New()
```

## Instance With Custom Config
```go
reqidMiddleware := requestid.New(
	requestid.WithKeyHeader(func() string { return "X-Request-ID" }),
	requestid.WithID(func() string {
		_id, _ := uuid.NewV7()
		return _id.String()
	}),
)
```

## Mount Instance to Ngamux
### Global Middleware
```go
mux := ngamux.New()
mux.Use(reqidMiddleware)
```

### Route Middleware
```go
mux := ngamux.New()
mux.Get("/", reqidMiddleware(handler))
```
or
```go
mux := ngamux.New()
mux.Get("/", ngamux.WithMiddlewares(reqidMiddleware)(handler))
```

