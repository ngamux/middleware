# Ping
Middleware to add new URL path and send "pong" as response.

Usefull for checking the health of server.

# Usage

## Import
```go
import "github.com/ngamux/middleware/authjwt"
```

## Instance With Default Config
```go
authjwtMiddleware := authjwt.New()
```

## Instance With Custom Config
```go
authjwtMiddleware := authjwt.New(authjwt.Config{
	SigningKey: []byte("123123123"),
	Header:     "X-API-Key",
})
```

## Mount to Ngamux
```go
mux := ngamux.New()
mux.Use(authjwtMiddleware)
```
