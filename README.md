<p align="center">
  <img src="assets/goomerang-logo-transparent.png" alt="Goomerang" width="400">
</p>

<h3 align="center">A <em>potentially</em> fast, configurable HTTP reverse proxy with built-in caching</h3>

---

## Features

- **Multi-upstream routing** - route requests to different upstream servers based on path prefixes (longest match wins)
- **In-memory LRU caching** - cache GET/HEAD responses with configurable global TTL and max size, and per-route TTL
- **Pluggable middleware system** - onion-style middleware chain, configurable per route with individual configs
- **Prefix stripping** - `strip_prefix` middleware removes path prefixes before forwarding to upstream
- **GeoIP lookup** - `geoip` middleware injects geoip headers using an embedded [MaxMind GeoLite2-City](https://dev.maxmind.com/geoip/geolite2-free-geolocation-data) database
- **Structured logging** - Fast logging using [Zerolog](https://github.com/rs/zerolog) with configurable levels (`trace`, `debug`, `info`, `warn`, `error`, `fatal`, `panic`) and format (`json`, `console`)
- **YAML configuration** - single config file `goomerang.yml` with sensible defaults for all settings
- **Cache-Control awareness _(sort of, not 100% as per HTTP spec)_** - respects `private`, `no-cache`, `no-store` directives and skips caching for responses with `Set-Cookie` and for requests with `Authorization`
- **Hop-by-hop header handling** - properly strips and forwards headers per HTTP spec
- **X-Forwarded-For** - injects client IP for proper identification
- **X-Cache-Status** - response header showing `HIT`, `MISS`, or `BYPASS`
- **Embedded GeoIP database** - GeoIP database is compiled into the binary, no external files needed at runtime

## Quick Start

### Download
- **Docker**: Run it with docker
```bash
docker run -v ./goomerang.yml:/goomerang.yml -p 8080:8080 ghcr.io/saifat29/goomerang
```

- **Binary**: Download the latest binary from [releases](https://github.com/saifat29/goomerang/releases)
```bash
./goomerang
```
Keep the `goomerang.yml` configuration file in the same directory as the binary. If the config file is not found, default config will be used.

- **Version flag**: Print the build version and exit
```bash
./goomerang -version
```

### Local Development
- Clone the source
```bash
git clone https://github.com/saifat29/goomerang.git
```

- Run locally
```bash
make run
```

- Run all unit tests
```bash
make test
```

- Format, Lint, Test, and Build (Binaries can be found in the `bin` directory)
```bash
make all
```

- Run in Docker
```bash
make docker-build
make docker-run
```

> Note: This project uses [Semantic Versioning](https://semver.org/)

## Configuration

Goomerang is configured via a YAML file called `goomerang.yml`

The `goomerang.yml` file:

```yaml
logging:
  level: info        # trace, debug, info, warn, error, fatal, panic
  format: json       # json, console

server:
  addr: :8080
  read_timeout: 30s
  write_timeout: 30s
  idle_timeout: 30s

cache:
  max_size_bytes: 104857600   # 100 MB
  ttl: 5m

upstream:
  dial_timeout: 30s
  keep_alive_probes: 30s
  disable_keep_alives: false
  max_idle_conns: 100
  max_idle_conns_per_host: 100
  max_conns_per_host: 100
  idle_conn_timeout: 90s

proxy:
  - path: "/data/json"
    upstream: "http://httpbin.org"
    middlewares:
      - strip_prefix:
          prefix: "/data"
      - logger: {}
      - geoip: {}
      - http_cache:
          ttl: 3s

  - path: "/ip"
    upstream: "http://httpbingo.org"
    middlewares:
      - logger: {}
      - http_cache:
          ttl: 10s

  - path: "/whoami"
    upstream: "http://whoami.localhost:8888"
    middlewares:
      - strip_prefix:
          prefix: "/whoami"
      - logger: {}
      - geoip: {}
```

All fields have sensible defaults. Any omitted field will use its default value.

> Note: All fields have defaults, EXCEPT the `proxy` block and it's children. You'll have to configure the proxy upstream servers for the reverse proxy to work.

## Middlewares

Middlewares are applied in the order listed in the `middlewares` list for each proxy route. Each proxy route can have its own independent middleware configuration.

| Middleware | Config Fields | Description |
|-----------|---------------|-------------|
| `logger` | (none) | Logs each request with method, path, remote address, user agent, duration, status, and response size |
| `http_cache` | `ttl` | Caches GET/HEAD responses in memory with per-route TTL. Adds `X-Cache-Status` header (`HIT`, `MISS`, or `BYPASS`) |
| `strip_prefix` | `prefix` | Strips the given prefix from the request path before forwarding to upstream |
| `geoip` | (none) | Looks up client IP in the embedded MaxMind GeoLite2-City database and injects `X-GeoIP-*` headers |

### Middleware Configuration

Middlewares with configuration use a YAML map syntax:

```yaml
middlewares:
  - strip_prefix:
      prefix: "/whoami" # prefix to strip
  - logger: {}          # no config
  - geoip: {}           # no config
  - http_cache:
      ttl: 5s           # per-route cache TTL
```

### Middleware Ordering

Middleware executes in the order listed. Place `strip_prefix` before other middlewares if you need the path modified before caching or logging:

```yaml
middlewares:
  - strip_prefix:
      prefix: "/whoami"
  - logger: {}
  - geoip: {}
  - http_cache:
      ttl: 5s
```

### GeoIP Headers

The `geoip` middleware injects the following headers into the request:

| Header |  Example |
|--------|----------|
| `X-GeoIP-Country` | `US` |
| `X-GeoIP-Country-Name` | `United States` |
| `X-GeoIP-City` | `Mountain View` |
| `X-GeoIP-Region` | `California` |
| `X-GeoIP-Latitude` | `37.3861` |
| `X-GeoIP-Longitude` | `-122.0838` |

> **Note**: Not all IPs have city or region data in the GeoLite2-City database. Missing fields will not have headers set.

## How it works

Incoming requests are matched against configured paths using longest prefix match. The matched route determines the middleware chain to execute and the upstream to forward to.

The request flows through the middleware chain in order, then gets forwarded to the upstream server. The response travels back through the chain in reverse order (_like a boomerang, hence the name_).

```
Client -> :8080/hello/whoami
  -> [strip_prefix "/hello"] -> path becomes /whoami
  -> [geoip]                 -> injects X-GeoIP-* headers
  -> [logger]                -> logs request
  -> [http_cache]            -> serves from cache (HIT) or forwards upstream (MISS)
  -> [upstream server]
```

## ⚠️ Check this Wiki to see the exact plan I made and followed for this project.
- ### [My Plan for building an HTTP Proxy](https://github.com/saifat29/goomerang/wiki)
