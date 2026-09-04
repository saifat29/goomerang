package proxy

import (
	"net/url"
	"strings"

	"github.com/saifat29/goomerang/config"
)

// Route represents a mapping between a path prefix and an upstream URL.
type Route struct {
	Path        string
	UpstreamURL *url.URL
}

// FromConfig converts a slice of config.Proxy to a slice of Route.
func FromConfig(cfg []config.Proxy) []Route {
	routes := make([]Route, len(cfg))

	for i, r := range cfg {
		routes[i] = Route{
			Path:        r.Path,
			UpstreamURL: r.Upstream.URL,
		}
	}

	return routes
}

// pathMatched checks if the given path matches the route's path prefix.
func (r *Route) pathMatched(path string) bool {
	if strings.HasPrefix(path, r.Path) {
		return true
	}

	return false
}
