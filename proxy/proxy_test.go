package proxy

import (
	"net/http"
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

func TestRemoveHopHeaders(t *testing.T) {
	tests := []struct {
		name        string
		header      http.Header
		wantPresent []string
		wantAbsent  []string
	}{
		{
			name: "removes all standard hop-by-hop headers",
			header: http.Header{
				"Connection":          {"keep-alive"},
				"Proxy-Connection":    {"keep-alive"},
				"Keep-Alive":          {"timeout=5"},
				"Proxy-Authenticate":  {"Basic"},
				"Proxy-Authorization": {"Basic dGVzdA=="},
				"Te":                  {"trailers"},
				"Trailer":             {"X-Trailer"},
				"Transfer-Encoding":   {"chunked"},
				"Upgrade":             {"websocket"},
			},
			wantPresent: []string{},
			wantAbsent: []string{
				"Connection",
				"Proxy-Connection",
				"Keep-Alive",
				"Proxy-Authenticate",
				"Proxy-Authorization",
				"Te",
				"Trailer",
				"Transfer-Encoding",
				"Upgrade",
			},
		},
		{
			name: "removes headers referenced in Connection header",
			header: http.Header{
				"Connection": {"X-Custom, X-Other"},
				"X-Custom":   {"value"},
				"X-Other":    {"value"},
			},
			wantPresent: []string{},
			wantAbsent:  []string{"Connection", "X-Custom", "X-Other"},
		},
		{
			name: "trims whitespace in Connection header values",
			header: http.Header{
				"Connection": {" X-Custom , X-Other "},
				"X-Custom":   {"value"},
				"X-Other":    {"value"},
			},
			wantPresent: []string{},
			wantAbsent:  []string{"Connection", "X-Custom", "X-Other"},
		},
		{
			name: "preserves non-hop headers",
			header: http.Header{
				"Authorization": {"Bearer token"},
				"Content-Type":  {"application/json"},
				"Connection":    {"keep-alive"},
				"Keep-Alive":    {"timeout=5"},
			},
			wantPresent: []string{"Authorization", "Content-Type"},
			wantAbsent:  []string{"Connection", "Keep-Alive"},
		},
		{
			name:        "handles empty headers gracefully",
			header:      http.Header{},
			wantPresent: []string{},
			wantAbsent:  []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			removeHopHeaders(tt.header)

			for _, h := range tt.wantPresent {
				assert.NotEmpty(t, tt.header.Get(h), "header %q should be present", h)
			}
			for _, h := range tt.wantAbsent {
				assert.Empty(t, tt.header.Get(h), "header %q should be removed", h)
			}
		})
	}
}

func TestCopyHeaders(t *testing.T) {
	tests := []struct {
		name string
		src  http.Header
		dst  http.Header
		want http.Header
	}{
		{
			name: "copies all headers to empty destination",
			src:  http.Header{"X-Foo": {"bar"}},
			dst:  http.Header{},
			want: http.Header{"X-Foo": {"bar"}},
		},
		{
			name: "preserves multiple values for same key",
			src:  http.Header{"X-Foo": {"a", "b"}},
			dst:  http.Header{},
			want: http.Header{"X-Foo": {"a", "b"}},
		},
		{
			name: "appends to existing destination headers",
			src:  http.Header{"X-Foo": {"bar"}},
			dst:  http.Header{"X-Foo": {"baz"}},
			want: http.Header{"X-Foo": {"baz", "bar"}},
		},
		{
			name: "handles empty source gracefully",
			src:  http.Header{},
			dst:  http.Header{},
			want: http.Header{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			copyHeaders(tt.dst, tt.src)

			assert.Equal(t, tt.want, tt.dst, "destination headers should match expected")
		})
	}
}

func TestFindRoute(t *testing.T) {
	upstreamA, err := url.Parse("http://a.com")
	if err != nil {
		t.Fatalf("failed to parse upstream URL: %v", err)
	}

	upstreamB, err := url.Parse("http://b.com")
	if err != nil {
		t.Fatalf("failed to parse upstream URL: %v", err)
	}

	upstreamC, err := url.Parse("http://c.com")
	if err != nil {
		t.Fatalf("failed to parse upstream URL: %v", err)
	}

	p := &ReverseProxy{
		routes: []Route{
			{Path: "/api", UpstreamURL: upstreamA},
			{Path: "/api/v1", UpstreamURL: upstreamB},
			{Path: "/static", UpstreamURL: upstreamC},
		},
	}

	tests := []struct {
		name     string
		path     string
		wantHost string
	}{
		{
			name:     "matches exact prefix",
			path:     "/api/users",
			wantHost: "a.com",
		},
		{
			name:     "longest prefix wins",
			path:     "/api/v1/users",
			wantHost: "b.com",
		},
		{
			name:     "matches different route",
			path:     "/static/image.png",
			wantHost: "c.com",
		},
		{
			name:     "matches root of prefix",
			path:     "/api",
			wantHost: "a.com",
		},
		{
			name:     "no match returns nil",
			path:     "/unknown/path",
			wantHost: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := p.findRoute(tt.path)

			if tt.wantHost == "" {
				assert.Nil(t, got)
			} else {
				require.NotNil(t, got)
				assert.Equal(t, tt.wantHost, got.UpstreamURL.Host)
			}
		})
	}
}
