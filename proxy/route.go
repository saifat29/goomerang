package proxy

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/saifat29/goomerang/config"
)

// Route represents a mapping between a path prefix and an upstream URL,
// it also has middlewares that are chained for requests.
type Route struct {
	Path        string
	UpstreamURL *url.URL
	middlewares []Middleware
}

// FromConfig converts a slice of config.Proxy to a slice of Route,
// and fetching the correct middleware from the registry.
func FromConfig(cfg []config.Proxy, registry MiddlewareRegistry) []Route {
	routes := make([]Route, len(cfg))

	for i, r := range cfg {
		route := Route{
			Path:        r.Path,
			UpstreamURL: r.Upstream.URL,
		}

		for _, name := range r.Middlewares {
			if mw, ok := registry[name]; ok {
				route.middlewares = append(route.middlewares, mw)
			} else {
				log.Warn().Str("middleware", name).Msg("middleware not found in registry")
			}
		}
		routes[i] = route
	}

	return routes
}

// Handler wraps the given upstream handler with the route's middleware chain.
func (r *Route) Handler(upstream http.Handler) http.Handler {
	return ChainMiddlewares(upstream, r.middlewares...)
}

// pathMatched checks if the given path matches the route's path prefix.
// More complex matching logic can be added here in the future if needed.
func (r *Route) pathMatched(path string) bool {
	return strings.HasPrefix(path, r.Path)
}
