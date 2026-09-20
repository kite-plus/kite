# Kite 总体架构设计

> 状态：设计中（M0 开工前的基线） · 最近更新：2026-09-21
> 主题系统详见 [theme-system.md](theme-system.md)，插件系统详见 [plugin-system.md](plugin-system.md)。
> `[EV]` 标记的结论有既有项目的实证支撑，来源见 [§34](#34-证据来源)。

---

## Context：Kite 要解决什么

现有的内容发布工具被劈成互不相容的两半：

- **SSG 阵营**（Hugo / Hexo / Astro）：Markdown + Git + 静态托管的工程体验极好，但内容管理体验差——写文章要直接编辑文件，没有真正的后台，媒体/分类/主题管理不直观，普通创作者门槛高。
- **CMS 阵营**（WordPress / Halo / Typecho）：后台完整、在线写作、点击发布，但天然绑定 Server Runtime + Database，和 Git / Markdown / 静态托管的结合非常别扭。

用户被迫二选一，而且**这个选择极难反悔**——换部署方式几乎等于换 CMS、换写作方式、换全部内容。

Kite 的目标结果：

> 用户不再选择 "Hugo 还是 WordPress"，只选择"我的内容今天想部署在哪里"。

**Content Management 与 Deployment Strategy 解耦**是本项目的第一性原理。本文档中每一个架构决策都必须能回溯到这一条。

---

## 0. 两个核心判断

整份设计建立在这两条之上。不认同这两条，后面所有细节都失去意义。

### 0.1 Store × Runtime 是两个正交轴

现有产品的根本缺陷不是功能少，而是把两件本应正交的事耦死了：

| | **Build Runtime**（产出静态 HTML） | **Serve Runtime**（HTTP 动态渲染） |
|---|---|---|
| **File Store**<br>（Markdown + Git 为真相源） | Hugo 模式 ← **M0** | `hugo server` / Grav / Kirby ← **M1**<br>**Kite 的本地写作 + 预览运行时** |
| **DB Store**<br>（SQLite/PG/MySQL 为真相源） | DB 写作 + 静态导出 | Halo / WordPress 模式 ← **M7** |

四个格子全部合法，都要上线。

**左中格（File + Serve）不是玩具**——它同时是本地开发循环、Admin 运行时和预览引擎。Grav / Kirby / `hugo server` 证明它是可用的生产架构 `[EV]`。先把这一格做出来，"正交轴"这个主张就被免费验证了。

所谓 Static / Dynamic / Headless 三种"模式"，在这个矩阵里其实是 **Store 的选择 + Runtime 的选择 + Publisher 的选择** 三个独立维度的组合，**不是三个产品**。

**架构含义（现在就必须做对）：**

1. Core 必须同时对 Store 和 Runtime 解耦。任何一个方向耦合了，另外几格就废掉。
2. 判断任何新代码该放哪，问两个问题：**它依赖具体 Store 吗？它依赖具体 Runtime 吗？** 两个都"否"才配进 Core。

### 0.2 只有一个读模型（Read Model）

**这是全文最高杠杆的一条决策。**

不要写两套查询实现。**做一份 SQL 读模型 schema，和恰好一个 `ContentReader` 实现：**

- **Static 模式**：这些行由 indexer 从 Markdown 文件派生写入（`.kite/cache/index.db`，可随时删除重建）。
- **Dynamic 模式**：这些行由 writer 直接写入（它就是主库）。

**为什么**：File Store 本来就**必须**有索引才能支撑 Admin 的列表/过滤/排序/分页。纯文件系统方案在 1~2k 篇量级就会垮——Grav 在这个量级出现 3~8 秒的页面加载，根因是"YAML 解析与文件处理"；Kirby 不得不发展出整个缓存插件生态 `[EV]`。TinaCMS 最终也收敛到 Bridge（存储）+ Database（索引）的同一结构 `[EV]`。

既然两种模式的查询最终都由 SQL 提供，那就**不要假装有两套实现**。

**推论**：主题、Admin、API 三者永远只看到同一个读模型，"Static 与 Dynamic 渲染分叉"这一整类问题在架构层面被消灭。

---

## 1. 产品定位

**A modern open-source publishing platform. One Content, Multiple Destinations.**

> Kite 管理你的内容；部署到哪里，是内容的一个属性，不是产品的一个分支。

- **对创作者**：像 WordPress 一样写作，像 Hugo 一样部署。
- **对开发者**：内容是纯 Markdown + Git，可 diff、可 review、可 CI，永不被平台绑架。
- **对生态**：一套 Content Model / Theme Contract / Plugin Contract，覆盖 Blog / Documentation / Knowledge Base / Portfolio / 企业站 / 项目站 / 摄影站。

**不是什么**（防止定位漂移）：

- 不是 Static Site Generator —— SSG 只是 Kite 的一个 Runtime
- 不是 Headless CMS —— Headless 只是一个 Runtime
- 不是 Git-based CMS GUI —— 那只是 Static 模式下 Admin 的实现形态
- 不做拖拽式可视化建站

---

## 2. 核心竞争力

按护城河深度排序。**"用 Go 写的"不在列表里——Go 是实现手段，不是竞争力。**

1. **统一 Static 与 Dynamic**：同一份内容、同一个后台、同一套主题、同一套插件契约，部署方式可随时切换而不迁移。这是唯一没有成熟竞品的位置。
2. **Git 是一等公民，但对用户不可见**：文件是唯一真相源，同时 Admin 把 commit / push / CI 状态包装成"发布"一个按钮。既满足开发者，也不吓跑普通用户。
3. **不弄脏用户的仓库**：精确提交、保留 front matter 原貌、不碰用户的暂存区。听起来是细节，实际是 Git-backed CMS 的口碑生死线（见 [§16](#16-git-workflow最高危模块)）。
4. **Local First + 零锁定**：单二进制、本地可跑、`kite export` 随时拿走 Markdown + Media + Config。即使 Kite Cloud 永不存在，产品也完整。
5. **Reproducible Build**：`kite.lock` + `kitew` 保证 Local Build = CI Build = Production Build。这是 SSG 生态长期的痛点。
6. **Content Type System 而非 Blog Post 模型**：从第一天就是通用内容抽象，决定了 Kite 能不能长成平台。

**v1.0 要验证的命题**：

> 一个不懂 Git 的创作者，能否在不打开终端的前提下完成"安装 → 写作 → 配图 → 发布 → 线上可见"，且产出物是一个标准的 Git 仓库 + 静态站点？

---

## 3. 产品边界

**做**：内容的创作 / 组织 / 存储 / 渲染 / 构建 / 发布；主题与插件扩展体系；面向单站点的完整后台；多种部署目标的适配。

**长期不做**：

- 拖拽式可视化建站（Wix / Webflow 方向）
- 电商、会员付费、订阅计费（交给插件）
- 社交网络 / 联邦宇宙（可由插件接入）
- 企业级 CMS 的审批工作流、内容合规、翻译管理
- 自建托管 SaaS（Kite Cloud 是 V4 的可选分支，不是产品主线）

**暂不做**（有明确时间点，见 [§32](#32-现在不要设计的东西)）：多站点、实时协作、细粒度 RBAC、Marketplace、Mobile App、GraphQL、Block Editor。

---

## 4. 用户类型

设计决策发生冲突时，按此优先级裁决。

| 优先级 | 用户 | 画像 | 关键诉求 | 对架构的强制约束 |
|---|---|---|---|---|
| **P0** | **Developer Blogger** | 会用 Git/Markdown，嫌 Hugo 后台缺失、嫌 WordPress 太重 | 文件真相源、可 diff、CI 友好、单二进制 | File Store 必须是一等公民，**不能是 DB 的导出产物** |
| **P0** | **Creator（半技术）** | 能装软件、看得懂 Markdown，但不想碰终端和 Git | 图形后台、所见即所得、一键发布 | Admin 必须完整，Git 必须被**完全**包装 |
| P1 | Team / Docs Site | 用 Kite 做项目文档站、公司站 | 自定义内容类型、多人、Review | Content Type 与 Taxonomy 必须泛化 |
| P2 | Theme / Plugin Developer | 生态建设者 | 稳定契约、好文档、SDK | 契约稳定性优先于内部实现优雅 |
| P3 | Headless 使用者 | 前端自己写，只要 API | 稳定 REST API、Webhook | Admin 必须走公开 API |

**P0 的两类用户必须同时满足**——这正是 Kite 的定位所在。任何"只讨好其中一类"的设计都要打回。

---

## 5. 核心使用场景

场景是验收标准的来源，每个场景对应后面里程碑的 Demo。

**S1 个人博客（Static + GitHub Pages）** — 主场景
`kite init` → `kite run` → 浏览器写文章、传图 → 点发布 → Git commit/push → GitHub Actions 跑 `kite build` → Pages 上线。用户全程没打开终端（除了第一次启动）。

**S2 开发者混合工作流**
同一个仓库，白天在 VS Code 写 Markdown + `git push`，晚上在 Kite Admin 补图和改标签。两边操作同一份文件，互不覆盖，冲突有提示。

**S3 动态站点（Dynamic + VPS/Docker）**
下载单二进制 + SQLite，`kite serve`，即时发布无需构建。适合更新频繁、需要评论/搜索/会员的站点。

**S4 项目文档站（Static + Cloudflare Pages）**
自定义 `document` 内容类型、多级目录、版本化 Taxonomy、构建期生成全文搜索索引。

**S5 Headless**
Kite 作为内容后端，Next.js / Nuxt 前端消费 REST API。

**S6 模式迁移**
用户从 Static 切到 Dynamic（或反向），`kite migrate` 搬运内容，主题和内容不变。**这个场景的存在本身就是对架构的约束**：Store 必须可换。

---

## 6. 系统架构

```
                          Kite Studio (React SPA, go:embed)
                     Editor / Media / Theme / Plugin / Git Panel
                                      │
                                 REST API  ← Admin 与 Headless 同源
                                      │
┌─────────────────────────────────────▼─────────────────────────────────────┐
│                                 Kite Core                                 │
│                                                                           │
│  读侧（唯一实现）      ContentReader  ←  ContentQuery 规格对象             │
│  写侧（唯一方法）      ContentWriter.Apply(ctx, ChangeSet) → Revision      │
│                                                                           │
│  Domain              Content · ContentType Registry · ChangeSet           │
│                      Revision · DeliveryState · Taxonomy(投影)            │
│                                                                           │
│  Contracts           Reader / Writer / Publisher / Renderer / HookBus     │
│                      BuildContext / MediaStore                            │
└───────┬───────────────────────┬────────────────────────┬──────────────────┘
        │                       │                        │
┌───────▼────────┐    ┌─────────▼─────────┐    ┌─────────▼──────────┐
│ 写入方 + 索引  │    │   Render Engine   │    │  Plugin Runtime    │
│                │    │                   │    │  (M8, wazero)      │
│ FileWriter     │    │ Markdown Pipeline │    │                    │
│  + Indexer ────┼───▶│ Theme Engine      │◀───┤ HookBus 消费者     │
│ SQLWriter ─────┼───▶│ URL Resolver      │    │ Capability + Perm  │
│  （同一读模型）│    │ Asset Pipeline    │    └────────────────────┘
└───────┬────────┘    └─────────┬─────────┘
        │                       │
        └───────────┬───────────┘
                    │
        ┌───────────▼────────────┐        ┌────────────────────┐
        │       Runtime          │        │     Publisher      │
        │  Build · Serve · API   │        │ Git · DB · Webhook │
        └───────────┬────────────┘        └─────────┬──────────┘
                    │                               │
     ┌──────────────┼───────────────┐    ┌──────────┴──────────┐
  Static         Dynamic         Headless   CI/CD   DeliveryState
```

**分层铁律：**

- 箭头只能从上往下。Core **不允许** import 任何 Store / Runtime / Publisher 的具体实现。**CI 强制检查这条**（见 [§25](#25-项目目录结构)）。
- Admin **只能**走 REST API，不能直接调 Service——保证未来的 Cloud Admin、Headless 客户端复用同一套接口。
- 只有一个 `ContentReader` 实现；`FileWriter` 和 `SQLWriter` 都往同一份读模型 schema 里写。

---

## 7. 模块边界

用户常见的困惑是"这段代码该放哪"。每一类都给出**可机械执行的判定标准**。

### Core（内核：稳定、唯一、不可替换）

| 模块 | 职责 | 为什么是 Core |
|---|---|---|
| Content Model & Type Registry | 领域模型与类型注册 | 所有模式共享，改动影响一切 |
| ContentReader / ContentQuery | 唯一读模型（[§0.2](#02-只有一个读模型read-model)） | 消灭 Static/Dynamic 查询分叉 |
| ContentWriter.Apply / ChangeSet | 唯一写入口（[§9.2](#92-写侧只有一个方法)） | 事务、冲突、审计、撤销的共同基底 |
| Markdown Pipeline | Source → AST → HTML | Build 和 Serve 必须用同一个，否则产出分叉 |
| RenderContext + URL Resolver | 模板数据与链接生成 | 主题契约的地基，最不能分叉的地方 |
| HookBus / Event Bus | 扩展点总线 | 插件的地基，V1 自用 |
| BuildContext | 构建期 I/O 的唯一通道 | 构建纯度的守门人 |
| Config / kite.lock 解析 | 配置与版本锁 | 决定可重现性 |
| DeliveryState | 发布投递状态机 | 绝不能塞进 `Status` 字段 |

**判定**：不依赖具体 Store、也不依赖具体 Runtime → Core。

### Extension（可插拔，可多实现，可第三方提供）

写入方（`FileWriter` / `SQLWriter`）、Media Storage 驱动（Local / S3 / R2 / OSS）、Publisher 实现（Git / DB / Webhook）、Theme、Plugin。

**判定**：能同时存在多个实现、且切换实现不改业务代码 → Extension。

### Build Time（只在 `kite build` 期间存在）

Build Planner、Dependency Recorder、Build Cache、Asset Pipeline（fingerprint / minify）、Search Index 生成、RSS / Sitemap 生成、Output Emitter。

**判定**：产物是文件，且进程结束即消失 → Build Time。

### Runtime（只在 `kite serve` / `kite run` 期间存在）

HTTP Router、Middleware、Session / Auth、Response Cache、Live Preview、File Watcher、Task Queue Worker。

**判定**：依赖请求生命周期或长驻进程 → Runtime。

### Deployment（不在 Go 代码里，在仓库模板和 CI 配置里）

GitHub Actions workflow、Cloudflare Pages 配置、Dockerfile / Compose / Helm、`kitew` wrapper、`setup-kite` Action。

**判定**：换一个托管商就要改 → Deployment，**必须留在数据/模板层，不能编译进 Core**。

---

## 8. Content Model

### 8.1 设计原则

1. 不绑定 Blog Post。第一天就是通用 Content + Kind。
2. ID 稳定，且与路径、slug、URL 完全解耦。
3. Taxonomy 泛化，不硬编码 tags / categories。
4. 扩展字段走 Schema，不走随意的 map。
5. 渲染产物永远是派生物，不进 Store。

### 8.2 领域模型

```go
// Core —— 与任何 Store 实现无关
type Content struct {
    ID          ContentID              // ULID/UUIDv7，稳定不变
    Kind        Kind                   // "post" | "page" | 未来插件注册
    Slug        string                 // URL 片段；改它不动文件
    Title       string
    Status      Status                 // draft | scheduled | published | archived
    Body        Body                   // { Format: markdown|html, Raw: string }
    Meta        Meta                   // 受 ContentType.Fields 约束
    Taxonomies  map[string][]string    // {"tag": [...], "category": [...]}，是字符串不是实体
    Aliases     []string               // 旧 URL → 301
    Locale      string                 // i18n 预留，V1 单值
    Path        ResourceLocator        // 字节存在哪；与 URL 无关
    Revision    Revision               // 不透明：File=内容 hash，SQL=版本号
    CreatedAt, UpdatedAt, PublishedAt, DeletedAt *time.Time
}
```

### 8.3 关键决策

#### D1. ID 稳定，且不从路径或 slug 派生 —— **第一个 commit 就要有** `[已冻结]`

**为什么**：Obsidian 是"路径即身份"的反面教材——重命名就断链，社区不得不用插件往 front matter 里塞 UUID。Kirby 则把 UUID 做成核心特性，官方理由原文就是"传统文件路径引用在内容重组时会断" `[EV]`。

**ID 具体用来干五件事**（不是为了好看）：

1. **跨重命名的内部引用** —— `page://01J8K…` 链接协议，渲染时解析
2. **挂载带外状态**（浏览量、评论、定时发布、自动保存草稿）**且不与路径耦合** —— 没有这一条，Static 与 Dynamic 无法共享同一个读模型
3. **媒体归属** —— 哪些附件属于哪篇内容，使孤儿回收成为可能
4. **重定向表** —— 旧 slug → id → 当前 URL，自动生成
5. **Admin 列表的稳定 React key** —— 重命名时列表不闪烁

**两条必须同时上线的规则：**

- `kite doctor` 为手写文件批量补 ID，生成一个用户可 review 的 ChangeSet。
- **索引期发现重复 ID 是硬错误，绝不静默修复。** 用户复制一个文件夹是完全正常的操作，Kirby 明确文档化了这个坑 `[EV]`。

#### D2. Slug / 物理路径 / URL 三者解耦 `[已冻结]`

- Hugo 的模型是"路径即 URL"，改 URL 就要改目录。**Kite 不采用。**
- 规则：**URL 来自 `slug`，路径只是字节的存放位置。** 改 slug 不动文件。
- 用户**显式**要求移动文件时：ChangeSet 里出现 delete + add 对，Publisher 用 `git mv` 保留历史，并自动写入 `aliases:` 让旧 URL 301。
- **为什么**：默认 rename-on-slug-change 会产出几百文件的 diff 和一地断链。

#### D3. Taxonomy 降级为「内容上的字符串列表 + 派生投影」，不是实体 `[已冻结]`

这是消除"文件模式没有外键、SQL 模式有外键"这一抽象泄漏的唯一干净办法。**referential integrity 的分歧会改变领域模型本身，是真正会导致重写的那种泄漏。**

- 两种模式下语义完全一致："重命名标签" = 改写 N 条内容记录。诚实、统一、与所有文件型系统一致。
- SQL 模式里的 `tags` 表是**物化索引**，可从内容重建，不是真相源。
- 将来要给标签加描述/封面：单独加 `content/tags/go/_index.md`（branch bundle 风格）。**但这件事必须在主题契约冻结前决定**，见 [theme-system.md](theme-system.md)。

#### D4. Status 与「是否真的上线」严格分离 `[已冻结]`

`Status = published` 只表达**内容意图**。内容**实际是否可见**由 Runtime + Publisher 决定——Static 模式下 push 完成、CI 构建完成前，published 的内容并未上线。

因此引入独立的 **DeliveryState**（[§15.2](#152-deliverystate)）。

> **把上线状态塞进 status 枚举，是本项目最容易犯、且代价最高的错误之一。**

#### D5. ContentType Registry：V1 落地、内置两种、不开放注册 `[设计中]`

```go
type ContentType struct {
    Kind, Label string
    Route       RoutePattern   // "/posts/:slug" / "/docs/:path"
    Layout      StorageLayout  // Bundle（目录）| SingleFile
    Fields      FieldSchema    // JSON Schema 子集 → 驱动 Admin 表单
    Taxonomies  []string       // 允许挂载的分类轴
    Templates   TemplateHints  // 默认模板查找名
    Sortable    []string
}
```

**为什么现在就要 Registry**：路由规则、模板查找、Admin 表单生成这三件事都必须从类型元数据读取。V1 若硬编码成 `if kind == "post"`，开放自定义类型时就是一次彻底重写。

**态度：先留接口，不要现在实现开放注册。**

#### D6. Meta / Theme Settings / Plugin Settings 三处共用一套 Schema → 表单渲染器

一次投入三处受益，也是"每加一种内容类型就要写一个 Admin 页面"的唯一解药。Halo 的 `settings.yaml` 驱动主题配置 UI 是这个模式的成功先例 `[EV]`。

---

## 9. Storage Model

### 9.1 读侧：一个读模型，一个实现

```go
type ContentReader interface {
    Get(ctx context.Context, id ContentID) (*Content, error)
    Query(ctx context.Context, q ContentQuery) (Page[ContentSummary], error)
    Caps() Capabilities
}

type ContentQuery struct {
    Kinds, Statuses  []string
    TagsAny, TagsAll []string
    CategoryPath     string
    AuthorID         string
    DateRange        *Range
    TextQuery        string
    Sort             []SortKey
    Cursor           string   // 不透明
    Limit            int
}
```

**三条硬规则：**

1. **Query 是规格对象，不是 method-per-query。** 否则接口无限膨胀。
2. **后端满足不了某个查询就返回 `ErrUnsupportedQuery`，绝不静默退化成"全量加载再在 Go 里过滤"。** 那种静默回退是 O(n) 重新爬回来的唯一路径。
3. **分页从第一天就用复合游标 `(published_at DESC, id DESC)` 的 base64 编码，不用 offset。** 可变集合上的 offset 分页必然产生重复行和漏行，而且一旦 API 公开就改不掉了。

### 9.2 写侧：只有一个方法

```go
type ContentWriter interface {
    Apply(ctx context.Context, cs ChangeSet) (Revision, error)
}

type ChangeSet struct {
    Ops     []Op      // PutContent | DeleteContent | RenameContent | PutMedia | DeleteMedia
    Message string
    BaseRev Revision  // 乐观并发：不匹配则 409
}
```

**没有 `UpdateTitle`，没有 `SetTags`。**

理由：ChangeSet 同时是——

- Git 的 **commit 单元**
- SQL 的 **事务单元**
- **冲突检测单元**
- **审计记录**
- **撤销记录**

把这五件事建立在一个结构上，它们将来全部是免费的；反过来，把它们回填到 40 个字段级 mutator 上，就是那次重写。

> **这是全项目第三重要的决策。任何一次"就加一个 setter 吧"的妥协，都要在 code review 里打回。**

`Revision` 保持不透明字符串：File 模式是内容 hash 或 commit oid，SQL 模式是版本号。

### 9.3 派生索引的一致性（Static 模式最容易出 bug 的地方）

**铁律：数据单向流动——文件 → 索引。绝不从索引回写文件。**

**索引里不允许存在任何不可从 `content/` + config + theme 推导出来的东西**——草稿状态、定时发布时间、编辑锁都不行，它们属于 front matter 或另一个非缓存存储。

**可验证的不变量（写进 CI）：**

```
rm -rf .kite/cache && kite index
```

产出的 DB dump 必须与之前逐字节一致。

**三层一致性 —— watcher 是优化，扫描才是真相：**

| 层 | 机制 | 成本 | 角色 |
|---|---|---|---|
| **T0** | fsnotify 监听 `content/` `static/` `themes/`，**目录级**，debounce 100ms | ~0 | 只负责降低延迟，**永远不是权威** |
| **T1** | 只 stat 的全树遍历，与索引行比对；仅 stat 不匹配才重算 hash | 10k 文件约 30~80ms | 任何权威读取前跑一次（打开编辑器、发布、刷新列表） |
| **T2** | Git 哨兵：持久化 `HEAD` oid + `.git/index` mtime；HEAD 变化时执行 `git diff --name-status <old> <new> -- content/` | ~1ms | **只失效真正变化的路径**，O(changed) 而非 O(n)——这是切分支能便宜的原因 |

**为什么 fsnotify 不能当正确性机制** `[EV]`：

- 编辑器用"写临时文件 + rename"原子保存，会直接干掉被监听的 inode，watch 静默失效。fsnotify 官方的建议就是**监听父目录而非文件**。
- macOS FSEvents 会合并快速变更，且不像 inotify 那样配对 MOVED_FROM / MOVED_TO。
- `git checkout` 会在毫秒内产生上千事件，撑爆队列。

**mtime 不是安全的缓存键。** 直接抄 Git 的 "racy git" 规则 `[EV]`：

- 索引行存 `path, dev, ino, mode, size, mtime_ns, content_sha256, frontmatter_json, indexed_at`
- **若 `mtime_ns >= 索引写入时间 - 1s`，无条件重算 hash**，不管 stat 是否匹配

亚秒精度文件系统允许"改 → 索引 → 再改"发生在同一时间刻度内，这就是 Git 自己要处理的那个 race。

**写入是原子且自抑制的：**

```
写 .kite/tmp/（必须同一文件系统）→ fsync → os.Rename → fsync 目录
  → 立即更新索引行 → 把 hash 放进短时抑制集
```

抑制集防止自己的 fsnotify 事件触发重索引循环。

### 9.4 并发编辑 = CAS，绝不 last-write-wins

- `GET /api/v1/contents/:id` 返回 `etag = sha256(文件原始字节)`
- `PUT` 必须带 `If-Match`
- 保存时重读文件比对 hash，不匹配 → **409 + `{base, ours, theirs}` 三方载荷**，Admin 给出 diff 与「覆盖 / 放弃 / 合并」

> **这一个特性，就是"我敢把 CMS 挂在我的 Git 仓库上"和 Decap 的区别** `[EV]`。必须 V1 就有。

### 9.5 目录布局

```
my-blog/
├── kite.yaml                  # 配置（人写，入 Git）
├── kite.lock                  # 版本锁（生成，入 Git，M6）
├── content/
│   ├── posts/
│   │   └── hello-world/       # Page Bundle：内容与其资源同目录
│   │       ├── index.md
│   │       └── cover.webp
│   └── pages/
│       └── about.md           # 单文件布局同样合法
├── static/                    # 原样拷贝到输出根
├── layouts/                   # 站点级模板覆盖（优先于主题）
├── themes/                    # 本地主题；受管主题在 .kite/themes
├── plugins/
├── .kite/                     # 全部派生物，gitignore
│   ├── cache/index.db         # 派生索引
│   ├── cache/wasm/            # wazero 编译缓存（M8）
│   ├── tmp/                   # 原子写临时区（同文件系统）
│   ├── themes/ plugins/       # lock 解析后下载的依赖
│   └── publish.lock
├── public/                    # 构建输出，gitignore，永不入仓库
└── .github/workflows/deploy.yml
```

---

## 10. Static / Dynamic 的统一方式

统一不是靠"写两套再抹平"，而是靠**四个单一来源**。任何一个破掉，两种模式立刻分叉。

1. **单一读模型**（[§0.2](#02-只有一个读模型read-model)）—— 最根本的一条
2. **单一渲染管线** —— `kite build` 和 `kite serve` 调用完全相同的 `Renderer`
3. **单一 URL 解析器** —— `URLResolver.For(content)`，主题内**禁止**手工拼接链接
4. **单一 API** —— Admin 在两种模式下调同一套 REST API

**Runtime 差异被收敛到三处，且只有三处：**

| 关注点 | Build Runtime | Serve Runtime |
|---|---|---|
| 数据来源 | 一次性全量加载为 SiteGraph | 按请求从 Reader 查询 |
| 产物 | 写入 `public/` | 写入 HTTP Response |
| 分页 / 聚合 | 构建期展开为静态页面 | 请求期计算 + 缓存 |

**请求态数据的唯一出口是 `.Request`**，文档化为"静态构建时为 nil"，主题只能用 `{{ with .Request }}` 访问。**绝不隐式暴露**——否则主题作者会不自觉地写出只能在 Dynamic 下工作的模板。详见 [theme-system.md](theme-system.md)。

---

## 11. Admin Studio

**技术栈**：React 19 + TypeScript + Vite + TanStack Query + Tailwind + shadcn/ui；产物 `//go:embed` 进二进制。

```
Studio (SPA)
 ├── API Client（由 OpenAPI 生成，禁止手写请求）
 ├── SchemaForm 渲染器  ← Theme Settings / ContentType Fields / Plugin Settings 共用
 ├── Editor（CodeMirror 6 + Markdown 扩展）
 ├── Preview（iframe → serve runtime 的预览路由）
 ├── Media Library
 └── Publish Panel（DeliveryState 可视化）
```

### A1. 预览必须由服务端渲染，前端不得二次实现 Markdown 渲染

如果前端用 `markdown-it` 渲染预览、构建用 Go 的 goldmark，两者在扩展语法、代码高亮、脚注、数学公式上必然不一致，会变成永远修不完的"预览和线上不一样"。

预览路由 `GET /_preview/:id` 走完整 Renderer + 当前主题。

### A2. Admin 只走公开 REST API

好处：Headless 用户、未来的 Cloud Admin、第三方客户端天然复用。
代价：Git 面板这类 Admin 专属能力也要设计成正式 API。**值得。**

### A3. Dev 模式

Vite dev server + proxy 到 `kite run`；Release 模式读 embed FS。用构建 tag 切换。

### A4. 编辑器路线

V1 只做 Markdown + 实时预览 + 图片拖拽上传 + 斜杠命令。**Block Editor 不是 V1 阻塞项**（[§32](#32-现在不要设计的东西)）。

---

## 12. Theme Engine（摘要）

> **完整设计见 [theme-system.md](theme-system.md)。** 这里只列对总体架构有约束的结论。

- **引擎**：Go `html/template`。决定性理由是 **contextual auto-escaping**——主题是第三方代码在渲染用户内容，这是 Go 里唯一按上下文正确转义的方案。`templ` 要编译，直接排除了可分发主题。
- **契约冻结时机：M5，不是 M0。** Hugo 在 v0.146 被迫重写整个模板系统并打断主题生态 `[EV]`。**没写过第二套主题就冻结的契约一定是错的。**
- **数据上下文用带方法的接口，不用导出字段的 struct**——方法可以新增、可以带警告地废弃，导出字段等于把内部内存布局永久冻结。
- **函数必须命名空间化**（`str.*` / `img.*` / `time.*` / `collections.*`）。Hugo 的扁平全局命名空间是永久的兼容性负债。
- **`kite theme verify` 从 V1 就要有**：同一个 fixture 站点在 build 与 serve 下渲染，逐字节 diff。**那个测试才是契约本身，文档不是。**
- `theme.yaml` 携带 `apiVersion` 与 `requires`，安装时**和**构建时都校验；未知 apiVersion 是硬拒绝。

---

## 13. Plugin System（摘要）

> **完整设计见 [plugin-system.md](plugin-system.md)。** 这里只列对总体架构有约束的结论。

- **运行时**：WebAssembly + wazero（纯 Go、零 CGO、可 embed）。**实现排在 M8。**
- **V1 必须做的只有三件事**（不引入 wazero 依赖）：
  1. **HookBus 落地，且 V1 的内置功能必须走它**——sitemap、RSS、代码高亮、图片处理全部注册为 Hook。**让内置功能成为"第一批插件"，是验证扩展点够不够用的唯一可靠办法**，成本约 200 行。
  2. Hook 的三个元数据现在就定：`Phase`（Build 必须纯 / Request）、`CacheKey()`、**形状必须是批量的**。
  3. `kite.lock` 预留 `plugins: [{id, version, sha256, capabilities}]` 字段。
- **所有 Hook 一律以文档为单位，绝不以 AST 节点为单位**——一篇 3000 字文章有上万个 inline 节点，节点级跨边界调用直接不可用。
- **Capability**（`build` / `runtime` / `client` / `admin`）决定插件在 Static 与 Dynamic 下的行为差异。
- **默认不开放完整 WASI**，权限走域名白名单 + 分组授权。

---

## 14. Build Engine

### 14.1 Pipeline

```
Load    从 Reader 加载全部内容与配置
  ↓
Resolve 解析 Taxonomy、关系、菜单、URL → SiteGraph
  ↓
Plan    生成 OutputTarget 列表（每个 URL 一个 target）+ 依赖边
  ↓
Render  并发渲染（worker pool）
  ↓
Post    Asset pipeline、fingerprint、minify、sitemap、RSS、search index
  ↓
Emit    原子写入 public/（先写 temp dir 再 swap）
```

### 14.2 现在必须做对的四件事（回填 = 重写）

Hugo 做了十年仍然没做对增量构建——bep 本人的 issue 原文：局部重建"有时太粗（拿不准就全量），有时又检测不到完整变更集"，大部分问题"可以用 cache buster 配置绕过，但很难配对" `[EV]`。`--disableFastRender` 的存在本身就是证据。Eleventy 则明确文档化了它的增量构建**不处理**数据文件变更、**不处理**分页级增量 `[EV]`。

所以：**V1 做全量构建，但下面四条现在就要做对。**

1. **构建纯度** `[已冻结]`
   `BuildContext` 是读取任何东西（文件、配置、env、时间）的**唯一通道**。`time.Now()` 冻结为单一构建时间戳，暴露为 `.Site.BuildTime`。模板和将来的插件**不允许**裸 I/O。
   **不可协商**——任何一个隐藏输入都会永久毒化可缓存性。

2. **顶层循环是「逐 OutputTarget」，不是 `renderSite()`** `[已冻结]`
   ```go
   for _, t := range plan.Targets {
       if cached(t) { continue }   // V1 里这个分支可以字面上就是 if false
       render(t)
   }
   ```
   V1 若写成一个单体渲染函数，增量就是一次重写。

3. **输出路径是内容记录的纯函数**，绝不由 map 迭代顺序决定。所有遍历显式排序。

4. **Taxonomy 页和 Paginator 页从第一天就是有独立缓存键的一等节点。**
   Eleventy 做不了分页增量，直接后果就是没把它们建模成节点。

### 14.3 依赖记录：投影级 hash

边在**渲染时记录**，不靠静态解析——Go 模板里数据驱动的 partial 名字静态分析看不见。把模板数据上下文包一层 recorder：每次 `.Site.Pages` / `partial "x"` / `.Params.y` 都登记一条边。

**缓存键：**

```
key = SHA256( build_abi_version ‖ config_hash ‖ theme_hash ‖ output_format ‖ locale
            ‖ sorted( (dep_node_id, dep_projection_hash)… ) )
```

**关键是 hash「投影」而不是「对象」**：列表模板只碰了每篇文章的 `{title, slug, date, summary}`，那么依赖 hash 就只覆盖这几个字段。于是**改一篇文章的正文不会让首页失效**。

这正是 Hugo 用 `.RelPermalink` 特判去绕的那一类问题——直接从数据模型解决。

**两个安全阀：**

- 缓存键里带 ABI 版本——缓存逻辑一改，全部失效
- `kite build --verify` = 冷构建 + 热构建 + 逐字节 diff，在 CI 跑

> **如果有一天你需要给用户提供 "cache buster" 配置，说明已经输了。** Hugo 的 cache buster 就是一次认输。

### 14.4 别高估增量构建的收益

Hugo 全量渲染数千页是个位数秒级；Go 模板管线跑 2k 篇文章全量构建应当远低于 2 秒，CI 里没人会在意。

**真正需要快的是 `kite serve` 的"敲键盘到看到预览"这一环**，而那里用内存渲染缓存就能解决，远早于需要持久化依赖图。

→ 所以 §14.2 的四条是"现在做"，真正的增量算法是"以后做"（M6+）。

---

## 15. Publisher

### 15.1 接口

```go
type Publisher interface {
    Name() string
    Preflight(ctx context.Context, cs ChangeSet) (*PublishPlan, error)  // 只检查，零副作用
    Apply(ctx context.Context, p *PublishPlan) (*PublishResult, error)
    State(ctx context.Context) (*DeliveryState, error)
}
```

**`Preflight` / `Apply` 两段式是刻意的**：Git 发布有大量可预见的失败（非快进、冲突、凭据缺失、工作区脏、平台限额），必须在动手前告诉用户，而不是失败到一半留下半提交状态。

### 15.2 DeliveryState

与 `Content.Status` 分离的**第二状态机**：

```go
type DeliveryState struct {
    Local     StepState  // 内容已写入文件/DB
    Committed StepState  // 已 commit（Git 模式）
    Pushed    StepState  // 已 push
    Deployed  StepState  // CI/CD 完成（通过 Actions API 或 webhook 回填）
    LastError *PublishError
}
```

对应后台展示的四个勾。**DB 模式下 Committed / Pushed 恒为 N/A，Deployed 在 Local 完成时即为 done**——同一个 UI 组件，不同的状态填充。

### 15.3 三个动作严格区分

| 动作 | 行为 | 副作用 |
|---|---|---|
| **Save Draft** | 只写 Store | 无 Git 操作，不上线 |
| **Preview** | 渲染到预览路由 | 无任何持久副作用 |
| **Publish** | Status → published + 触发 Publisher | 产生 commit/push 或 DB 状态变更 |

---

## 16. Git Workflow（最高危模块）

你在操作**别人的仓库**。下面每一种状态都是真实可达的。

| 失败模式 | 未处理的后果 |
|---|---|
| 非快进 push | 被拒；天真地 `--force` 会摧毁协作者的提交 |
| Detached HEAD | commit 落在无分支处，下次 checkout 即丢失 |
| 进行中的 merge / rebase / cherry-pick | 部分提交被拒或仓库损坏 |
| 工作区有**其他**未完成的文章 | `git add .` 会把半成品草稿一起发布 |
| 已有的暂存条目 | 你的 commit 静默包含了它们 |
| LFS 跟踪的媒体 | 提交原始 blob 而非 pointer → 仓库损坏 |
| `content/` 是 submodule | 你提交到了错误的仓库 |
| `.gitattributes` / CRLF | 整文件的伪 diff |
| 用户终端里的 `index.lock` | 发布中途随机失败 |
| credential helper / SSH agent | 发布在 HTTP handler 里挂死在 TTY 提示上 |
| 平台限额 | commit+push 之后才失败，无法回退 |

### 16.1 实现方式：调用 `git` 二进制

**不用 go-git 做任何写操作。**

决定性理由不是功能数量，而是**这个仓库属于用户**，配着他的 credential helper、他的签名密钥、他的 hooks、他的 `.gitattributes` filter、他的 LFS。自己实现 git 等于静默忽略这一切。

go-git 官方 `COMPATIBILITY.md` 的事实 `[EV]`：

- merge 仅支持 fast-forward；pull 同样只能 fast-forward
- rebase / stash / gc / fsck **不支持**
- `add` —— "除 plain add 外的任何 flag 都不支持"
- **LFS 不支持** —— 没有 clean/smudge filter，暂存一张 LFS 跟踪的图片会把**原始 blob** 写进本该是 pointer 的位置。对用户而言这就是仓库损坏。
- global / system config 只读
- **credential helper 至今不支持**（issue 开着、标着 `help wanted`、已 stale）——意味着每个用 `osxkeychain`、用 `gh auth`、用企业 SSO 的用户都得手动粘 PAT。既是体验倒退，也是你不想承担的密钥保管责任。
- 性能上还有 ~8× 内存、~4× 耗时的记录

→ `kite doctor` 检测 git 是否存在；不存在时 Static 模式仍可本地编辑，只提示"请自行提交"。

### 16.2 Preflight 闸门（大声拒绝，绝不猜）

```bash
git rev-parse --git-dir                         # 是不是仓库
git rev-parse --show-superproject-working-tree  # content/ 是 submodule？拒绝或改投目标
git symbolic-ref -q HEAD                        # detached HEAD？提示建分支
test -e $GIT_DIR/{MERGE_HEAD,REBASE_HEAD,CHERRY_PICK_HEAD,BISECT_LOG}   # 进行中的操作？拒绝
git check-attr filter -- <每个媒体路径>          # filter=lfs 且没装 git-lfs？拒绝
```

### 16.3 暂存：绝不 `git add .`，绝不 `git add -A`

**主路径 —— 用 Git 自己的部分提交：**

```bash
git commit --only -- <ChangeSet 里的精确路径>
```

这正是想要的语义：

- **只提交列出的路径**
- **用户已暂存的其他条目和所有其他脏文件原封不动**
- 用户的 clean filter、`.gitattributes`、LFS **全部正常生效**
- hooks 正常触发

唯一限制是 merge 进行中会拒绝，而这已经被 Preflight 挡掉了。

**逃生舱 —— 需要跳过 hooks 或完全不碰真实 index 时：**

```bash
GIT_INDEX_FILE=$GIT_DIR/kite.tmpindex git read-tree HEAD
GIT_INDEX_FILE=$GIT_DIR/kite.tmpindex git add -- <paths>
TREE=$(GIT_INDEX_FILE=$GIT_DIR/kite.tmpindex git write-tree)
NEW=$(git commit-tree $TREE -p $OLD -m "$msg")
git update-ref -m "kite: publish" refs/heads/<branch> $NEW $OLD   # CAS
git update-index --add --remove -- <paths>
rm -f $GIT_DIR/kite.tmpindex
```

`update-ref` 的 `$OLD` 参数是 **compare-and-swap**——这是防"用户在终端里同时提交"的并发保护。

另外：当 `git diff --name-only --cached -- <paths>` 非空时**要警告而不是静默覆盖**，因为用户对你即将提交的路径有不同的暂存内容。

### 16.4 凭据：根本不要碰

用用户的环境运行 `git`，让他的 credential helper 和 SSH agent 正常工作。同时：

```
GIT_TERMINAL_PROMPT=0
GIT_ASKPASS=<指向一个空操作>
```

**这样缺凭据时会立刻报一个可捕获的错误，而不是让一个 HTTP handler 永远挂在 TTY 提示上。** 这一个细节就避免了服务端进程最糟糕的一类 bug。

### 16.5 Front matter 保真（被低估的头号口碑杀手）

**只重写真正变化的 key，保留 key 顺序、注释、以及 flow / block 风格。**

一次朴素的 YAML unmarshal → marshal 往返会把 `tags: [a, b]` 变成块状列表，生成巨大的无意义 diff。

> **"这个 CMS 把我的文件搞乱了"是每一个 Git-backed CMS 的头号投诉。**

换行符同理：只写 `\n`，剩下的交给 git filter，并附带推荐的 `.gitattributes`（`*.md text eol=lf`）。

对应的验收用例见 [§29 M3](#29-每阶段验收标准)。

### 16.6 推送策略

默认普通 push，**永不 `--force`**。

非快进被拒时**不自动合并**：

1. `git fetch`
2. 计算远端新提交是否触碰 ChangeSet 里的路径
3. **没有重叠**且你的发布提交是唯一领先的 → 提供一键 `git rebase --onto`
4. **有重叠** → 展示 diff 并交还用户

`--force-with-lease --force-if-includes` 只保留给用户明确确认的"重新发布 / 修补"动作。

### 16.7 锁与限额

- 应用级 advisory lock：`.kite/publish.lock`，保证两个 Kite 操作不重叠
- 每次 git 调用遇到 `index.lock` 用带抖动的重试（5 次，50→800ms）。**永远不要自动删除陈旧的锁**
- **平台限额在发布前检查，别等 push 完才失败** `[EV]`：
  - GitHub Pages：站点 ≤1GB、部署超时 10 分钟、约 10 次/小时软限、100GB/月软带宽
  - Cloudflare Pages：20,000 文件（免费）/ 100,000（付费）、单文件 ≤25MiB、构建超时 20 分钟

### 16.8 Commit Message

默认模板可配置：

```
publish: <title>
content: update <slug>
media: add assets for <slug>
```

---

## 17. CI/CD

```
Admin Publish → GitPublisher → push → GitHub
   → CI（setup-kite 或 ./kitew）→ kite build → public/ → Pages/CDN
   → 部署完成事件回填 DeliveryState.Deployed
```

**仓库只存 Source，`public/` 永不入库。** CI 产出物作为 Deployment Artifact。

**V1 只做 GitHub Pages**，而且只生成一个 GitHub Actions workflow 文件，不做任何 API 集成。

Cloudflare Pages 放到 M6 和 `kitew` 一起做——因为 CF Pages 的构建容器没法预装 Kite，正是 wrapper 的用武之地。

### `kitew`（Gradle Wrapper 思路）

```
kitew            # POSIX
kitew.ps1        # Windows
.kite/version    # 锁定的 Kite 版本
```

流程：读版本 → 下载对应平台二进制 → **强制校验 checksum（不可跳过）** → 缓存 → exec。

价值：CI 与新机器零安装、版本随仓库走、`setup-kite` Action 不可用时（GitLab / 自建 CI / Cloudflare Pages 构建容器）仍然可用。

---

## 18. Deployment Model

| 模式 | 组成 | 场景 | 里程碑 |
|---|---|---|---|
| Static | Git 仓库 + CI + 静态托管 | S1 / S4 | M4（v1.0） |
| Dynamic 单机 | 单二进制 + SQLite + uploads/themes/plugins | S3 | M7（v2.0） |
| Dynamic 容器 | Docker / Compose | S3 | M7 |
| Dynamic 集群 | K8s + PG + S3 | 团队 | V4 |
| Headless | API Server + 外部前端 | S5 | M7 之后 |

**Single Binary 是硬性目标**：Admin 前端、默认主题、迁移脚本全部 embed。下载即用，不依赖运行时环境。

SQLite 用 **`modernc.org/sqlite`**（纯 Go，零 CGO，交叉编译不破，且默认编入 FTS5）`[EV]`。

---

## 19. API

- 路径 `/api/v1/...`，**Admin 与 Headless 同源**
- 规范：OpenAPI 3.1。由代码生成 spec，再由 spec 生成前端 client——**禁止手写请求**
- 资源：`contents` / `taxonomies` / `media` / `content-types` / `themes` / `plugins` / `settings` / `builds` / `publishes` / `users`
- **分页一律复合游标**（[§9.1](#91-读侧一个读模型一个实现)），**公开 API 里不允许出现 offset**
- 并发控制：`ETag` + `If-Match`（[§9.4](#94-并发编辑--cas绝不-last-write-wins)）
- 认证：V1 本地单用户（本地 token / session）；M7 多用户；V3 API Token + Scope；V4 OAuth
- 版本策略：v1 内**只增不改**；破坏性变更开 v2 并保留 v1 至少一个大版本
- Webhook（V3）：内容变更、构建完成、发布完成

---

## 20. Media System

### 20.1 两种布局的取舍

| | **Page Bundle**<br>`content/posts/foo/cover.webp` | **集中式**<br>`static/uploads/2026/09/cover.webp` |
|---|---|---|
| 内容自包含 | ✅ 删文章即删资源，整篇可复制迁移 | ❌ 资源散落，易产生孤儿文件 |
| Markdown 可移植性 | ✅ 相对路径 `![](cover.webp)`，GitHub / VS Code 直接可见 | ❌ 需绝对路径，编辑器预览常挂 |
| 跨文章复用 | ❌ 困难，会产生副本 | ✅ 天然复用 |
| 媒体库管理 | ❌ 需跨目录聚合（索引可解决） | ✅ 目录即媒体库 |
| 大量图片时的目录体积 | ⚠️ 单目录膨胀 | ✅ 可按日期分片 |
| CDN / 对象存储迁移 | ⚠️ 路径改写成本高 | ✅ 前缀替换即可 |

**决策：V1 默认 Page Bundle，同时支持集中式，由 ContentType 的 `Layout` 与站点配置决定。**

理由：默认值要服务 P0 的两类用户——Page Bundle 在 GitHub 网页、VS Code 预览、整篇迁移时体验明显更好；而"跨文章复用"和"媒体库聚合"可以由派生索引补齐。

### 20.2 分期

**V1 的媒体功能只有一件事：拖入图片 → 拷进 page bundle → 插入路径。其余全部推迟。**

"完整媒体库"（上传 + 缩放 + EXIF 清理 + srcset + 存储抽象 + 孤儿回收 + LFS 交互）本身是一个子产品，硬塞进 V1 会拖垮进度。

### 20.3 MediaService（接口先定，驱动后补）

- 驱动：`LocalStorage`（V1）/ `S3` / `R2` / `OSS`（V3）
- 处理链：上传 → 校验（类型/大小）→ 提取元数据 → 生成缩略图 → WebP/AVIF 转换 → 写入 → 索引
- **引用追踪**：保存内容时解析 Markdown 与 Meta 中的资源引用，写入 `media_reference` 表 → 支持"未被引用的媒体"清理与"删除前警告"
- CDN URL 由 `URLResolver` 统一产出，主题不拼路径

---

## 21. Cache / Task / Event

### Cache —— 三层，不要混

1. **Build Cache**（`.kite/cache/`）：渲染中间产物与依赖图，跨构建复用
2. **Runtime Cache**（Serve 模式）：内存 LRU + 可选 Redis，页面片段与查询结果
3. **HTTP Cache**：ETag / Last-Modified / Cache-Control；带 fingerprint 的静态资源可 immutable

### Task Queue

本地内嵌队列（SQLite 持久化）。用途：publish、图片处理、索引重建、CI 状态轮询。

要求：可重试、可观测（Admin 有任务列表）、进程重启不丢。

**不引入外部 MQ 依赖**——违反 Single Binary。

### Event Bus

进程内，同步/异步双模式，是 HookBus 的底座。Core 内部的解耦也走它（如"内容保存后刷新索引"）。

---

## 22. CLI

```
kite init [--template <starter>]      # 初始化项目
kite new <kind> "<title>"             # 新建内容
kite run                              # 开发主入口：serve + admin + watch + 打开浏览器
kite build [--force] [--verify] [--drafts]
kite preview                          # 预览 public/
kite serve                            # 生产动态服务
kite publish [--message]
kite index [--rebuild]                # 重建派生索引
kite theme <list|add|remove|new|verify>
kite plugin <list|add|remove|new>     # M8
kite lock <update|verify>             # M6
kite migrate <file-to-sql|sql-to-file>  # M7
kite export                           # 导出 Markdown + Media + Config
kite doctor                           # 环境/配置/主题/Git/重复 ID 体检
kite version
```

**`kite run` 是产品的第一体验**，必须做到：一条命令 → 浏览器自动打开 `/admin` → 立刻可以写作。在空目录里跑 `kite run` 应当**引导初始化**，而不是报错。

设计原则：Simple / Predictable / Scriptable —— 所有命令支持 `--json` 输出。

---

## 23. Configuration

`kite.yaml`（人写，入 Git）：

```yaml
site:
  title: My Blog
  baseURL: https://example.com
  language: zh-CN

content:
  store: file            # file | sqlite | postgres | mysql
  dir: content

theme:
  name: paper
  settings:
    primary_color: "#2563eb"
    show_toc: true

plugins:
  - seo
  - mermaid

publish:
  publisher: git         # git | database | webhook
  git:
    branch: main
    commitMessage: "publish: {{.Title}}"

build:
  output: public
  minify: true
```

**优先级**：CLI flag > 环境变量（`KITE_*`）> `kite.yaml` > 内置默认值。

**约束：**

- 配置是 Core 的输入，**Core 之外的模块不得直接读文件**，一律通过 `ConfigService`
- 敏感值（S3 密钥、Token）**不写进 `kite.yaml`**，走环境变量或 `.kite/secrets`（gitignore）

---

## 24. kite.lock（M6）

```yaml
lockfileVersion: 1
kite: 1.0.0
theme:
  paper:
    version: 1.2.0
    resolved: https://github.com/kite-plus/themes/releases/download/paper-1.2.0/paper.tar.gz
    checksum: sha256-xxxx
plugins:
  seo:
    version: 1.0.0
    resolved: ...
    checksum: sha256-xxxx
    capabilities: [build]
```

- 入 Git。`kite build` 默认按 lock 安装，**checksum 不匹配直接失败**
- `kite lock update` 显式更新
- 保证 Local Build = CI Build = Production Build

**注意分期**：V1 只需要一个 `.kite/version` 文件加 30 行 shell wrapper。完整的 lock 机制只有在存在多个版本、多个主题/插件时才有意义——提前做等于浪费。

---

## 25. 项目目录结构

```
kite/
├── cmd/kite/                  # CLI 入口
├── internal/
│   ├── content/               # ← 领域核心，不允许 import 下面任何一个
│   │   ├── content.go         # Content / ContentType Registry
│   │   ├── query.go           # ContentQuery / SortKey / 游标编码 / Capabilities
│   │   └── changeset.go       # ChangeSet / Op / ContentWriter.Apply / Revision
│   ├── index/
│   │   ├── schema.sql         # 唯一读模型 schema
│   │   └── indexer.go         # stat+hash 一致性、racy-git 规则、HEAD 哨兵
│   ├── reader/                # 唯一的 ContentReader 实现
│   ├── store/
│   │   ├── file/              # FileWriter：原子写、front matter 保真编解码
│   │   └── sql/               # SQLWriter（M7）
│   ├── render/
│   │   ├── markdown/          # goldmark pipeline
│   │   ├── theme/             # contract.go：查找顺序 / 上下文接口 / funcmap
│   │   └── url/               # URLResolver
│   ├── build/
│   │   ├── context.go         # BuildContext：唯一 I/O 通道、冻结时钟、依赖记录
│   │   ├── plan.go            # OutputTarget + 缓存键
│   │   └── emit.go
│   ├── serve/                 # HTTP router / middleware / preview
│   ├── api/                   # REST handlers + OpenAPI
│   ├── publish/
│   │   ├── git/               # preflight / commit --only / CAS / 推送策略
│   │   └── database/
│   ├── media/
│   ├── hook/                  # HookBus：Phase / CacheKey / 批量形状
│   ├── task/                  # 内嵌队列
│   └── plugin/                # M8：wazero runtime + host api
├── web/                       # React Admin（pnpm）
│   └── dist/                  # 构建产物，go:embed
├── themes/default/            # 内置默认主题，embed
├── docs/design/               # 本设计文档
├── scripts/
└── .github/workflows/
```

**强制的架构守护**：CI 中加一条 import 规则检查（`go-arch-lint` 或自写脚本），**禁止 `internal/content` 与 `internal/hook` import 任何 `store|render|serve|publish|plugin`**。

> 这条规则比任何文档都有效。

---

## 26. GitHub Organization 规划

**`kite-plus/kite` 只放核心主程序，其余分仓。**

| 仓库 | 内容 | 何时建 |
|---|---|---|
| **kite-plus/kite** | Go Core + CLI + Build + Serve + API + **React Admin（`web/`）** + 默认主题 | 现在 |
| kite-plus/explore | 技术预研、原型、设计文档存档 | 现在 |
| kite-plus/website | kite.plus 官网 + 文档（**用 Kite 自己搭 —— dogfooding**） | M4 |
| kite-plus/starters | Starter 模板（blog / docs / portfolio） | M4 |
| kite-plus/setup-kite | GitHub Action | M4 |
| kite-plus/themes | 官方主题集合 | M5 |
| kite-plus/theme-sdk | 主题开发工具链 + 脚手架 | M5 |
| kite-plus/plugin-sdk | 插件 SDK（Go / Rust / Zig） | M8 |
| kite-plus/plugins | 官方插件集合 | M8 |

**一个例外说明**：React Admin 放在 `kite-plus/kite` 内而非独立仓库——它必须 `go:embed` 进二进制、与 Go 版本强绑定，拆仓会立刻产生版本对齐地狱。**它是"主程序"的一部分，不是生态组件。**

**建仓时机原则：有第一个真实用户之前不要建仓。** 空仓库是维护负担。

---

## 27. MVP 范围

MVP（= M0~M4）要回答唯一的问题：

> **Kite 是否真的能做到「WordPress 的写作体验 + Hugo 的部署体验」？**

**包含**：File Store + 派生索引 + Markdown 渲染 + post/page + tag/category + 拖图进 bundle + 一套内置主题 + React Admin + 服务端实时预览 + 409 冲突流程 + 全量 `kite build` + `kite serve` + GitPublisher + DeliveryState + GitHub Actions workflow 模板 + `kite doctor`。

**不包含**：Dynamic 生产模式、SQLite 内容库、插件运行时、公开的主题契约、完整媒体库、`kite.lock` / `kitew`、Cloudflare Pages、多用户、搜索、评论、对象存储、自定义内容类型注册、增量构建、i18n。

---

## 28. Roadmap

### 三处对"直觉版本序"的修正

**修正一：Dynamic Serve 不放 V1，推到 M7。**

同时做 Static Build 和 Dynamic Serve 意味着 SQLite schema、动态路由、认证、会话、CSRF、限流、备份、迁移必须同期定型。更要命的是：**DB 模型一旦先固化，很容易反过来污染文件模型**（例如为了 DB 方便把 taxonomy 拍平成外键表——而 [§8 D3](#d3-taxonomy-降级为内容上的字符串列表--派生投影不是实体-已冻结) 正好要求相反）。

**这不影响"三种模式现在就要在架构上考虑"**：正交矩阵、唯一读模型、URLResolver、`.Request` 的 nil 语义全部按三模式设计，只是实现排期靠后。而且 M1 的 `serve × files` 会**免费验证**这个抽象。

**修正二：主题契约的公开推迟到 M5。**

Hugo 在 v0.146 不得不重写整个模板系统并打断主题生态 `[EV]`。**没写过第二套主题就冻结的契约一定是错的。**

**修正三：Plugin Runtime 推到 M8，中间插入 M5/M6。**

主题生态比插件更紧急——主题是用户第一眼看到的东西，插件是纯技术投入。**先有主题生态，插件才有被需要的场景。** 但 [§13](#13-plugin-system摘要) 的三件事 V1 就做，成本约 200 行。

### 里程碑序列

| # | 范围 | 估时 | Demo / 意义 |
|---|---|---|---|
| **M0** | `kite build` 纯 CLI：`content/**.md` + 一套内置主题 → `public/`。front matter、tag/category、taxonomy 页、分页、RSS、sitemap。无 Admin、无 Git、无 DB | 3~4 周 | "把一个现成的 Hugo 内容目录指给它，产出站点"。在零 UI 噪音下逼出模板上下文与查找顺序的决策 |
| **M1** | `kite serve`：按请求从文件渲染 + watcher + live reload | 2 周 | 即时预览。**这就是 `serve × files` 格子——正交轴主张免费被验证** |
| **M2** | Admin **只读**：React + `go:embed`、`.kite/cache/index.db`、列表/过滤/排序/分页/搜索、在带外编辑下保持一致 | 3~4 周 | "用一个真正的 UI 打开你现有的 Hugo 博客"。**在押上任何东西之前，先证明全项目最难的技术主张** |
| **M3** | Admin **可写**：Markdown 编辑器、schema 驱动的 front matter 表单、增删改、原子写、409 三方冲突流、拖图进 bundle。仍然没有 Git | 4~5 周 | 完整本地写作。此时已可配用户自己的 git 工作流使用 |
| **M4** | **Git Publisher**：ChangeSet → `git commit --only` → push，[§16](#16-git-workflow最高危模块) 的全部闸门 + 生成 GitHub Actions workflow | 3~4 周 | 浏览器里编辑 → 线上站点。**这就是 v1.0**：一个连贯的产品——"给你的静态站配一个真正的后台" |
| **M5** | **主题契约 v1**：`apiVersion` + `requires`、settings schema → Admin 配置页、**第二套主题**、`kite theme verify` 黄金文件测试 | 4~5 周 | v1.1。第三方可以写主题了 |
| **M6** | `kite.lock` + `kitew` + Cloudflare Pages + 增量构建起步 | 2~3 周 | v1.2。可重现的 CI 构建 |
| **M7** | **Dynamic 模式**：SQLite 作真相源、认证、单二进制服务、`kite migrate` | 8~12 周 | v2.0。主要是"第二个 writer + 认证"，**不是重写**，因为读模型和主题契约已经验证过了 |
| **M8** | wazero 插件，跑在已有的 HookBus 上 | — | v3.0 |

**估时按单人专注投入给出，含测试与文档。到 v1.0 约 15~19 周（4~5 个月）。**

> **这个排序最关键的性质**：**M2（索引一致性）和 M4（发布器）是业界还没有人把它们同时做好的两件事。** 它们排在主题契约冻结之前、第二个 Store 之前——所以如果哪一个的判断是错的，你在第 2 个月就知道，而不是第 14 个月。

---

## 29. 每阶段验收标准

**M0**
- 在一个真实的 Hugo 内容仓库上 `kite build` 产出可用站点
- 2000 篇全量构建 < 2s
- `kite build --verify`（冷/热构建逐字节 diff）在 CI 通过
- `internal/content` 的 import 边界检查通过

**M1**
- 改一个 Markdown 文件，浏览器 < 500ms 反映变化
- 同一个 fixture 站点在 build 与 serve 下的 HTML 输出**逐字节一致**

**M2**
- 打开一个 2000 篇的真实仓库，列表首屏 < 300ms
- `rm -rf .kite/cache && kite index` 产出的 DB dump 与之前逐字节一致
- Admin 开着时 `git checkout` 另一个分支，3 秒内列表正确反映新分支内容，**且只重索引变化的路径**

**M3**
- **不打开终端**完成：新建文章 → 正文 → 拖入两张图 → 设分类标签 → 保存 → 实时预览与最终渲染逐字节一致
- VS Code 改同一文件并保存，Admin 3 秒内反映
- 在陈旧版本上保存得到 409 与三方 diff
- **保存一篇只改了标题的文章，`git diff` 只显示 title 那一行**（front matter 保真）

**M4**
- **端到端零终端**：点「发布」→ 5 分钟内 GitHub Pages 可见
- `git log` 显示本次 commit **只包含**该文章的文件
- 发布前手工修改另一篇文章并 `git add` 另一个文件，发布后这些改动**原封不动**
- 远端有新提交时被 Preflight 拦截并给出选项，**不产生任何半完成状态**
- 没有配置凭据时**立刻报错而不是挂起**

**M5**
- 第三方开发者仅凭文档 2 小时内做出可用主题
- 主题在 Admin 中自动生成配置页
- `kite theme verify` 对两套官方主题在 build/serve 下的输出零 diff

**M7**
- 同一份内容、同一套主题，Static 产物与 Dynamic 渲染输出在结构与链接上一致（自动化 diff）
- `kite migrate` 往返后内容 hash 一致

**M8**
- 第三方插件按声明的 Capability 在两种模式下正确工作
- 越权调用被拒绝并有审计记录
- 构建期插件参与缓存键，改插件版本触发正确的重建

---

## 30. 核心技术风险

按"发生概率 × 修复代价"排序。

| # | 风险 | 后果 | 缓解 |
|---|---|---|---|
| **R1** | 派生索引与文件不一致 | Admin 显示错内容、发布错内容——**信任崩塌** | 三层一致性（[§9.3](#93-派生索引的一致性static-模式最容易出-bug-的地方)）+ racy-git 规则 + "索引可重建"不变量写进 CI |
| **R2** | Git 自动化破坏用户工作 | 丢失未提交工作——**不可挽回，口碑一次性死亡** | `git commit --only`（[§16.3](#163-暂存绝不-git-add-绝不-git-add--a)）+ Preflight 闸门 + `update-ref` CAS + 绝不 force |
| **R3** | Front matter 被搅乱 | 巨大无意义 diff，"这个 CMS 把我文件搞乱了" | 只改动真正变化的 key（[§16.5](#165-front-matter-保真被低估的头号口碑杀手)），M3 验收里有专门用例 |
| **R4** | 主题契约冻结过早 | v1.1 就是破坏性发布 | 推迟到 M5；带方法的接口；`apiVersion`/`requires` |
| **R5** | Store 抽象泄漏 | 加 DB 实现时被迫重写 Service 层 | 唯一读模型 + `Apply(ChangeSet)` 单一写入口 + Taxonomy 降级为投影 |
| **R6** | 增量构建漏构建 | 用户看到过期页面，**比慢更致命** | BuildContext 纯度 + 逐 target 循环 + 投影级依赖 hash + `--verify` 在 CI |
| **R7** | WASM 插件性能 / ABI | 节点级 Hook 不可用；wazero 在非 amd64/arm64 平台 panic | 所有 Hook 批量形状；实例池化 + 编译缓存；能力检测降级解释器 |
| **R8** | Single Binary 体积膨胀 | Admin + 主题 + wazero embed 后过大 | 体积预算（目标 < 50MB）；主题按需下载而非全量 embed |
| **R9** | CGO 破坏交叉编译 | 发不出全平台二进制 | SQLite 用 `modernc.org/sqlite` |
| **R10** | 分页用 offset | 公开 API 上改不掉，且可变集合必然重复/漏行 | 第一天就用复合游标 |

---

## 31. 必须现在确定的架构决策

**这些如果现在不定或定错，12~18 个月后是重写级代价。**

### Top 5（真正的重写风险）

| # | 决策 | 一句话结论 |
|---|---|---|
| **1** | **跨两种 Store 的统一读模型** | 一份 SQL 读模型 schema、**恰好一个 `ContentReader` 实现**；Static 模式由 indexer 写行，Dynamic 模式由 writer 写行 |
| **2** | **内容身份** | 第一个 commit 起就在 front matter 里放 ULID/UUIDv7 `id`，path / slug / URL 三者与它、以及彼此之间全部解耦——**事后往上千个文件里回填 ID 是用户会拒绝的迁移** |
| **3** | **写入接缝** | 唯一写 API 是 `Apply(ctx, ChangeSet) (Revision, error)` + 类型化 Op，**永不做字段级 mutator** |
| **4** | **构建纯度** | 所有 I/O 走 `BuildContext`（冻结时钟，模板/插件禁止裸 `os.ReadFile`），顶层循环逐「带缓存键的 OutputTarget」，依赖边在渲染期以**投影粒度**记录——哪怕 V1 的 skip 判断字面上就是 `if false` |
| **5** | **主题契约的形状与时机** | 带方法的接口 + 命名空间化 funcmap + `apiVersion`/`requires` + `kite theme verify` 黄金文件测试，**而且不到 M5 不公开** |

### 同样现在要定，但影响范围限于单个 package

6. Git 走二进制而非 go-git；主路径 `git commit --only`
7. HookBus day-one，内置功能吃自己的狗粮
8. Taxonomy 是投影不是实体
9. 分页用复合游标不用 offset
10. `ErrUnsupportedQuery` 永不静默退化为全量加载
11. Admin 只走公开 REST API
12. CI 强制 `internal/content` 的 import 边界

---

## 32. 现在不要设计的东西

**判断标准：如果将来做时只需新增代码、不需修改已有接口，那就现在不要做。反之必须现在做。**

| 不要做 | 现在只需要 | 何时 |
|---|---|---|
| WASM ABI 实现、wazero 依赖、SDK、插件注册表 | 冻结 ABI 常量 + HookBus + lock 字段 | M8 |
| 完整媒体库（缩放/EXIF/srcset/孤儿回收/对象存储） | 拖图进 bundle，`MediaStore` 接口定义 | V1.5 / V3 |
| **主题继承 / 子主题** | —— | **永远不要在 v1 发布，极难收回** |
| SCSS / PostCSS 工具链 | 纯 CSS + 拷贝 + 可选 fingerprint | M5 之后 |
| Block Editor / 富文本 | Markdown + 实时预览 | V2 之后评估 |
| 多站点 / 多租户 | 单站点，但配置不写死全局单例 | V4 |
| 细粒度 RBAC | 单用户；权限检查点预留 | M7+ |
| GraphQL | REST + OpenAPI | 有真实需求再说 |
| 实时协作 / CRDT | 乐观锁 + 409 三方冲突 | V4+ |
| Marketplace / Kite Cloud | Admin 走公开 API，为复用留路 | V4 |
| 增量构建算法、磁盘持久化依赖图、并行调度 | 只记录依赖 | M6+ |
| PostgreSQL / MySQL | 接口按多方言设计，只实现 SQLite | V3 |
| FTS5 全文检索 | 2k 篇以下用 `LIKE`/`instr`（索引是派生的，schema 随时能改） | 需要时 |
| 编辑流 / PR-per-draft、多作者锁、定时发布 | —— | V3+ |
| i18n 完整方案 | 函数、目录约定、**URL 策略**先定，实现单语言 | M5 |

---

## 33. 从 0 到 v1.0 的开发顺序

### M0（3~4 周）

1. 工程基建：Go module、目录骨架、lint、CI、**import 边界检查**、goreleaser
2. `internal/content`：Content、ContentType Registry（内置 post/page）、`query.go`（Query 规格 + 复合游标）、`changeset.go`（ChangeSet + Op + Revision）
3. **Front matter 保真编解码器**（保留 key 顺序/注释/风格）—— 这是 R3 的根
4. `FileWriter`：原子写（temp + fsync + rename + fsync dir）、ID 生成
5. `index/schema.sql` + indexer：stat+hash、racy-git 规则、**重复 ID 硬错误**
6. 唯一的 `ContentReader` 实现
7. Markdown Pipeline（goldmark + GFM / 脚注 / 高亮 / TOC），**裸 HTML 策略现在定**
8. `URLResolver` + RenderContext（**带方法的接口**）
9. Theme Engine v0：查找顺序 + 命名空间 funcmap + 内置主题
10. `build/context.go`（BuildContext + 冻结时钟 + 依赖记录）、`plan.go`（逐 OutputTarget + 缓存键）、taxonomy/paginator 作为一等节点
11. HookBus，**sitemap / RSS / 代码高亮全部注册为 Hook**
12. `kite build` + `kite build --verify`

### M1（2 周）

13. `serve` runtime：路由、按请求渲染、watcher + live reload
14. build / serve 输出一致性的自动化 diff 测试

### M2（3~4 周）

15. REST API + OpenAPI（只读部分）
16. React 工程 + embed 管道 + dev proxy
17. 列表 / 过滤 / 排序 / 游标分页 / 搜索
18. 三层索引一致性（fsnotify / stat walk / **`.git` HEAD 哨兵 + `git diff --name-status`**）
19. "索引可重建"不变量的 CI 测试

### M3（4~5 周）

20. 写侧 API：`PUT` + `If-Match` → `Apply(ChangeSet)`
21. SchemaForm 渲染器（三处复用）
22. CodeMirror 编辑器 + **服务端**实时预览
23. 409 三方冲突 UI
24. 拖图 → 拷进 page bundle → 插入相对路径
25. 站点设置 + 主题设置页（由 `theme.yaml` 自动生成）
26. `kite doctor`：补 ID、查重复 ID、查环境

### M4（3~4 周）

27. ChangeSet → Git：Preflight 闸门全套
28. `git commit --only` 主路径 + 临时 index 逃生舱 + `update-ref` CAS
29. `GIT_TERMINAL_PROMPT=0` + `GIT_ASKPASS` 空操作
30. 推送策略（非快进处理、绝不 force）+ `.kite/publish.lock` + `index.lock` 重试
31. 平台限额 Preflight（GitHub Pages）
32. DeliveryState + Publish Panel
33. GitHub Actions workflow 模板 + `kite init` + starters
34. 打磨：错误信息、首次运行引导、用 Kite 自己搭文档站

### 贯穿全程的纪律

- **`Apply(ChangeSet)` 是唯一写入口**——任何一次"就加一个 setter 吧"的妥协都要在 review 里打回
- **每个里程碑结束录一个 2 分钟 Demo**——检验"是否真的可用"最诚实的方式
- 写 Go 代码时应用 Modern Go Guidelines（Go 1.26）

---

## 34. 证据来源

文中 `[EV]` 标注的结论出处：

**文件真相源 / 索引**
[TinaCMS Data Layer](https://tina.io/docs/reference/content-api/data-layer) ·
[Grav 大站性能 #931](https://github.com/getgrav/grav/issues/931) ·
[kirby3-boost](https://github.com/bnomei/kirby3-boost) ·
[Kirby UUIDs](https://getkirby.com/docs/guide/uuids) ·
[fsnotify 原子保存丢 watch #372](https://github.com/fsnotify/fsnotify/issues/372) ·
[fsnotify #255](https://github.com/fsnotify/fsnotify/issues/255) ·
[Git racy-git](https://github.com/git/git/blob/master/Documentation/technical/racy-git.adoc) ·
[Decap 编辑流限制](https://decapcms.org/docs/editorial-workflows/)

**主题契约**
[Hugo 模板系统重写 PR #13541](https://github.com/gohugoio/hugo/pull/13541) ·
[Hugo 新模板系统概览](https://gohugo.io/templates/new-templatesystem-overview/) ·
[Docsy 被打断 #2243](https://github.com/google/docsy/issues/2243) ·
[Hugo fingerprint/purge 缺陷 #11268](https://github.com/gohugoio/hugo/issues/11268) ·
[Halo 主题开发](https://docs.halo.run/developer-guide/theme/prepare)

**增量构建**
[Hugo 局部重建 #11543](https://github.com/gohugoio/hugo/issues/11543) ·
[Hugo #3325](https://github.com/gohugoio/hugo/issues/3325) ·
[Eleventy incremental 的明确缺口](https://www.11ty.dev/docs/usage/incremental/) ·
[Hugo 依赖追踪](https://deepwiki.com/gohugoio/hugo/3.6-dependency-tracking-and-caching)

**Git**
[go-git COMPATIBILITY.md](https://github.com/go-git/go-git/blob/master/COMPATIBILITY.md) ·
[go-git credential helper #1420](https://github.com/go-git/go-git/issues/1420) ·
[git 环境变量](https://git-scm.com/docs/git#_environment_variables) ·
[force-with-lease](https://thoughtbot.com/blog/git-push-force-with-lease) ·
[GitHub Pages 限额](https://docs.github.com/en/pages/getting-started-with-github-pages/github-pages-limits) ·
[Cloudflare Pages 限额](https://developers.cloudflare.com/pages/platform/limits/)

**WASM** —— 详见 [plugin-system.md](plugin-system.md#证据来源)

**SQLite**
[modernc.org/sqlite（纯 Go）](https://pkg.go.dev/modernc.org/sqlite)
