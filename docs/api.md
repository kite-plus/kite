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
- 前台文章：`COALESCE(published_at, created_at) DESC, created_at DESC`
- 后台文章：`created_at DESC`
- 友情链接：`sort ASC, created_at DESC`

### 4.3 过滤

当前已实现的过滤能力：
- 前台文章：`keyword`、`tag_id`、`category_id`，并强制追加公开约束
- 后台文章：`status`、`keyword`、`tag_id`、`category_id`
- 前台友情链接：`keyword`，并强制追加 `is_active=true`
- 后台友情链接：`keyword`、`is_active`
- 标签：`keyword`
- 分类：`keyword`

### 4.4 管理员配置

Kite 当前采用单管理员设计，管理员身份与个人资料通过配置文件提供。

推荐配置示例：

```yaml
admin:
  enabled: true
  username: admin
  password_hash: "$2a$10$replace.with.bcrypt.hash"
  session_secret: "replace-with-a-long-random-secret"
  session_ttl_hours: 168
  profile:
    display_name: "Amigoer"
    email: "hello@example.com"
    bio: "独立开发者，喜欢写作、设计和 Go。"
    avatar: "https://example.com/avatar.png"
    website: "https://example.com"
    location: "Shanghai"
```

字段说明：
- `enabled=true` 时，`/api/v1/admin/...` 路由启用登录态校验
- `username` 与 `password_hash` 用于单管理员登录
- `password_hash` 推荐使用 `bcrypt`
- `session_secret` 用于管理员会话签名，需使用高强度随机字符串
- `profile` 用于当前管理员资料展示，`/api/v1/admin/auth/me` 会返回这些字段

## 5. 数据模型草案

以下是当前已经落地或即将扩展的核心资源。

### 5.1 Post

```json
{
  "id": "0195f3ff-4f17-7f0b-9e5f-15db2f2fb6a1",
  "title": "Hello Kite",
  "slug": "hello-kite",
  "summary": "A lightweight AI-native blog engine.",
  "content_markdown": "# Hello Kite",
  "content_html": "<h1 id=\"hello-kite\">Hello Kite</h1>\n",
  "status": "draft",
  "cover_image": "",
  "published_at": null,
  "show_comments": true,
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
- `content_markdown`: 文章 Markdown 原文，作为唯一编辑源
- `content_html`: 服务端根据 `content_markdown` 渲染并缓存的 HTML，用于前台展示
- `published_at`: 文章发布时间；当前台接口查询时，只有 `status=published` 且 `published_at<=当前时间` 的文章会被公开
- `show_comments`: 是否展示文章底部评论区
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
  "status": "active",
  "created_at": "2026-03-15T10:00:00Z",
  "updated_at": "2026-03-15T10:00:00Z"
}
```

字段说明：
- `url`: 全局唯一
- `sort`: 数值越小越靠前
- `status`: `active` / `pending` / `down`

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

### 6.1.1 Admin Auth

#### `POST /api/v1/admin/auth/login`

用途：
- 管理员登录
- 登录成功后由服务端写入 `HttpOnly Cookie`

请求体示例：

```json
{
  "username": "admin",
  "password": "your-password"
}
```

响应示例：

```json
{
  "code": 200,
  "data": {
    "auth_enabled": true,
    "authenticated": true,
    "user": {
      "username": "admin",
      "display_name": "Amigoer",
      "email": "hello@example.com",
      "bio": "独立开发者，喜欢写作、设计和 Go。",
      "avatar": "https://example.com/avatar.png",
      "website": "https://example.com",
      "location": "Shanghai"
    },
    "session_expires": "2026-03-22T10:00:00Z"
  },
  "msg": "ok"
}
```

#### `GET /api/v1/admin/auth/me`

用途：
- 获取当前管理员登录态与个人资料

说明：
- 当 `admin.enabled=true` 且未登录时，返回 `401`
- 当 `admin.enabled=false` 时，返回当前配置中的管理员资料，`authenticated=false`

#### `POST /api/v1/admin/auth/logout`

用途：
- 退出当前管理员会话
- 需要已登录

### 6.2 Posts

#### `GET /api/v1/posts`

用途：
- 获取前台公开文章列表
- 支持分页、关键字搜索、标签和分类过滤
- 仅返回 `status=published` 且 `published_at<=当前时间` 的文章

查询参数：
- `page`
- `page_size`
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
        "content_markdown": "# Hello Kite",
        "content_html": "<h1 id=\"hello-kite\">Hello Kite</h1>\n",
        "status": "published",
        "cover_image": "",
        "published_at": "2026-03-15T10:00:00Z",
        "show_comments": true,
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
- 根据 UUID 获取前台公开文章详情
- 草稿、归档和未来发布时间的文章统一返回 `404`

#### `GET /api/v1/posts/slug/:slug`

用途：
- 根据 slug 获取前台公开文章详情
- 供 classic 前台和 headless 前端共用
- 草稿、归档和未来发布时间的文章统一返回 `404`

说明：
- 公共文章接口仅提供只读能力
- 创建、更新、删除文章请使用 `/api/v1/admin/posts` 这组后台接口

### 6.2.1 Admin Posts

#### `GET /api/v1/admin/posts`

用途：
- 获取后台文章列表
- 支持分页、状态过滤、关键字搜索
- 不附加前台公开约束，可查询草稿、归档和定时发布文章

查询参数：
- `page`
- `page_size`
- `status`
- `keyword`
- `tag_id`
- `category_id`

#### `GET /api/v1/admin/posts/:id`

用途：
- 根据 UUID 获取后台文章详情

#### `GET /api/v1/admin/posts/slug/:slug`

用途：
- 根据 slug 获取后台文章详情

#### `POST /api/v1/admin/posts`

用途：
- 创建文章

请求体示例：

```json
{
  "title": "Hello Kite",
  "slug": "hello-kite",
  "summary": "A lightweight AI-native blog engine.",
  "content_markdown": "# Hello Kite",
  "status": "draft",
  "cover_image": "",
  "published_at": null,
  "show_comments": true,
  "category_id": "0195f400-0d80-730a-bf8a-4d9776db8f4d",
  "tag_ids": [
    "0195f400-13ad-7cbb-9b9f-04d15b759cb8"
  ]
}
```

请求体字段补充说明：
- `content_markdown` 为后台唯一编辑源，`content_html` 由服务端自动渲染与缓存
- `status=published` 且未显式传入 `published_at` 时，服务端会自动将发布时间补为当前时间
- `show_comments=false` 时，前台可据此隐藏文章底部评论区

#### `PUT /api/v1/admin/posts/:id`

用途：
- 全量更新文章

#### `PATCH /api/v1/admin/posts/:id`

用途：
- 局部更新文章

#### `DELETE /api/v1/admin/posts/:id`

用途：
- 软删除文章

### 6.3 Friend Links

#### `GET /api/v1/friend-links`

用途：
- 获取前台公开友情链接列表
- 支持分页与关键字搜索
- 仅返回 `is_active=true` 的友情链接

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
- 根据 UUID 获取前台公开友情链接详情
- 未启用友情链接统一返回 `404`

说明：
- 公共友情链接接口仅提供只读能力
- 创建、更新、删除友情链接请使用 `/api/v1/admin/friend-links`

### 6.3.1 Admin Friend Links

#### `GET /api/v1/admin/friend-links`

用途：
- 获取后台友情链接列表
- 支持分页、关键字搜索、启用状态过滤

查询参数：
- `page`
- `page_size`
- `keyword`
- `is_active`

#### `GET /api/v1/admin/friend-links/:id`

用途：
- 根据 UUID 获取后台友情链接详情

#### `POST /api/v1/admin/friend-links`

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

#### `PUT /api/v1/admin/friend-links/:id`

用途：
- 全量更新友情链接

#### `PATCH /api/v1/admin/friend-links/:id`

用途：
- 局部更新友情链接

#### `DELETE /api/v1/admin/friend-links/:id`

用途：
- 软删除友情链接

### 6.4 Tags

#### `GET /api/v1/tags`

用途：
- 获取公开标签列表
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
- 根据 UUID 获取公开标签详情

说明：
- 公共标签接口仅提供只读能力
- 创建、更新、删除标签请使用 `/api/v1/admin/tags`

### 6.4.1 Admin Tags

#### `GET /api/v1/admin/tags`

用途：
- 获取后台标签列表
- 支持分页与关键字搜索

#### `GET /api/v1/admin/tags/:id`

用途：
- 根据 UUID 获取后台标签详情

#### `POST /api/v1/admin/tags`

用途：
- 创建标签

请求体示例：

```json
{
  "name": "golang",
  "slug": "golang"
}
```

#### `PUT /api/v1/admin/tags/:id`

用途：
- 全量更新标签

#### `PATCH /api/v1/admin/tags/:id`

用途：
- 局部更新标签

#### `DELETE /api/v1/admin/tags/:id`

用途：
- 软删除标签

### 6.5 Categories

#### `GET /api/v1/categories`

用途：
- 获取公开分类列表
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
- 根据 UUID 获取公开分类详情

说明：
- 公共分类接口仅提供只读能力
- 创建、更新、删除分类请使用 `/api/v1/admin/categories`

### 6.5.1 Admin Categories

#### `GET /api/v1/admin/categories`

用途：
- 获取后台分类列表
- 支持分页与关键字搜索

#### `GET /api/v1/admin/categories/:id`

用途：
- 根据 UUID 获取后台分类详情

#### `POST /api/v1/admin/categories`

用途：
- 创建分类

请求体示例：

```json
{
  "name": "Backend",
  "slug": "backend"
}
```

#### `PUT /api/v1/admin/categories/:id`

用途：
- 全量更新分类

#### `PATCH /api/v1/admin/categories/:id`

用途：
- 局部更新分类

#### `DELETE /api/v1/admin/categories/:id`

用途：
- 软删除分类

## 7. 管理端与前台接口边界

当前已将文章、友情链接、标签、分类接口按使用场景分为两类：
- 公共接口：面向前台展示与外部 headless 消费
- 管理接口：面向后台管理端，后续可接入鉴权

建议路径：
- 公共接口：`/api/v1/...`
- 管理接口：`/api/v1/admin/...`

当前接口示例：
- `POST /api/v1/admin/auth/login` 供后台登录使用
- `GET /api/v1/posts/slug/:slug` 供前台使用
- `GET /api/v1/admin/posts` 供后台管理使用
- `GET /api/v1/friend-links` 供前台读取已启用友情链接
- `GET /api/v1/admin/friend-links` 供后台管理全量友情链接
- `GET /api/v1/tags` 与 `GET /api/v1/categories` 供前台读取基础元数据
- `GET /api/v1/admin/tags` 与 `GET /api/v1/admin/categories` 供后台管理使用

### 6.6 Comment（评论）

#### `GET /api/v1/posts/:id/comments`

用途：
- 获取指定文章的**已审核**评论列表

查询参数：
- `page`: 页码，默认 `1`
- `page_size`: 每页数量，默认 `20`

#### `POST /api/v1/posts/:id/comments`

用途：
- 前台访客提交评论（默认进入 `pending` 待审核状态）

请求体示例：

```json
{
  "author": "访客",
  "email": "visitor@example.com",
  "content": "写得真好！"
}
```

说明：
- 服务端自动记录 IP 和 User-Agent
- 新评论状态为 `pending`，需管理员审核后才会前台展示

#### 6.6.1 Admin Comments

#### `GET /api/v1/admin/comments`

用途：
- 获取后台评论列表
- 支持按状态、关键字、文章筛选

查询参数：
- `page`, `page_size`: 分页
- `status`: 可选值 `approved` / `pending` / `spam`
- `keyword`: 搜索评论内容或作者
- `post_id`: 按文章 UUID 筛选

#### `GET /api/v1/admin/comments/stats`

用途：
- 获取评论统计

响应示例：

```json
{
  "code": 200,
  "data": {
    "total": 42,
    "approved": 30,
    "pending": 10,
    "spam": 2
  },
  "msg": "ok"
}
```

#### `PATCH /api/v1/admin/comments/:id`

用途：
- 审核评论（修改状态）

请求体示例：

```json
{
  "status": "approved"
}
```

可选值：`approved` / `pending` / `spam`

#### `DELETE /api/v1/admin/comments/:id`

用途：
- 删除评论

### 6.7 Page（独立页面）

数据模型：

```json
{
  "id": "uuid",
  "title": "关于",
  "slug": "about",
  "content_markdown": "# 关于我\n...",
  "content_html": "<h1>关于我</h1>...",
  "status": "published",
  "sort_order": 0,
  "show_in_nav": true,
  "published_at": "2026-03-16T10:00:00Z",
  "template": "default",
  "config": "{}"
}
```

特殊字段说明：
- `template`: 模板名，对应 `templates/pages/{template}.html`，默认 `"default"`
- `config`: JSON 格式的模板参数，由模板清单（`*.json`）声明所需字段

#### `GET /api/v1/pages`

用途：
- 前台获取**已发布**页面列表

#### `GET /api/v1/pages/slug/:slug`

用途：
- 前台根据 slug 获取已发布页面详情

#### 6.7.1 Admin Pages

#### `GET /api/v1/admin/pages`

用途：
- 获取后台页面列表
- 支持分页、状态筛选、关键字搜索

#### `GET /api/v1/admin/pages/:id`

用途：
- 根据 UUID 获取页面详情

#### `POST /api/v1/admin/pages`

用途：
- 创建独立页面

请求体示例：

```json
{
  "title": "我的开源项目",
  "slug": "github",
  "content_markdown": "以下是我的 GitHub 项目展示。",
  "status": "published",
  "sort_order": 10,
  "show_in_nav": true,
  "template": "github",
  "config": "{\"username\": \"amigoer\", \"count\": 6}"
}
```

#### `PUT /api/v1/admin/pages/:id`

用途：
- 全量更新页面

#### `PATCH /api/v1/admin/pages/:id`

用途：
- 局部更新页面

#### `DELETE /api/v1/admin/pages/:id`

用途：
- 软删除页面

### 6.8 Settings（系统设置）

#### `GET /api/v1/admin/settings`

用途：
- 获取全部系统设置

响应示例：

```json
{
  "code": 200,
  "data": {
    "site": {
      "site_name": "Kite",
      "site_url": "https://example.com",
      "description": "一个 AI 原生博客引擎",
      "keywords": "blog, kite",
      "favicon": "/favicon.ico",
      "logo": "",
      "icp": "京ICP备xxxxxxxx号",
      "footer": "© 2026 Kite"
    },
    "post": {
      "posts_per_page": 10,
      "enable_comment": true,
      "enable_toc": true,
      "summary_length": 200,
      "default_cover_url": ""
    },
    "render": {
      "render_mode": "classic",
      "api_prefix": "/api/v1",
      "enable_cors": true
    },
    "ai": {
      "enabled": false,
      "provider": "",
      "api_key": "****",
      "model": "",
      "auto_summary": false,
      "auto_tag": false
    }
  },
  "msg": "ok"
}
```

说明：
- `ai.api_key` 返回掩码值（如 `sk-4****abcd`），不会暴露完整密钥

#### `PUT /api/v1/admin/settings`

用途：
- 更新全部系统设置（运行时覆盖，不持久化到配置文件）

请求体：与 GET 返回的 `data` 结构一致

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

## 9. V1 已实现能力总结

- 全部 CRUD：文章、分类、标签、评论、独立页面、友情链接、系统设置
- 鉴权：Cookie-based Session Auth（login/logout/middleware）
- SSR 模板渲染 + Headless API 双模式
- Admin SPA 嵌入 Go 二进制，生产环境单进程部署
- Markdown → HTML 自动转换

## 10. OpenAPI 规划

当前文档为手写设计文档。后续建议补充：
- `openapi.yaml`
- Swagger UI 或 Redoc
- 请求/响应 DTO 与 OpenAPI schema 对齐

建议在接口稳定后再生成 OpenAPI，避免早期频繁返工。
