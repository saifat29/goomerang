package proxy

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/saifat29/goomerang/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFromConfig(t *testing.T) {
	cfg := []*config.Proxy{
		{Path: "/api", Upstream: &config.URL{URL: &url.URL{Scheme: "http", Host: "example.com"}}},
		{Path: "/static", Upstream: &config.URL{URL: &url.URL{Scheme: "http", Host: "other.com"}}},
	}

	routes := FromConfig(cfg, nil)

	require.Len(t, routes, 2)
	assert.Equal(t, "/api", routes[0].Path)
	assert.Equal(t, "example.com", routes[0].UpstreamURL.Host)
	assert.Equal(t, "/static", routes[1].Path)
	assert.Equal(t, "other.com", routes[1].UpstreamURL.Host)
}

func TestFromConfigEmpty(t *testing.T) {
	routes := FromConfig([]*config.Proxy{}, nil)

	assert.Empty(t, routes)
}

func TestFromConfigWithMiddlewares(t *testing.T) {
	called := false
	mw := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
			next.ServeHTTP(w, r)
		})
	}

	registry := MiddlewareRegistry{"test-mw": mw}
	cfg := []*config.Proxy{
		{Path: "/api", Upstream: &config.URL{URL: &url.URL{Scheme: "http", Host: "example.com"}}, Middlewares: []string{"test-mw"}},
	}

	routes := FromConfig(cfg, registry)

	require.Len(t, routes, 1)
	require.Len(t, routes[0].middlewares, 1)

	// Verify the middleware is invoked when Handler is called.
	handler := routes[0].Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodGet, "/api", http.NoBody)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.True(t, called)
}

func TestFromConfigUnknownMiddleware(t *testing.T) {
	registry := MiddlewareRegistry{}
	cfg := []*config.Proxy{
		{Path: "/api", Upstream: &config.URL{URL: &url.URL{Scheme: "http", Host: "example.com"}}, Middlewares: []string{"nonexistent"}},
	}

	routes := FromConfig(cfg, registry)

	require.Len(t, routes, 1)
	assert.Empty(t, routes[0].middlewares)
}

func TestRouteHandler(t *testing.T) {
	handlerCalled := false
	upstream := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	route := &Route{Path: "/api"}
	handler := route.Handler(upstream)

	req := httptest.NewRequest(http.MethodGet, "/api", http.NoBody)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.True(t, handlerCalled)
	assert.Equal(t, http.StatusOK, rec.Code)
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
