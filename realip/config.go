package realip

import (
	"net"
	"strings"
)

type Config struct {
	TrustedProxies []string
	trusted        []*net.IPNet
}

func buildConfig(config Config) Config {
	config.trusted = nil
	for _, value := range config.TrustedProxies {
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
			config.trusted = append(config.trusted, &net.IPNet{
				IP:   ip,
				Mask: net.CIDRMask(bits, bits),
			})
			continue
		}

		if _, network, err := net.ParseCIDR(value); err == nil {
			config.trusted = append(config.trusted, network)
		}
	}

	return config
}

func (c Config) isTrusted(ip net.IP) bool {
	if len(c.TrustedProxies) == 0 {
		return true
	}

	for _, network := range c.trusted {
		if network.Contains(ip) {
			return true
		}
	}

	return false
}
