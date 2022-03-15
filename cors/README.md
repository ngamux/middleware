# CORS
Middleware to manage [CORS](https://developer.mozilla.org/en-US/docs/Web/HTTP/CORS).

# Usage
## Instance With Default Config
```go
corsMiddleware := cors.New()
```

## Instance With Custom Config
```go
corsMiddleware := cors.New(cors.Config{
  AllowOrigins: "https://github.com, https://apps.github.com",
  AllowHeaders:  "Origin, Content-Type, Accept",
  AllowMethods:     "GET,POST,HEAD,PUT,DELETE,PATCH",
})
```

## Mount Instance to Ngamux
### Global Middleware
```go
mux := ngamux.New()
mux.Use(corsMiddleware)
```

### Route Middleware
```go
mux := ngamux.New()
mux.Get("/", corsMiddleware(handler))
```
or
```go
mux := ngamux.New()
mux.Get("/", ngamux.WithMiddlewares(corsMiddleware)(handler))
```
