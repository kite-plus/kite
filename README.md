<div align="center">
  <img src="web/admin/public/logo.png" alt="Kite logo" width="88" height="88" />
  <h1>Kite Static Asset Hosting</h1>
  <p>A lightweight, fast, and modern static asset hosting platform.</p>
  <p>
    <img src="https://img.shields.io/badge/Go-1.25+-00ADD8?logo=go&logoColor=white" alt="Go 1.25+" />
    <img src="https://img.shields.io/badge/Gin-1.12-008ECF?logo=gin&logoColor=white" alt="Gin 1.12" />
    <img src="https://img.shields.io/badge/React-19-20232A?logo=react&logoColor=61DAFB" alt="React 19" />
    <img src="https://img.shields.io/badge/Vite-8-646CFF?logo=vite&logoColor=white" alt="Vite 8" />
    <img src="https://img.shields.io/badge/License-MIT-16A34A" alt="MIT License" />
    <img src="https://img.shields.io/badge/Storage-Local%20%7C%20S3%20%7C%20FTP-111827" alt="Storage Drivers" />
  </p>
  <p>
    <a href="README.zh-CN.md">简体中文</a> | English
  </p>
</div>

## 🪁 Introduction

Kite is a self-hosted media platform for images, audio, video, and general static files. A Go backend, an embedded React admin panel, and a pluggable storage layer (local disk, FTP, and the major object-storage providers) make it a good fit for self-hosted image hosting, internal media libraries, and small-team deployments.

## ✨ Features

- **All media types** — images, audio, video, and arbitrary static files in one console.
- **Lightweight & fast** — single Go binary with the SPA embedded; no extra runtime beyond your storage backend.
- **Bilingual UI** — English and Simplified Chinese; one switcher flips the public site, admin console, and backend responses together.
- **First-run install wizard** — `/setup` provisions the admin account, site URL, and default storage on a fresh deployment without exposing temporary credentials.
- **Pluggable storage** — local, S3-compatible, FTP, and the major cloud providers, with priority / round-robin / mirror upload policies.

## 🧱 Tech Stack

- **Backend**: Go, Gin, GORM, SQLite / MySQL / PostgreSQL
- **Frontend**: React, TypeScript, Vite, TanStack Query, Radix UI
- **Storage**: Local, S3-compatible, FTP, OSS / COS / OBS / BOS / R2 / GCS / B2 / Wasabi / Spaces
- **Media**: Thumbnailing, optional WebP transcoding, public upload endpoints

## 🚀 Quick Start

### Development

```bash
make dev            # backend + admin SPA dev server
make hooks-install  # one-time: enable pre-commit / pre-push hooks
```

`pre-commit` formats staged Go / admin files and runs `go mod tidy`. `pre-push` runs `go test`, Go build, and the admin build in an isolated checkout.

### Production

```bash
make build
./build/kite
```

The build compiles the SPA and embeds it into the Go binary.

### First-run install wizard

A fresh deployment routes the operator through `/setup` to capture the site identity, the first admin account, and the default storage backend. The wizard then stamps `is_installed=true` and stops responding so the form can't be re-driven later. No temporary credentials are ever written to the database or the logs.

To skip the wizard in scripted / container deployments, set `KITE_LEGACY_BOOTSTRAP=1` — Kite seeds `admin / admin` on first boot and forces a password reset on first login.

## ⚙️ Configuration

Three layers, last wins: compiled-in defaults → optional JSON config file (path: `KITE_CONFIG_FILE`, default `data/config.json`, written by the install wizard) → environment variables. Runtime-tunable settings (site name, JWT secret, registration policy, …) also live in the `settings` table and are editable from the admin console.

### Environment variables

| Variable | Purpose | Example |
| --- | --- | --- |
| `KITE_PORT` | HTTP listener port. | `8080` |
| `KITE_HOST` | Bind address. | `0.0.0.0` |
| `KITE_DSN` | Database connection string (format depends on the driver). | `data/kite.db` |
| `KITE_DB_DRIVER` | `sqlite`, `mysql`, or `postgres`. | `sqlite` |
| `KITE_SITE_URL` | Public site URL — used for OAuth callbacks and absolute links. | `https://kite.example.com` |
| `KITE_LOG_LEVEL` | `debug` / `info` / `warn` / `error`. Default `info`. | `info` |
| `KITE_LOG_FORMAT` | `text` or `json`. Default `text`. | `json` |
| `KITE_CORS_ORIGINS` | Comma-separated origin allowlist for `/i/`, `/v/`, `/a/`, `/f/`. | `https://kite.example.com` |
| `KITE_LEGACY_BOOTSTRAP` | Set to `1` to skip the install wizard. | `1` |
| `KITE_CONFIG_FILE` | Path to the on-disk JSON config file. | `/etc/kite/config.json` |

### Internationalization

Locale resolves from the `kite_locale` cookie → `Accept-Language` header → English. Switching the language on any surface updates the cookie, so the rest follows on the next render. Adding a new locale is one file under `internal/i18n` plus its sibling under `web/admin/src/i18n/locales`.

## 📦 Project Structure

```text
cmd/kite     application entrypoint
internal/    backend handlers, services, repos, storage drivers
web/admin/   React admin panel
template/    landing and upload page templates
deploy/      docker-compose and nginx examples
```

## 🐳 Deployment

- [`deploy/docker-compose.yml`](deploy/docker-compose.yml) — containerized setup.
- [`Dockerfile`](Dockerfile) — standalone image.
- [`deploy/nginx/conf.d/www.kite.plus.conf`](deploy/nginx/conf.d/www.kite.plus.conf) — reverse-proxy reference.

## 🤝 Contributing

Contributions are welcome — please open an issue or pull request.

## 📄 License

[MIT](LICENSE).
