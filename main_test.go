package main

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildUpstreamURL(t *testing.T) {
	tests := []struct {
		name      string
		upstream  string
		request   string
		wantPath  string
		wantQuery string
	}{
		{
			name:      "merges upstream and request paths",
			upstream:  "http://httpbin.org/api",
			request:   "http://example.com/users",
			wantPath:  "/api/users",
			wantQuery: "",
		},
		{
			name:      "trims trailing slash on upstream path",
			upstream:  "http://httpbin.org/api/",
			request:   "http://example.com/users",
			wantPath:  "/api/users",
			wantQuery: "",
		},
		{
			name:      "produces root path when both paths are empty",
			upstream:  "http://httpbin.org",
			request:   "http://example.com",
			wantPath:  "/",
			wantQuery: "",
		},
		{
			name:      "produces root path when both are trailing slash",
			upstream:  "http://httpbin.org/",
			request:   "http://example.com/",
			wantPath:  "/",
			wantQuery: "",
		},
		{
			name:      "combines query params from upstream and request",
			upstream:  "http://httpbin.org/api?key=upstream",
			request:   "http://example.com/users?page=2",
			wantPath:  "/api/users",
			wantQuery: "key=upstream&page=2",
		},
		{
			name:      "preserves upstream query when request has none",
			upstream:  "http://httpbin.org/api?key=upstream",
			request:   "http://example.com/users",
			wantPath:  "/api/users",
			wantQuery: "key=upstream",
		},
		{
			name:      "preserves request query when upstream has none",
			upstream:  "http://httpbin.org/api",
			request:   "http://example.com/users?page=2",
			wantPath:  "/api/users",
			wantQuery: "page=2",
		},
		{
			name:      "URL encodes special characters in query values",
			upstream:  "http://httpbin.org/api",
			request:   "http://example.com/search?q=hello world&lang=go",
			wantPath:  "/api/search",
			wantQuery: "lang=go&q=hello+world",
		},
		{
			name:      "sorts merged query params alphabetically",
			upstream:  "http://httpbin.org/api?z=1&a=2",
			request:   "http://example.com/users?m=3&b=4",
			wantPath:  "/api/users",
			wantQuery: "a=2&b=4&m=3&z=1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			upstreamURL, err := url.Parse(tt.upstream)
			require.NoError(t, err, "failed to parse upstream URL")

			reqURL, err := url.Parse(tt.request)
			require.NoError(t, err, "failed to parse request URL")

			got := buildUpstreamURL(reqURL, upstreamURL)

			assert.Equal(t, "http", got.Scheme, "scheme should always be http")
			assert.Equal(t, "httpbin.org", got.Host, "host should come from upstream")
			assert.Equal(t, tt.wantPath, got.Path, "path should merge upstream and request paths")

			if tt.wantQuery == "" {
				assert.Empty(t, got.RawQuery, "query should be empty when neither side has params")
			} else {
				assert.Equal(t, tt.wantQuery, got.RawQuery, "query should combine params from both sides")
			}
		})
	}
}
