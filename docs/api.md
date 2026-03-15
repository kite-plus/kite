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

当前已实现的列表接口采用服务端固定排序：
- 文章：`created_at DESC`
- 友情链接：`sort ASC, created_at DESC`

### 4.3 过滤

当前已实现的过滤能力：
- 文章：`status`、`keyword`、`tag_id`、`category_id`
- 友情链接：`keyword`、`is_active`

## 5. 数据模型草案

以下是当前已经落地或即将扩展的核心资源。

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
  "category_id": "0195f400-0d80-730a-bf8a-4d9776db8f4d",
  "category": {
    "id": "0195f400-0d80-730a-bf8a-4d9776db8f4d",
    "name": "Backend",
    "slug": "backend"
  },
  "tags": [
    {
      "id": "0195f400-13ad-7cbb-9b9f-04d15b759cb8",
      "name": "golang",
      "slug": "golang"
    }
  ],
  "created_at": "2026-03-15T10:00:00Z",
  "updated_at": "2026-03-15T10:00:00Z"
}
```

字段说明：
- `status`: `draft` / `published` / `archived`
- `slug`: 全局唯一，用于前台路由
- `summary`: 可人工填写，也可由 AI 自动生成
- `category_id`: 可为空，表示文章所属分类
- `tags`: 多对多标签集合

### 5.2 FriendLink

```json
{
  "id": "0195f401-2f0a-7a1d-9d8f-2d62d8f1bc2a",
  "name": "Example Blog",
  "url": "https://example.com",
  "description": "A friendly site about writing and engineering.",
  "logo": "https://example.com/logo.png",
  "sort": 10,
  "is_active": true,
  "created_at": "2026-03-15T10:00:00Z",
  "updated_at": "2026-03-15T10:00:00Z"
}
```

字段说明：
- `url`: 全局唯一
- `sort`: 数值越小越靠前
- `is_active`: 是否启用该友情链接

### 5.3 Tag

```json
{
  "id": "0195f400-13ad-7cbb-9b9f-04d15b759cb8",
  "name": "golang",
  "slug": "golang",
  "created_at": "2026-03-15T10:00:00Z",
  "updated_at": "2026-03-15T10:00:00Z"
}
```

### 5.4 Category

```json
{
  "id": "0195f400-0d80-730a-bf8a-4d9776db8f4d",
  "name": "Backend",
  "slug": "backend",
  "created_at": "2026-03-15T10:00:00Z",
  "updated_at": "2026-03-15T10:00:00Z"
}
```

## 6. 当前已实现 API

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
        "content": "# Hello Kite",
        "status": "published",
        "cover_image": "",
        "published_at": "2026-03-15T10:00:00Z",
        "category_id": "0195f400-0d80-730a-bf8a-4d9776db8f4d",
        "category": {
          "id": "0195f400-0d80-730a-bf8a-4d9776db8f4d",
          "name": "Backend",
          "slug": "backend"
        },
        "tags": [
          {
            "id": "0195f400-13ad-7cbb-9b9f-04d15b759cb8",
            "name": "golang",
            "slug": "golang"
          }
        ],
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
  "cover_image": "",
  "published_at": null,
  "category_id": "0195f400-0d80-730a-bf8a-4d9776db8f4d",
  "tag_ids": [
    "0195f400-13ad-7cbb-9b9f-04d15b759cb8"
  ]
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

### 6.3 Friend Links

#### `GET /api/v1/friend-links`

用途：
- 获取友情链接列表
- 支持分页、关键字搜索、启用状态过滤

查询参数：
- `page`
- `page_size`
- `keyword`
- `is_active`

响应示例：

```json
{
  "code": 200,
  "data": {
    "items": [
      {
        "id": "0195f401-2f0a-7a1d-9d8f-2d62d8f1bc2a",
        "name": "Example Blog",
        "url": "https://example.com",
        "description": "A friendly site about writing and engineering.",
        "logo": "https://example.com/logo.png",
        "sort": 10,
        "is_active": true,
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

#### `GET /api/v1/friend-links/:id`

用途：
- 根据 UUID 获取友情链接详情

#### `POST /api/v1/friend-links`

用途：
- 创建友情链接

请求体示例：

```json
{
  "name": "Example Blog",
  "url": "https://example.com",
  "description": "A friendly site about writing and engineering.",
  "logo": "https://example.com/logo.png",
  "sort": 10,
  "is_active": true
}
```

#### `PUT /api/v1/friend-links/:id`

用途：
- 全量更新友情链接

#### `PATCH /api/v1/friend-links/:id`

用途：
- 局部更新友情链接

#### `DELETE /api/v1/friend-links/:id`

用途：
- 软删除友情链接

### 6.4 Tags

#### `GET /api/v1/tags`

用途：
- 获取标签列表
- 支持分页与关键字搜索

查询参数：
- `page`
- `page_size`
- `keyword`

响应示例：

```json
{
  "code": 200,
  "data": {
    "items": [
      {
        "id": "0195f400-13ad-7cbb-9b9f-04d15b759cb8",
        "name": "golang",
        "slug": "golang",
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

#### `GET /api/v1/tags/:id`

用途：
- 根据 UUID 获取标签详情

#### `POST /api/v1/tags`

用途：
- 创建标签

请求体示例：

```json
{
  "name": "golang",
  "slug": "golang"
}
```

#### `PUT /api/v1/tags/:id`

用途：
- 全量更新标签

#### `PATCH /api/v1/tags/:id`

用途：
- 局部更新标签

#### `DELETE /api/v1/tags/:id`

用途：
- 软删除标签

### 6.5 Categories

#### `GET /api/v1/categories`

用途：
- 获取分类列表
- 支持分页与关键字搜索

查询参数：
- `page`
- `page_size`
- `keyword`

响应示例：

```json
{
  "code": 200,
  "data": {
    "items": [
      {
        "id": "0195f400-0d80-730a-bf8a-4d9776db8f4d",
        "name": "backend",
        "slug": "backend",
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

#### `GET /api/v1/categories/:id`

用途：
- 根据 UUID 获取分类详情

#### `POST /api/v1/categories`

用途：
- 创建分类

请求体示例：

```json
{
  "name": "Backend",
  "slug": "backend"
}
```

#### `PUT /api/v1/categories/:id`

用途：
- 全量更新分类

#### `PATCH /api/v1/categories/:id`

用途：
- 局部更新分类

#### `DELETE /api/v1/categories/:id`

用途：
- 软删除分类

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
- `GET /api/v1/friend-links` 可作为公开友情链接读取接口
- `GET /api/v1/admin/friend-links` 可作为后台管理接口的后续扩展

## 8. 错误处理约定

错误处理原则：
- 控制层负责解析请求与返回响应
- 业务校验放在 `service`
- 数据库错误在 `repo` 中包装后上抛
- 禁止直接把底层数据库错误原文暴露给客户端

当前常见错误消息：
- `invalid request payload`
- `resource not found`
- `duplicate slug`
- `duplicate friend link url`
- `duplicate tag slug`
- `duplicate category slug`
- `internal server error`

## 9. 后续实现顺序建议

建议后续按以下顺序继续扩展：
1. 文章与标签/分类关联
2. 管理端接口拆分到 `/api/v1/admin`
3. 鉴权与权限控制
4. OpenAPI 文档生成

## 10. OpenAPI 规划

当前文档为手写设计文档。后续建议补充：
- `openapi.yaml`
- Swagger UI 或 Redoc
- 请求/响应 DTO 与 OpenAPI schema 对齐

建议在接口稳定后再生成 OpenAPI，避免早期频繁返工。
