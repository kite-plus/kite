# Kite

> **Flight on Code, Focused on Words.**
> Kite 是一款由 Go 驱动、极致扁平化的 AI 原生博客引擎。

[![Go Report Card](https://goreportcard.com/badge/github.com/amigoer/kite-blog)](https://goreportcard.com/report/github.com/amigoer/kite-blog)
[![License](https://img.shields.io/badge/license-MIT-black.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/go-1.25%2B-blue.svg)](https://golang.org)

## 项目概览

Kite 的后端采用严格分层架构：

- `api -> service -> repo -> model`
- 全局使用 UUID 主键
- 同时支持 SQLite 与 PostgreSQL
- 支持 `classic` SSR 与 `headless` JSON 双模输出
- 通过 `go:embed` 打包模板与静态资源

当前项目已经完成后端基础骨架，包括配置加载、数据库初始化、Gin 服务启动和基础健康检查接口。

## 当前进度

已完成：

- `internal/model` 中的 UUID v7 基础模型
- `internal/config` 中的 YAML 配置加载
- `internal/repo` 中的数据库初始化
- `internal/api` 中的统一响应封装
- `GET /api/v1/health` 健康检查接口
- `classic` 模式下的 `/` 最小首页渲染

待继续实现：

- 文章、标签、分类模型
- 面向前台的内容 API
- `/api/v1/admin` 管理接口
- React 后台构建接入

## 目录结构

```text
├── cmd/kite/main.go          # 程序入口
├── docs/api.md               # 后端 API 设计文档
├── internal/                 # 私有后端核心逻辑
├── templates/                # SSR 模板
├── ui/admin/                 # 管理后台源码
├── embed.go                  # 嵌入模板与静态资源
└── Makefile                  # 构建与开发脚本
```

## 快速开始

### 环境要求

- Go 1.25+
- Node.js 20+，仅在后续需要构建管理后台时使用

### 编译后端

```bash
make build-server
```

### 运行服务

```bash
./kite
```

默认监听地址：

```text
http://localhost:8080
```

当前可访问路由：

- `GET /`
- `GET /api/v1/health`

## 配置说明

Kite 通过环境变量 `KITE_CONFIG` 读取 YAML 配置文件。

示例：

```bash
KITE_CONFIG=./config.yaml ./kite
```

如果未设置 `KITE_CONFIG`，程序会回退到默认配置。

当前支持的配置项包括：

- `render_mode`: `classic` 或 `headless`
- `database.driver`: `sqlite` 或 `postgres`
- `database.path`
- `database.host`
- `database.port`
- `database.user`
- `database.password`
- `database.name`
- `database.ssl_mode`

## API 文档

后端 API 设计文档位于：

- `docs/api.md`

当前统一响应格式：

```json
{
  "code": 200,
  "data": {},
  "msg": "ok"
}
```

## 开发命令

常用命令：

```bash
make tidy
make test
make build-server
make run
```

如果管理后台尚未初始化，`make build-ui` 会自动安全跳过。

## 设计理念

Kite 遵循严格的 Flat Design：

- 零圆角
- 零阴影
- 高对比度硬边框
- 回归内容与阅读本身

## 参与贡献

开发前请先遵循根目录的 `.cursorrules` 约束。

## 开源协议

项目基于 MIT License 开源，详见 `LICENSE`。
