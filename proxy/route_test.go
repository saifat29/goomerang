package proxy

import (
	"net/url"
	"testing"

	"github.com/saifat29/goomerang/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFromConfig(t *testing.T) {
	cfg := []config.Proxy{
		{Path: "/api", Upstream: &config.URL{URL: &url.URL{Scheme: "http", Host: "example.com"}}},
		{Path: "/static", Upstream: &config.URL{URL: &url.URL{Scheme: "http", Host: "other.com"}}},
	}

	routes := FromConfig(cfg)

	require.Len(t, routes, 2)
	assert.Equal(t, "/api", routes[0].Path)
	assert.Equal(t, "example.com", routes[0].UpstreamURL.Host)
	assert.Equal(t, "/static", routes[1].Path)
	assert.Equal(t, "other.com", routes[1].UpstreamURL.Host)
}

func TestFromConfigEmpty(t *testing.T) {
	routes := FromConfig([]config.Proxy{})

	assert.Empty(t, routes)
}

func TestPathMatched(t *testing.T) {
	tests := []struct {
		name  string
		route string
		path  string
		want  bool
	}{
		{
			name:  "exact match",
			route: "/api",
			path:  "/api",
			want:  true,
		},
		{
			name:  "prefix match",
			route: "/api",
			path:  "/api/users",
			want:  true,
		},
		{
			name:  "deep prefix match",
			route: "/api",
			path:  "/api/v1/users/123",
			want:  true,
		},
		{
			name:  "no match",
			route: "/api",
			path:  "/static/image.png",
			want:  false,
		},
		{
			name:  "partial prefix still matches",
			route: "/api",
			path:  "/api-v2/users",
			want:  true,
		},
		{
			name:  "empty path",
			route: "/api",
			path:  "",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Route{Path: tt.route}
			got := r.pathMatched(tt.path)

			assert.Equal(t, tt.want, got)
		})
	}
}
