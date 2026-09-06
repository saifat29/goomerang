package middleware

import (
	"net/http"

	"github.com/saifat29/goomerang/config"
)

// Middleware takes an http.Handler and returns http.Handler,
// allowing chaining of multiple middleware functions.
type Middleware func(http.Handler) http.Handler

// Builder builds a middleware from its configuration.
type Builder func(*config.Middleware) Middleware

// Registry contains middleware factories keyed by name.
type Registry map[config.MiddlewareName]Builder

// NewRegistry returns a new empty Registry.
func NewRegistry() Registry {
	return make(Registry)
}

// Register adds a new middleware builder to the registry with the given name.
func (r Registry) Register(name config.MiddlewareName, builder Builder) {
	r[name] = builder
}

// Chain applies a series of middleware functions to an http.Handler.
func Chain(handler http.Handler, middlewares ...Middleware) http.Handler {
	for i := len(middlewares) - 1; i >= 0; i-- {
		handler = middlewares[i](handler)
	}

	return handler
}
