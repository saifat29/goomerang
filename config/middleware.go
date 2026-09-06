package config

import "time"

type MiddlewareName string

const (
	MiddlewareLogger      MiddlewareName = "logger"
	MiddlewareHTTPCache   MiddlewareName = "http_cache"
	MiddlewareStripPrefix MiddlewareName = "strip_prefix"
)

func (mn MiddlewareName) String() string {
	return string(mn)
}

// Middleware is the container for middleware configuration fields.
type Middleware struct {
	Logger      *Logger      `json:"logger,omitempty" yaml:"logger"`
	HTTPCache   *HTTPCache   `json:"http_cache,omitempty" yaml:"http_cache"`
	StripPrefix *StripPrefix `json:"strip_prefix,omitempty" yaml:"strip_prefix"`
}

// Active returns the middleware that is activated through the configuration.
// TODO: Need to find a better alternative to this.
// I keep forgetting adding a new middleware here, this is not intuitive, this is bad.
func (m *Middleware) Active() MiddlewareName {
	if m.Logger != nil {
		return MiddlewareLogger
	}

	if m.HTTPCache != nil {
		return MiddlewareHTTPCache
	}

	if m.StripPrefix != nil {
		return MiddlewareStripPrefix
	}

	return ""
}

// HTTPCache contains the configuration fields for Logger.
type Logger struct{}

// HTTPCache contains the configuration fields for HTTPCache.
type HTTPCache struct {
	TTL time.Duration `json:"ttl" yaml:"ttl"`
}

// StripPrefix contains the configuration fields for StripPrefix.
type StripPrefix struct {
	Prefix string `json:"prefix" yaml:"prefix"`
}
