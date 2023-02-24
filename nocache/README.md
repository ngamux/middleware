# Nocache
Middleware to set no cache headers.

It follows http://wiki.nginx.org/HttpProxyModule

# Usage

## Import
```go
import "github.com/ngamux/middleware/nocache"
```

## Instance
```go
nocacheMiddleware := nocache.New()
```

## Mount to Ngamux
```go
mux := ngamux.New()
mux.Use(nocacheMiddleware)
```
