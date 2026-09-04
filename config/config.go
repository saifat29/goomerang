package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"time"

	"go.yaml.in/yaml/v3"
)

const (
	DefaultServerAddr   = ":8080"
	DefaultReadTimeout  = 30 * time.Second
	DefaultWriteTimeout = 30 * time.Second
	DefaultIdleTimeout  = 30 * time.Second

	DefaultCacheSize = 100 * (1 << 20)
	DefaultCacheTTL  = 5 * time.Minute

	DefaultDialTimeout         = 30 * time.Second
	DefaultKeepAliveProbe      = 30 * time.Second
	DefaultDisableKeepAlive    = false
	DefaultMaxIdleConns        = 100
	DefaultMaxIdleConnsPerHost = 100
	DefaultMaxConnsPerHost     = 100
	DefaultIdleConnTimeout     = 90 * time.Second
)

// Config is the container for all the configuration fields required by the application.
type Config struct {
	Server   Server   `json:"server" yaml:"server"`
	Cache    Cache    `json:"cache" yaml:"cache"`
	Upstream Upstream `json:"upstream" yaml:"upstream"`
	Proxy    []Proxy  `json:"proxy" yaml:"proxy"`
}

// Server contains the configuration fields for the HTTP server.
type Server struct {
	Addr         string        `json:"addr" yaml:"addr"`
	ReadTimeout  time.Duration `json:"read_timeout" yaml:"read_timeout"`
	WriteTimeout time.Duration `json:"write_timeout" yaml:"write_timeout"`
	IdleTimeout  time.Duration `json:"idle_timeout" yaml:"idle_timeout"`
}

// Cache contains the configuration fields for the cache.
type Cache struct {
	MaxSizeBytes int           `json:"max_size_bytes" yaml:"max_size_bytes"`
	TTL          time.Duration `json:"ttl" yaml:"ttl"`
}

// Upstream contains the configuration fields for the upstream server.
type Upstream struct {
	DialTimeout         time.Duration `json:"dial_timeout" yaml:"dial_timeout"`
	KeepAliveProbes     time.Duration `json:"keep_alive_probes" yaml:"keep_alive_probes"`
	DisableKeepAlives   bool          `json:"disable_keep_alives" yaml:"disable_keep_alives"`
	MaxIdleConns        int           `json:"max_idle_conns" yaml:"max_idle_conns"`
	MaxIdleConnsPerHost int           `json:"max_idle_conns_per_host" yaml:"max_idle_conns_per_host"`
	MaxConnsPerHost     int           `json:"max_conns_per_host" yaml:"max_conns_per_host"`
	IdleConnTimeout     time.Duration `json:"idle_conn_timeout" yaml:"idle_conn_timeout"`
}

// Proxy contains the configuration fields for the proxy.
type Proxy struct {
	Path     string `json:"path" yaml:"path"`
	Upstream *URL   `json:"upstream" yaml:"upstream"`
}

func (c *Config) String() string {
	b, err := json.Marshal(c)
	if err != nil {
		return fmt.Sprintf("error formatting config: %v", err)
	}

	return string(b)
}

// URL contains a parsed URL.
type URL struct {
	*url.URL
}

// UnmarshalYAML implements the yaml.Unmarshaler interface for the URL type.
func (u *URL) UnmarshalYAML(node *yaml.Node) error {
	var s string
	if err := node.Decode(&s); err != nil {
		return err
	}

	parsed, err := url.Parse(s)
	if err != nil {
		return fmt.Errorf("invalid url %q: %w", s, err)
	}

	u.URL = parsed
	return nil
}

// MarshalYAML implements the yaml.Marshaler interface to print the URL as a clean string.
func (u *URL) MarshalYAML() (interface{}, error) {
	if u == nil || u.URL == nil {
		return "", nil
	}
	return u.URL.String(), nil
}

func (u *URL) MarshalJSON() ([]byte, error) {
	if u == nil || u.URL == nil {
		return nil, nil
	}
	return json.Marshal(u.URL.String())
}

// Load reads the configuration from the specified YAML file.
func Load(path string) (*Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}
	defer file.Close()

	var cfg Config
	if err := yaml.NewDecoder(file).Decode(&cfg); err != nil {
		return nil, fmt.Errorf("failed to decode YAML: %w", err)
	}

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	cfg.applyDefaults()

	return &cfg, err
}

// validate checks if the required fields in the configuration are set
// and valid and returns an error if not.
func (c *Config) validate() error {
	if len(c.Proxy) == 0 {
		return errors.New("at least one proxy configuration is required")
	}

	for _, proxy := range c.Proxy {
		if proxy.Path == "" {
			return errors.New("proxy path is required")
		}
		if proxy.Upstream == nil || proxy.Upstream.URL == nil {
			return errors.New("proxy upstream URL is required")
		}
	}

	return nil
}

// applyDefaults sets default values for the configuration fields if they are not provided.
func (c *Config) applyDefaults() {
	if c.Server.Addr == "" {
		c.Server.Addr = DefaultServerAddr
	}
	if c.Server.ReadTimeout == 0 {
		c.Server.ReadTimeout = DefaultReadTimeout
	}
	if c.Server.WriteTimeout == 0 {
		c.Server.WriteTimeout = DefaultWriteTimeout
	}
	if c.Server.IdleTimeout == 0 {
		c.Server.IdleTimeout = DefaultIdleTimeout
	}

	if c.Cache.MaxSizeBytes < DefaultCacheSize {
		c.Cache.MaxSizeBytes = DefaultCacheSize
	}
	if c.Cache.TTL == 0 {
		c.Cache.TTL = DefaultCacheTTL
	}

	if c.Upstream.DialTimeout == 0 {
		c.Upstream.DialTimeout = DefaultDialTimeout
	}
	if c.Upstream.KeepAliveProbes == 0 {
		c.Upstream.KeepAliveProbes = DefaultKeepAliveProbe
	}

	if c.Upstream.MaxIdleConns == 0 {
		c.Upstream.MaxIdleConns = DefaultMaxIdleConns
	}
	if c.Upstream.MaxIdleConnsPerHost == 0 {
		c.Upstream.MaxIdleConnsPerHost = DefaultMaxIdleConnsPerHost
	}
	if c.Upstream.MaxConnsPerHost == 0 {
		c.Upstream.MaxConnsPerHost = DefaultMaxConnsPerHost
	}
	if c.Upstream.IdleConnTimeout == 0 {
		c.Upstream.IdleConnTimeout = DefaultIdleConnTimeout
	}
}
