<p align="center">
  <img src="assets/goomerang-logo-transparent.png" alt="Goomerang" width="400">
</p>

<h3 align="center">A <em>potentially</em> fast, configurable HTTP reverse proxy with built-in caching</h3>

---

## Features

- **Multi-upstream routing** - route requests to different upstream servers based on path prefixes (longest match wins)
- **In-memory LRU caching** - cache GET/HEAD responses with configurable TTL
- **YAML configuration** - single config file with sensible defaults for settings
- **Cache-Control awareness _(sort of, not 100% as per HTTP spec)_** - respects `private`, `no-cache`, `no-store` directives and skips caching for responses with `Set-Cookie` and for requests with `Authorization`
- **Hop-by-hop header handling** - properly strips and forwards headers per HTTP spec
- **X-Forwarded-For** - injects client IP for proper identification
- **X-Cache-Status** - response header showing `HIT`, `MISS`, or `BYPASS`

## Quick Start

- Run locally
```bash
make run
```

- Run all unit tests
```bash
make test
```

- Format, Lint, Test, and Build (Binaries can be found in the `/bin` directory)
```bash
make all
```

## Configuration

Goomerang is configured via a YAML file called `goomerang.yml`

The `goomerang.yml` file:

```yaml
server:
  addr: :8080
  read_timeout: 30s
  write_timeout: 30s
  idle_timeout: 60s

cache:
  max_size_bytes: 104857600
  ttl: 5s

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
  - path: "/ip"
    upstream: "http://httpbingo.org"
```

## How it works

Incoming requests are matched against configured paths. A simple route matching method (longest prefix match) is used to route the incoming request to the correct upstream.

An LRU + TTL cache is used for caching responses for GET/HEAD requests.

```
Client -> :8080/json -> httpbin.org/json -> 200 OK
Client -> :8080/ip -> httpbingo.org/ip -> 200 OK
Client -> :8080/unknown -> 404 Not Found
```

## ⚠️ Check this Wiki to see the exact plan I made and followed for this project.
- ### [My Plan for building an HTTP Proxy](https://github.com/saifat29/goomerang/wiki)
