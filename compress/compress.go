package compress

import (
	"bufio"
	"compress/gzip"
	"compress/zlib"
	"errors"
	"io"
	"mime"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/andybalholm/brotli"
	"github.com/klauspost/compress/zstd"
	"github.com/ngamux/ngamux"
)

const varyAcceptEncoding = "Accept-Encoding"

var supportedEncodings = []string{"br", "zstd", "gzip", "deflate"}

type compressor interface {
	io.WriteCloser
	Flush() error
	Reset(io.Writer)
}

type compressResponseWriter struct {
	http.ResponseWriter

	encoding string
	types    []string
	pool     *sync.Pool

	encoder     compressor
	status      int
	started     bool
	wroteHeader bool
	compress    bool
}

func newCompressorPool(encoding string, config Config) *sync.Pool {
	return &sync.Pool{
		New: func() any {
			switch encoding {
			case "br":
				return brotli.NewWriterLevel(io.Discard, config.BrotliLevel)
			case "zstd":
				writer, err := zstd.NewWriter(
					io.Discard,
					zstd.WithEncoderLevel(zstd.EncoderLevelFromZstd(config.ZstdLevel)),
					zstd.WithEncoderConcurrency(1),
				)
				if err != nil {
					return nil
				}
				return writer
			case "gzip":
				writer, _ := gzip.NewWriterLevel(io.Discard, config.Level)
				return writer
			case "deflate":
				writer, _ := zlib.NewWriterLevel(io.Discard, config.Level)
				return writer
			default:
				return nil
			}
		},
	}
}

func (cw *compressResponseWriter) WriteHeader(status int) {
	if cw.started || cw.wroteHeader {
		return
	}

	cw.status = status
	cw.wroteHeader = true
}

func (cw *compressResponseWriter) Write(p []byte) (int, error) {
	if !cw.wroteHeader {
		cw.status = http.StatusOK
		cw.wroteHeader = true
	}

	if !cw.started {
		cw.sniffContentType(p)
		cw.start(len(p) > 0)
	}

	if cw.compress {
		return cw.encoder.Write(p)
	}

	return cw.ResponseWriter.Write(p)
}

func (cw *compressResponseWriter) Flush() {
	if !cw.started {
		cw.start(true)
	}

	if cw.encoder != nil {
		_ = cw.encoder.Flush()
	}

	if flusher, ok := cw.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (cw *compressResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if cw.encoder != nil {
		return nil, nil, errors.New("compress: cannot hijack a compressed response")
	}

	hijacker, ok := cw.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, errors.New("compress: underlying response writer does not support hijacking")
	}

	return hijacker.Hijack()
}

func (cw *compressResponseWriter) Unwrap() http.ResponseWriter {
	return cw.ResponseWriter
}

func (cw *compressResponseWriter) Close() error {
	if !cw.started {
		if !cw.wroteHeader {
			return nil
		}
		cw.start(false)
	}

	if cw.encoder == nil {
		return nil
	}

	err := cw.encoder.Close()
	cw.encoder.Reset(io.Discard)
	cw.pool.Put(cw.encoder)
	cw.encoder = nil

	return err
}

func (cw *compressResponseWriter) sniffContentType(p []byte) {
	if len(p) == 0 || cw.Header().Get("Content-Type") != "" {
		return
	}

	cw.Header().Set("Content-Type", http.DetectContentType(p))
}

func (cw *compressResponseWriter) start(hasBody bool) {
	if cw.started {
		return
	}
	cw.started = true

	if !cw.wroteHeader {
		cw.status = http.StatusOK
		cw.wroteHeader = true
	}

	header := cw.Header()
	var encoder compressor
	if hasBody && cw.shouldCompress(header) {
		encoder, _ = cw.pool.Get().(compressor)
	}

	if encoder != nil {
		header.Set("Content-Encoding", cw.encoding)
		header.Del("Content-Length")
		cw.compress = true
		cw.encoder = encoder
	}

	cw.ResponseWriter.WriteHeader(cw.status)

	if cw.encoder != nil {
		cw.encoder.Reset(cw.ResponseWriter)
	}
}

func (cw *compressResponseWriter) shouldCompress(header http.Header) bool {
	switch cw.status {
	case http.StatusNoContent, http.StatusResetContent, http.StatusPartialContent, http.StatusNotModified:
		return false
	}

	if cw.status < http.StatusOK {
		return false
	}

	if header.Get("Content-Encoding") != "" {
		return false
	}

	if hasNoTransform(header.Get("Cache-Control")) {
		return false
	}

	return isCompressibleContentType(header.Get("Content-Type"), cw.types)
}

func isCompressibleContentType(contentType string, types []string) bool {
	if contentType == "" {
		return false
	}

	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		mediaType = contentType
		if index := strings.IndexByte(mediaType, ';'); index >= 0 {
			mediaType = mediaType[:index]
		}
	}
	mediaType = strings.ToLower(strings.TrimSpace(mediaType))

	for _, compressibleType := range types {
		if strings.HasPrefix(mediaType, strings.ToLower(compressibleType)) {
			return true
		}
	}

	return false
}

func hasNoTransform(cacheControl string) bool {
	for _, directive := range strings.Split(cacheControl, ",") {
		if strings.EqualFold(strings.TrimSpace(directive), "no-transform") {
			return true
		}
	}

	return false
}

func addVary(header http.Header, value string) {
	for _, vary := range header.Values("Vary") {
		for _, item := range strings.Split(vary, ",") {
			item = strings.TrimSpace(item)
			if item == "*" || strings.EqualFold(item, value) {
				return
			}
		}
	}

	header.Add("Vary", value)
}

func parseAcceptEncoding(header string) map[string]float64 {
	weights := make(map[string]float64)

	for _, part := range strings.Split(header, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		fields := strings.Split(part, ";")
		encoding := strings.ToLower(strings.TrimSpace(fields[0]))
		if encoding == "" {
			continue
		}

		quality := 1.0
		for _, parameter := range fields[1:] {
			key, value, ok := strings.Cut(strings.TrimSpace(parameter), "=")
			if !ok || !strings.EqualFold(strings.TrimSpace(key), "q") {
				continue
			}

			if parsed, err := strconv.ParseFloat(strings.TrimSpace(value), 64); err == nil {
				quality = parsed
			}
		}

		if _, exists := weights[encoding]; !exists {
			weights[encoding] = quality
		}
	}

	return weights
}

func negotiateEncoding(header string) string {
	if header == "" {
		return ""
	}

	weights := parseAcceptEncoding(header)
	wildcard, hasWildcard := weights["*"]

	best := ""
	bestQuality := 0.0
	for _, encoding := range supportedEncodings {
		quality, ok := weights[encoding]
		if !ok {
			if !hasWildcard {
				continue
			}
			quality = wildcard
		}

		if quality > bestQuality {
			best = encoding
			bestQuality = quality
		}
	}

	return best
}

func shouldSkipRequest(r *http.Request) bool {
	if r.Method == http.MethodHead {
		return true
	}

	if r.Header.Get("Range") != "" {
		return true
	}

	if r.Header.Get("Upgrade") != "" {
		return true
	}

	return false
}

func New(configs ...Config) ngamux.MiddlewareFunc {
	var config Config
	if len(configs) > 0 {
		config = configs[0]
	}
	config = buildConfig(config)

	pools := make(map[string]*sync.Pool, len(supportedEncodings))
	for _, encoding := range supportedEncodings {
		pools[encoding] = newCompressorPool(encoding, config)
	}

	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(rw http.ResponseWriter, r *http.Request) {
			addVary(rw.Header(), varyAcceptEncoding)

			encoding := negotiateEncoding(strings.Join(r.Header.Values("Accept-Encoding"), ","))
			if encoding == "" || shouldSkipRequest(r) {
				next(rw, r)
				return
			}

			cw := &compressResponseWriter{
				ResponseWriter: rw,
				encoding:       encoding,
				types:          config.ContentTypes,
				pool:           pools[encoding],
			}
			defer func() {
				_ = cw.Close()
			}()

			next(cw, r)
		}
	}
}
