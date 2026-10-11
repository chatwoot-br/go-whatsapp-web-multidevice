package whatsapp

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mau.fi/whatsmeow/store"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func waVersionTestClient(status int, body string, err error) *http.Client {
	return &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if err != nil {
			return nil, err
		}
		return &http.Response{
			StatusCode: status,
			Body:       io.NopCloser(strings.NewReader(body)),
			Header:     make(http.Header),
			Request:    r,
		}, nil
	})}
}

func TestResolveWAVersion(t *testing.T) {
	embedded := store.WAVersionContainer{2, 3000, 1049558099}

	tests := []struct {
		name    string
		client  *http.Client
		want    store.WAVersionContainer
		wantErr bool
	}{
		{
			name:   "newer version is returned",
			client: waVersionTestClient(200, `{"client_revision":1050000000,"x":1}`, nil),
			want:   store.WAVersionContainer{2, 3000, 1050000000},
		},
		{
			name:   "older version is still returned; the caller decides not to adopt it",
			client: waVersionTestClient(200, `{"client_revision":1041871181,}`, nil),
			want:   store.WAVersionContainer{2, 3000, 1041871181},
		},
		{
			name:    "transport error falls back to embedded",
			client:  waVersionTestClient(0, "", errors.New("dial tcp: connection refused")),
			want:    embedded,
			wantErr: true,
		},
		{
			name:    "non-200 response falls back to embedded",
			client:  waVersionTestClient(503, "unavailable", nil),
			want:    embedded,
			wantErr: true,
		},
		{
			name:    "page without a version falls back to embedded",
			client:  waVersionTestClient(200, "<html>no version here</html>", nil),
			want:    embedded,
			wantErr: true,
		},
		{
			name:    "zero version falls back to embedded",
			client:  waVersionTestClient(200, `{"client_revision":0,}`, nil),
			want:    embedded,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolveWAVersion(context.Background(), tt.client, embedded)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestResolveWAVersionHonoursContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return nil, r.Context().Err()
	})}
	embedded := store.WAVersionContainer{2, 3000, 1}
	got, err := resolveWAVersion(ctx, client, embedded)
	require.Error(t, err)
	assert.Equal(t, embedded, got)
}

func TestWAVersionHTTPClient(t *testing.T) {
	direct, err := waVersionHTTPClient("")
	require.NoError(t, err)
	assert.Same(t, http.DefaultClient, direct)

	for _, raw := range []string{"http://proxy.example:3128", "socks5://user:pass@proxy.example:1080"} {
		proxied, err := waVersionHTTPClient(raw)
		require.NoError(t, err, raw)
		transport, ok := proxied.Transport.(*http.Transport)
		require.True(t, ok, raw)
		require.NotNil(t, transport.Proxy, raw)
		req, _ := http.NewRequest(http.MethodGet, "https://web.whatsapp.com", nil)
		got, err := transport.Proxy(req)
		require.NoError(t, err, raw)
		assert.Equal(t, raw, got.String(), raw)
	}

	_, err = waVersionHTTPClient("://bad")
	assert.Error(t, err)
}
