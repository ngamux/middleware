# Log
Middleware for logging request and response data.

# Usage

## Import
```go
import "github.com/ngamux/middleware/log"
```

## Instance With Default Config
```go
logMiddleware := log.New()
```

## Instance With Custom Config
```go
logMiddleware := log.New(log.Config{
  Format: "${method} ${path} ${status}",
})
```

## Mount to Ngamux
```go
mux := ngamux.New()
mux.Use(logMiddleware)
```
