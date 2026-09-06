package main

import (
	"flag"
	"fmt"
	"net/http"

	"github.com/rs/zerolog/log"

	"github.com/saifat29/goomerang/cache"
	"github.com/saifat29/goomerang/config"
	"github.com/saifat29/goomerang/proxy"
	"github.com/saifat29/goomerang/proxy/middleware/http_cache"
	"github.com/saifat29/goomerang/proxy/middleware/logger"
)

func main() {
	versionFlag := flag.Bool("version", false, "print version and exit")
	flag.Parse()

	if *versionFlag {
		fmt.Printf("goomerang %s\n", version)
		return
	}

	// Load config, if err then use default config.
	cfg, err := config.Load("goomerang.yml")
	if err != nil {
		log.Warn().Err(err).Msg("loading default config")
		cfg = config.DefaultConfig()
	}

	// Setup logger
	setupLogger(cfg)

	log.Info().Msg("starting goomerang")
	log.Info().Str("config", cfg.String()).Msg("configuration loaded")

	// Initialise cache.
	proxyCache := cache.NewMemoryLRU(cfg.Cache.MaxSizeBytes, cfg.Cache.TTL)

	// Initialise middleware registry and register middlewares.
	mwRegistry := proxy.NewMiddlewareRegistry()
	mwRegistry.Register("logger", logger.New())
	mwRegistry.Register("http_cache", http_cache.New(proxyCache, cfg.Cache.TTL))

	// Initialize reverse proxy with routes and transport.
	reverseProxy := proxy.NewReverseProxy(
		proxy.FromConfig(cfg.Proxy, mwRegistry),
		proxy.RoundTripper(cfg.Upstream),
	)

	// Initialise HTTP server
	server := &http.Server{
		Addr:         cfg.Server.Addr,
		Handler:      reverseProxy,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	log.Info().Str("addr", cfg.Server.Addr).Msg("starting http server")

	// Start listening and serving.
	if err := server.ListenAndServe(); err != nil {
		log.Fatal().Err(err).Msg("failed to start http server")
	}
}
