package main

import (
	"log"
	"net"
	"net/http"

	"github.com/saifat29/goomerang/cache"
	"github.com/saifat29/goomerang/config"
	"github.com/saifat29/goomerang/proxy"
)

func main() {
	log.Println("starting boomerang")

	cfg, err := config.Load("goomerang.yml")
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}
	log.Printf("configuration loaded: %s", cfg)

	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   cfg.Upstream.DialTimeout,
			KeepAlive: cfg.Upstream.KeepAliveProbes,
		}).DialContext,
		DisableKeepAlives:   false,
		MaxIdleConns:        cfg.Upstream.MaxIdleConns,
		MaxIdleConnsPerHost: cfg.Upstream.MaxIdleConnsPerHost,
		MaxConnsPerHost:     cfg.Upstream.MaxConnsPerHost,
		IdleConnTimeout:     cfg.Upstream.IdleConnTimeout,
	}

	c := cache.NewMemoryLRU(cfg.Cache.MaxSizeBytes, cfg.Cache.TTL)

	reverseProxy := proxy.NewReverseProxy(cfg.Proxy.UpstreamURL.URL, transport, c)

	server := &http.Server{
		Addr:         cfg.Server.Addr,
		Handler:      reverseProxy,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	log.Printf("starting http server on %s", cfg.Server.Addr)

	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("failed to start http server: %v", err)
	}
}
