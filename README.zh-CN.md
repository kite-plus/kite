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

Kite 是一个轻量、快速、现代化的静态资源托管平台。它不仅仅是一个图床，更是一个涵盖了图片、音视频以及各类静态文件托管的综合性解决方案。就像天空中翱翔的风筝一样，Kite 旨在为您提供无缝、轻松的全类型媒体资源管理和分发体验。

Kite 由 Go 后端、内嵌式 React 管理后台和可扩展的存储抽象层组成，支持本地磁盘、FTP 以及主流对象存储服务。它适合自托管图床、静态资源分发、内部媒体库和轻量团队部署等场景。

## ✨ 特性

- **全格式支持**：轻松托管和分发图片、音频、视频以及其他标准的静态文件格式。
- **轻量且快速**：采用现代 Web 技术构建，提供卓越的性能与极致的资源加载速度。
- **简单易用**：简洁直观的用户界面，方便您上传、管理和查看各类资源。
- **安全可靠**：强大的安全措施确保您的数据和资产安全。
- **开源免费**：自由使用、修改和分发。

## 🧱 技术栈

- **后端**：Go、Gin、GORM、SQLite / MySQL / PostgreSQL
- **前端**：React、TypeScript、Vite、TanStack Query、Radix UI
- **存储**：本地、S3、FTP
- **媒体能力**：缩略图、静态文件分发、公开上传接口

## 🚀 快速开始

### 开发模式

```bash
make dev
```

该命令会同时启动 Go 后端和管理后台前端开发服务。

首次克隆仓库后，建议执行一次下面的命令启用 Git Hook：

```bash
make hooks-install
```

`pre-commit` 负责提交前的快速修正：自动格式化已暂存的 Go 与管理后台代码，并执行 `go mod tidy`。

`pre-push` 会在一个隔离的 `HEAD` 临时检出目录里执行较慢的完整校验，包括 `go test`、Go 构建校验，以及管理后台构建校验。

### 生产构建

```bash
make build
./build/kite
```

生产构建会先编译前端，并将其嵌入到 Go 二进制中。

### 初始管理员凭据

首次启动（users 表为空）时，Kite 会自动创建一个临时管理员账号：

```
用户名：admin
密码：  admin
```

该账号带有 `password_must_change` 标记，首次登录会强制跳转至重置页面。出于安全考虑，这组密码不会写入应用日志，请在首次登录后立刻修改，并将新凭据妥善保存（切勿提交进版本库）。

## 📦 项目结构

```text
cmd/kite            应用入口
internal/           后端处理器、服务、仓储与存储驱动
web/admin/          React 管理后台
template/           落地页与上传页模板
deploy/             docker-compose 与 nginx 示例
```

## 🐳 部署

- 使用 [deploy/docker-compose.yml](deploy/docker-compose.yml) 快速容器化部署。
- 使用 [Dockerfile](Dockerfile) 构建独立镜像。
- 使用 [deploy/nginx/conf.d/www.kite.plus.conf](deploy/nginx/conf.d/www.kite.plus.conf) 作为反向代理参考。

## 🤝 参与贡献

欢迎任何形式的贡献！

## 📄 开源协议

本项目采用 MIT 协议 - 详情请见 [LICENSE](LICENSE) 文件。
