package realip

import (
	"net"
	"net/http"
	"strings"

	"github.com/ngamux/ngamux"
)

func New(configs ...Config) ngamux.MiddlewareFunc {
	var config Config
	if len(configs) > 0 {
		config = configs[0]
	}
	config = buildConfig(config)

	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(rw http.ResponseWriter, r *http.Request) {
			if ip := config.clientIP(r); ip != nil {
				r.RemoteAddr = replaceHost(r.RemoteAddr, ip.String())
			}

			next(rw, r)
		}
	}
}

func (c Config) clientIP(r *http.Request) net.IP {
	peer := parseIP(r.RemoteAddr)
	if peer == nil {
		if len(c.TrustedProxies) > 0 {
			return nil
		}
	} else if !c.isTrusted(peer) {
		return nil
	}

	if value := headerValue(r, "X-Forwarded-For"); value != "" {
		if ip := c.forwardedForIP(value); ip != nil {
			return ip
		}
	}

	if value := headerValue(r, "X-Real-IP"); value != "" {
		if ip := parseIP(value); ip != nil {
			return ip
		}
	}

	if value := headerValue(r, "Forwarded"); value != "" {
		if ip := c.forwardedIP(value); ip != nil {
			return ip
		}
	}

	return nil
}

func (c Config) forwardedForIP(header string) net.IP {
	parts := strings.Split(header, ",")

	var leftmost net.IP
	for i := len(parts) - 1; i >= 0; i-- {
		ip := parseIP(parts[i])
		if ip == nil {
			continue
		}

		leftmost = ip
		if !c.isTrusted(ip) {
			return ip
		}
	}

	if len(c.TrustedProxies) == 0 {
		return leftmost
	}

	return nil
}

func (c Config) forwardedIP(header string) net.IP {
	elements := strings.Split(header, ",")

	var leftmost net.IP
	for i := len(elements) - 1; i >= 0; i-- {
		for _, parameter := range strings.Split(elements[i], ";") {
			key, value, ok := strings.Cut(strings.TrimSpace(parameter), "=")
			if !ok || !strings.EqualFold(strings.TrimSpace(key), "for") {
				continue
			}

			ip := parseIP(value)
			if ip == nil {
				continue
			}

			leftmost = ip
			if !c.isTrusted(ip) {
				return ip
			}
		}
	}

	if len(c.TrustedProxies) == 0 {
		return leftmost
	}

	return nil
}

func headerValue(r *http.Request, name string) string {
	return strings.Join(r.Header.Values(name), ",")
}

func parseIP(value string) net.IP {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}

	value = strings.Trim(value, `"`)
	if host, _, err := net.SplitHostPort(value); err == nil {
		value = host
	} else {
		value = strings.TrimPrefix(value, "[")
		value = strings.TrimSuffix(value, "]")
	}

	if zone := strings.IndexByte(value, '%'); zone >= 0 {
		value = value[:zone]
	}

	return net.ParseIP(value)
}

func replaceHost(remoteAddr, ip string) string {
	_, port, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		return ip
	}

	return net.JoinHostPort(ip, port)
}
