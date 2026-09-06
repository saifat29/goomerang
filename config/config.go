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
	DefaultLoggingLevel  = "info"
	DefaultLoggingFormat = "json"

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
	Logging  *Logging  `json:"logging" yaml:"logging"`
	Server   *Server   `json:"server" yaml:"server"`
	Cache    *Cache    `json:"cache" yaml:"cache"`
	Upstream *Upstream `json:"upstream" yaml:"upstream"`
	Proxy    []*Proxy  `json:"proxy" yaml:"proxy"`
}

// Default returns a Config with all fields populated with defaults values.
func Default() *Config {
	cfg := &Config{
		Logging:  &Logging{},
		Server:   &Server{},
		Cache:    &Cache{},
		Upstream: &Upstream{},
	}
	cfg.SetDefaults()

	return cfg
}

// Load reads the configuration from the specified YAML file.
// It handles validation, and merging with default values.
func Load(path string) (*Config, error) {
	file, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("config file %q does not exist: %w", path, err)
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}
	defer file.Close()

	var userCfg Config
	if err = yaml.NewDecoder(file).Decode(&userCfg); err != nil {
		return nil, fmt.Errorf("failed to decode YAML: %w", err)
	}

	cfg := Default()
	cfg = Merge(cfg, &userCfg)
	cfg.SetDefaults()

	if err := Validate(cfg); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return cfg, nil
}

func (c *Config) SetDefaults() {
	if c.Logging != nil {
		c.Logging.SetDefaults()
	}
	if c.Server != nil {
		c.Server.SetDefaults()
	}
	if c.Cache != nil {
		c.Cache.SetDefaults()
	}
	if c.Upstream != nil {
		c.Upstream.SetDefaults()
	}
}

func (c *Config) String() string {
	b, err := json.Marshal(c)
	if err != nil {
		return fmt.Sprintf("error formatting config: %v", err)
	}

	return string(b)
}

// Logging contains the configuration fields for the logger.
type Logging struct {
	Level  string `json:"level" yaml:"level"`
	Format string `json:"format" yaml:"format"`
}

func (l *Logging) SetDefaults() {
	if l.Level == "" {
		l.Level = DefaultLoggingLevel
	}
	if l.Format == "" {
		l.Format = DefaultLoggingFormat
	}
}

// Server contains the configuration fields for the HTTP server.
type Server struct {
	Addr         string        `json:"addr" yaml:"addr"`
	ReadTimeout  time.Duration `json:"read_timeout" yaml:"read_timeout"`
	WriteTimeout time.Duration `json:"write_timeout" yaml:"write_timeout"`
	IdleTimeout  time.Duration `json:"idle_timeout" yaml:"idle_timeout"`
}

func (s *Server) SetDefaults() {
	if s.Addr == "" {
		s.Addr = DefaultServerAddr
	}
	if s.ReadTimeout == 0 {
		s.ReadTimeout = DefaultReadTimeout
	}
	if s.WriteTimeout == 0 {
		s.WriteTimeout = DefaultWriteTimeout
	}
	if s.IdleTimeout == 0 {
		s.IdleTimeout = DefaultIdleTimeout
	}
}

// Cache contains the configuration fields for the cache.
type Cache struct {
	MaxSizeBytes int           `json:"max_size_bytes" yaml:"max_size_bytes"`
	TTL          time.Duration `json:"ttl" yaml:"ttl"`
}

func (c *Cache) SetDefaults() {
	if c.MaxSizeBytes == 0 {
		c.MaxSizeBytes = DefaultCacheSize
	}
	if c.TTL == 0 {
		c.TTL = DefaultCacheTTL
	}
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

func (u *Upstream) SetDefaults() {
	if u.DialTimeout == 0 {
		u.DialTimeout = DefaultDialTimeout
	}
	if u.KeepAliveProbes == 0 {
		u.KeepAliveProbes = DefaultKeepAliveProbe
	}
	if u.MaxIdleConns == 0 {
		u.MaxIdleConns = DefaultMaxIdleConns
	}
	if u.MaxIdleConnsPerHost == 0 {
		u.MaxIdleConnsPerHost = DefaultMaxIdleConnsPerHost
	}
	if u.MaxConnsPerHost == 0 {
		u.MaxConnsPerHost = DefaultMaxConnsPerHost
	}
	if u.IdleConnTimeout == 0 {
		u.IdleConnTimeout = DefaultIdleConnTimeout
	}
}

// Proxy contains the configuration fields for the proxy.
type Proxy struct {
	Path        string        `json:"path" yaml:"path"`
	Upstream    *URL          `json:"upstream" yaml:"upstream"`
	Middlewares []*Middleware `json:"middlewares" yaml:"middlewares"`
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
	return u.String(), nil
}

func (u *URL) MarshalJSON() ([]byte, error) {
	if u == nil || u.URL == nil {
		return nil, nil
	}
	return json.Marshal(u.String())
}
