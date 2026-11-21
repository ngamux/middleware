package compress

import (
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"strconv"
	"sync"
	"testing"

	"github.com/andybalholm/brotli"
	"github.com/klauspost/compress/zstd"
)

func serve(t *testing.T, middleware func(http.HandlerFunc) http.HandlerFunc, handler http.HandlerFunc, acceptEncoding string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if acceptEncoding != "" {
		req.Header.Set("Accept-Encoding", acceptEncoding)
	}

	rec := httptest.NewRecorder()
	middleware(handler)(rec, req)

	return rec
}

func mustDecompress(t *testing.T, encoding string, data []byte) string {
	t.Helper()

	var (
		body []byte
		err  error
	)
	switch encoding {
	case "br":
		body, err = io.ReadAll(brotli.NewReader(bytes.NewReader(data)))
	case "zstd":
		var decoder *zstd.Decoder
		decoder, err = zstd.NewReader(bytes.NewReader(data))
		if err == nil {
			defer decoder.Close()
			body, err = io.ReadAll(decoder)
		}
	case "gzip":
		var reader *gzip.Reader
		reader, err = gzip.NewReader(bytes.NewReader(data))
		if err == nil {
			defer func() {
				_ = reader.Close()
			}()
			body, err = io.ReadAll(reader)
		}
	case "deflate":
		var reader io.ReadCloser
		reader, err = zlib.NewReader(bytes.NewReader(data))
		if err == nil {
			defer func() {
				_ = reader.Close()
			}()
			body, err = io.ReadAll(reader)
		}
	default:
		t.Fatalf("unsupported encoding %q", encoding)
	}
	if err != nil {
		t.Fatalf("decompress %s body: %v", encoding, err)
	}

	return string(body)
}

func TestNewCompressesWithEncoding(t *testing.T) {
	cases := []struct {
		name     string
		accept   string
		encoding string
	}{
		{"brotli", "br", "br"},
		{"zstd", "zstd", "zstd"},
		{"gzip", "gzip", "gzip"},
		{"deflate", "deflate", "deflate"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			const content = "hello hello hello"
			rec := serve(t, New(), func(rw http.ResponseWriter, r *http.Request) {
				rw.Header().Set("Content-Type", "text/plain")
				_, _ = rw.Write([]byte(content))
			}, c.accept)

			if got := rec.Header().Get("Content-Encoding"); got != c.encoding {
				t.Errorf("Content-Encoding = %q, want %q", got, c.encoding)
			}
			if got := rec.Header().Get("Content-Length"); got != "" {
				t.Errorf("Content-Length = %q, want empty", got)
			}
			if got := rec.Header().Get("Vary"); got != "Accept-Encoding" {
				t.Errorf("Vary = %q, want %q", got, "Accept-Encoding")
			}
			if got := mustDecompress(t, c.encoding, rec.Body.Bytes()); got != content {
				t.Errorf("body = %q, want %q", got, content)
			}
		})
	}
}

func TestNegotiateEncoding(t *testing.T) {
	cases := []struct {
		header string
		want   string
	}{
		{"", ""},
		{"br", "br"},
		{"zstd", "zstd"},
		{"gzip", "gzip"},
		{"deflate", "deflate"},
		{"br, zstd, gzip, deflate", "br"},
		{"zstd, br", "br"},
		{"zstd;q=1, br;q=0.5", "zstd"},
		{"gzip, deflate", "gzip"},
		{"deflate;q=0.8, gzip;q=0.5", "deflate"},
		{"gzip;q=0, deflate", "deflate"},
		{"gzip;q=0, *;q=0.5", "br"},
		{"br;q=0, *;q=0.5", "zstd"},
		{"identity", ""},
		{"*", "br"},
		{"*;q=0", ""},
		{"br;q=0, zstd;q=0, gzip;q=0, deflate;q=0", ""},
		{"GZIP", "gzip"},
	}

	for _, c := range cases {
		if got := negotiateEncoding(c.header); got != c.want {
			t.Errorf("negotiateEncoding(%q) = %q, want %q", c.header, got, c.want)
		}
	}
}

func TestNewSkipsWhenNotAccepted(t *testing.T) {
	const content = "hello"
	rec := serve(t, New(), func(rw http.ResponseWriter, r *http.Request) {
		rw.Header().Set("Content-Type", "text/plain")
		_, _ = rw.Write([]byte(content))
	}, "")

	if got := rec.Header().Get("Content-Encoding"); got != "" {
		t.Errorf("Content-Encoding = %q, want empty", got)
	}
	if got := rec.Body.String(); got != content {
		t.Errorf("body = %q, want %q", got, content)
	}
	if got := rec.Header().Get("Vary"); got != "Accept-Encoding" {
		t.Errorf("Vary = %q, want %q", got, "Accept-Encoding")
	}
}

func TestNewSkipsNonCompressibleContentType(t *testing.T) {
	const content = "not compressible"
	rec := serve(t, New(), func(rw http.ResponseWriter, r *http.Request) {
		rw.Header().Set("Content-Type", "image/png")
		_, _ = rw.Write([]byte(content))
	}, "gzip")

	if got := rec.Header().Get("Content-Encoding"); got != "" {
		t.Errorf("Content-Encoding = %q, want empty", got)
	}
	if got := rec.Body.String(); got != content {
		t.Errorf("body = %q, want %q", got, content)
	}
}

func TestNewSkipsAlreadyEncodedResponse(t *testing.T) {
	const content = "already encoded"
	rec := serve(t, New(), func(rw http.ResponseWriter, r *http.Request) {
		rw.Header().Set("Content-Type", "text/plain")
		rw.Header().Set("Content-Encoding", "br")
		_, _ = rw.Write([]byte(content))
	}, "gzip")

	if got := rec.Header().Get("Content-Encoding"); got != "br" {
		t.Errorf("Content-Encoding = %q, want %q", got, "br")
	}
	if got := rec.Body.String(); got != content {
		t.Errorf("body = %q, want %q", got, content)
	}
}

func TestNewSkipsNoTransform(t *testing.T) {
	const content = "no transform"
	rec := serve(t, New(), func(rw http.ResponseWriter, r *http.Request) {
		rw.Header().Set("Content-Type", "text/plain")
		rw.Header().Set("Cache-Control", "public, no-transform")
		_, _ = rw.Write([]byte(content))
	}, "gzip")

	if got := rec.Header().Get("Content-Encoding"); got != "" {
		t.Errorf("Content-Encoding = %q, want empty", got)
	}
	if got := rec.Body.String(); got != content {
		t.Errorf("body = %q, want %q", got, content)
	}
}

func TestNewSkipsStatusesWithoutBody(t *testing.T) {
	for _, status := range []int{
		http.StatusNoContent,
		http.StatusResetContent,
		http.StatusPartialContent,
		http.StatusNotModified,
	} {
		t.Run(strconv.Itoa(status), func(t *testing.T) {
			rec := serve(t, New(), func(rw http.ResponseWriter, r *http.Request) {
				rw.Header().Set("Content-Type", "text/plain")
				rw.WriteHeader(status)
				_, _ = rw.Write([]byte("body"))
			}, "gzip")

			if got := rec.Header().Get("Content-Encoding"); got != "" {
				t.Errorf("Content-Encoding = %q, want empty", got)
			}
		})
	}
}

func TestNewSkipsRequests(t *testing.T) {
	cases := []struct {
		name   string
		method string
		header string
		value  string
	}{
		{"head", http.MethodHead, "", ""},
		{"range", http.MethodGet, "Range", "bytes=0-10"},
		{"upgrade", http.MethodGet, "Upgrade", "websocket"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			req := httptest.NewRequest(c.method, "/", nil)
			req.Header.Set("Accept-Encoding", "gzip")
			if c.header != "" {
				req.Header.Set(c.header, c.value)
			}

			rec := httptest.NewRecorder()
			New()(func(rw http.ResponseWriter, r *http.Request) {
				rw.Header().Set("Content-Type", "text/plain")
				_, _ = rw.Write([]byte("body"))
			})(rec, req)

			if got := rec.Header().Get("Content-Encoding"); got != "" {
				t.Errorf("Content-Encoding = %q, want empty", got)
			}
		})
	}
}

func TestNewRemovesContentLength(t *testing.T) {
	const content = "hello hello"
	rec := serve(t, New(), func(rw http.ResponseWriter, r *http.Request) {
		rw.Header().Set("Content-Type", "text/plain")
		rw.Header().Set("Content-Length", strconv.Itoa(len(content)))
		_, _ = rw.Write([]byte(content))
	}, "gzip")

	if got := rec.Header().Get("Content-Length"); got != "" {
		t.Errorf("Content-Length = %q, want empty", got)
	}
	if got := mustDecompress(t, "gzip", rec.Body.Bytes()); got != content {
		t.Errorf("body = %q, want %q", got, content)
	}
}

func TestNewSniffsContentType(t *testing.T) {
	const content = "<html><body>hello</body></html>"
	rec := serve(t, New(), func(rw http.ResponseWriter, r *http.Request) {
		_, _ = rw.Write([]byte(content))
	}, "gzip")

	if got := rec.Header().Get("Content-Type"); got != "text/html; charset=utf-8" {
		t.Errorf("Content-Type = %q, want %q", got, "text/html; charset=utf-8")
	}
	if got := rec.Header().Get("Content-Encoding"); got != "gzip" {
		t.Errorf("Content-Encoding = %q, want %q", got, "gzip")
	}
	if got := mustDecompress(t, "gzip", rec.Body.Bytes()); got != content {
		t.Errorf("body = %q, want %q", got, content)
	}
}

func TestNewUsesCustomContentTypes(t *testing.T) {
	middleware := New(Config{ContentTypes: []string{"application/x-custom"}})

	rec := serve(t, middleware, func(rw http.ResponseWriter, r *http.Request) {
		rw.Header().Set("Content-Type", "application/x-custom")
		_, _ = rw.Write([]byte("custom"))
	}, "gzip")

	if got := rec.Header().Get("Content-Encoding"); got != "gzip" {
		t.Errorf("custom type Content-Encoding = %q, want %q", got, "gzip")
	}

	rec = serve(t, middleware, func(rw http.ResponseWriter, r *http.Request) {
		rw.Header().Set("Content-Type", "text/plain")
		_, _ = rw.Write([]byte("plain"))
	}, "gzip")

	if got := rec.Header().Get("Content-Encoding"); got != "" {
		t.Errorf("default type Content-Encoding = %q, want empty", got)
	}
}

func TestAddVary(t *testing.T) {
	cases := []struct {
		name     string
		existing []string
		want     []string
	}{
		{"none", nil, []string{"Accept-Encoding"}},
		{"same", []string{"Accept-Encoding"}, []string{"Accept-Encoding"}},
		{"other", []string{"Origin"}, []string{"Origin", "Accept-Encoding"}},
		{"wildcard", []string{"*"}, []string{"*"}},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			header := http.Header{}
			for _, vary := range c.existing {
				header.Add("Vary", vary)
			}

			addVary(header, "Accept-Encoding")

			if got := header.Values("Vary"); !slices.Equal(got, c.want) {
				t.Errorf("Vary = %v, want %v", got, c.want)
			}
		})
	}
}

func TestNewPreservesExistingVary(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rec := httptest.NewRecorder()
	rec.Header().Set("Vary", "Origin")

	New()(func(rw http.ResponseWriter, r *http.Request) {
		rw.Header().Set("Content-Type", "text/plain")
		_, _ = rw.Write([]byte("body"))
	})(rec, req)

	vary := rec.Header().Values("Vary")
	if !slices.Contains(vary, "Origin") || !slices.Contains(vary, "Accept-Encoding") {
		t.Errorf("Vary = %v, want Origin and Accept-Encoding", vary)
	}
}

func TestNewFlushes(t *testing.T) {
	var gotWriter http.ResponseWriter
	rec := serve(t, New(), func(rw http.ResponseWriter, r *http.Request) {
		gotWriter = rw
		rw.Header().Set("Content-Type", "text/plain")
		_, _ = rw.Write([]byte("one"))

		flusher, ok := rw.(http.Flusher)
		if !ok {
			t.Fatal("response writer does not implement http.Flusher")
		}
		flusher.Flush()

		_, _ = rw.Write([]byte("two"))
	}, "gzip")

	if !rec.Flushed {
		t.Error("Flush was not propagated to the underlying response writer")
	}
	if got := mustDecompress(t, "gzip", rec.Body.Bytes()); got != "onetwo" {
		t.Errorf("body = %q, want %q", got, "onetwo")
	}
	if unwrapper, ok := gotWriter.(interface{ Unwrap() http.ResponseWriter }); !ok {
		t.Error("response writer does not implement Unwrap")
	} else if unwrapper.Unwrap() != rec {
		t.Error("Unwrap did not return the underlying response writer")
	}
}

func TestNewEmptyBody(t *testing.T) {
	rec := serve(t, New(), func(rw http.ResponseWriter, r *http.Request) {
		rw.Header().Set("Content-Type", "text/plain")
		rw.WriteHeader(http.StatusCreated)
	}, "gzip")

	if rec.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusCreated)
	}
	if got := rec.Header().Get("Content-Encoding"); got != "" {
		t.Errorf("Content-Encoding = %q, want empty", got)
	}
	if rec.Body.Len() != 0 {
		t.Errorf("body = %q, want empty", rec.Body.String())
	}
}

func TestBuildConfig(t *testing.T) {
	config := buildConfig(Config{})
	if config.Level != gzip.DefaultCompression {
		t.Errorf("Level = %d, want %d", config.Level, gzip.DefaultCompression)
	}
	if config.BrotliLevel != brotli.DefaultCompression {
		t.Errorf("BrotliLevel = %d, want %d", config.BrotliLevel, brotli.DefaultCompression)
	}
	if config.ZstdLevel != defaultZstdLevel {
		t.Errorf("ZstdLevel = %d, want %d", config.ZstdLevel, defaultZstdLevel)
	}
	if len(config.ContentTypes) != len(defaultContentTypes) {
		t.Errorf("ContentTypes length = %d, want %d", len(config.ContentTypes), len(defaultContentTypes))
	}

	config = buildConfig(Config{
		Level:        100,
		BrotliLevel:  100,
		ZstdLevel:    100,
		ContentTypes: []string{"custom"},
	})
	if config.Level != gzip.DefaultCompression {
		t.Errorf("invalid Level = %d, want %d", config.Level, gzip.DefaultCompression)
	}
	if config.BrotliLevel != brotli.DefaultCompression {
		t.Errorf("invalid BrotliLevel = %d, want %d", config.BrotliLevel, brotli.DefaultCompression)
	}
	if config.ZstdLevel != defaultZstdLevel {
		t.Errorf("invalid ZstdLevel = %d, want %d", config.ZstdLevel, defaultZstdLevel)
	}
	if len(config.ContentTypes) != 1 || config.ContentTypes[0] != "custom" {
		t.Errorf("ContentTypes = %v, want [custom]", config.ContentTypes)
	}

	config = buildConfig(Config{
		Level:       gzip.BestSpeed,
		BrotliLevel: brotli.BestCompression,
		ZstdLevel:   maxZstdLevel,
	})
	if config.Level != gzip.BestSpeed {
		t.Errorf("Level = %d, want %d", config.Level, gzip.BestSpeed)
	}
	if config.BrotliLevel != brotli.BestCompression {
		t.Errorf("BrotliLevel = %d, want %d", config.BrotliLevel, brotli.BestCompression)
	}
	if config.ZstdLevel != maxZstdLevel {
		t.Errorf("ZstdLevel = %d, want %d", config.ZstdLevel, maxZstdLevel)
	}
}

func TestNewIsConcurrencySafe(t *testing.T) {
	const requests = 25
	middleware := New()

	var wg sync.WaitGroup
	for _, encoding := range supportedEncodings {
		for i := 0; i < requests; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()

				req := httptest.NewRequest(http.MethodGet, "/", nil)
				req.Header.Set("Accept-Encoding", encoding)
				rec := httptest.NewRecorder()
				middleware(func(rw http.ResponseWriter, r *http.Request) {
					rw.Header().Set("Content-Type", "text/plain")
					_, _ = rw.Write([]byte("hello"))
				})(rec, req)
			}()
		}
	}
	wg.Wait()
}

func BenchmarkNew(b *testing.B) {
	middleware := New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	handler := middleware(func(rw http.ResponseWriter, r *http.Request) {
		rw.Header().Set("Content-Type", "text/plain")
		_, _ = rw.Write([]byte("hello hello hello"))
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		handler(httptest.NewRecorder(), req)
	}
}
