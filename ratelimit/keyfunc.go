package ratelimit

import (
	"net"
	"net/http"
	"strings"
)

func KeyFunc_ByClientIP(r *http.Request) string {
	return KeyFunc_ByClientIPFromRemoteAddr(r)
}

func KeyFunc_ByClientIPFromRemoteAddr(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}

	return host
}

func KeyFunc_ByClientIPFromHeader(header string) func(*http.Request) string {
	return func(r *http.Request) string {
		if ip := firstIP(r.Header.Values(header)); ip != nil {
			return ip.String()
		}

		return KeyFunc_ByClientIP(r)
	}
}

func KeyFunc_ByClientIPFromXFF(trustedProxies ...string) func(*http.Request) string {
	trusted := newProxySet(trustedProxies)

	return func(r *http.Request) string {
		if trusted.configured {
			peer := parseIP(r.RemoteAddr)
			if peer == nil || !trusted.contains(peer) {
				return KeyFunc_ByClientIP(r)
			}
		}

		if ip := clientIPFromXFF(r.Header.Values("X-Forwarded-For"), trusted); ip != nil {
			return ip.String()
		}

		return KeyFunc_ByClientIP(r)
	}
}

func KeyFunc_ByClientIPFromXFFTrustedProxies(trustedProxies string) func(*http.Request) string {
	var values []string
	for _, value := range strings.Split(trustedProxies, ",") {
		if value = strings.TrimSpace(value); value != "" {
			values = append(values, value)
		}
	}

	return KeyFunc_ByClientIPFromXFF(values...)
}

type proxySet struct {
	configured bool
	networks   []*net.IPNet
}

func newProxySet(values []string) proxySet {
	set := proxySet{configured: len(values) > 0}

	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}

		if ip := net.ParseIP(value); ip != nil {
			bits := 128
			if ip.To4() != nil {
				ip = ip.To4()
				bits = 32
			}
			set.networks = append(set.networks, &net.IPNet{
				IP:   ip,
				Mask: net.CIDRMask(bits, bits),
			})
			continue
		}

		if _, network, err := net.ParseCIDR(value); err == nil {
			set.networks = append(set.networks, network)
		}
	}

	return set
}

func (s proxySet) contains(ip net.IP) bool {
	for _, network := range s.networks {
		if network.Contains(ip) {
			return true
		}
	}

	return false
}

func firstIP(values []string) net.IP {
	for _, value := range values {
		for _, part := range strings.Split(value, ",") {
			if ip := parseIP(part); ip != nil {
				return ip
			}
		}
	}

	return nil
}

func clientIPFromXFF(values []string, trusted proxySet) net.IP {
	parts := splitIPValues(values)

	if !trusted.configured {
		for _, part := range parts {
			if ip := parseIP(part); ip != nil {
				return ip
			}
		}

		return nil
	}

	for i := len(parts) - 1; i >= 0; i-- {
		ip := parseIP(parts[i])
		if ip == nil {
			continue
		}

		if !trusted.contains(ip) {
			return ip
		}
	}

	return nil
}

func splitIPValues(values []string) []string {
	var parts []string
	for _, value := range values {
		for _, part := range strings.Split(value, ",") {
			if part = strings.TrimSpace(part); part != "" {
				parts = append(parts, part)
			}
		}
	}

	return parts
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
