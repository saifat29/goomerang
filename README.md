<p align="center">
  <img src="assets/goomerang-logo-transparent.png" alt="Goomerang" width="400">
</p>

<h3 align="center">A <em>potentially</em> fast, configurable HTTP reverse proxy with built-in caching</h3>

---

## Features

- **Multi-upstream routing** - route requests to different upstream servers based on path prefixes (longest match wins)
- **In-memory LRU caching** - cache GET/HEAD responses with configurable TTL and max size
- **Pluggable middleware system** - onion-style middleware chain, configurable per proxy
- **Structured logging** - Fast logging using [Zerolog](https://github.com/rs/zerolog) with configurable levels (`trace`, `debug`, `info`, `warn`, `error`, `fatal`, `panic`) and format (`json`, `console`)
- **YAML configuration** - single config file with sensible defaults for all settings
- **Cache-Control awareness _(sort of, not 100% as per HTTP spec)_** - respects `private`, `no-cache`, `no-store` directives and skips caching for responses with `Set-Cookie` and for requests with `Authorization`
- **Hop-by-hop header handling** - properly strips and forwards headers per HTTP spec
- **X-Forwarded-For** - injects client IP for proper identification
- **X-Cache-Status** - response header showing `HIT`, `MISS`, or `BYPASS`

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
  - path: "/json"
    upstream: "http://httpbin.org"
    middlewares:
      - logger
      - cache
  - path: "/ip"
    upstream: "http://httpbingo.org"
    middlewares:
      - logger
      - cache
```

## Middlewares

| Name | Description |
|------|-------------|
| `logger` | Logs each request with method, path, remote address, user agent, duration, status, and response size |
| `cache` | Caches GET/HEAD responses in memory. Adds `X-Cache-Status` header (`HIT`, `MISS`, or `BYPASS`) |

Middlewares are applied in the order listed in the `middlewares` list.

## How it works

Incoming requests are matched against configured paths. A simple route matching method (longest prefix match) is used to determine the middleware chain to execute and then forward the request to the correct upstream.

The request flows through the middleware chain (e.g., logger -> cache -> upstream handler), then gets forwarded to the upstream server. The response travels back through the chain in reverse order.


The complete cycle looks like this (_like a boomerang_)-
```
Client -> :8080/hello -> [route] -> [logger] -> [cache] -> [upstream server]

Client <- :8080/hello <- [route] <- [logger] <- [cache] <- [upstream server]
```

## ⚠️ Check this Wiki to see the exact plan I made and followed for this project.
- ### [My Plan for building an HTTP Proxy](https://github.com/saifat29/goomerang/wiki)
