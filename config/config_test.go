package config

import (
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefault(t *testing.T) {
	cfg := Default()

	assert.Equal(t, DefaultLoggingLevel, cfg.Logging.Level)
	assert.Equal(t, DefaultLoggingFormat, cfg.Logging.Format)

	assert.Equal(t, DefaultServerAddr, cfg.Server.Addr)
	assert.Equal(t, DefaultReadTimeout, cfg.Server.ReadTimeout)
	assert.Equal(t, DefaultWriteTimeout, cfg.Server.WriteTimeout)
	assert.Equal(t, DefaultIdleTimeout, cfg.Server.IdleTimeout)

	assert.Equal(t, DefaultCacheSize, cfg.Cache.MaxSizeBytes)
	assert.Equal(t, DefaultCacheTTL, cfg.Cache.TTL)

	assert.Equal(t, DefaultDialTimeout, cfg.Upstream.DialTimeout)
	assert.Equal(t, DefaultKeepAliveProbe, cfg.Upstream.KeepAliveProbes)
	assert.Equal(t, DefaultMaxIdleConns, cfg.Upstream.MaxIdleConns)
	assert.Equal(t, DefaultMaxIdleConnsPerHost, cfg.Upstream.MaxIdleConnsPerHost)
	assert.Equal(t, DefaultMaxConnsPerHost, cfg.Upstream.MaxConnsPerHost)
	assert.Equal(t, DefaultIdleConnTimeout, cfg.Upstream.IdleConnTimeout)
}

func TestSetDefaults_ZeroValuesOnly(t *testing.T) {
	s := &Server{
		Addr:         ":3000",
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 0,
		IdleTimeout:  0,
	}
	s.SetDefaults()

	assert.Equal(t, ":3000", s.Addr, "should not overwrite user value")
	assert.Equal(t, 10*time.Second, s.ReadTimeout, "should not overwrite user value")
	assert.Equal(t, DefaultWriteTimeout, s.WriteTimeout, "should set default for zero value")
	assert.Equal(t, DefaultIdleTimeout, s.IdleTimeout, "should set default for zero value")
}

func TestSetDefaults_Logging(t *testing.T) {
	l := &Logging{Level: "debug"}
	l.SetDefaults()

	assert.Equal(t, "debug", l.Level, "should not overwrite user value")
	assert.Equal(t, DefaultLoggingFormat, l.Format, "should set default for zero value")
}

func TestSetDefaults_Cache(t *testing.T) {
	c := &Cache{TTL: 10 * time.Minute}
	c.SetDefaults()

	assert.Equal(t, DefaultCacheSize, c.MaxSizeBytes, "should set default for zero value")
	assert.Equal(t, 10*time.Minute, c.TTL, "should not overwrite user value")
}

func TestSetDefaults_Upstream(t *testing.T) {
	u := &Upstream{MaxIdleConns: 50}
	u.SetDefaults()

	assert.Equal(t, DefaultDialTimeout, u.DialTimeout, "should set default for zero value")
	assert.Equal(t, 50, u.MaxIdleConns, "should not overwrite user value")
	assert.Equal(t, DefaultMaxIdleConnsPerHost, u.MaxIdleConnsPerHost, "should set default for zero value")
}

func TestSetDefaults_Config(t *testing.T) {
	cfg := &Config{
		Server: &Server{Addr: ":9090"},
	}
	cfg.SetDefaults()

	assert.Equal(t, ":9090", cfg.Server.Addr, "should not overwrite user value")
	assert.Equal(t, DefaultReadTimeout, cfg.Server.ReadTimeout, "should set default for zero value")
	assert.Nil(t, cfg.Logging, "nil sub-struct should remain nil")
}

func TestMerge_NilUser(t *testing.T) {
	base := Default()
	result := Merge(base, nil)

	assert.Equal(t, base, result)
}

func TestMerge_PartialServer(t *testing.T) {
	base := Default()
	user := &Config{
		Server: &Server{Addr: ":9090"},
	}

	result := Merge(base, user)

	assert.Equal(t, ":9090", result.Server.Addr)
	assert.Equal(t, DefaultReadTimeout, result.Server.ReadTimeout)
	assert.Equal(t, DefaultWriteTimeout, result.Server.WriteTimeout)
	assert.Equal(t, DefaultIdleTimeout, result.Server.IdleTimeout)
}

func TestMerge_PartialUpstream(t *testing.T) {
	base := Default()
	user := &Config{
		Upstream: &Upstream{MaxIdleConns: 200},
	}

	result := Merge(base, user)

	assert.Equal(t, 200, result.Upstream.MaxIdleConns)
	assert.Equal(t, DefaultDialTimeout, result.Upstream.DialTimeout)
	assert.Equal(t, DefaultMaxIdleConnsPerHost, result.Upstream.MaxIdleConnsPerHost)
}

func TestMerge_FullUserValues(t *testing.T) {
	base := Default()
	user := &Config{
		Logging:  &Logging{Level: "debug", Format: "console"},
		Server:   &Server{Addr: ":9090", ReadTimeout: 10 * time.Second, WriteTimeout: 20 * time.Second, IdleTimeout: 60 * time.Second},
		Cache:    &Cache{MaxSizeBytes: 500, TTL: 1 * time.Minute},
		Upstream: &Upstream{DialTimeout: 5 * time.Second, MaxIdleConns: 50},
	}

	result := Merge(base, user)

	assert.Equal(t, "debug", result.Logging.Level)
	assert.Equal(t, "console", result.Logging.Format)
	assert.Equal(t, ":9090", result.Server.Addr)
	assert.Equal(t, 10*time.Second, result.Server.ReadTimeout)
	assert.Equal(t, 500, result.Cache.MaxSizeBytes)
	assert.Equal(t, 1*time.Minute, result.Cache.TTL)
	assert.Equal(t, 5*time.Second, result.Upstream.DialTimeout)
	assert.Equal(t, 50, result.Upstream.MaxIdleConns)
}

func TestMerge_ProxyReplaces(t *testing.T) {
	base := Default()
	base.Proxy = []*Proxy{
		{Path: "/old", Upstream: &URL{URL: &url.URL{Scheme: "http", Host: "old.example.com"}}},
	}

	userProxy := &Proxy{Path: "/new", Upstream: &URL{URL: &url.URL{Scheme: "http", Host: "new.example.com"}}}
	user := &Config{Proxy: []*Proxy{userProxy}}

	result := Merge(base, user)

	require.Len(t, result.Proxy, 1)
	assert.Equal(t, "/new", result.Proxy[0].Path)
}

func TestMerge_BuildIsolation(t *testing.T) {
	base := Default()
	user := &Config{
		Server: &Server{Addr: ":9090"},
	}

	result := Merge(base, user)

	assert.Equal(t, ":9090", result.Server.Addr)
	assert.Equal(t, DefaultReadTimeout, result.Server.ReadTimeout)
	assert.Equal(t, DefaultWriteTimeout, result.Server.WriteTimeout)
	assert.Equal(t, DefaultIdleTimeout, result.Server.IdleTimeout)
}

func TestValidate_Valid(t *testing.T) {
	cfg := Default()
	err := Validate(cfg)
	assert.NoError(t, err)
}

func TestValidate_NegativeReadTimeout(t *testing.T) {
	cfg := Default()
	cfg.Server.ReadTimeout = -1 * time.Second

	err := Validate(cfg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "read_timeout")
}

func TestValidate_NegativeWriteTimeout(t *testing.T) {
	cfg := Default()
	cfg.Server.WriteTimeout = -1 * time.Second

	err := Validate(cfg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "write_timeout")
}

func TestValidate_NegativeIdleTimeout(t *testing.T) {
	cfg := Default()
	cfg.Server.IdleTimeout = -1 * time.Second

	err := Validate(cfg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "idle_timeout")
}

func TestValidate_NegativeCacheSize(t *testing.T) {
	cfg := Default()
	cfg.Cache.MaxSizeBytes = -1

	err := Validate(cfg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "max_size_bytes")
}

func TestValidate_NegativeCacheTTL(t *testing.T) {
	cfg := Default()
	cfg.Cache.TTL = -1 * time.Second

	err := Validate(cfg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ttl")
}

func TestValidate_NegativeMaxIdleConns(t *testing.T) {
	cfg := Default()
	cfg.Upstream.MaxIdleConns = -1

	err := Validate(cfg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "max_idle_conns")
}

func TestValidate_ProxyMissingPath(t *testing.T) {
	cfg := Default()
	cfg.Proxy = []*Proxy{
		{Upstream: &URL{URL: &url.URL{Scheme: "http", Host: "example.com"}}},
	}

	err := Validate(cfg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "path is required")
}

func TestValidate_ProxyMissingUpstream(t *testing.T) {
	cfg := Default()
	cfg.Proxy = []*Proxy{
		{Path: "/test"},
	}

	err := Validate(cfg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "upstream URL is required")
}

func TestLoad_FullYAML(t *testing.T) {
	content := `
logging:
  level: debug
  format: console
server:
  addr: :9090
  read_timeout: 10s
  write_timeout: 20s
  idle_timeout: 60s
cache:
  max_size_bytes: 500
  ttl: 1m
upstream:
  dial_timeout: 5s
  max_idle_conns: 50
proxy:
  - path: "/api"
    upstream: "http://example.com"
    middlewares:
      - logger
`
	path := writeTempYAML(t, content)
	defer os.Remove(path)

	cfg, err := Load(path)
	require.NoError(t, err)

	assert.Equal(t, "debug", cfg.Logging.Level)
	assert.Equal(t, "console", cfg.Logging.Format)
	assert.Equal(t, ":9090", cfg.Server.Addr)
	assert.Equal(t, 10*time.Second, cfg.Server.ReadTimeout)
	assert.Equal(t, 20*time.Second, cfg.Server.WriteTimeout)
	assert.Equal(t, 60*time.Second, cfg.Server.IdleTimeout)
	assert.Equal(t, 500, cfg.Cache.MaxSizeBytes)
	assert.Equal(t, 1*time.Minute, cfg.Cache.TTL)
	assert.Equal(t, 5*time.Second, cfg.Upstream.DialTimeout)
	assert.Equal(t, 50, cfg.Upstream.MaxIdleConns)
	require.Len(t, cfg.Proxy, 1)
	assert.Equal(t, "/api", cfg.Proxy[0].Path)
	assert.Equal(t, "http://example.com", cfg.Proxy[0].Upstream.String())
	assert.Equal(t, []string{"logger"}, cfg.Proxy[0].Middlewares)
}

func TestLoad_PartialYAML(t *testing.T) {
	content := `
server:
  addr: :3000
`
	path := writeTempYAML(t, content)
	defer os.Remove(path)

	cfg, err := Load(path)
	require.NoError(t, err)

	assert.Equal(t, ":3000", cfg.Server.Addr)
	assert.Equal(t, DefaultReadTimeout, cfg.Server.ReadTimeout)
	assert.Equal(t, DefaultWriteTimeout, cfg.Server.WriteTimeout)
	assert.Equal(t, DefaultIdleTimeout, cfg.Server.IdleTimeout)
	assert.Equal(t, DefaultLoggingLevel, cfg.Logging.Level)
	assert.Equal(t, DefaultCacheSize, cfg.Cache.MaxSizeBytes)
}

func TestLoad_EmptyYAML(t *testing.T) {
	content := `{}`
	path := writeTempYAML(t, content)
	defer os.Remove(path)

	cfg, err := Load(path)
	require.NoError(t, err)

	assert.Equal(t, DefaultServerAddr, cfg.Server.Addr)
	assert.Equal(t, DefaultReadTimeout, cfg.Server.ReadTimeout)
	assert.Equal(t, DefaultLoggingLevel, cfg.Logging.Level)
	assert.Equal(t, DefaultCacheSize, cfg.Cache.MaxSizeBytes)
}

func TestLoad_InvalidYAML(t *testing.T) {
	content := `invalid: [yaml: {{{`
	path := writeTempYAML(t, content)
	defer os.Remove(path)

	_, err := Load(path)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to decode YAML")
}

func TestLoad_NegativeTimeout(t *testing.T) {
	content := `
server:
  read_timeout: -1s
proxy:
  - path: "/test"
    upstream: "http://example.com"
`
	path := writeTempYAML(t, content)
	defer os.Remove(path)

	_, err := Load(path)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid config")
}

func TestLoad_FileNotFound(t *testing.T) {
	_, err := Load("/nonexistent/path/config.yml")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not exist")
}

func TestLoad_InvalidProxy(t *testing.T) {
	content := `
proxy:
  - path: ""
    upstream: "http://example.com"
`
	path := writeTempYAML(t, content)
	defer os.Remove(path)

	_, err := Load(path)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "path is required")
}

func TestLoad_MissingProxyUpstream(t *testing.T) {
	content := `
proxy:
  - path: "/test"
`
	path := writeTempYAML(t, content)
	defer os.Remove(path)

	_, err := Load(path)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "upstream URL is required")
}

func writeTempYAML(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yml")
	err := os.WriteFile(path, []byte(content), 0644)
	require.NoError(t, err)
	return path
}
