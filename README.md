<div align="center">
  <img src="web/admin/public/favicon.svg" alt="Kite logo" width="88" height="88" />
  <h1>Kite Static Asset Hosting</h1>
  <p>A lightweight, fast, and modern static asset hosting platform.</p>
  <p>
    <img src="https://img.shields.io/badge/Go-1.25+-00ADD8?logo=go&logoColor=white" alt="Go 1.25+" />
    <img src="https://img.shields.io/badge/Gin-1.12-008ECF?logo=gin&logoColor=white" alt="Gin 1.12" />
    <img src="https://img.shields.io/badge/React-19-20232A?logo=react&logoColor=61DAFB" alt="React 19" />
    <img src="https://img.shields.io/badge/Vite-8-646CFF?logo=vite&logoColor=white" alt="Vite 8" />
    <img src="https://img.shields.io/badge/License-MIT-16A34A" alt="MIT License" />
    <img src="https://img.shields.io/badge/Storage-Local%20%7C%20S3%20%7C%20OSS%20%7C%20COS%20%7C%20OBS%20%7C%20BOS%20%7C%20FTP-111827" alt="Storage Drivers" />
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
- **Storage**: Local, FTP, S3, MinIO, Cloudflare R2, OSS, COS, OBS, BOS
- **Media**: Image thumbnailing, static file serving, public upload endpoints

## 🚀 Quick Start

### Development

```bash
make dev
```

This starts the Go backend and the admin frontend development server.

### Production Build

```bash
make build
./build/kite
```

The production build compiles the frontend and embeds it into the Go binary.

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
- Use [deploy/nginx/conf.d/demo.kite.plus.conf](deploy/nginx/conf.d/demo.kite.plus.conf) as a reverse-proxy reference.

## 🤝 Contributing

We welcome contributions! 

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
