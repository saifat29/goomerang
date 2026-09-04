package main

import (
	"log"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/saifat29/goomerang/cache"
	"github.com/saifat29/goomerang/proxy"
)

const (
	serverAddr  string = ":8080"
	upstreamURL string = "http://httpbin.org"
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

	c := cache.NewMemoryLRU(1<<30, 5*time.Minute)

	reverseProxy := proxy.NewReverseProxy(parsedURL, transport, c)

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
