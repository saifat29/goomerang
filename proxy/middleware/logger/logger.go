package logger

import (
	"net/http"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/saifat29/goomerang/config"
	"github.com/saifat29/goomerang/proxy/middleware"
)

// New returns a middleware that logs request and response details.
func New(cfg *config.Logger) middleware.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now().UTC()

			recorder := newResponseRecorder(w)

			next.ServeHTTP(recorder, r)

			took := time.Since(start)

			log.Info().
				Str("method", r.Method).
				Str("path", r.URL.Path).
				Str("remote_addr", r.RemoteAddr).
				Str("user_agent", r.UserAgent()).
				Dur("duration", took).
				Int("status", recorder.statusCode).
				Int("size", recorder.size).
				Msg("request completed")
		})
	}
}

// Embeds the `http.ResponseWriter` interface for capturing response data.
type responseRecorder struct {
	http.ResponseWriter
	statusCode int
	size       int
}

// newResponseRecorder returns new instance of `responseRecorder`.
func newResponseRecorder(w http.ResponseWriter) *responseRecorder {
	return &responseRecorder{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
		size:           0,
	}
}

// Write is the implementation of the `http.ResponseWriter` interface method.
func (rr *responseRecorder) Write(body []byte) (int, error) {
	size, err := rr.ResponseWriter.Write(body)
	rr.size += size

	return size, err
}

// WriteHeader is the imlplementation of the `http.ResponseWriter` interface method.
func (rr *responseRecorder) WriteHeader(statusCode int) {
	rr.statusCode = statusCode
	rr.ResponseWriter.WriteHeader(statusCode)
}
