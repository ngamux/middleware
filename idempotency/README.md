# Idempotency
Middleware to make `POST` and `PATCH` requests safe to retry using an idempotency key.

The first request with a key runs normally. 
Its response is stored, and later requests with the same key, method, host, and URL receive the stored response without running the handler again.

# Usage
## Import
```go
import "github.com/ngamux/middleware/idempotency"
```

## Instance With Default Config
```go
idempotencyMiddleware := idempotency.New()
```

Defaults:

- Header: `Idempotency-Key`
- Methods: `POST`, `PATCH`
- TTL: 24 hours
- Missing keys pass through without idempotency handling

## Instance With Custom Config
```go
idempotencyMiddleware := idempotency.New(idempotency.Config{
	Header:   "X-Idempotency-Key",
	TTL:      12 * time.Hour,
	Methods:  []string{http.MethodPost, http.MethodPatch, http.MethodPut},
	Required: true,
	ErrorHandler: func(rw http.ResponseWriter, r *http.Request, err error) {
		http.Error(rw, err.Error(), http.StatusBadRequest)
	},
})
```

Set `Required` to reject requests that don't send a key.

## Custom Store
By default, responses are kept in an in-memory store. Provide `Store` to use another backend:

```go
idempotencyMiddleware := idempotency.New(idempotency.Config{
	Store: myStore,
})
```

```go
type Store interface {
	Begin(key string) (response *StoredResponse, state State)
	Complete(key string, response *StoredResponse)
	Abort(key string)
}
```

Implementations must be safe for concurrent use and must not evict a key while it is in progress. `TTL` only applies to the default `MemoryStore`.

A shared `MemoryStore` can be created with `idempotency.NewMemoryStore(ttl)`.

# Behavior
- Keys longer than 255 characters, or containing only whitespace, are rejected with `400 Bad Request`.
- A request whose key is still being processed receives `409 Conflict` with `Retry-After: 1`.
- Completed responses with status below `500` are stored.
- `5xx` responses are not stored, so the request can be retried.
- Handler panics clear the in-progress entry before the panic is re-raised.
- A stored response includes its status code, headers, and body.

The default store is in memory and local to each middleware instance. It is not shared across processes or servers unless a custom `Store` is provided.

Responses are buffered for replay, so this middleware is intended for bounded API responses rather than streaming endpoints.

# Mount Instance to Ngamux
### Global Middleware
```go
mux := ngamux.New()
mux.Use(idempotencyMiddleware)
```

### Route Middleware
```go
mux := ngamux.New()
mux.Post("/orders", idempotencyMiddleware(handler))
```
