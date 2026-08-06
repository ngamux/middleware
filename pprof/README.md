# PPROF
Middleware to show profiling data using [standard pprof](https://pkg.go.dev/net/http/pprof).

# Usage
## Import
```go
import "github.com/ngamux/middleware/pprof"
```

## Instance Creation
```go
pprofMiddleware := pprof.New()
```

## Mount Instance to Ngamux
### Global Middleware
```go
mux := ngamux.New()
mux.Use(pprofMiddleware)
```

### Test
Now send request to `/debug/pprof` path, it will shows profiling interface (web page).
