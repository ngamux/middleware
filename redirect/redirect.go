package redirect

import (
	"context"
	"errors"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/ngamux/ngamux"
)

func newProxy(target *url.URL, config Config) *httputil.ReverseProxy {
	return &httputil.ReverseProxy{
		Transport:     config.Transport,
		FlushInterval: config.FlushInterval,
		Rewrite: func(pr *httputil.ProxyRequest) {
			incomingQuery := pr.Out.URL.RawQuery

			pr.Out.URL.Scheme = target.Scheme
			pr.Out.URL.Host = target.Host
			pr.Out.URL.Path = target.Path
			pr.Out.URL.RawPath = target.RawPath
			pr.Out.URL.RawQuery = joinQuery(target.RawQuery, incomingQuery)

			if config.PreserveHost {
				pr.Out.Host = pr.In.Host
			} else {
				pr.Out.Host = ""
			}

			pr.SetXForwarded()
		},
		ErrorHandler: config.ErrorHandler,
	}
}

func joinQuery(target, incoming string) string {
	if target == "" {
		return incoming
	}

	if incoming == "" {
		return target
	}

	return target + "&" + incoming
}

func defaultErrorHandler(rw http.ResponseWriter, _ *http.Request, err error) {
	switch {
	case errors.Is(err, context.Canceled):
		return
	case errors.Is(err, context.DeadlineExceeded):
		ngamux.Res(rw).Status(http.StatusGatewayTimeout).JSON(ngamux.Map{
			"error": err.Error(),
		})
	default:
		ngamux.Res(rw).Status(http.StatusBadGateway).JSON(ngamux.Map{
			"error": err.Error(),
		})
	}
}

func New(configs ...Config) ngamux.MiddlewareFunc {
	var config Config
	if len(configs) > 0 {
		config = configs[0]
	}
	config = buildConfig(config)

	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(rw http.ResponseWriter, r *http.Request) {
			proxy, ok := config.proxies[r.URL.Path]
			if !ok {
				next(rw, r)
				return
			}

			if config.Timeout > 0 {
				ctx, cancel := context.WithTimeout(r.Context(), config.Timeout)
				defer cancel()
				r = r.WithContext(ctx)
			}

			proxy.ServeHTTP(rw, r)
		}
	}
}
