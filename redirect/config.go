package redirect

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"
)

type Rewrite map[string]string

type Config struct {
	Rewrite       Rewrite
	Transport     http.RoundTripper
	Timeout       time.Duration
	PreserveHost  bool
	FlushInterval time.Duration
	ErrorHandler  func(rw http.ResponseWriter, r *http.Request, err error)

	proxies map[string]*httputil.ReverseProxy
}

func buildConfig(config Config) Config {
	if config.Rewrite == nil {
		config.Rewrite = Rewrite{}
	}

	if config.ErrorHandler == nil {
		config.ErrorHandler = defaultErrorHandler
	}

	config.proxies = make(map[string]*httputil.ReverseProxy, len(config.Rewrite))
	for path, target := range config.Rewrite {
		parsed, err := url.Parse(target)
		if err != nil || parsed.Scheme == "" || parsed.Host == "" {
			panic(fmt.Sprintf("redirect: invalid target %q for path %q", target, path))
		}

		config.proxies[path] = newProxy(parsed, config)
	}

	return config
}
