package cache

import (
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/saifat29/goomerang/cache"
	"github.com/saifat29/goomerang/proxy"
)

const (
	xCacheStatus      = "X-Cache-Status"
	cacheStatusHit    = "HIT"
	cacheStatusMiss   = "MISS"
	cacheStatusBypass = "BYPASS"
)

// Embeds the `http.ResponseWriter` interface for capturing response data.
type responseRecorder struct {
	http.ResponseWriter
	statusCode  int
	body        []byte
	injectAfter http.Header
}

// newResponseRecorder returns new instance of `responseRecorder`.
func newResponseRecorder(w http.ResponseWriter) *responseRecorder {
	return &responseRecorder{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
		body:           make([]byte, 0),
		injectAfter:    make(http.Header),
	}
}

// Write is the implementation of the `http.ResponseWriter` interface method.
func (r *responseRecorder) Write(b []byte) (int, error) {
	r.body = append(r.body, b...)

	return r.ResponseWriter.Write(b)
}

// WriteHeader is the imlplementation of the `http.ResponseWriter` interface method.
func (r *responseRecorder) WriteHeader(code int) {
	r.statusCode = code

	proxy.AddHeaders(r.Header(), r.injectAfter)

	r.ResponseWriter.WriteHeader(code)
}

// SetInjectHeader holds a header temporarily till our `responseRecoder.WriteHeader()`
// method is called by `next.ServeHTTP()`. This is to ensure that our header is injected
// before the `Write()` method is called.
func (r *responseRecorder) SetInjectHeader(k, v string) {
	r.injectAfter.Set(k, v)
}

// New returns a middleware that caches upstream responses.
func New(c *cache.MemoryLRU, ttl time.Duration) proxy.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// BYPASS: Caching bypassed for non-cacheable requests.
			if !serveFromCache(r) {
				log.Debug().Str("path", r.URL.Path).Msg("cache bypassed")

				w.Header().Set(xCacheStatus, cacheStatusBypass)
				next.ServeHTTP(w, r)

				return
			}

			cacheKey := cache.NewCacheKey(r)

			// HIT: Serving response from cache.
			if entry := c.Get(cacheKey); entry != nil {
				log.Debug().Str("path", r.URL.Path).Msg("cache hit")

				proxy.CopyHeaders(w.Header(), entry.Headers)
				w.Header().Set(xCacheStatus, cacheStatusHit)
				w.WriteHeader(entry.StatusCode)

				if _, err := w.Write(entry.Body); err != nil {
					log.Error().Err(err).Msg("failed to write cached response")
				}

				return
			}

			// MISS: Forwarding to upstream and caching the response if applicable.
			log.Debug().Str("path", r.URL.Path).Msg("cache miss")

			rec := newResponseRecorder(w)
			rec.SetInjectHeader(xCacheStatus, cacheStatusMiss)
			next.ServeHTTP(rec, r)

			if saveToCache(r, rec) {
				log.Debug().Str("path", r.URL.Path).Msg("storing response in cache")

				headers := rec.Header().Clone()

				// Before saving response to cache, we remove the `X-Cache-Status` header.
				// This is to ensure that when the response is served from cache it doesn't
				// sends the `MISS` cache status.
				headers.Del(xCacheStatus)
				entry := cache.NewEntry(cacheKey, rec.statusCode, headers, rec.body, ttl)
				c.Set(cacheKey, entry)
			}
		})
	}
}

// serveFromCache checks if the request can be served from cache.
func serveFromCache(req *http.Request) bool {
	if req.Method != http.MethodGet && req.Method != http.MethodHead {
		return false
	}

	return true
}

// saveToCache checks if the response can be saved to cache.
func saveToCache(req *http.Request, rec *responseRecorder) bool {
	if req.Method != http.MethodGet && req.Method != http.MethodHead {
		return false
	}

	if rec.statusCode != http.StatusOK {
		return false
	}

	cacheCtrlHeader := parseCacheCtrlHeader(rec.Header())

	if slices.Contains(cacheCtrlHeader, "private") ||
		slices.Contains(cacheCtrlHeader, "no-cache") ||
		slices.Contains(cacheCtrlHeader, "no-store") ||
		rec.Header().Get("Set-Cookie") != "" ||
		req.Header.Get("Authorization") != "" {
		return false
	}

	return true
}

// parseCacheCtrlHeader parses the cache control headers.
func parseCacheCtrlHeader(header http.Header) []string {
	var result []string

	for _, values := range header.Values("Cache-Control") {
		for _, value := range strings.Split(values, ",") {
			if trimmed := strings.TrimSpace(value); trimmed != "" {
				result = append(result, trimmed)
			}
		}
	}

	return result
}
