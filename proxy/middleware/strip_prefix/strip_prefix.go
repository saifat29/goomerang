package strip_prefix

import (
	"net/http"

	"github.com/saifat29/goomerang/config"
	"github.com/saifat29/goomerang/proxy/middleware"
)

// New returns a middleware that strips prefix from the path.
func New(cfg *config.StripPrefix) middleware.Middleware {
	return func(next http.Handler) http.Handler {
		return http.StripPrefix(cfg.Prefix, next)
	}
}
