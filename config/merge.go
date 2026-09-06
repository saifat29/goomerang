package config

// Merge applies the user provided configs on top of our default base configs.
func Merge(base, user *Config) *Config {
	if user == nil {
		return base
	}

	if user.Logging != nil {
		mergeLogging(base.Logging, user.Logging)
	}

	if user.Server != nil {
		mergeServer(base.Server, user.Server)
	}

	if user.Cache != nil {
		mergeCache(base.Cache, user.Cache)
	}

	if user.Upstream != nil {
		mergeUpstream(base.Upstream, user.Upstream)
	}

	if user.Proxy != nil {
		base.Proxy = user.Proxy
	}

	return base
}

func mergeLogging(base, user *Logging) {
	if user.Level != "" {
		base.Level = user.Level
	}
	if user.Format != "" {
		base.Format = user.Format
	}
}

func mergeServer(base, user *Server) {
	if user.Addr != "" {
		base.Addr = user.Addr
	}
	if user.ReadTimeout != 0 {
		base.ReadTimeout = user.ReadTimeout
	}
	if user.WriteTimeout != 0 {
		base.WriteTimeout = user.WriteTimeout
	}
	if user.IdleTimeout != 0 {
		base.IdleTimeout = user.IdleTimeout
	}
}

func mergeCache(base, user *Cache) {
	if user.MaxSizeBytes != 0 {
		base.MaxSizeBytes = user.MaxSizeBytes
	}
	if user.TTL != 0 {
		base.TTL = user.TTL
	}
}

func mergeUpstream(base, user *Upstream) {
	if user.DialTimeout != 0 {
		base.DialTimeout = user.DialTimeout
	}
	if user.KeepAliveProbes != 0 {
		base.KeepAliveProbes = user.KeepAliveProbes
	}
	if user.DisableKeepAlives {
		base.DisableKeepAlives = user.DisableKeepAlives
	}
	if user.MaxIdleConns != 0 {
		base.MaxIdleConns = user.MaxIdleConns
	}
	if user.MaxIdleConnsPerHost != 0 {
		base.MaxIdleConnsPerHost = user.MaxIdleConnsPerHost
	}
	if user.MaxConnsPerHost != 0 {
		base.MaxConnsPerHost = user.MaxConnsPerHost
	}
	if user.IdleConnTimeout != 0 {
		base.IdleConnTimeout = user.IdleConnTimeout
	}
}
