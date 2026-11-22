# RealIP
Middleware to resolve the real client IP from proxy headers.

It rewrites `r.RemoteAddr` while preserving the original port, so middlewares such as [`ratelimit`](../ratelimit) and [`log`](../log) see the client address instead of the proxy address.

# Usage
## Import
```go
import "github.com/ngamux/middleware/realip"
```

## Instance With Default Config
```go
realIPMiddleware := realip.New()
```

With the default config, proxy headers are trusted from any peer.
Use `TrustedProxies` when the server is reachable directly as well.

## Instance With Trusted Proxies
```go
realIPMiddleware := realip.New(realip.Config{
	TrustedProxies: []string{
		"10.0.0.0/8",
		"192.168.1.0/24",
		"2001:db8::/32",
	},
})
```

`TrustedProxies` accepts single IP addresses and CIDR ranges. 
Invalid entries are ignored. 
If the list is non-empty but contains only invalid entries, no peer is trusted and proxy headers are ignored.

# Behavior
Headers are checked in this order:

1. `X-Forwarded-For`
2. `X-Real-IP`
3. `Forwarded` (RFC 7239)

When `TrustedProxies` is configured:

- Headers are ignored unless the direct peer is a trusted proxy.
- `X-Forwarded-For` is walked from right to left, skipping trusted proxies, so a spoofed leftmost value is not trusted.
- If every address in the chain is a trusted proxy, no client address can be identified and `RemoteAddr` is left unchanged.

When `TrustedProxies` is empty, the leftmost valid address in the header is used.

# Mount Instance to Ngamux
### Global Middleware
```go
mux := ngamux.New()
mux.Use(realIPMiddleware)
```

### Route Middleware
```go
mux := ngamux.New()
mux.Get("/", realIPMiddleware(handler))
```
