# Kite

> **Flight on Code, Focused on Words.**
> A flat-designed, AI-native blog engine powered by Go.

[![Go Report Card](https://goreportcard.com/badge/github.com/amigoer/kite-blog)](https://goreportcard.com/report/github.com/amigoer/kite-blog)
[![License](https://img.shields.io/badge/license-MIT-black.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/go-1.25%2B-blue.svg)](https://golang.org)

## Overview

Kite is a Go-first blog engine with a strict layered backend architecture:

- `api -> service -> repo -> model`
- UUID-only primary keys
- SQLite and PostgreSQL support
- Classic SSR and headless JSON modes
- Embedded templates and admin assets via `go:embed`

The backend foundation is already in place, including config loading, database initialization, a minimal Gin server, and a health check endpoint.

## Current Status

Implemented now:

- Base model with UUID v7 primary key hook
- YAML config loader in `internal/config`
- Database initialization in `internal/repo`
- Unified API response wrapper in `internal/api`
- `GET /api/v1/health`
- Minimal classic-mode SSR homepage at `/`

Planned next:

- Post, tag, and category models
- Public content APIs
- Admin APIs under `/api/v1/admin`
- React admin build integration

## Project Structure

```text
├── cmd/kite/main.go          # Application entry point
├── docs/api.md               # Backend API design document
├── internal/                 # Private backend logic
├── templates/                # SSR templates
├── ui/admin/                 # Admin frontend source
├── embed.go                  # Embedded static/template assets
└── Makefile                  # Build and development automation
```

## Quick Start

### Prerequisites

- Go 1.25+
- Node.js 20+ if you want to build the admin frontend later

### Build the server

```bash
make build-server
```

### Run the server

```bash
./kite
```

Server default address:

```text
http://localhost:8080
```

Available routes right now:

- `GET /`
- `GET /api/v1/health`

## Configuration

Kite reads config from the `KITE_CONFIG` environment variable.

Example:

```bash
KITE_CONFIG=./config.yaml ./kite
```

If `KITE_CONFIG` is not set, Kite falls back to default config values.

Current config capabilities:

- `render_mode`: `classic` or `headless`
- `database.driver`: `sqlite` or `postgres`
- `database.path`
- `database.host`
- `database.port`
- `database.user`
- `database.password`
- `database.name`
- `database.ssl_mode`

## API

Backend API design is documented in:

- `docs/api.md`

Current response format:

```json
{
  "code": 200,
  "data": {},
  "msg": "ok"
}
```

## Development

Useful commands:

```bash
make tidy
make test
make build-server
make run
```

If the admin frontend has not been initialized yet, `make build-ui` will be skipped safely.

## Design Philosophy

Kite follows a strict flat visual language:

- Zero border radius
- Zero shadow
- Hard borders and high contrast
- Clean, minimal reading experience

## Contributing

Please follow the development rules defined in `.cursorrules`.

## License

Distributed under the MIT License. See `LICENSE` for details.
