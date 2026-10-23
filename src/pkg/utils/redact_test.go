package utils

import (
	"errors"
	"fmt"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRedactURLCredentials(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"empty", "", ""},
		{"socks5 with user and password", "socks5://user:s3cret@proxy.example.com:1080", "socks5://***:***@proxy.example.com:1080"},
		{"http with user and password", "http://user:s3cret@10.0.0.1:3128", "http://***:***@10.0.0.1:3128"},
		{"https with path", "https://user:s3cret@proxy.example.com:443/path?x=1", "https://***:***@proxy.example.com:443/path?x=1"},
		{"percent-encoded password", "socks5://user:p%40ss%3Aw0rd@proxy.example.com:1080", "socks5://***:***@proxy.example.com:1080"},
		{"empty password", "socks5://user:@proxy.example.com:1080", "socks5://***:***@proxy.example.com:1080"},
		{"user only", "socks5://user@proxy.example.com:1080", "socks5://***@proxy.example.com:1080"},
		{"ipv6 host", "socks5://user:s3cret@[2001:db8::1]:1080", "socks5://***:***@[2001:db8::1]:1080"},
		{"scheme-relative", "//user:s3cret@proxy.example.com:1080", "//***:***@proxy.example.com:1080"},
		{"missing host", "socks5://user:s3cret@", "socks5://***:***@"},
		{"no credentials", "socks5://proxy.example.com:1080", "socks5://proxy.example.com:1080"},
		{"no credentials http", "http://proxy.example.com:3128", "http://proxy.example.com:3128"},
		{"unparseable password with space", "socks5://user:pa ss@proxy.example.com:1080", "socks5://***:***@proxy.example.com:1080"},
		{"unparseable invalid port", "socks5://user:s3cret@proxy.example.com:port", "socks5://***:***@proxy.example.com:port"},
		{"unparseable unescaped slash in password", "socks5://user:pa/ss@proxy.example.com:1080", "socks5://***:***@proxy.example.com:1080"},
		{"unescaped at in password", "socks5://user:p@ss@proxy.example.com:1080", "socks5://***:***@proxy.example.com:1080"},
		{"missing scheme", "user:s3cret@proxy.example.com:1080", "***:***@proxy.example.com:1080"},
		{"unparseable without at", "socks5://user:s3cret", "<redacted>"},
		{"unparseable without at or credentials", "://bad", "<redacted>"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RedactURLCredentials(tt.in)
			assert.Equal(t, tt.want, got)
			assert.NotContains(t, got, "s3cret")
			assert.NotContains(t, got, "user")
		})
	}
}

func TestRedactURLError(t *testing.T) {
	raw := "socks5://user:s3cret@proxy.example.com:port"
	_, parseErr := url.Parse(raw)
	require.Error(t, parseErr)
	require.Contains(t, parseErr.Error(), "s3cret", "url.Parse error is expected to embed the raw URL")

	got := RedactURLError(parseErr)
	assert.NotContains(t, got.Error(), "s3cret")
	assert.NotContains(t, got.Error(), "user")
	assert.Contains(t, got.Error(), "proxy.example.com:port")

	wrapped := RedactURLError(fmt.Errorf("set proxy: %w", parseErr))
	assert.NotContains(t, wrapped.Error(), "s3cret")

	other := errors.New("unsupported proxy scheme \"ftp\"")
	assert.Same(t, other, RedactURLError(other))
	assert.NoError(t, RedactURLError(nil))
}
