package config

import "time"

type MiddlewareName string

const (
	MiddlewareLogger    MiddlewareName = "logger"
	MiddlewareHTTPCache MiddlewareName = "http_cache"
)

func (mn MiddlewareName) String() string {
	return string(mn)
}

// Middleware is the container for middleware configuration fields.
type Middleware struct {
	Logger    *Logger    `json:"logger" yaml:"logger"`
	HTTPCache *HTTPCache `json:"http_cache" yaml:"http_cache"`
}

// Active returns the middleware that is activated through the configuration.
func (m *Middleware) Active() MiddlewareName {
	if m.Logger != nil {
		return MiddlewareLogger
	}

	if m.HTTPCache != nil {
		return MiddlewareHTTPCache
	}

	return ""
}

// HTTPCache contains the configuration fields for Logger.
type Logger struct{}

// HTTPCache contains the configuration fields for HTTPCache.
type HTTPCache struct {
	TTL time.Duration `json:"ttl" yaml:"ttl"`
}
