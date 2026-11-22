package realip

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
)

func resolve(t *testing.T, config Config, remoteAddr string, headers map[string]string) string {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = remoteAddr
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	New(config)(func(http.ResponseWriter, *http.Request) {})(httptest.NewRecorder(), req)

	return req.RemoteAddr
}

func TestNewResolvesClientIP(t *testing.T) {
	cases := []struct {
		name    string
		config  Config
		remote  string
		headers map[string]string
		want    string
	}{
		{
			name:    "x-forwarded-for",
			remote:  "10.0.0.1:1234",
			headers: map[string]string{"X-Forwarded-For": "203.0.113.9"},
			want:    "203.0.113.9:1234",
		},
		{
			name:    "x-real-ip",
			remote:  "10.0.0.1:1234",
			headers: map[string]string{"X-Real-IP": "203.0.113.9"},
			want:    "203.0.113.9:1234",
		},
		{
			name:    "forwarded",
			remote:  "10.0.0.1:1234",
			headers: map[string]string{"Forwarded": "for=203.0.113.9;proto=https"},
			want:    "203.0.113.9:1234",
		},
		{
			name:    "forwarded quoted ipv6",
			remote:  "10.0.0.1:1234",
			headers: map[string]string{"Forwarded": `for="[2001:db8::2]:443";proto=https`},
			want:    "[2001:db8::2]:1234",
		},
		{
			name:    "prefers x-forwarded-for",
			remote:  "10.0.0.1:1234",
			headers: map[string]string{"X-Forwarded-For": "203.0.113.9", "X-Real-IP": "198.51.100.7"},
			want:    "203.0.113.9:1234",
		},
		{
			name:    "header with port",
			remote:  "10.0.0.1:1234",
			headers: map[string]string{"X-Forwarded-For": "203.0.113.9:5555"},
			want:    "203.0.113.9:1234",
		},
		{
			name:    "trusted proxy",
			config:  Config{TrustedProxies: []string{"10.0.0.0/8"}},
			remote:  "10.0.0.1:1234",
			headers: map[string]string{"X-Forwarded-For": "203.0.113.9"},
			want:    "203.0.113.9:1234",
		},
		{
			name:    "trusted proxy chain",
			config:  Config{TrustedProxies: []string{"10.0.0.0/8"}},
			remote:  "10.0.0.1:1234",
			headers: map[string]string{"X-Forwarded-For": "203.0.113.9, 10.0.0.2, 10.0.0.3"},
			want:    "203.0.113.9:1234",
		},
		{
			name:    "ignores spoofed leftmost entry",
			config:  Config{TrustedProxies: []string{"10.0.0.0/8"}},
			remote:  "10.0.0.1:1234",
			headers: map[string]string{"X-Forwarded-For": "1.2.3.4, 198.51.100.7, 10.0.0.2"},
			want:    "198.51.100.7:1234",
		},
		{
			name:    "single trusted ip",
			config:  Config{TrustedProxies: []string{"10.0.0.1"}},
			remote:  "10.0.0.1:1234",
			headers: map[string]string{"X-Forwarded-For": "203.0.113.9"},
			want:    "203.0.113.9:1234",
		},
		{
			name:    "untrusted peer",
			config:  Config{TrustedProxies: []string{"10.0.0.0/8"}},
			remote:  "192.0.2.1:1234",
			headers: map[string]string{"X-Forwarded-For": "203.0.113.9"},
			want:    "192.0.2.1:1234",
		},
		{
			name:    "untrusted peer without port",
			config:  Config{TrustedProxies: []string{"10.0.0.0/8"}},
			remote:  "192.0.2.1",
			headers: map[string]string{"X-Forwarded-For": "203.0.113.9"},
			want:    "192.0.2.1",
		},
		{
			name:    "ipv6",
			remote:  "[2001:db8::1]:1234",
			headers: map[string]string{"X-Forwarded-For": "2001:db8::2"},
			want:    "[2001:db8::2]:1234",
		},
		{
			name:   "no headers",
			remote: "10.0.0.1:1234",
			want:   "10.0.0.1:1234",
		},
		{
			name:    "invalid header",
			remote:  "10.0.0.1:1234",
			headers: map[string]string{"X-Forwarded-For": "not-an-ip"},
			want:    "10.0.0.1:1234",
		},
		{
			name:    "all trusted chain fails closed",
			config:  Config{TrustedProxies: []string{"10.0.0.0/8"}},
			remote:  "10.0.0.1:1234",
			headers: map[string]string{"X-Forwarded-For": "10.0.0.2, 10.0.0.3"},
			want:    "10.0.0.1:1234",
		},
		{
			name:    "invalid trusted proxies fail closed",
			config:  Config{TrustedProxies: []string{"not-a-proxy"}},
			remote:  "10.0.0.1:1234",
			headers: map[string]string{"X-Forwarded-For": "203.0.113.9"},
			want:    "10.0.0.1:1234",
		},
		{
			name:    "unparseable peer with trusted proxies",
			config:  Config{TrustedProxies: []string{"10.0.0.0/8"}},
			remote:  "",
			headers: map[string]string{"X-Forwarded-For": "203.0.113.9"},
			want:    "",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := resolve(t, c.config, c.remote, c.headers); got != c.want {
				t.Errorf("RemoteAddr = %q, want %q", got, c.want)
			}
		})
	}
}

func TestNewJoinsMultipleHeaderLines(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.1:1234"
	req.Header.Add("X-Forwarded-For", "203.0.113.9")
	req.Header.Add("X-Forwarded-For", "10.0.0.2")

	New(Config{TrustedProxies: []string{"10.0.0.0/8"}})(func(http.ResponseWriter, *http.Request) {})(httptest.NewRecorder(), req)

	if got := req.RemoteAddr; got != "203.0.113.9:1234" {
		t.Errorf("RemoteAddr = %q, want %q", got, "203.0.113.9:1234")
	}
}

func TestNewForwardedChain(t *testing.T) {
	headers := "for=203.0.113.9, for=10.0.0.2"
	got := resolve(t, Config{TrustedProxies: []string{"10.0.0.0/8"}}, "10.0.0.1:1234", map[string]string{
		"Forwarded": headers,
	})

	if got != "203.0.113.9:1234" {
		t.Errorf("RemoteAddr = %q, want %q", got, "203.0.113.9:1234")
	}
}

func TestForwardedAllTrustedFailsClosed(t *testing.T) {
	got := resolve(t, Config{TrustedProxies: []string{"10.0.0.0/8"}}, "10.0.0.1:1234", map[string]string{
		"Forwarded": "for=10.0.0.2, for=10.0.0.3",
	})

	if got != "10.0.0.1:1234" {
		t.Errorf("RemoteAddr = %q, want %q", got, "10.0.0.1:1234")
	}
}

func TestBuildConfigTrustedProxies(t *testing.T) {
	config := buildConfig(Config{TrustedProxies: []string{
		"10.0.0.1",
		"192.168.0.0/16",
		"2001:db8::1",
		"2001:db8:1::/48",
		"not-a-proxy",
		"",
	}})

	if got := len(config.trusted); got != 4 {
		t.Fatalf("trusted networks = %d, want 4", got)
	}

	cases := []struct {
		ip   string
		want bool
	}{
		{"10.0.0.1", true},
		{"10.0.0.2", false},
		{"192.168.1.5", true},
		{"192.169.0.1", false},
		{"2001:db8::1", true},
		{"2001:db8::2", false},
		{"2001:db8:1::abcd", true},
		{"203.0.113.1", false},
	}

	for _, c := range cases {
		if got := config.isTrusted(net.ParseIP(c.ip)); got != c.want {
			t.Errorf("isTrusted(%s) = %v, want %v", c.ip, got, c.want)
		}
	}
}

func TestIsTrustedEmptyConfig(t *testing.T) {
	config := buildConfig(Config{})

	if !config.isTrusted(net.ParseIP("203.0.113.1")) {
		t.Error("empty TrustedProxies should trust every peer")
	}
}

func TestIsTrustedOnlyInvalidEntries(t *testing.T) {
	config := buildConfig(Config{TrustedProxies: []string{"not-a-proxy"}})

	if config.isTrusted(net.ParseIP("10.0.0.1")) {
		t.Error("invalid TrustedProxies should not trust any peer")
	}
}
