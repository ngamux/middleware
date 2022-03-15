# Recover
Middleware to recover while panic.

# Usage

## Import
```go
import "github.com/ngamux/middleware/recover"
```

## Instance With Default Config
```go
recoverMiddleware := recover.New()
```

## Instance With Custom Config
```go
recoverMiddleware := recover.New(recover.Config{
  ErrorHandler: func(rw http.ResponseWriter, r *http.Request, e error) {
		rw.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintln(rw, e)
		log.Println("error:", e)
	},
})
```

## Mount to Ngamux
```go
mux := ngamux.New()
mux.Use(recoverMiddleware)
```
