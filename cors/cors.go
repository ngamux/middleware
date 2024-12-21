package cors

import (
	"net/http"
	"strings"

	"github.com/ngamux/ngamux"
)

var configDefault = Config{
	AllowOrigins: "*",
	AllowMethods: strings.Join([]string{
		http.MethodGet,
		http.MethodPost,
		http.MethodHead,
		http.MethodPut,
		http.MethodPatch,
		http.MethodDelete,
		http.MethodOptions,
	}, ","),
	AllowHeaders: "",
}

func New(config ...Config) func(next http.HandlerFunc) http.HandlerFunc {
	cfg := configDefault

	if len(config) > 0 {
		cfg = config[0]
		if cfg.AllowMethods == "" {
			cfg.AllowMethods = configDefault.AllowMethods
		}
		if cfg.AllowOrigins == "" {
			cfg.AllowOrigins = configDefault.AllowOrigins
		}

		cfg.AllowOrigins = strings.ReplaceAll(cfg.AllowOrigins, " ", "")
		cfg.AllowMethods = strings.ReplaceAll(cfg.AllowMethods, " ", "")
		cfg.AllowHeaders = strings.ReplaceAll(cfg.AllowHeaders, " ", "")
	}
	allowedOrigins := strings.Split(cfg.AllowOrigins, ",")

	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(rw http.ResponseWriter, r *http.Request) {
			allowed := false
			origin := r.Referer()
			if origin == "" {
				origin = r.Header.Get("Origin")
			}
			origin = strings.TrimRight(origin, "/")
			for _, o := range allowedOrigins {
				o = strings.TrimSpace(o)
				if o == "*" || o == origin {
					allowed = true
					break
				}
			}

			if allowed {
				rw.Header().Set("Access-Control-Allow-Origin", origin)
			}
			rw.Header().Set("Access-Control-Allow-Methods", cfg.AllowMethods)
			rw.Header().Set("Access-Control-Allow-Headers", cfg.AllowHeaders)

			if r.Method == http.MethodOptions {
				ngamux.Res(rw).Status(http.StatusNoContent).Text("")
				return
			}

			next(rw, r)
		}
	}
}
