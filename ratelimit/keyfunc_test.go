package ratelimit

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func requestWith(remoteAddr string, headers map[string]string) *http.Request {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = remoteAddr
	for key, value := range headers {
		r.Header.Set(key, value)
	}

	return r
}

func TestKeyFuncByClientIPFromRemoteAddr(t *testing.T) {
	cases := []struct {
		remoteAddr string
		want       string
	}{
		{"192.0.2.1:1234", "192.0.2.1"},
		{"192.0.2.1", "192.0.2.1"},
		{"[::1]:8080", "::1"},
		{"", ""},
	}

	for _, c := range cases {
		r := requestWith(c.remoteAddr, nil)
		if got := KeyFunc_ByClientIPFromRemoteAddr(r); got != c.want {
			t.Errorf("KeyFunc_ByClientIPFromRemoteAddr(%q) = %q, want %q", c.remoteAddr, got, c.want)
		}
	}
}

func TestKeyFuncByClientIPFromHeader(t *testing.T) {
	keyFunc := KeyFunc_ByClientIPFromHeader("X-Forwarded-For")

	cases := []struct {
		name       string
		remoteAddr string
		header     string
		want       string
	}{
		{"header", "10.0.0.1:1234", "203.0.113.9", "203.0.113.9"},
		{"first value", "10.0.0.1:1234", "203.0.113.9, 10.0.0.2", "203.0.113.9"},
		{"invalid header", "10.0.0.1:1234", "not-an-ip", "10.0.0.1"},
		{"empty header", "10.0.0.1:1234", "", "10.0.0.1"},
		{"ipv6", "[2001:db8::1]:1234", "[2001:db8::2]:443", "2001:db8::2"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := requestWith(c.remoteAddr, map[string]string{"X-Forwarded-For": c.header})
			if c.header == "" {
				r.Header.Del("X-Forwarded-For")
			}
			if got := keyFunc(r); got != c.want {
				t.Errorf("key = %q, want %q", got, c.want)
			}
		})
	}
}

func TestKeyFuncByClientIPFromXFF(t *testing.T) {
	cases := []struct {
		name       string
		remoteAddr string
		header     string
		want       string
	}{
		{"single", "10.0.0.1:1234", "203.0.113.9", "203.0.113.9"},
		{"leftmost", "10.0.0.1:1234", "1.2.3.4, 198.51.100.7, 10.0.0.2", "1.2.3.4"},
		{"missing", "10.0.0.1:1234", "", "10.0.0.1"},
		{"invalid", "10.0.0.1:1234", "not-an-ip", "10.0.0.1"},
		{"ipv6", "[2001:db8::1]:1234", "2001:db8::2", "2001:db8::2"},
	}

	keyFunc := KeyFunc_ByClientIPFromXFF()
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			headers := map[string]string{}
			if c.header != "" {
				headers["X-Forwarded-For"] = c.header
			}

			if got := keyFunc(requestWith(c.remoteAddr, headers)); got != c.want {
				t.Errorf("key = %q, want %q", got, c.want)
			}
		})
	}
}

func TestKeyFuncByClientIPFromXFFTrustedProxies(t *testing.T) {
	cases := []struct {
		name       string
		trusted    []string
		remoteAddr string
		header     string
		want       string
	}{
		{
			name:       "trusted chain",
			trusted:    []string{"10.0.0.0/8"},
			remoteAddr: "10.0.0.1:1234",
			header:     "203.0.113.9, 10.0.0.2, 10.0.0.3",
			want:       "203.0.113.9",
		},
		{
			name:       "spoofed leftmost",
			trusted:    []string{"10.0.0.0/8"},
			remoteAddr: "10.0.0.1:1234",
			header:     "1.2.3.4, 198.51.100.7, 10.0.0.2",
			want:       "198.51.100.7",
		},
		{
			name:       "all trusted",
			trusted:    []string{"10.0.0.0/8"},
			remoteAddr: "10.0.0.1:1234",
			header:     "10.0.0.2, 10.0.0.3",
			want:       "10.0.0.1",
		},
		{
			name:       "untrusted peer",
			trusted:    []string{"10.0.0.0/8"},
			remoteAddr: "192.0.2.1:1234",
			header:     "203.0.113.9",
			want:       "192.0.2.1",
		},
		{
			name:       "missing header",
			trusted:    []string{"10.0.0.0/8"},
			remoteAddr: "10.0.0.1:1234",
			header:     "",
			want:       "10.0.0.1",
		},
		{
			name:       "single trusted ip",
			trusted:    []string{"10.0.0.1"},
			remoteAddr: "10.0.0.1:1234",
			header:     "203.0.113.9",
			want:       "203.0.113.9",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			keyFunc := KeyFunc_ByClientIPFromXFF(c.trusted...)

			headers := map[string]string{}
			if c.header != "" {
				headers["X-Forwarded-For"] = c.header
			}

			if got := keyFunc(requestWith(c.remoteAddr, headers)); got != c.want {
				t.Errorf("key = %q, want %q", got, c.want)
			}
		})
	}
}

func TestKeyFuncByClientIPFromXFFTrustedProxiesList(t *testing.T) {
	keyFunc := KeyFunc_ByClientIPFromXFFTrustedProxies("10.0.0.0/8, 192.168.1.10")

	r := requestWith("192.168.1.10:1234", map[string]string{
		"X-Forwarded-For": "203.0.113.9, 10.0.0.2",
	})
	if got := keyFunc(r); got != "203.0.113.9" {
		t.Errorf("key = %q, want %q", got, "203.0.113.9")
	}

	r = requestWith("203.0.113.1:1234", map[string]string{
		"X-Forwarded-For": "198.51.100.7",
	})
	if got := keyFunc(r); got != "203.0.113.1" {
		t.Errorf("untrusted peer key = %q, want %q", got, "203.0.113.1")
	}
}
