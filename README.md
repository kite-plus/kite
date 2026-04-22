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

Kite is a lightweight, fast, and modern static asset hosting platform. Much more than just an image host, Kite is designed to handle images, audio, video, and general static files. Just like a kite soaring in the sky, Kite aims to provide a seamless and effortless experience for managing and serving all your media assets.

Kite ships with a Go backend, an embedded React admin panel, and a storage abstraction layer that supports local disk, FTP, and mainstream object storage providers. It is suitable for self-hosted image hosting, static asset delivery, internal media libraries, and lightweight team deployments.

## ✨ Features

- **Comprehensive Support**: Host and serve images, audio, video, and standard static files with ease.
- **Lightweight & Fast**: Built with modern web technologies for optimal performance and rapid delivery.
- **Easy to Use**: A clean and intuitive user interface for uploading, managing, and viewing your files.
- **Secure**: Robust security measures to keep your data and assets safe.
- **Open Source**: Free to use and modify.

## 🧱 Tech Stack

- **Backend**: Go, Gin, GORM, SQLite/MySQL/PostgreSQL
- **Frontend**: React, TypeScript, Vite, TanStack Query, Radix UI
- **Storage**: Local, S3, FTP
- **Media**: Image thumbnailing, static file serving, public upload endpoints

## 🚀 Quick Start

### Development

```bash
make dev
```

This starts the Go backend and the admin frontend development server.

Enable the repository Git hooks once after cloning:

```bash
make hooks-install
```

The `pre-commit` hook focuses on quick fixes before a commit: it formats staged Go and admin frontend files and runs `go mod tidy`.

The `pre-push` hook runs the slower validation steps in an isolated checkout of `HEAD`: `go test`, Go build verification, and the admin frontend build check.

### Production Build

```bash
make build
./build/kite
```

The production build compiles the frontend and embeds it into the Go binary.

### First-boot admin credentials

On the first start (when the user table is empty) Kite seeds a bootstrap administrator account with these temporary credentials:

```
username: admin
password: admin
```

The account is flagged `password_must_change`, so the first login redirects to a mandatory reset page. The password is intentionally not written to the application logs — change it immediately after the first login and keep the new credentials out of version control.

## 📦 Project Structure

```text
cmd/kite            application entrypoint
internal/           backend handlers, services, repos, storage drivers
web/admin/          React admin panel
template/           landing and upload page templates
deploy/             docker-compose and nginx examples
```

## 🐳 Deployment

- Use [deploy/docker-compose.yml](deploy/docker-compose.yml) for a containerized setup.
- Use [Dockerfile](Dockerfile) to build a standalone image.
- Use [deploy/nginx/conf.d/www.kite.plus.conf](deploy/nginx/conf.d/www.kite.plus.conf) as a reverse-proxy reference.

## 🤝 Contributing

We welcome contributions! 

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
