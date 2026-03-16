<p align="center">
  <img src="https://img.icons8.com/color/96/kite.png" width="80" alt="Kite Logo" />
</p>

<h1 align="center">Kite Blog</h1>

<p align="center">
  Lightweight AI-native blog engine · Go + React + SQLite · Single binary deployment
</p>

<p align="center">
  <strong>English</strong> | <a href="README.zh-CN.md">中文</a>
</p>

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go&logoColor=white" />
  <img src="https://img.shields.io/badge/React-19-61DAFB?logo=react&logoColor=white" />
  <img src="https://img.shields.io/badge/SQLite-embedded-003B57?logo=sqlite&logoColor=white" />
  <img src="https://img.shields.io/badge/License-MIT-green" />
</p>

---

## ✨ Features

- 🚀 **Single binary** — Frontend & backend compiled into one executable, just drop it on a server
- 📦 **Zero config** — Embedded SQLite, all settings stored in DB, backup = copy one `kite.db` file
- 🌐 **Web installer** — Browser-based interactive setup on first run, no config files needed
- 🎨 **Dual render modes** — Classic (Go Template SSR) / Headless (pure JSON API)
- 🤖 **AI native** — One-click integration with OpenAI / DeepSeek / Ollama, auto summary & tag suggestions
- ✍️ **Tiptap editor** — Markdown / WYSIWYG dual mode, syntax highlighting, tables, callouts, drag-and-drop image upload
- 🖼️ **Image management** — Cover image upload, in-editor drag & paste auto-upload
- 📡 **RSS & Sitemap** — Auto-generated, SEO friendly
- 🎭 **Theme system** — Custom Go Template themes

## 🏗️ Architecture

```
┌─────────────────────────────────────────────┐
│              kite (single binary)            │
├─────────────┬───────────────────────────────┤
│  Admin SPA  │         Go Backend            │
│  (React 19) │  ┌──────────┬──────────┐      │
│  Semi Design│  │ Gin HTTP │ GORM ORM │      │
│  Tiptap     │  │  Router  │          │      │
│  TanStack   │  └────┬─────┴────┬─────┘      │
│  Query      │       │          │             │
├─────────────┤  ┌────┴────┐ ┌───┴──────┐     │
│  Classic    │  │ Service │ │ Template │     │
│  Theme (SSR)│  │  Layer  │ │  Engine  │     │
├─────────────┤  └────┬────┘ └──────────┘     │
│             │       │                        │
│             │  ┌────┴────┐                   │
│             │  │ SQLite  │ ← one kite.db     │
│             │  └─────────┘   for everything  │
└─────────────┴───────────────────────────────┘
```

## 🚀 Quick Start

### Build & Run

```bash
# Clone
git clone https://github.com/amigoer/kite-blog.git
cd kite-blog

# Build (frontend + backend)
make build

# Run
./kite
```

On first run, the web installer launches automatically:

```
🪁 Kite first run — visit the installer: http://localhost:8080
```

Open your browser, 2 steps to complete setup:
1. Enter your site name
2. Set admin username & password

Restart after installation.

### Manual Build (without Make)

```bash
# Build admin frontend
cd ui/admin && npm install && npm run build && cd ../..

# Compile Go binary (frontend embedded)
go build -ldflags="-s -w" -o kite cmd/kite/main.go
```

## 🛠️ Development

### Backend

```bash
go run ./cmd/kite
```

### Admin Frontend

```bash
cd ui/admin
npm install
npm run dev    # Vite dev server, API auto-proxied to :8080
```

Dev frontend at `http://localhost:5173/admin`, Vite proxies `/api` to Go backend.

### Make Targets

```bash
make build         # Build frontend + backend
make build-server  # Backend only
make build-ui      # Frontend only
make run           # Run backend
make test          # Run tests
make clean         # Clean build artifacts
```

## 📁 Project Structure

```
kite-blog/
├── cmd/kite/               # Entry point & installer
│   ├── main.go             # Startup logic
│   └── setup_web.go        # Web installation wizard
├── internal/
│   ├── api/                # HTTP handlers & routing
│   │   ├── router.go       # Route registration
│   │   ├── post.go         # Post API
│   │   ├── ai.go           # AI summary/tags API
│   │   ├── upload.go       # Image upload API
│   │   ├── feed.go         # RSS & Sitemap
│   │   └── search.go       # Full-text search API
│   ├── config/             # Config struct definitions
│   ├── model/              # Data models (GORM)
│   ├── repo/               # Data access layer
│   └── service/            # Business logic layer
│       ├── ai.go           # AI service (OpenAI compatible)
│       ├── settings.go     # Settings service (SQLite R/W)
│       └── upload.go       # File upload service
├── ui/admin/               # React Admin SPA
│   └── src/
│       ├── components/     # Shared components
│       ├── pages/          # Page components
│       ├── hooks/          # Data hooks
│       ├── extensions/     # Tiptap extensions
│       └── lib/            # API client
├── templates/              # SSR theme templates
├── embed.go                # Embed frontend assets
├── Makefile
└── docs/
    ├── api.md              # API documentation
    ├── ssr.md              # SSR interface docs
    └── theme-dev.md        # Theme development guide
```

## 🔌 API Overview

| Method | Path | Description | Auth |
|--------|------|-------------|------|
| `GET` | `/api/v1/posts` | List posts | Public |
| `GET` | `/api/v1/search?q=` | Full-text search | Public |
| `GET` | `/feed.xml` | RSS 2.0 feed | Public |
| `GET` | `/sitemap.xml` | Sitemap | Public |
| `POST` | `/api/v1/auth/login` | Admin login | — |
| `POST` | `/api/v1/admin/posts` | Create post | 🔒 |
| `POST` | `/api/v1/admin/upload/image` | Upload image | 🔒 |
| `POST` | `/api/v1/admin/ai/summary` | AI summary | 🔒 |
| `POST` | `/api/v1/admin/ai/tags` | AI tag suggestions | 🔒 |
| `GET` | `/api/v1/admin/settings` | Get settings | 🔒 |
| `PUT` | `/api/v1/admin/settings` | Save settings | 🔒 |

Full documentation: [`docs/api.md`](docs/api.md)

## 🤖 AI Configuration

Configure via admin panel: **Settings → AI Integration**

1. Enable AI
2. Enter API endpoint (supports any OpenAI-compatible API)
3. Enter API Key and model name
4. Save — takes effect immediately

| Provider | API Endpoint | Recommended Model |
|----------|-------------|-------------------|
| DeepSeek | `https://api.deepseek.com` | `deepseek-chat` |
| OpenAI | `https://api.openai.com` | `gpt-4o-mini` |
| Qwen | `https://dashscope.aliyuncs.com/compatible-mode` | `qwen-turbo` |
| Ollama (local) | `http://localhost:11434` | `llama3` |

## 💾 Data & Backup

Everything (data + config) lives in a single `kite.db` file:

```bash
# Backup
cp kite.db kite.db.bak

# Migrate to new server
scp kite.db user@new-server:/path/to/kite/

# Restore
cp kite.db.bak kite.db
```

## 🐳 Docker

```dockerfile
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY . .
RUN apk add --no-cache nodejs npm
RUN cd ui/admin && npm ci && npm run build && cd ../..
RUN go build -ldflags="-s -w" -o kite ./cmd/kite

FROM alpine:3.19
WORKDIR /app
COPY --from=builder /app/kite .
EXPOSE 8080
VOLUME ["/app/data"]
CMD ["./kite"]
```

```bash
docker build -t kite .
docker run -d -p 8080:8080 -v kite-data:/app/data kite
```

## 🧰 Tech Stack

| Layer | Technology |
|-------|-----------|
| Backend | Go 1.22+, Gin |
| ORM | GORM |
| Database | SQLite (glebarez/sqlite, pure Go) |
| Frontend | React 19, TypeScript, Vite |
| UI Library | Semi Design (@douyinfe/semi-ui) |
| Rich Editor | Tiptap (ProseMirror) |
| Data Fetching | TanStack Query |
| Template Engine | Go html/template |
| Auth | Cookie Session + bcrypt |

## 📄 Documentation

- [API Reference](docs/api.md)
- [SSR Interface](docs/ssr.md)
- [Theme Development Guide](docs/theme-dev.md)

## 📜 License

[MIT](LICENSE)
