package proxy

import (
	"net/http"
)

// Middleware takes an http.Handler and returns http.Handler,
// allowing chaining of multiple middleware functions.
type Middleware func(http.Handler) http.Handler

// MiddlewareRegistry contains all the middlewares.
type MiddlewareRegistry map[string]Middleware

// NewMiddlewareRegistry returns a new MiddlewareRegistry with the default middlewares registered.
func NewMiddlewareRegistry() MiddlewareRegistry {
	return make(MiddlewareRegistry)
}

// Register adds a new middleware to the registry with the given name.
func (m MiddlewareRegistry) Register(name string, mw Middleware) {
	m[name] = mw
}

// ChainMiddlewares applies a series of middleware functions to an http.Handler.
func ChainMiddlewares(handler http.Handler, middlewares ...Middleware) http.Handler {
	for i := len(middlewares) - 1; i >= 0; i-- {
		handler = middlewares[i](handler)
	}

	return handler
}
