<p align="center">
  <img src="https://img.icons8.com/color/96/kite.png" width="80" alt="Kite Logo" />
</p>

<h1 align="center">Kite Blog</h1>

<p align="center">
  轻量级 AI 原生博客引擎 · Go + React + SQLite · 单文件部署
</p>

<p align="center">
  <a href="README.md">English</a> | <strong>中文</strong>
</p>

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go&logoColor=white" />
  <img src="https://img.shields.io/badge/React-19-61DAFB?logo=react&logoColor=white" />
  <img src="https://img.shields.io/badge/SQLite-embedded-003B57?logo=sqlite&logoColor=white" />
  <img src="https://img.shields.io/badge/License-MIT-green" />
</p>

---

## ✨ 特性

- 🚀 **单二进制部署** — 前后端编译为一个可执行文件，扔到服务器就能跑
- 📦 **零配置** — 内嵌 SQLite，所有设置存数据库，备份只需复制 `kite.db`
- 🌐 **Web 安装引导** — 首次运行浏览器交互式设置，无需任何配置文件
- 🎨 **双渲染模式** — Classic（Go Template SSR）/ Headless（纯 JSON API）
- 🤖 **AI 原生** — 一键接入 OpenAI / DeepSeek 等大模型，自动摘要 & 标签推荐
- ✍️ **Tiptap 富文本编辑器** — Markdown / 所见即所得双模式、代码高亮、表格、Callout、拖拽上传图片
- 🖼️ **图片管理** — 封面图上传、编辑器内拖拽 / 粘贴自动上传
- 📡 **RSS & Sitemap** — 自动生成，SEO 友好
- 🎭 **主题系统** — 支持自定义 Go Template 主题

## 🏗️ 架构

```
┌─────────────────────────────────────────────┐
│                  kite (单二进制)              │
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
│             │  │ SQLite  │ ← 一个 kite.db    │
│             │  └─────────┘   搞定一切        │
└─────────────┴───────────────────────────────┘
```

## 🚀 快速开始

### 编译 & 运行

```bash
# 克隆仓库
git clone https://github.com/amigoer/kite-blog.git
cd kite-blog

# 一键构建（前端 + 后端）
make build

# 运行
./kite
```

首次运行会自动启动安装引导：

```
🪁 Kite 首次运行 — 请访问安装引导页: http://localhost:8080
```

打开浏览器，2 步完成安装：
1. 填写站点名称
2. 设置管理员账号密码

安装完成后重启即可使用。

### 一行编译（不用 Make）

```bash
# 构建 Admin 前端
cd ui/admin && npm install && npm run build && cd ../..

# 编译 Go 二进制（前端已嵌入）
go build -ldflags="-s -w" -o kite cmd/kite/main.go
```

## 🛠️ 开发

### 后端

```bash
go run ./cmd/kite
```

### 前端 Admin

```bash
cd ui/admin
npm install
npm run dev    # Vite 开发服务器，API 自动代理到 :8080
```

开发时前端 `http://localhost:5173/admin`，Vite 已配置 `/api` 代理到后端。

### 常用命令

```bash
make build         # 构建前端 + 后端
make build-server  # 仅构建后端
make build-ui      # 仅构建前端
make run           # 运行后端
make test          # 运行测试
make clean         # 清理构建产物
```

## 📁 项目结构

```
kite-blog/
├── cmd/kite/               # 入口 & 安装引导
│   ├── main.go             # 启动逻辑
│   └── setup_web.go        # Web 安装引导页
├── internal/
│   ├── api/                # HTTP 处理器 & 路由
│   │   ├── router.go       # 路由注册
│   │   ├── post.go         # 文章 API
│   │   ├── ai.go           # AI 摘要/标签 API
│   │   ├── upload.go       # 图片上传 API
│   │   ├── feed.go         # RSS & Sitemap
│   │   └── search.go       # 全文搜索 API
│   ├── config/             # 配置结构定义
│   ├── model/              # 数据模型（GORM）
│   ├── repo/               # 数据访问层
│   └── service/            # 业务逻辑层
│       ├── ai.go           # AI 服务（OpenAI 兼容）
│       ├── settings.go     # 设置服务（SQLite 读写）
│       └── upload.go       # 文件上传服务
├── ui/admin/               # React Admin SPA
│   └── src/
│       ├── components/     # 通用组件
│       ├── pages/          # 页面组件
│       ├── hooks/          # 数据 Hooks
│       ├── extensions/     # Tiptap 扩展
│       └── lib/            # API 客户端
├── templates/              # SSR 主题模板
├── embed.go                # 嵌入前端资源
├── Makefile
└── docs/
    ├── api.md              # API 文档
    ├── ssr.md              # SSR 接口文档
    └── theme-dev.md        # 主题开发指南
```

## 🔌 API 概览

| 方法 | 路径 | 说明 | 鉴权 |
|------|------|------|------|
| `GET` | `/api/v1/posts` | 文章列表 | 公开 |
| `GET` | `/api/v1/search?q=` | 全文搜索 | 公开 |
| `GET` | `/feed.xml` | RSS 2.0 | 公开 |
| `GET` | `/sitemap.xml` | Sitemap | 公开 |
| `POST` | `/api/v1/auth/login` | 管理员登录 | — |
| `POST` | `/api/v1/admin/posts` | 创建文章 | 🔒 |
| `POST` | `/api/v1/admin/upload/image` | 上传图片 | 🔒 |
| `POST` | `/api/v1/admin/ai/summary` | AI 生成摘要 | 🔒 |
| `POST` | `/api/v1/admin/ai/tags` | AI 推荐标签 | 🔒 |
| `GET` | `/api/v1/admin/settings` | 获取设置 | 🔒 |
| `PUT` | `/api/v1/admin/settings` | 保存设置 | 🔒 |

完整文档见 [`docs/api.md`](docs/api.md)

## 🤖 AI 配置

在后台 **设置 → AI 集成** 中可视化配置：

1. 开启 AI 功能
2. 填写 API 地址（支持 OpenAI / DeepSeek / 通义千问等兼容接口）
3. 填写 API Key 和模型名称
4. 保存即生效

| 服务商 | API 地址 | 推荐模型 |
|--------|----------|----------|
| DeepSeek | `https://api.deepseek.com` | `deepseek-chat` |
| OpenAI | `https://api.openai.com` | `gpt-4o-mini` |
| 通义千问 | `https://dashscope.aliyuncs.com/compatible-mode` | `qwen-turbo` |
| Ollama (本地) | `http://localhost:11434` | `llama3` |

## 💾 数据 & 备份

所有数据和配置存储在单一 `kite.db` 文件中：

```bash
# 备份
cp kite.db kite.db.bak

# 迁移到新服务器
scp kite.db user@new-server:/path/to/kite/

# 恢复
cp kite.db.bak kite.db
```

## 🐳 Docker（示例）

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

## 🧰 技术栈

| 层 | 技术 |
|----|------|
| 后端框架 | Go 1.22+, Gin |
| ORM | GORM |
| 数据库 | SQLite (glebarez/sqlite, 纯 Go) |
| 前端框架 | React 19, TypeScript, Vite |
| UI 组件库 | Semi Design (@douyinfe/semi-ui) |
| 富文本编辑器 | Tiptap (ProseMirror) |
| 数据请求 | TanStack Query |
| 模板引擎 | Go html/template |
| 认证 | Cookie Session + bcrypt |

## 📄 文档

- [API 文档](docs/api.md) — RESTful API 完整参考
- [SSR 接口文档](docs/ssr.md) — 服务端渲染变量说明
- [主题开发指南](docs/theme-dev.md) — 自定义主题模板开发

## 📜 License

[MIT](LICENSE)
