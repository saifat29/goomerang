package proxy

import (
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/saifat29/goomerang/cache"
)

const (
	XForwardedFor string = "X-Forwarded-For"
	XCacheStatus  string = "X-Cache-Status"

	CacheStatusHit    string = "HIT"
	CacheStatusMiss   string = "MISS"
	CacheStatusBypass string = "BYPASS"
)

// ReverseProxy implements an HTTP-only proxy.
type ReverseProxy struct {
	upstreamURL *url.URL
	transport   http.RoundTripper
	cache       *cache.MemoryLRU
}

// NewReverseProxy returns a new ReverseProxy with the provided upstream URL,
// transport, and cache, if the transport and cache is not provided, a default is used.
func NewReverseProxy(upstreamURL *url.URL, transport http.RoundTripper, c *cache.MemoryLRU) *ReverseProxy {
	return &ReverseProxy{
		upstreamURL: upstreamURL,
		transport:   transport,
		cache:       c,
	}
}

func (p *ReverseProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	newReq := r.Clone(r.Context())

	cacheStatus := CacheStatusBypass

	if serveFromCache(newReq) {
		log.Println("serving from cache")
		cacheKey := cache.NewCacheKey(r)

		if cachedRes := p.cache.Get(cacheKey); cachedRes != nil {
			log.Println("cached response found")
			copyHeaders(w.Header(), cachedRes.Headers)

			w.Header().Set(XCacheStatus, CacheStatusHit)
			w.WriteHeader(cachedRes.StatusCode)

			if _, err := w.Write(cachedRes.Body); err != nil {
				log.Printf("failed to write response: %v", err)
				http.Error(w, "failed to write response", http.StatusInternalServerError)
				return
			}
			return
		}
		log.Println("not found in cache")
		cacheStatus = CacheStatusMiss
	} else {
		log.Println("cache bypassed")
	}

	removeHopHeaders(newReq.Header)

	newReq.URL = buildUpstreamURL(newReq.URL, p.upstreamURL)
	newReq.RequestURI = ""
	newReq.Host = p.upstreamURL.Host

	clientIP, _, err := net.SplitHostPort(newReq.RemoteAddr)
	if err == nil {
		// Assuming this proxy is running on the edge. Ideally, we must check `X-Forwarded-For`
		// and append the `RemoteAddr` if we're behind another proxy.
		newReq.Header.Set(XForwardedFor, clientIP)
	}

	outDumpReq, _ := httputil.DumpRequestOut(newReq, true)
	log.Println(string(outDumpReq))

	res, err := p.transport.RoundTrip(newReq)
	if err != nil {
		log.Printf("upstream server error: %v", err)
		http.Error(w, "upstream server error", http.StatusInternalServerError)
		return
	}
	defer res.Body.Close()

	outDumpRes, _ := httputil.DumpResponse(res, true)
	log.Println(string(outDumpRes))

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		log.Printf("failed to read response body from upstream server: %v", err)
		http.Error(w, "failed to read response body from upstream server", http.StatusInternalServerError)
		return
	}

	removeHopHeaders(res.Header)
	copyHeaders(w.Header(), res.Header)

	if saveToCache(newReq, res) {
		log.Println("caching response")

		cacheKey := cache.NewCacheKey(r)
		p.cache.Set(
			cacheKey,
			cache.NewEntry(cacheKey, res.StatusCode, res.Header, resBody, 5*time.Minute),
		)
	}

	w.Header().Set(XCacheStatus, cacheStatus)
	w.WriteHeader(res.StatusCode)

	if _, err := w.Write(resBody); err != nil {
		log.Printf("failed to write response: %v", err)
		http.Error(w, "failed to write response", http.StatusInternalServerError)
		return
	}
}

// buildUpstreamURL creates the upstream URL that will be used for sending
// the request. It accepts the request and upstream URL and creates the final URL.
func buildUpstreamURL(reqURL, upstreamURL *url.URL) *url.URL {
	var joinedURL url.URL

	joinedURL.Scheme = upstreamURL.Scheme
	joinedURL.Host = upstreamURL.Host

	joinedURL.Path = strings.TrimRight(upstreamURL.Path, "/") + "/" + strings.TrimLeft(reqURL.Path, "/")
	joinedURL.RawQuery = upstreamURL.RawQuery + "&" + reqURL.RawQuery

	// Sort the query keys for consistency using `Encode()`.
	joinedURL.RawQuery = joinedURL.Query().Encode()

	return &joinedURL
}

// List of hop-by-hop headers taken from the official Go stdlib.
var hopHeaders = []string{
	"Connection",
	"Proxy-Connection",
	"Keep-Alive",
	"Proxy-Authenticate",
	"Proxy-Authorization",
	"Te",
	"Trailer",
	"Transfer-Encoding",
	"Upgrade",
}

// removeHopHeaders deletes hop-by-hop headers. Hop-by-hop headers are only for a single
// connection between client and server, and must not be retransmitted upstream.
func removeHopHeaders(header http.Header) {
	for _, h := range header.Values("Connection") {
		for dirtyHeader := range strings.SplitSeq(h, ",") {
			header.Del(strings.TrimSpace(dirtyHeader))
		}
	}

	for _, hopHeader := range hopHeaders {
		header.Del(hopHeader)
	}
}

// copyHeaders does a deep-copy of all the headers from `src` to `dst`.
func copyHeaders(dst, src http.Header) {
	for key, values := range src {
		for _, value := range values {
			dst.Add(key, value)
		}
	}
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

// serveFromCache checks if the request can be served from cache.
func serveFromCache(req *http.Request) bool {
	if req.Method != http.MethodGet && req.Method != http.MethodHead {
		return false
	}

	return true
}

// saveToCache checks if the response can be saved to cache.
func saveToCache(req *http.Request, res *http.Response) bool {
	if req.Method != http.MethodGet && req.Method != http.MethodHead {
		return false
	}

	cacheCtrlHeader := parseCacheCtrlHeader(res.Header)

	if slices.Contains(cacheCtrlHeader, "private") ||
		slices.Contains(cacheCtrlHeader, "no-cache") ||
		slices.Contains(cacheCtrlHeader, "no-store") ||
		res.Header.Get("Set-Cookie") != "" ||
		req.Header.Get("Authorization") != "" {
		return false
	}

	return true
}
