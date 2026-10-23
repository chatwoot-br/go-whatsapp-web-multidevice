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
		{"password with space", "socks5://user:pa ss3cret@proxy.example.com:1080", "socks5://***:***@proxy.example.com:1080"},
		{"invalid port", "socks5://user:s3cret@proxy.example.com:port", "socks5://***:***@proxy.example.com:port"},
		{"unescaped slash in password", "socks5://user:s3cret/x@proxy.example.com:1080", "socks5://***:***@proxy.example.com:1080"},
		// url.Parse reads "user:2024" as host:port for these, leaving User nil.
		{"unescaped hash after numeric password prefix", "socks5://user:2024#s3cret@proxy.example.com:1080", "socks5://***:***@proxy.example.com:1080"},
		{"unescaped slash after numeric password prefix", "http://user:2024/s3cret@proxy.example.com:3128", "http://***:***@proxy.example.com:3128"},
		{"unescaped question mark in password", "socks5://user:?s3cret@proxy.example.com:1080", "socks5://***:***@proxy.example.com:1080"},
		{"unescaped at in password", "socks5://user:s3@cret@proxy.example.com:1080", "socks5://***:***@proxy.example.com:1080"},
		{"unescaped at then slash in password", "socks5://user:s3cret@dor/s3cret2@proxy.example.com:1080", "socks5://***:***@proxy.example.com:1080"},
		{"comma in password", "socks5://user:2024,s3cret@proxy.example.com:1080", "socks5://***:***@proxy.example.com:1080"},
		{"missing scheme", "user:s3cret@proxy.example.com:1080", "***:***@proxy.example.com:1080"},
		{"scheme delimiter inside password", "user:pa://ss@proxy.example.com:1080", "***:***@proxy.example.com:1080"},
		{"invalid leading scheme token", "1user://s3cret@proxy.example.com:1080", "***:***@proxy.example.com:1080"},
		{"overlapping user and misparsed host", "socks5://frag:pa@fragsecret.invalid/rest@proxy.example.com:1080", "socks5://***:***@proxy.example.com:1080"},
		{"scheme delimiter after comma in password", "socks5://proxyuser:2024,https://secret@proxy.example.com:1080", "socks5://***:***@proxy.example.com:1080"},
		{"missing at and host", "socks5://user:s3cret", "<redacted>"},
		{"unparseable without credentials", "://bad", "<redacted>"},
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
	for _, raw := range []string{
		"socks5://user:s3cret@proxy.example.com:port",
		"socks5://user:s3cret",                           // inner error quotes `:s3cret`
		"http://user:s3cret/x@proxy.example.com:3128",    // inner error quotes `:s3cret`
		"socks5://user:ab%zzs3cret@proxy.example.com:80", // inner error quotes `%zz`
	} {
		t.Run(raw, func(t *testing.T) {
			_, parseErr := url.Parse(raw)
			require.Error(t, parseErr)
			require.Contains(t, parseErr.Error(), "s3cret", "url.Parse error is expected to embed the raw URL")

			got := RedactURLError(parseErr, raw)
			assert.NotContains(t, got.Error(), "s3cret")
			assert.NotContains(t, got.Error(), "user")
			assert.NotContains(t, got.Error(), "%zz")

			wrapped := RedactURLError(fmt.Errorf("set proxy: %w", parseErr), raw)
			assert.NotContains(t, wrapped.Error(), "s3cret")
		})
	}

	raw := "socks5://user:s3cret@proxy.example.com:port"
	_, parseErr := url.Parse(raw)
	assert.Contains(t, RedactURLError(parseErr, raw).Error(), "proxy.example.com:port", "host stays visible for diagnosis")

	for _, tt := range []struct {
		name   string
		raw    string
		detail string
	}{
		{"user parsed as host", "socks5://user:2024#s3cret@proxy.example.com:1080", "proxyconnect tcp: dial tcp: lookup user: no such host"},
		{"password parsed as host", "socks5://user:pa@ss3/cret@proxy.example.com:1080", "proxyconnect tcp: dial tcp: lookup ss3: no such host"},
		{"overlapping user and host", "socks5://frag:pa@fragsecret.invalid/rest@proxy.example.com:1080", "proxyconnect tcp: dial tcp: lookup fragsecret.invalid: no such host"},
		{"schemeless credentials", "user:s3cret@proxy.example.com:1080", `unsupported proxy scheme "user"`},
		{"scheme delimiter in password", "user:pa://ss@proxy.example.com:1080", `unsupported proxy scheme "user"`},
	} {
		t.Run(tt.name, func(t *testing.T) {
			for _, err := range []error{
				errors.New(tt.detail),
				&url.Error{Op: "Get", URL: "https://web.whatsapp.com", Err: errors.New(tt.detail)},
			} {
				assert.EqualError(t, RedactURLError(err, tt.raw), "invalid URL (details withheld: they may contain credentials)")
			}
		})
	}

	for _, tt := range []struct {
		name   string
		raw    string
		detail string
		want   string
	}{
		{"overlapping credentials", "socks5://frag:fragsecret@proxy.example.com:1080", "authentication failed for frag: fragsecret", "authentication failed for ***: ***"},
		{"encoded credentials", "socks5://u%40ser:pa%3Ass@proxy.example.com:1080", "authentication failed for u@ser: pa:ss (u%40ser:pa%3Ass)", "authentication failed for ***: *** (***)"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			assert.EqualError(t, RedactURLError(errors.New(tt.detail), tt.raw), tt.want)
		})
	}

	other := errors.New(`unsupported proxy scheme "ftp"`)
	assert.Same(t, other, RedactURLError(other, "ftp://proxy.example.com:21"))
	assert.Same(t, other, RedactURLError(other, "socks5://user:s3cret@proxy.example.com:1080"))
	assert.NoError(t, RedactURLError(nil, raw))
}
