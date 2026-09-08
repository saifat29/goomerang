package proxy

import (
	"net/http"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/saifat29/goomerang/config"
	"github.com/saifat29/goomerang/proxy/middleware"
)

// Route represents a mapping between a path prefix and an upstream URL,
// it also has middlewares that are chained for requests.
// And contains balancer (load balancer) for choosing an upstream server.
type Route struct {
	path        string
	middlewares []middleware.Middleware
	balancer    Balancer
}

// FromConfig converts a slice of `config.Proxy` to a slice of `Route`,
// fetching the correct middleware from the registry, and creating the
// balancer as per the config.
func FromConfig(cfg []*config.Proxy, registry middleware.Registry) []Route {
	routes := make([]Route, len(cfg))

	for i, c := range cfg {
		route := Route{
			path:     c.Path,
			balancer: NewBalancer(c),
		}

		for _, mwCfg := range c.Middlewares {
			if builder, ok := registry[mwCfg.Active()]; ok {
				route.middlewares = append(route.middlewares, builder(mwCfg))
			} else {
				log.Warn().Str("middleware", mwCfg.Active().String()).Msg("middleware not found in registry")
			}
		}
		routes[i] = route
	}

	return routes
}

// Handler wraps the given upstream handler with the route's middleware chain.
func (r *Route) Handler(upstream http.Handler) http.Handler {
	return middleware.Chain(upstream, r.middlewares...)
}

// pathMatched checks if the given path matches the route's path prefix.
// More complex matching logic can be added here in the future if needed.
func (r *Route) pathMatched(path string) bool {
	return strings.HasPrefix(path, r.path)
}
