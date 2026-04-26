<div align="center">
  <img src="web/admin/public/logo.png" alt="Kite logo" width="88" height="88" />
  <h1>Kite 静态资源托管平台</h1>
  <p>一个轻量、快速、现代化的静态资源托管平台。</p>
  <p>
    <img src="https://img.shields.io/badge/Go-1.25+-00ADD8?logo=go&logoColor=white" alt="Go 1.25+" />
    <img src="https://img.shields.io/badge/Gin-1.12-008ECF?logo=gin&logoColor=white" alt="Gin 1.12" />
    <img src="https://img.shields.io/badge/React-19-20232A?logo=react&logoColor=61DAFB" alt="React 19" />
    <img src="https://img.shields.io/badge/Vite-8-646CFF?logo=vite&logoColor=white" alt="Vite 8" />
    <img src="https://img.shields.io/badge/License-MIT-16A34A" alt="MIT License" />
    <img src="https://img.shields.io/badge/Storage-Local%20%7C%20S3%20%7C%20FTP-111827" alt="Storage Drivers" />
  </p>
  <p>
    简体中文 | <a href="README.md">English</a>
  </p>
</div>

## 🪁 简介

Kite 是一个自托管的媒体托管平台，覆盖图片、音频、视频与各类静态文件。Go 后端 + 内嵌 React 管理后台 + 可插拔存储层（本地磁盘、FTP 与主流对象存储），适合自托管图床、内部媒体库与小团队部署。

## ✨ 特性

- **全格式支持**：图片、音视频与任意静态文件，统一在一个控制台中管理。
- **轻量且快速**：单一 Go 二进制内嵌 SPA，除存储后端外无额外运行时依赖。
- **中英双语界面**：一个切换按钮即可让公共站点、管理后台与后端响应同步切换语言。
- **首次安装向导**：`/setup` 一站式配置管理员账号、站点 URL 与默认存储，全程不会出现临时凭据。
- **可插拔存储**：本地、S3 兼容、FTP 以及主流云厂商，支持优先级 / 轮询 / 镜像写入策略。

## 🧱 技术栈

- **后端**：Go、Gin、GORM、SQLite / MySQL / PostgreSQL
- **前端**：React、TypeScript、Vite、TanStack Query、Radix UI
- **存储**：本地、S3 兼容、FTP、OSS / COS / OBS / BOS / R2 / GCS / B2 / Wasabi / Spaces
- **媒体能力**：缩略图、可选 WebP 转码、公开上传接口

## 🚀 快速开始

### 开发模式

```bash
make dev            # 同时启动 Go 后端与管理后台 SPA
make hooks-install  # 首次执行：启用 pre-commit / pre-push 钩子
```

`pre-commit` 会格式化已暂存的 Go / 前端文件并执行 `go mod tidy`；`pre-push` 会在隔离目录中跑 `go test`、Go 构建与前端构建。

### 生产构建

```bash
make build
./build/kite
```

构建过程会先编译 SPA，再将其嵌入 Go 二进制中。

### 首次安装向导

全新部署会引导操作者通过 `/setup` 完成站点信息、首位管理员账号与默认存储后端的初始化。提交成功后向导会写入 `is_installed=true` 并停止响应，避免再次被提交。整个流程不会向数据库或日志写入任何临时凭据。

如需在脚本 / 容器化部署中跳过向导，可设置 `KITE_LEGACY_BOOTSTRAP=1`：首启动会按旧逻辑创建 `admin / admin` 账号，并在首次登录时强制重置密码。

## ⚙️ 配置

三层叠加，后一层覆盖前一层：编译期默认值 → 可选 JSON 配置文件（路径由 `KITE_CONFIG_FILE` 指定，默认 `data/config.json`，由安装向导写入）→ 环境变量。部分运行时可调字段（站点名称、JWT 秘钥、注册策略等）还保存在 `settings` 表中，可在管理后台直接修改。

### 环境变量

| 变量 | 作用 | 示例 |
| --- | --- | --- |
| `KITE_PORT` | HTTP 监听端口。 | `8080` |
| `KITE_HOST` | 监听地址。 | `0.0.0.0` |
| `KITE_DSN` | 数据库连接串，格式取决于驱动。 | `data/kite.db` |
| `KITE_DB_DRIVER` | `sqlite`、`mysql` 或 `postgres`。 | `sqlite` |
| `KITE_SITE_URL` | 站点对外 URL，用于 OAuth 回调与绝对链接。 | `https://kite.example.com` |
| `KITE_LOG_LEVEL` | `debug` / `info` / `warn` / `error`，默认 `info`。 | `info` |
| `KITE_LOG_FORMAT` | `text` 或 `json`，默认 `text`。 | `json` |
| `KITE_CORS_ORIGINS` | `/i/`、`/v/`、`/a/`、`/f/` 的跨域来源列表，逗号分隔。 | `https://kite.example.com` |
| `KITE_LEGACY_BOOTSTRAP` | 设为 `1` 时跳过安装向导。 | `1` |
| `KITE_CONFIG_FILE` | 磁盘上 JSON 配置文件的路径。 | `/etc/kite/config.json` |

### 国际化

语言按 `kite_locale` Cookie → `Accept-Language` → 英文兜底的顺序解析。任意界面切换语言都会更新 Cookie，其它界面下一次渲染时即可同步。新增语言只需在 `internal/i18n` 与 `web/admin/src/i18n/locales` 各加一个文件。

## 📦 项目结构

```text
cmd/kite     应用入口
internal/    后端处理器、服务、仓储与存储驱动
web/admin/   React 管理后台
template/    落地页与上传页模板
deploy/      docker-compose 与 nginx 示例
```

## 🐳 部署

- [`deploy/docker-compose.yml`](deploy/docker-compose.yml) — 容器化部署。
- [`Dockerfile`](Dockerfile) — 独立镜像构建。
- [`deploy/nginx/conf.d/www.kite.plus.conf`](deploy/nginx/conf.d/www.kite.plus.conf) — 反向代理参考。

## 🤝 参与贡献

欢迎提交 Issue 或 Pull Request 一起完善。

## 📄 开源协议

[MIT](LICENSE)。
