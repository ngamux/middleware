package compress

import (
	"compress/gzip"

	"github.com/andybalholm/brotli"
)

const (
	defaultZstdLevel = 3
	maxZstdLevel     = 22
)

type Config struct {
	Level        int
	BrotliLevel  int
	ZstdLevel    int
	ContentTypes []string
}

var defaultContentTypes = []string{
	"text/",
	"application/json",
	"application/javascript",
	"application/x-javascript",
	"application/xml",
	"application/xhtml+xml",
	"application/rss+xml",
	"application/atom+xml",
	"application/manifest+json",
	"application/ld+json",
	"image/svg+xml",
	"application/wasm",
}

func buildConfig(config Config) Config {
	if config.Level == 0 || config.Level < gzip.HuffmanOnly || config.Level > gzip.BestCompression {
		config.Level = gzip.DefaultCompression
	}

	if config.BrotliLevel == 0 || config.BrotliLevel < brotli.BestSpeed || config.BrotliLevel > brotli.BestCompression {
		config.BrotliLevel = brotli.DefaultCompression
	}

	if config.ZstdLevel == 0 || config.ZstdLevel < 1 || config.ZstdLevel > maxZstdLevel {
		config.ZstdLevel = defaultZstdLevel
	}

	if len(config.ContentTypes) == 0 {
		config.ContentTypes = defaultContentTypes
	}

	return config
}
