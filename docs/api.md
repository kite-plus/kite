# Kite Backend API 文档

## 1. 概述

Kite 后端采用严格分层结构：`api -> service -> repo -> model`。

当前 API 设计目标：
- 统一支持 `classic` 与 `headless` 两种渲染模式
- 所有资源主键均使用 `uuid.UUID`
- 所有错误都以统一 JSON 结构返回，禁止 `panic`
- 同时兼容 SQLite 与 PostgreSQL
- 优先为后台管理端与未来 AI 能力预留稳定接口

基础约定：
- Base Path: `/api/v1`
- Content-Type: `application/json`
- Time Format: RFC3339
- ID Type: UUID v7

## 2. 渲染模式

### 2.1 `classic`

当 `config.render_mode=classic` 时：
- 页面型路由返回 HTML 页面
- 数据型路由仍返回 JSON
- `gin.HTMLRender` 用于渲染 `templates/` 下模板

适用示例：
- `GET /` -> HTML 首页
- `GET /posts/:slug` -> HTML 文章详情页
- `GET /api/v1/posts` -> JSON

### 2.2 `headless`

当 `config.render_mode=headless` 时：
- 不输出 SSR 页面
- 仅保留 JSON API
- 所有前台能力通过 API 提供给外部前端

约定行为：
- 若访问仅在 classic 模式下存在的页面路由，返回 JSON 错误响应
- 推荐错误码：`404` 或 `501`，由具体实现策略决定；当前建议返回 `404`

## 3. 统一响应结构

所有 API 响应统一使用以下格式：

```json
{
  "code": 200,
  "data": {},
  "msg": "ok"
}
```

字段说明：
- `code`: 业务状态码，通常与 HTTP 状态协同表达结果
- `data`: 具体业务数据；无数据时可返回 `null`、`{}` 或 `[]`
- `msg`: 人类可读消息，适合前端直接展示或日志记录

### 3.1 成功响应示例

```json
{
  "code": 200,
  "data": {
    "id": "0195f3ff-4f17-7f0b-9e5f-15db2f2fb6a1",
    "title": "Hello Kite"
  },
  "msg": "ok"
}
```

### 3.2 失败响应示例

```json
{
  "code": 400,
  "data": null,
  "msg": "invalid request payload"
}
```

### 3.3 HTTP 状态码约定

| HTTP Status | code | 含义 |
| --- | --- | --- |
| 200 | 200 | 请求成功 |
| 201 | 201 | 创建成功 |
| 400 | 400 | 请求参数错误 |
| 401 | 401 | 未认证 |
| 403 | 403 | 无权限 |
| 404 | 404 | 资源不存在 |
| 409 | 409 | 资源冲突 |
| 422 | 422 | 业务校验失败 |
| 500 | 500 | 服务内部错误 |

## 4. 通用请求约定

### 4.1 分页

列表接口统一支持：
- `page`: 页码，从 `1` 开始
- `page_size`: 每页数量，默认 `10`，最大建议 `100`

分页响应建议结构：

```json
{
  "code": 200,
  "data": {
    "items": [],
    "pagination": {
      "page": 1,
      "page_size": 10,
      "total": 0
    }
  },
  "msg": "ok"
}
```

### 4.2 排序

列表接口建议支持：
- `sort_by`: 排序字段，如 `created_at`、`updated_at`
- `sort_order`: `asc` 或 `desc`

### 4.3 过滤

列表接口按资源特性支持过滤，例如：
- `status=published`
- `tag_id=<uuid>`
- `keyword=golang`

## 5. 数据模型草案

以下是后续后端 API 的核心资源。

### 5.1 Post

```json
{
  "id": "0195f3ff-4f17-7f0b-9e5f-15db2f2fb6a1",
  "title": "Hello Kite",
  "slug": "hello-kite",
  "summary": "A lightweight AI-native blog engine.",
  "content": "# Hello Kite",
  "status": "draft",
  "cover_image": "",
  "published_at": null,
  "created_at": "2026-03-15T10:00:00Z",
  "updated_at": "2026-03-15T10:00:00Z"
}
```

字段建议：
- `status`: `draft` / `published` / `archived`
- `slug`: 全局唯一，用于前台路由
- `summary`: 可人工填写，也可由 AI 自动生成

### 5.2 Tag

```json
{
  "id": "0195f400-13ad-7cbb-9b9f-04d15b759cb8",
  "name": "golang",
  "slug": "golang",
  "created_at": "2026-03-15T10:00:00Z",
  "updated_at": "2026-03-15T10:00:00Z"
}
```

### 5.3 Category

```json
{
  "id": "0195f400-0d80-730a-bf8a-4d9776db8f4d",
  "name": "Backend",
  "slug": "backend",
  "created_at": "2026-03-15T10:00:00Z",
  "updated_at": "2026-03-15T10:00:00Z"
}
```

## 6. 首批 API 规划

本阶段优先定义基础能力与文章系统接口。

### 6.1 Health Check

#### `GET /api/v1/health`

用途：
- 健康检查
- 供部署平台探活
- 供管理后台确认服务已启动

响应示例：

```json
{
  "code": 200,
  "data": {
    "status": "ok",
    "render_mode": "classic"
  },
  "msg": "ok"
}
```

### 6.2 Posts

#### `GET /api/v1/posts`

用途：
- 获取文章列表
- 支持分页、状态过滤、关键字搜索

查询参数：
- `page`
- `page_size`
- `status`
- `keyword`
- `tag_id`
- `category_id`
- `sort_by`
- `sort_order`

响应示例：

```json
{
  "code": 200,
  "data": {
    "items": [
      {
        "id": "0195f3ff-4f17-7f0b-9e5f-15db2f2fb6a1",
        "title": "Hello Kite",
        "slug": "hello-kite",
        "summary": "A lightweight AI-native blog engine.",
        "status": "published",
        "published_at": "2026-03-15T10:00:00Z",
        "created_at": "2026-03-15T10:00:00Z",
        "updated_at": "2026-03-15T10:00:00Z"
      }
    ],
    "pagination": {
      "page": 1,
      "page_size": 10,
      "total": 1
    }
  },
  "msg": "ok"
}
```

#### `GET /api/v1/posts/:id`

用途：
- 根据 UUID 获取文章详情

#### `GET /api/v1/posts/slug/:slug`

用途：
- 根据 slug 获取文章详情
- 供 classic 前台和 headless 前端共用

#### `POST /api/v1/posts`

用途：
- 创建文章

请求体示例：

```json
{
  "title": "Hello Kite",
  "slug": "hello-kite",
  "summary": "A lightweight AI-native blog engine.",
  "content": "# Hello Kite",
  "status": "draft",
  "tag_ids": [],
  "category_id": null,
  "published_at": null
}
```

#### `PUT /api/v1/posts/:id`

用途：
- 全量更新文章

#### `PATCH /api/v1/posts/:id`

用途：
- 局部更新文章
- 推荐给后台表单自动保存场景

#### `DELETE /api/v1/posts/:id`

用途：
- 软删除文章

### 6.3 Tags

#### `GET /api/v1/tags`
#### `GET /api/v1/tags/:id`
#### `POST /api/v1/tags`
#### `PUT /api/v1/tags/:id`
#### `DELETE /api/v1/tags/:id`

### 6.4 Categories

#### `GET /api/v1/categories`
#### `GET /api/v1/categories/:id`
#### `POST /api/v1/categories`
#### `PUT /api/v1/categories/:id`
#### `DELETE /api/v1/categories/:id`

## 7. 管理端与前台接口边界

建议将接口按使用场景分为两类：
- 公共接口：面向前台展示与外部 headless 消费
- 管理接口：面向后台管理端，后续可接入鉴权

建议路径：
- 公共接口：`/api/v1/...`
- 管理接口：`/api/v1/admin/...`

例如：
- `GET /api/v1/posts/slug/:slug` 供前台使用
- `GET /api/v1/admin/posts` 供后台管理使用

这样可以避免后台字段误暴露给公共 API。

## 8. 错误处理约定

错误处理原则：
- 控制层负责解析请求与返回响应
- 业务校验放在 `service`
- 数据库错误在 `repo` 中包装后上抛
- 禁止直接把底层数据库错误原文暴露给客户端

建议错误消息：
- `invalid request payload`
- `resource not found`
- `duplicate slug`
- `database operation failed`
- `internal server error`

## 9. 后续实现顺序建议

建议按以下顺序落地：
1. `api/response.go`，统一响应封装
2. `api/handler/health.go`，实现健康检查
3. `model/post.go`，定义文章模型
4. `repo/post.go`，封装文章存储层
5. `service/post.go`，实现文章业务逻辑
6. `api/handler/post.go`，暴露文章接口
7. `api/router.go`，统一注册路由

## 10. OpenAPI 规划

当前文档为手写设计文档。后续建议补充：
- `openapi.yaml`
- Swagger UI 或 Redoc
- 请求/响应 DTO 与 OpenAPI schema 对齐

建议在接口稳定后再生成 OpenAPI，避免早期频繁返工。
