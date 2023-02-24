# Ping
Middleware to add new URL path and send "pong" as response.

Usefull for checking the health of server.

# Usage

## Import
```go
import "github.com/ngamux/middleware/ping"
```

## Instance With Default Config
```go
pingMiddleware := ping.New()
```

## Instance With Custom Config
```go
pingMiddleware := ping.New(ping.Config{
	Path: "/check",
})
```

## Mount to Ngamux
```go
mux := ngamux.New()
mux.Use(pingMiddleware)
```
