package config

import "fmt"

// Validate validates if the configuration is valid after defaults and
// user configuration is merged.
func Validate(cfg *Config) error {
	if cfg.Server != nil {
		if cfg.Server.ReadTimeout < 0 {
			return fmt.Errorf("server.read_timeout must be non-negative")
		}
		if cfg.Server.WriteTimeout < 0 {
			return fmt.Errorf("server.write_timeout must be non-negative")
		}
		if cfg.Server.IdleTimeout < 0 {
			return fmt.Errorf("server.idle_timeout must be non-negative")
		}
	}

	if cfg.Cache != nil {
		if cfg.Cache.MaxSizeBytes < 0 {
			return fmt.Errorf("cache.max_size_bytes must be non-negative")
		}
		if cfg.Cache.TTL < 0 {
			return fmt.Errorf("cache.ttl must be non-negative")
		}
	}

	if cfg.Upstream != nil {
		if cfg.Upstream.MaxIdleConns < 0 {
			return fmt.Errorf("upstream.max_idle_conns must be non-negative")
		}
		if cfg.Upstream.MaxIdleConnsPerHost < 0 {
			return fmt.Errorf("upstream.max_idle_conns_per_host must be non-negative")
		}
		if cfg.Upstream.MaxConnsPerHost < 0 {
			return fmt.Errorf("upstream.max_conns_per_host must be non-negative")
		}
		if cfg.Upstream.DialTimeout < 0 {
			return fmt.Errorf("upstream.dial_timeout must be non-negative")
		}
		if cfg.Upstream.IdleConnTimeout < 0 {
			return fmt.Errorf("upstream.idle_conn_timeout must be non-negative")
		}
	}

	if len(cfg.Proxy) > 0 {
		for i, proxy := range cfg.Proxy {
			if proxy.Path == "" {
				return fmt.Errorf("proxy[%d].path is required", i)
			}
			if proxy.Upstream == nil || proxy.Upstream.URL == nil {
				return fmt.Errorf("proxy[%d].upstream URL is required", i)
			}
		}
	}

	return nil
}
