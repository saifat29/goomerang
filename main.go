package main

import (
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"
)

const (
	serverAddr  string = ":8080"
	upstreamURL string = "http://httpbin.org"

	XForwardedFor string = "X-Forwarded-For"
)

func main() {
	parsedURL, err := url.Parse(upstreamURL)
	if err != nil {
		log.Fatalf("failed to parse upstream URL: %s", upstreamURL)
	}

	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		DisableKeepAlives:   false,
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 100,
		MaxConnsPerHost:     100,
		IdleConnTimeout:     90 * time.Second,
	}

	reverseProxy := NewReverseProxy(parsedURL, transport)

	server := &http.Server{
		Addr:         serverAddr,
		Handler:      reverseProxy,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  30 * time.Second,
	}

	log.Printf("starting http server on %s", serverAddr)

	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("failed to start http server: %v", err)
	}
}

// ReverseProxy implements an HTTP-only proxy.
type ReverseProxy struct {
	upstreamURL *url.URL
	transport   http.RoundTripper
}

// NewReverseProxy returns a new ReverseProxy with the provided upstream URL
// and transport, if the transport is not provided, a default is used.
func NewReverseProxy(upstreamURL *url.URL, transport http.RoundTripper) *ReverseProxy {
	if transport == nil {
		transport = &http.Transport{
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			DisableKeepAlives:   false,
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 100,
			MaxConnsPerHost:     100,
			IdleConnTimeout:     90 * time.Second,
		}
	}

	return &ReverseProxy{
		upstreamURL: upstreamURL,
		transport:   transport,
	}
}

func (p *ReverseProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	newReq := r.Clone(r.Context())

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
		log.Printf("upstream server error:%v", err)
		http.Error(w, "upstream server error", http.StatusInternalServerError)
		return
	}
	defer res.Body.Close()

	outDumpRes, _ := httputil.DumpResponse(res, true)
	log.Println(string(outDumpRes))

	removeHopHeaders(res.Header)
	copyHeaders(w.Header(), res.Header)

	w.WriteHeader(res.StatusCode)

	if _, err := io.Copy(w, res.Body); err != nil {
		log.Printf("failed to write response:%v", err)
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
	for _, h := range header["Connection"] {
		for dirtyHeader := range strings.SplitSeq(h, ",") {
			header.Del(strings.TrimSpace(dirtyHeader))
		}
	}

	for _, hopHeader := range hopHeaders {
		header.Del(hopHeader)
	}
}

// copyHeaders does a deep-copy of all the headers from `src“ to `dst`.
func copyHeaders(dst, src http.Header) {
	for key, values := range src {
		for _, value := range values {
			dst.Add(key, value)
		}
	}
}
