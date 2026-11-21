# Compress
Middleware to compress HTTP responses with Brotli, zstd, gzip, or deflate.

The encoding is negotiated from the request's `Accept-Encoding` header. Compressed responses set `Content-Encoding`, remove `Content-Length`, and include `Vary: Accept-Encoding`.

# Usage
## Import
```go
import "github.com/ngamux/middleware/compress"
```

## Instance With Default Config
```go
compressMiddleware := compress.New()
```

## Instance With Custom Config
```go
compressMiddleware := compress.New(compress.Config{
	Level:       5,
	BrotliLevel: 6,
	ZstdLevel:   3,
	ContentTypes: []string{
		"text/",
		"application/json",
	},
})
```

Compression levels:

- `Level` is the gzip/deflate level. Zero or invalid values fall back to `gzip.DefaultCompression`.
- `BrotliLevel` is the Brotli quality. Zero or invalid values fall back to `brotli.DefaultCompression`.
- `ZstdLevel` is the zstd level from `1` to `22`. Zero or invalid values fall back to `3`.

When `Accept-Encoding` weights are equal, the preference order is Brotli, zstd, gzip, then deflate.

`ContentTypes` replaces the default allowlist when non-empty. Defaults:

- `text/*`
- `application/json`, `application/javascript`, `application/x-javascript`
- `application/xml`, `application/xhtml+xml`, `application/rss+xml`, `application/atom+xml`
- `application/manifest+json`, `application/ld+json`
- `image/svg+xml`
- `application/wasm`

## Mount Instance to Ngamux
### Global Middleware
```go
mux := ngamux.New()
mux.Use(compressMiddleware)
```

### Route Middleware
```go
mux := ngamux.New()
mux.Get("/", compressMiddleware(handler))
```

# Behavior
Responses are not compressed when:

- the client does not accept Brotli, zstd, gzip, or deflate
- the request uses `HEAD`, `Range`, or `Upgrade`
- the status is `1xx`, `204`, `205`, `206`, or `304`
- the handler already set `Content-Encoding`
- the response has `Cache-Control: no-transform`
- the content type is not in the allowlist

The middleware supports streaming: `http.Flusher` is forwarded, and `Unwrap` exposes the underlying response writer for `http.ResponseController`.

Compressor instances are pooled per middleware instance and reused across requests.
