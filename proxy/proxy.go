package proxy

import (
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/rs/zerolog/log"

	"github.com/saifat29/goomerang/config"
)

const (
	XForwardedFor string = "X-Forwarded-For"
)

// ReverseProxy implements an HTTP-only proxy.
type ReverseProxy struct {
	routes    []Route
	transport http.RoundTripper
}

// NewReverseProxy returns a new ReverseProxy with the provided routes and transport.
func NewReverseProxy(routes []Route, transport http.RoundTripper) *ReverseProxy {
	return &ReverseProxy{
		routes:    routes,
		transport: transport,
	}
}

// RoundTripper returns a new http.RoundTripper with the provided upstream configuration.
func RoundTripper(cfg *config.Upstream) http.RoundTripper {
	return &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   cfg.DialTimeout,
			KeepAlive: cfg.KeepAliveProbes,
		}).DialContext,
		DisableKeepAlives:   cfg.DisableKeepAlives,
		MaxIdleConns:        cfg.MaxIdleConns,
		MaxIdleConnsPerHost: cfg.MaxIdleConnsPerHost,
		MaxConnsPerHost:     cfg.MaxConnsPerHost,
		IdleConnTimeout:     cfg.IdleConnTimeout,
	}
}

func (p *ReverseProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	route := p.findRoute(r.URL.Path)
	if route == nil {
		log.Warn().Str("path", r.URL.Path).Msg("no upstream route found")
		http.Error(w, "no upstream route found", http.StatusNotFound)
		return
	}

	upstream := p.upstreamHandler(route)

	route.Handler(upstream).ServeHTTP(w, r)
}

// upstreamHandler accepts a `Route` and returns an `http.HandlerFunc`
// that proxies the request to the upstream server.
func (p *ReverseProxy) upstreamHandler(route *Route) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		newReq := r.Clone(r.Context())

		removeHopHeaders(newReq.Header)

		newReq.URL = buildUpstreamURL(newReq.URL, route.UpstreamURL)
		newReq.RequestURI = ""
		newReq.Host = route.UpstreamURL.Host

		clientIP, _, err := net.SplitHostPort(newReq.RemoteAddr)
		if err == nil {
			// Assuming this proxy is running on the edge. Ideally, we must check `X-Forwarded-For`
			// and append the `RemoteAddr` if we're behind another proxy.
			newReq.Header.Set(XForwardedFor, clientIP)
		}

		res, err := p.transport.RoundTrip(newReq)
		if err != nil {
			log.Error().Err(err).Str("upstream", newReq.URL.String()).Msg("upstream server error")
			http.Error(w, "upstream server error", http.StatusInternalServerError)
			return
		}
		defer res.Body.Close()

		resBody, err := io.ReadAll(res.Body)
		if err != nil {
			log.Error().Err(err).Msg("failed to read upstream response body")
			http.Error(w, "failed to read response body from upstream server", http.StatusInternalServerError)
			return
		}

		removeHopHeaders(res.Header)
		CopyHeaders(w.Header(), res.Header)

		w.WriteHeader(res.StatusCode)

		if _, err := w.Write(resBody); err != nil {
			log.Error().Err(err).Msg("failed to write response")
			http.Error(w, "failed to write response", http.StatusInternalServerError)
			return
		}
	})
}

// findRoute finds the best matching route for the given path.
// It returns the route with the longest matching prefix.
// At high traffic this will surely be a bottleneck, so a more
// efficient solution would be needed.
func (p *ReverseProxy) findRoute(path string) *Route {
	var bestMatch *Route
	var bestLen int

	for i := range p.routes {
		if p.routes[i].pathMatched(path) && len(p.routes[i].Path) > bestLen {
			bestMatch = &p.routes[i]
			bestLen = len(p.routes[i].Path)
		}
	}

	return bestMatch
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

// CopyHeaders does a deep-copy of all the headers from `src` to `dst`.
func CopyHeaders(dst, src http.Header) {
	for key, values := range src {
		for _, value := range values {
			dst.Add(key, value)
		}
	}
}

// AddHeaders adds all the headers from `src` to `dst`, while keeping the
// existing headers in `dst` intact. If a header already exists in `dst`,
// it will be overwritten.
func AddHeaders(dst, src http.Header) {
	for key, values := range src {
		for _, value := range values {
			dst.Set(key, value)
		}
	}
}
