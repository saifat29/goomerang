package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os/signal"
	"syscall"

	"github.com/oschwald/maxminddb-golang/v2"
	"github.com/rs/zerolog/log"

	"github.com/saifat29/goomerang/assets"
	"github.com/saifat29/goomerang/cache"
	"github.com/saifat29/goomerang/config"
	"github.com/saifat29/goomerang/proxy"
	"github.com/saifat29/goomerang/proxy/middleware"
	"github.com/saifat29/goomerang/proxy/middleware/geoip"
	"github.com/saifat29/goomerang/proxy/middleware/http_cache"
	"github.com/saifat29/goomerang/proxy/middleware/logger"
	"github.com/saifat29/goomerang/proxy/middleware/strip_prefix"
)

func main() {
	if err := run(context.Background()); err != nil {
		log.Fatal().Err(err).Msg("failed to start goomerang")
	}
	log.Info().Msg("goomerang stopped")
}

func run(ctx context.Context) error {
	ctx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	versionFlag := flag.Bool("version", false, "print version and exit")
	flag.Parse()

	if *versionFlag {
		fmt.Printf("goomerang %s\n", version)
		return nil
	}

	// Load config, if err then use default config.
	cfg, err := config.Load("goomerang.yml")
	if err != nil {
		log.Warn().Err(err).Msg("loading default config")
		cfg = config.Default()
	}

	// Setup logger
	setupLogger(cfg)

	log.Info().Msg("starting goomerang")
	log.Info().Str("config", cfg.String()).Msg("configuration loaded")

	// Open GeoIP database.
	geoipDB, err := maxminddb.OpenBytes(assets.GeoIPDB)
	if err != nil {
		return fmt.Errorf("failed to open geoip database: %w", err)
	}
	defer geoipDB.Close()

	// Initialise cache.
	proxyCache := cache.NewMemoryLRU(cfg.Cache.MaxSizeBytes, cfg.Cache.TTL)

	// Initialise middleware registry and register middlewares.)
	mwRegistry := middleware.NewRegistry()
	mwRegistry.Register(
		config.MiddlewareLogger, func(mwCfg *config.Middleware) middleware.Middleware {
			return logger.New(mwCfg.Logger)
		})

	mwRegistry.Register(
		config.MiddlewareHTTPCache, func(mwCfg *config.Middleware) middleware.Middleware {
			return http_cache.New(mwCfg.HTTPCache, proxyCache)
		})

	mwRegistry.Register(
		config.MiddlewareStripPrefix, func(mwCfg *config.Middleware) middleware.Middleware {
			return strip_prefix.New(mwCfg.StripPrefix)
		})

	mwRegistry.Register(
		config.MiddlewareGeoIP, func(mwCfg *config.Middleware) middleware.Middleware {
			return geoip.New(geoipDB)
		})

	// Initialize reverse proxy with routes and transport.
	reverseProxy := proxy.NewReverseProxy(
		proxy.FromConfig(cfg.Proxy, mwRegistry),
		proxy.RoundTripper(cfg.Upstream),
	)
	defer reverseProxy.Shutdown()

	// Start the HTTP server.
	if err := runServer(ctx, cfg.Server, reverseProxy); err != nil {
		return fmt.Errorf("failed to start http server: %w", err)
	}

	return nil
}

func runServer(ctx context.Context, cfg *config.Server, handler http.Handler) error {
	server := &http.Server{
		Addr:         cfg.Addr,
		Handler:      handler,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Info().Str("addr", cfg.Addr).Msg("starting http server")
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		log.Error().Err(err).Msg("failed to start http server")
		return err
	case <-ctx.Done():
		log.Info().Msg("starting graceful shutdown")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Error().Err(err).Msg("failed to shutdown http server gracefully")
	}

	return nil
}
