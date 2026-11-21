# Auth
Authentication middlewares for Ngamux.

This package provides:
- [Basic authentication](#basic-authentication)
- [Bearer authentication](#bearer-authentication)

# Usage

## Import
```go
import "github.com/ngamux/middleware/auth"
```

# Basic Authentication
## Instance With Default Config
```go
basicMiddleware := auth.Basic()
```

The default credential is `root:root`.
Passwords are compared in constant time.

## Instance With Custom Config
```go
basicMiddleware := auth.Basic(auth.BasicConfig{
	Realm: "My App",
	Creds: map[string]string{
		"admin": "secret",
	},
	Authorizer: func(username, password string) bool {
		return username == "admin" && password == "secret"
	},
	ErrorHandler: func(rw http.ResponseWriter, err error) {
		http.Error(rw, err.Error(), http.StatusUnauthorized)
	},
})
```

When `Realm` is set, the default error handler adds a `WWW-Authenticate` challenge header.

# Bearer Authentication
The token is read from the configured header, with an optional `Bearer ` prefix, and passed to `Authorizer`. Validation is required: when `Authorizer` is nil, every request is rejected with `403 Forbidden`.

## Instance With Config
```go
bearerMiddleware := auth.Bearer(auth.BearerConfig{
	Authorizer: func(token string) (any, error) {
		return jwt.Parse(token, func(t *jwt.Token) (any, error) {
			return []byte("secret"), nil
		})
	},
})
```

Defaults:
- Header: `authorization`
- Context key: `token`
- Rejection response: `403 Forbidden` with a JSON body

The value returned by `Authorizer` is stored in the request locals under `ContextKey`. Returning an error rejects the request through `ErrorHandler`.

## Other Options
- `Header` changes the request header. Default: `authorization`.
- `ContextKey` changes the key used in request locals. Default: `token`.
- `ErrorHandler` replaces the default rejection response.

A valid token is retrieved with:

```go
token := ngamux.Req(r).Locals("token")
```

# Mount Instance to Ngamux
### Global Middleware
```go
mux := ngamux.New()
mux.Use(basicMiddleware)
mux.Use(bearerMiddleware)
```

### Route Middleware
```go
mux := ngamux.New()
mux.Get("/", basicMiddleware(handler))
mux.Get("/private", bearerMiddleware(handler))
```
