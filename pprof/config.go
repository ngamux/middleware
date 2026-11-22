package pprof

import "strings"

const defaultPrefix = "/debug/pprof"

// Config is a configuration used in New for the pprof middleware.
//
// Prefix is the URL path where profiling endpoints are mounted. It defaults to
// "/debug/pprof".
type Config struct {
	Prefix string
}

func buildConfig(config Config) Config {
	prefix := strings.TrimSpace(config.Prefix)
	if prefix == "" {
		prefix = defaultPrefix
	}

	if !strings.HasPrefix(prefix, "/") {
		prefix = "/" + prefix
	}

	prefix = strings.TrimRight(prefix, "/")
	if prefix == "" {
		prefix = "/"
	}

	config.Prefix = prefix

	return config
}
