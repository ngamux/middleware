# PPROF
Middleware to expose profiling data using [standard pprof](https://pkg.go.dev/net/http/pprof).

> **Security:** pprof exposes runtime internals and can be used to trigger CPU and trace collection. Do not mount it publicly. Put it behind authentication or restrict it to a trusted network.

# Usage
## Import
```go
import "github.com/ngamux/middleware/pprof"
```

## Instance With Default Config
```go
pprofMiddleware := pprof.New()
```

The default prefix is `/debug/pprof`.

## Instance With Custom Prefix
```go
pprofMiddleware := pprof.New(pprof.Config{
	Prefix: "/admin/pprof",
})
```

## Mount Instance to Ngamux
### Global Middleware
```go
mux := ngamux.New()
mux.Use(pprofMiddleware)
```

### Route Middleware
```go
mux := ngamux.New()
mux.Get("/", pprofMiddleware(handler))
```

# Endpoints
With the default prefix:

| Path | Description |
| --- | --- |
| `/debug/pprof/` | Index |
| `/debug/pprof/cmdline` | Command line |
| `/debug/pprof/profile` | CPU profile |
| `/debug/pprof/symbol` | Symbol lookup |
| `/debug/pprof/trace` | Execution trace |
| `/debug/pprof/<profile>` | Named runtime profile |

The prefix is normalized to start with `/` and to have no trailing slash. `/debug/pprof` redirects to `/debug/pprof/`.
