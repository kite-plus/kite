# Kite 主题系统设计

> 状态：设计中，契约计划于 **M5** 冻结 · 最近更新：2026-09-21
> 上级文档：[architecture.md](architecture.md) · 姊妹文档：[plugin-system.md](plugin-system.md)
> `[EV]` 标记的结论有既有项目的实证支撑，来源见 [证据来源](#证据来源)。

---

## 1. 目标与非目标

### 目标

1. **主题可分发、免编译** —— 用户下载一个目录就能用，不需要 Go 工具链。
2. **同一套主题在 Build 与 Serve 两种 Runtime 下产出一致** —— 这是 Kite "统一 Static 与 Dynamic" 主张在渲染层的落点。
3. **主题开发者不写任何 Admin UI** —— `theme.yaml` 声明 settings schema，Admin 自动生成配置页。这是 Kite 相对 Hugo 的核心体验优势。
4. **契约稳定性优先于内部实现优雅** —— 一旦第三方主题存在，契约就是永久负债，内部实现可以随便重构。

### 非目标

- 不做可视化主题编辑器 / 拖拽布局
- 不支持主题里嵌入 Go 代码或原生插件（扩展走 [plugin-system.md](plugin-system.md)）
- **不做主题继承 / 子主题** —— 见 [§11.3](#113-明确不做的东西)
- v1 不提供 SCSS / PostCSS 工具链

---

## 2. 引擎选型

### 结论：Go 标准库 `html/template` `[已冻结]`

决定性理由**不是**生态或性能，而是 **contextual auto-escaping**。

主题是**第三方代码在渲染用户内容**。`html/template` 是 Go 里唯一按上下文（HTML body / 属性 / JS / URL / CSS）自动选择正确转义规则的模板引擎。这在一个要发展主题市场的产品里不是"加分项"，是**前提条件**。

### 被排除的方案

| 方案 | 排除理由 |
|---|---|
| `templ` | **需要编译**。主题必须先编译成 Go 代码再链接进二进制——直接排除了"下载一个主题目录就能用"和主题市场的可能。 |
| Jet / Pongo2 | 语法更好用，但**没有上下文感知转义**。把 XSS 防护的责任推给主题作者，在第三方主题场景下不可接受。 |
| Liquid（`osteele/liquid`） | 沙箱性好、Jekyll 作者熟悉，但 Go 实现生态小、性能较差，且仍需自己补一套转义层。**保留为未来的第二引擎选项**，不作为 v1 方案。 |
| 自研 DSL | 成本高、生态为零、文档要从头写。没有任何理由。 |

### 代价与对冲

`html/template` 的代价是**表达力弱、错误信息差**。对冲手段：

1. **day-one 就提供一套完整的辅助函数**（[§7](#7-函数命名空间)），不要让主题作者用 `printf` 拼逻辑。
2. **`kite doctor --explain-lookup <路径>`** 打印模板查找链，解决"为什么我的模板没被用到"这一类最常见的困惑。
3. **模板错误必须带文件名 + 行号 + 上下文片段**，而不是 Go 原生那种 `template: :12:5: executing "..." at <.Foo>` 的天书。这是 M0 就要做的事。

---

## 3. 什么算契约，什么不算

区分清楚，才知道哪些能改、哪些不能改。

| 属于契约（**冻结后改动 = 破坏性版本**） | 不属于契约（随时可改） |
|---|---|
| 模板目录布局与查找顺序 | 查找算法的内部实现 |
| `.Site` / `.Page` / `.Paginator` 等上下文对象的**方法签名** | 这些对象的内部 struct 字段与内存布局 |
| 函数命名空间与函数签名 | 函数内部实现、性能特征 |
| `theme.yaml` 的字段与语义 | 解析器实现 |
| Settings schema 的字段类型集合 | Admin 表单控件的外观 |
| `Resource` 接口 | Asset pipeline 的 transform 列表与顺序 |
| 分页、Taxonomy 页、RSS、Sitemap 的模板名与数据结构 | 它们的生成时机与缓存策略 |
| i18n 的函数名、目录约定、**URL 策略** | 翻译文件的加载实现 |
| Markdown 裸 HTML 策略（`unsafe` 开关的默认值） | goldmark 的扩展实现细节 |

---

## 4. 主题目录结构

```
themes/paper/
├── theme.yaml              # 元数据 + 兼容范围 + settings schema
├── layouts/                # 模板；与站点 layouts/ 同名同结构
│   ├── baseof.html         # 基础骨架
│   ├── home.html
│   ├── single.html
│   ├── list.html
│   ├── taxonomy.html       # 某个分类轴的所有 term 列表（如「所有标签」）
│   ├── term.html           # 某个 term 下的内容列表（如「标签 go」）
│   ├── 404.html
│   ├── single.rss.xml      # 非 html 输出格式
│   ├── post/               # 按 ContentType 覆盖
│   │   └── single.html
│   └── _partials/          # 下划线前缀 = 不可路由
│       ├── header.html
│       └── post-card.html
├── assets/                 # 参与 asset pipeline（fingerprint / minify）
│   ├── css/main.css
│   └── js/app.js
├── static/                 # 原样拷贝，不处理
├── i18n/
│   ├── en.yaml
│   └── zh-CN.yaml
└── screenshot.png          # 1280×800，Admin 主题列表用
```

### 关键决策：站点与主题使用**相同的目录名** `[已冻结]`

站点根目录下的 `layouts/` 与主题里的 `layouts/` 结构完全一致。覆盖规则因此可以一句话说清：

> **同一个相对路径，站点 `layouts/` 永远优先于主题 `layouts/`。**

> **与早期草案的差异**：最初设想主题用 `templates/`、站点用 `layouts/`。统一成 `layouts/` 之后，覆盖变成纯粹的路径前缀查找——既好解释，也让 `--explain-lookup` 的输出可读。

### 下划线前缀 = 保留，不可路由 `[已冻结]`

`_partials/` `_markup/` `_shortcodes/` 是保留目录。`layouts/` 下**非下划线开头的目录是可路由的页面路径**。

这是直接抄 Hugo 在 v0.146 重写后的**终点**，而不是它的弯路 `[EV]`——Hugo 早期把 partials 和页面模板混在同一层，后来不得不做破坏性迁移。

---

## 5. 模板查找顺序

### 5.1 只有四个维度 `[已冻结]`

| 维度 | 取值 |
|---|---|
| **kind** | `home` / `single` / `list` / `taxonomy` / `term` / `404` |
| **type** | ContentType 的 Kind，或 front matter 的 `type` 覆盖（如 `post` / `page` / `document`） |
| **layout** | front matter 的 `layout` 字段（可选，如 `wide`） |
| **output format** | `html`（默认，无后缀）/ `rss.xml` / `json` / … |

加一个**语言后缀**。

**明确拒绝 section 路径递归查找。** Hugo 的查找顺序有六个互相作用的维度、优先级不可调整，官方文档自己承认那是一次 "complete overhaul" `[EV]`。递归 section 查找正是让规则无法向用户解释的那一部分——Kite 不要。

### 5.2 查找链的构造

对一个 `kind=single`、`type=post`、`layout=wide`、`format=html`、`lang=zh` 的页面：

```
1.  layouts/post/wide.zh.html
2.  layouts/post/wide.html
3.  layouts/post/single.zh.html
4.  layouts/post/single.html
5.  layouts/single.zh.html
6.  layouts/single.html
```

每一条候选路径都**先查站点 `layouts/`，再查主题 `layouts/`**。也就是说实际检查顺序是：

```
site:layouts/post/wide.zh.html
theme:layouts/post/wide.zh.html
site:layouts/post/wide.html
theme:layouts/post/wide.html
...
```

> **为什么是"先按精确度、再按来源"而不是反过来**：如果站点的所有模板都优先于主题的所有模板，那么站点里一个笼统的 `layouts/single.html` 会覆盖掉主题精心写的 `layouts/post/single.html`，用户会莫名其妙丢掉主题效果。按精确度优先更符合直觉。

其他 kind 的链条同理：

| kind | 候选（由精确到笼统） |
|---|---|
| `home` | `home.html` → `list.html` |
| `list` | `<type>/list.html` → `list.html` |
| `taxonomy` | `<taxonomy>/taxonomy.html` → `taxonomy.html` → `list.html` |
| `term` | `<taxonomy>/term.html` → `term.html` → `list.html` |
| `404` | `404.html` |

### 5.3 `baseof.html` 与块

`baseof.html` 是骨架，页面模板通过 `{{ define "main" }}` 填充：

```html
<!-- layouts/baseof.html -->
<!DOCTYPE html>
<html lang="{{ .Site.Language }}">
<head>
  {{ partial "head.html" . }}
  {{ block "head_extra" . }}{{ end }}
</head>
<body>
  {{ partial "header.html" . }}
  <main>{{ block "main" . }}{{ end }}</main>
  {{ partial "footer.html" . }}
</body>
</html>
```

`baseof.html` 的查找遵循与页面模板相同的链条（`layouts/post/baseof.html` → `layouts/baseof.html`，各自先站点后主题）。

### 5.4 可解释性是硬要求

```bash
kite doctor --explain-lookup /posts/hello-world/
```

必须输出完整的候选链、每一条的命中/未命中、以及最终选中的文件。**"为什么我的模板没生效"是主题开发最高频的问题，把它变成一条命令就能自答，收益远大于成本。**

---

## 6. RenderContext 数据契约

### 6.1 用「带方法的接口」，不用「导出字段的 struct」 `[已冻结]`

```go
// 对的
type Page interface {
    Title() string
    Permalink() string
}

// 错的
type Page struct {
    Title     string
    Permalink string
}
```

**理由：**

- 方法可以**新增**而不破坏任何东西
- 方法可以**带日志告警地废弃**（`kite theme verify --strict` 可以把告警升级为失败）
- 方法可以**惰性计算**（`.Content()` 才触发渲染，`.Related()` 才触发查询）
- 导出字段等于把**内部内存布局永久冻结**——以后想改 `Content` 的存储方式都做不到

### 6.2 顶层对象

模板拿到的 `.` 永远是一个 `RenderContext`：

```go
type RenderContext interface {
    Site() Site
    Page() Page              // 当前页面；list 类页面也有（代表列表页自身）
    Pages() PageList         // list/taxonomy/term 页面的条目；single 页为空
    Paginator() Paginator    // 需要分页时非 nil
    Request() Request        // 静态构建时为 nil —— 见 §9.1
}
```

> 为方便书写，模板里 `.Title` 是 `.Page.Title` 的简写（在 single 上下文中）。这个简写规则本身也是契约的一部分。

### 6.3 `Site`

```go
type Site interface {
    Title() string
    Description() string
    BaseURL() string
    Language() string                       // "zh-CN"
    Languages() []LanguageInfo              // i18n 预留
    Params() ParamMap                       // kite.yaml 的 site.params
    ThemeSettings() ParamMap                // theme.yaml settings 的当前值
    Menus() map[string][]MenuItem
    Pages() PageList                        // 全站内容（惰性）
    Taxonomies() []string                   // ["tag", "category"]
    Taxonomy(name string) Taxonomy
    BuildTime() time.Time                   // 冻结的构建时间戳
    Kite() BuildInfo                        // Version / IsBuild / IsServe
}
```

**`Site.BuildTime()` 是模板获取时间的唯一途径。** funcmap 里**没有** `time.Now`——见 [§7.1](#71-命名空间设计-已冻结)。

### 6.4 `Page`

```go
type Page interface {
    ID() string                  // ULID，跨重命名稳定
    Kind() string                // home/single/list/taxonomy/term/404
    Type() string                // ContentType
    Layout() string

    Title() string
    Slug() string
    Description() string
    Permalink() string           // 绝对 URL
    RelPermalink() string        // 站内相对 URL
    Aliases() []string

    Content() template.HTML      // 渲染后的正文（惰性）
    Summary() template.HTML
    Truncated() bool
    TableOfContents() template.HTML
    Plain() string
    WordCount() int
    ReadingTime() time.Duration

    Date() time.Time
    PublishDate() time.Time
    Lastmod() time.Time
    Draft() bool

    Params() ParamMap            // front matter 的 meta 字段
    Terms(taxonomy string) TermList
    Resources() ResourceList     // page bundle 内的媒体

    Parent() Page
    Ancestors() PageList
    Children() PageList
    Next() Page
    Prev() Page
    Translations() PageList
}
```

### 6.5 `Paginator` `[已冻结]`

```go
type Paginator interface {
    Pages() PageList
    PageNumber() int
    TotalPages() int
    TotalItems() int
    PageSize() int
    First() Paginator
    Last() Paginator
    Prev() Paginator
    Next() Paginator
    HasPrev() bool
    HasNext() bool
    URL(n int) string     // 第 n 页的 URL；URL 模式是配置，这个方法是契约
}
```

**URL 的生成规则（`/page/2/` 还是 `?page=2`）是站点配置**，但模板可见的 API 固定。这样 Build 模式展开成静态页、Serve 模式走查询参数，主题**完全不需要知道**。

### 6.6 `Taxonomy` / `Term`

```go
type Taxonomy interface {
    Name() string                 // "tag"
    Hierarchical() bool
    Terms() TermList
    Get(slug string) Term
}

type Term interface {
    Name() string
    Slug() string
    Count() int
    Permalink() string
    Page() Page                   // 可能为 nil —— 见下方决策
    Pages() PageList
    Parent() Term
    Children() TermList
}
```

### 决策：**Term 是「可以带自己内容文件的页面」** `[已冻结]`

`content/tags/go/_index.md` 存在时，`Term.Page()` 返回它，主题可以拿到标签的描述、封面、自定义 meta；不存在时返回 nil，`{{ with .Page }}` 优雅降级。

> **这件事必须在契约冻结前决定。** 将来补"标签可以有描述和封面"会改变每一个 taxonomy / term 模板的数据形状，属于破坏性变更。

### 6.7 `Resource`（Asset Pipeline 的产物）`[已冻结]`

```go
type Resource interface {
    Name() string
    RelPermalink() string
    Permalink() string
    Content() string
    MediaType() string
    Data() ParamMap       // 图片的 Width/Height/EXIF 等
}
```

**冻结 `Resource` 接口，不冻结 transform 列表。** 这样以后加 AVIF、加 CSS bundling、换 minifier 都不是破坏性变更。

---

## 7. 函数命名空间

### 7.1 命名空间设计 `[已冻结]`

**所有函数必须挂在命名空间下**，只有模板机制本身必需的几个例外。

> **为什么**：Hugo 有约 200 个扁平的全局函数名，这是**永久的兼容性负债**——它永远不能再新增一个叫 `title` 或 `where` 的函数，也不能把 `title` 的语义改掉。命名空间让这个问题从一开始就不存在。`[EV]`

### 7.2 完整清单（v1 基线）

**顶层例外**（模板机制必需，无法命名空间化）：

```
dict  slice  default  partial  partialCached
safeHTML  safeURL  safeCSS  safeJS  safeHTMLAttr
T                              # i18n 的简写，高频到必须顶层
```

**`str.*`** —— 字符串

```
str.Title  str.Upper  str.Lower  str.Trim  str.TrimPrefix  str.TrimSuffix
str.Replace  str.Split  str.Join  str.Truncate  str.Slugify
str.HasPrefix  str.HasSuffix  str.Contains  str.Repeat  str.Pad
str.Markdownify  str.Plainify  str.CountWords
```

**`collections.*`** —— 集合（别名 `coll.*`）

```
collections.Where  collections.Sort  collections.First  collections.Last
collections.After  collections.Shuffle  collections.Uniq  collections.Reverse
collections.Union  collections.Intersect  collections.Symdiff
collections.GroupBy  collections.GroupByDate  collections.Len  collections.In
collections.Apply  collections.Seq
```

**`time.*`** —— 时间

```
time.Format  time.Parse  time.Since  time.Unix  time.Duration  time.AsTime
```

> **`time.Now` 不存在。** 模板获取当前时间的唯一途径是 `.Site.BuildTime`。
> 这不是疏漏，是[构建纯度](architecture.md#142-现在必须做对的四件事回填--重写)的强制执行点——`time.Now()` 是一个隐藏输入，会永久毒化增量构建的可缓存性。

**`url.*`** —— 链接

```
url.For        # ← 主题生成站内链接的唯一正确方式
url.Abs  url.Rel  url.JoinPath  url.Query  url.Anchorize  url.Parse
```

> **主题里禁止手工拼接站内链接。** Static 模式（目录索引 / `.html` 后缀）与 Dynamic 模式（路由）的 URL 形态不同，只有 `url.For` / `.Permalink` 能保证两边一致。这是 [§9 跨 Runtime 一致性](#9-跨-runtime-一致性)的第一道防线。

已实现的三个的语义：

- `url.For "home"`、`url.For "list" KIND`、`url.For "taxonomy" NAME`、`url.For "term" NAME TERM`：Kite 规划出的页面的链接，按站点配置的路由、URL 风格和 `baseURL` 的路径生成。名字或参数个数不对时模板报错。
- `url.Rel PATH`：站点里任意路径的链接，如 `"rss.xml"`，或作者在主题设置里填的 `/about/`，前面加上 `baseURL` 的路径。开头有没有 `/` 都一样，这一点和 Hugo 的 `relURL` 不同：交给 Kite 的路径一律是站内路径。完整网址、`//` 开头的地址、只有 `#` 或 `?` 的引用原样返回。
- `url.Abs PATH`：`url.Rel` 的结果再用 `baseURL` 补成绝对地址。

> 站点部署在子路径下时（GitHub Pages 的项目站点 `user.github.io/repo/`），`.RelPermalink`、分页、term 链接和上面三个函数给出的链接都以这段路径开头，输出文件的位置不变。`kite theme verify` 的夹具站点就发布在子路径下，并报告从域名根开始写的链接。

**`img.*`** —— 图片处理

```
img.Resize  img.Fit  img.Fill  img.Crop  img.Format  img.Quality  img.Filter
```

**`asset.*`** —— 资源管线

```
asset.Get  asset.CSS  asset.JS  asset.Fingerprint  asset.Minify  asset.Bundle  asset.Inline
```

**`i18n.*`**

```
i18n.T  i18n.Lang  i18n.Translate
```

**`math.*`** / **`debug.*`**

```
math.Add  math.Sub  math.Mul  math.Div  math.Mod  math.Ceil  math.Floor  math.Round  math.Max  math.Min
debug.Dump  debug.Timer
```

### 7.3 新增函数的规则

- **只增不改**：已发布的函数签名与语义不可变更
- 新函数必须挂在已有命名空间下，或开一个新命名空间
- 废弃：保留实现 + 运行时告警，`kite theme verify --strict` 下升级为失败；至少保留一个大版本

---

## 8. `theme.yaml`

### 8.1 完整 schema

```yaml
# ── 身份 ──
name: paper
version: 1.2.0                      # 主题自身版本（semver）
apiVersion: kite/v1                 # ← 主题契约版本；未知即硬拒绝
requires: ">=1.0.0 <2.0.0"          # 对 Kite 主程序的版本要求

# ── 元数据（Admin 展示用）──
title: Paper
description: A clean, reading-focused theme
author:
  name: Someone
  url: https://example.com
license: MIT
homepage: https://github.com/kite-plus/themes/tree/main/paper
tags: [blog, minimal, dark-mode]
screenshot: screenshot.png

# ── 能力声明 ──
capabilities: [static, dynamic]     # 支持的 Runtime；缺省两者都支持
contentTypes: [post, page]          # 该主题能渲染的内容类型
taxonomies: [tag, category]         # 该主题会渲染的分类轴
outputFormats: [html, rss]

# ── 模板清单（可选，用于 kite doctor 校验完整性）──
templates:
  - home
  - single
  - list
  - taxonomy
  - term
  - 404

# ── 设置 schema → Admin 自动生成配置页 ──
settings:
  - key: primary_color
    type: color
    label: 主色
    default: "#2563eb"

  - key: show_toc
    type: boolean
    label: 显示目录
    default: true
    help: 在文章页右侧显示自动生成的目录

  - key: posts_per_page
    type: number
    label: 每页文章数
    default: 10
    min: 1
    max: 100

  - key: header_style
    type: select
    label: 页头样式
    default: simple
    options:
      - { value: simple, label: 简洁 }
      - { value: cover,  label: 大图 }

  - key: social
    type: group
    label: 社交链接
    fields:
      - { key: github,   type: url, label: GitHub }
      - { key: twitter,  type: url, label: X / Twitter }

  - key: nav
    type: repeat
    label: 导航菜单
    fields:
      - { key: text, type: string, label: 文字 }
      - { key: url,  type: url,    label: 链接 }
```

### 8.2 Settings 字段类型（v1 基线） `[已冻结]`

| type | Admin 控件 | 模板取值 |
|---|---|---|
| `string` | 单行输入 | string |
| `text` | 多行输入 | string |
| `number` | 数字输入（`min`/`max`/`step`） | float64 |
| `boolean` | 开关 | bool |
| `color` | 取色器 | string |
| `select` | 下拉（`options`） | string |
| `multiselect` | 多选 | []string |
| `image` | 媒体选择器 | Resource |
| `url` | URL 输入（带校验） | string |
| `code` | 代码编辑器（`language`） | string |
| `group` | 嵌套分组（`fields`） | ParamMap |
| `repeat` | 可增删的重复项（`fields`） | []ParamMap |

通用可选属性：`label` / `help` / `default` / `required` / `placeholder` / `showIf`（条件显示）。

**主题里的取值**：`{{ .Site.ThemeSettings.primary_color }}`

> 同一套 schema 定义与渲染器被 **ContentType.Fields** 与 **Plugin Settings** 复用。一次投入，三处受益——这是"每加一种内容类型就要写一个 Admin 页面"的唯一解药。

### 8.3 校验时机

`apiVersion` 与 `requires` 在**安装时和每次构建时都校验**：

- 未知 `apiVersion` → **硬拒绝**，不是"尽力渲染"
- `requires` 不满足 → 拒绝并给出明确的升级/降级提示

Halo 的 `requires` 是已验证有效的模式 `[EV]`。

---

## 9. 跨 Runtime 一致性

**同一套主题必须在 `kite build` 和 `kite serve` 下产出一致结果。** 这不是靠自觉，靠四个机制。

### 9.1 模板永远不知道自己在哪个 Runtime

- 数据**只**从唯一的 `ContentReader` 读模型来（见 [architecture.md §0.2](architecture.md#02-只有一个读模型read-model)）
- 模板里**不允许惰性访问 Store**——要么预解析，要么所有惰性访问器都走同一个 reader

### 9.2 请求态数据的唯一出口是 `.Request` `[已冻结]`

```html
{{ with .Request }}
  <p>你正在访问 {{ .Path }}</p>
{{ end }}
```

- `.Request` 在**静态构建时为 nil**，这一点必须写进主题文档的显眼处
- `{{ with .Request }}` 是**唯一**允许的访问方式
- **绝不隐式暴露请求态数据**（比如把当前 URL 塞进 `.Page` 的某个字段）——否则主题作者会不自觉地写出只能在 Dynamic 下工作的模板，而且要到部署成静态站才发现

### 9.3 URL 只能由 `url.For` / `.Permalink` 生成

见 [§7.2](#72-完整清单v1-基线)。

### 9.4 `kite theme verify` —— **契约的真身**

```bash
kite theme verify ./themes/paper
kite theme verify ./themes/paper --strict   # 废弃告警升级为失败
```

行为：

1. 拿一个内置的 **fixture 站点**（覆盖 single / list / taxonomy / term / 分页 / 404 / RSS / 多语言 / 有图与无图 / 有 term 页与无 term 页）
2. 在 **build 模式**下渲染全站
3. 在 **serve 模式**下逐 URL 请求 build 写出的每个文件
4. **逐字节 diff**

> **那个测试才是契约本身，文档不是。** 文档会过时、会被误读；一个跑在 CI 里的黄金文件测试不会。
>
> `kite theme verify` **从 M0 就存在**（即使那时只有一套内置主题），不能等到 M5 才补。

---

## 10. Asset Pipeline

### 10.1 管线是声明式 DAG，输出 hash 必须覆盖每一个 transform `[已冻结]`

```html
{{ $css := asset.Get "css/main.css" | asset.Minify | asset.Fingerprint }}
<link rel="stylesheet" href="{{ $css.RelPermalink }}">
```

**核心规则：`Resource` 的内容 hash 必须覆盖它经历过的全部 transform 及其参数。**

反面教材：Hugo 曾经出现 fingerprint 在 PurgeCSS **之前**执行，导致产物变了但 hash 没变——CDN 继续投递旧文件 `[EV]`。这类 bug 是**静默**的，用户只会看到"样式偶尔不更新"。

实现要求：

```
resource_hash = SHA256( source_bytes ‖ transform_chain_spec ‖ transform_params )
```

其中 `transform_chain_spec` 是整条 DAG 的规范化描述，而不是"最后一步的输出"。

### 10.2 v1 提供的 transform

| 函数 | 作用 | v1 |
|---|---|---|
| `asset.Get` | 从 `assets/` 取资源 | ✅ |
| `asset.Fingerprint` | 内容 hash 进文件名 | ✅ |
| `asset.Minify` | CSS/JS/HTML 压缩 | ✅ |
| `asset.Inline` | 内联进 HTML | ✅ |
| `asset.Bundle` | 多文件合并 | M5 |
| `img.Resize` / `Fit` / `Fill` | 图片缩放 | M5 |
| `img.Format` | WebP / AVIF 转换 | M5 |
| SCSS / PostCSS | —— | **不做**，见 [§11.3](#113-明确不做的东西) |

**v1 的主题用纯 CSS。** 需要构建步骤的主题可以自己在发布前编译好，产物放进 `assets/`。

### 10.3 带 fingerprint 的产物可以 `immutable`

`Cache-Control: public, max-age=31536000, immutable` —— 这是 fingerprint 的全部意义。

---

## 11. 版本、兼容与冻结

### 11.1 版本轴

| 版本 | 含义 | 谁声明 |
|---|---|---|
| `apiVersion: kite/v1` | **主题契约的大版本**。`v1` → `v2` 是破坏性变更 | 主题在 `theme.yaml` |
| `requires: ">=1.0.0 <2.0.0"` | 主题对 **Kite 主程序**的版本要求 | 主题在 `theme.yaml` |
| `version: 1.2.0` | 主题**自身**版本 | 主题在 `theme.yaml` |

### 11.2 冻结时间表

| 阶段 | 状态 | 做什么 |
|---|---|---|
| **M0** | 引擎可用，契约**内部** | 实现查找顺序、RenderContext、funcmap、`kite theme verify`。文档标注"内部 API，可能变更" |
| **M1~M4** | 随实现演进 | 遇到不顺手就改，不承担任何兼容义务 |
| **M5** | **写第二套主题** | 用第一套主题的契约去写一套风格完全不同的主题。**每一处别扭都是契约缺陷的证据** |
| **M5 末** | **冻结，发布 `kite/v1`** | 打版本、写文档、发 theme-sdk、上官方主题仓库 |
| M5 之后 | 只增不改 | 新增方法/函数可以；改名/改语义要走 `kite/v2` |

> **为什么必须等第二套主题：** Hugo 在 v0.146 做了一次**彻底的模板系统重写**——`_default/` 去掉、`layouts/partials` → `layouts/_partials`、`index.html` → `home.html`、`list-baseof.html` → `baseof.list.html`。即便做了新旧映射，仍然打断了包括 Docsy 在内的大量主题 `[EV]`。
>
> **没写过第二套主题就冻结的契约一定是错的**——第一套主题的作者就是引擎作者，他会不自觉地绕开所有缺陷。

### 11.3 明确不做的东西

| 不做 | 理由 |
|---|---|
| **主题继承 / 子主题** | **极难收回**。一旦支持，查找顺序要多一个维度、覆盖语义要处理多层、`--explain-lookup` 的输出会失控。用户真正的需求（改一点样式）用站点 `layouts/` 覆盖 + settings 就能满足。 |
| SCSS / PostCSS 工具链 | 要么引入 Node 依赖（破坏 Single Binary），要么嵌一个不完整的实现（比没有更糟）。主题自己编译好再发布。 |
| 主题提供 Go 代码 / 原生插件 | 扩展走 [plugin-system.md](plugin-system.md)，主题只管渲染。 |
| Admin 里的在线主题编辑 | 会让"主题是 Git 里的文件"这个模型崩掉。 |
| 带命名参数校验的 shortcode | v1 只做最简单的 shortcode；复杂的交给插件的节点匹配器。 |
| 每主题自定义内容类型 | 内容模型属于站点，不属于主题。主题只能**声明它支持哪些类型**。 |
| 主题市场 / 注册表 | V4。v1 用 Git URL + checksum 安装。 |

---

## 12. i18n

v1 只发一种语言，**但以下三件事必须在冻结前定死**：

1. **Catalog 格式**：go-i18n 风格的 YAML，放 `i18n/<lang>.yaml`
2. **函数与上下文**：`T "key" count` / `i18n.T` / `.Site.Language` / `.Page.Translations` / `.Site.Languages`
3. **多语言 URL 策略**：路径前缀（`/zh/posts/...`）还是子域（`zh.example.com`）

> **第 3 条最容易被忽略，代价也最大**：它影响**每一个页面的 `.Permalink`**。等有了多语言站点再决定，等于要求所有已存在的站点改 URL。
>
> **建议默认：路径前缀 + 默认语言不加前缀**，可配置切换为子域。

```yaml
# i18n/zh-CN.yaml
read_more: 阅读全文
posts_count:
  one: "{{.Count}} 篇文章"
  other: "{{.Count}} 篇文章"
```

---

## 13. 反面教材清单

写实现前应该看一遍，这些都是别人已经付过的学费。

| 教训 | 出处 | Kite 的对策 |
|---|---|---|
| 查找顺序维度过多 → 无法向用户解释 → 被迫重写 | Hugo v0.146 `[EV]` | 只保留 4 个维度，拒绝 section 递归 |
| 扁平全局 funcmap → 永远不能再用那些名字 | Hugo ~200 个全局函数 `[EV]` | 强制命名空间 |
| 导出 struct 字段 → 内部布局被永久冻结 | —— | 带方法的接口 |
| fingerprint 在 transform 之前 → 静默的缓存失效 | Hugo #11268 `[EV]` | hash 覆盖整条 DAG |
| 模板能拿到请求态数据 → 主题只在 Dynamic 下能用 | —— | `.Request` 显式 nil + `{{ with }}` 唯一访问方式 |
| 模板能读时间 → 破坏构建纯度与增量构建 | —— | funcmap 里没有 `time.Now` |
| 契约靠文档维护 → 实现漂移无人发现 | —— | `kite theme verify` 黄金文件测试 |
| 主题继承 → 覆盖语义爆炸 | 多个 CMS | 不做 |
| Term 不能带内容 → 后来补描述/封面是破坏性变更 | —— | Term 从一开始就可以有 `_index.md` |

---

## 14. 开放问题 `[待定]`

这些**不在 M0 决定**，但要在 M5 冻结前有结论：

1. **Shortcode 的语法与能力边界** —— 用 `{{< >}}` 风格还是走 Markdown 扩展？与插件的节点匹配器如何分工？
2. **`partialCached` 的缓存键** —— 由用户显式传入，还是从依赖记录自动推导？后者更安全但实现复杂。
3. **主题能否声明自己需要的 Hook / 插件** —— 例如一个主题依赖 `mermaid` 插件。倾向于：可以声明为**建议**，不能声明为**硬依赖**。
4. **`.Site.Pages` 在 Serve 模式下的语义** —— 全量加载不可接受，但主题会天真地这么写。倾向于：返回惰性 PageList，并对无 limit 的遍历发出告警。
5. **暗色模式** —— 完全交给主题 CSS，还是提供一个标准的 settings key 与 `data-theme` 约定？倾向于后者（约定优于各自发明）。

---

## 证据来源

- [Hugo 模板系统重写 PR #13541](https://github.com/gohugoio/hugo/pull/13541) —— `_default/` 移除、`layouts/partials` → `_partials`、`index.html` → `home.html` 等破坏性迁移
- [Hugo 新模板系统概览](https://gohugo.io/templates/new-templatesystem-overview/)
- [Hugo 旧查找顺序文档](https://gohugo.io/templates/lookup-order/) —— 六个互相作用的维度
- [Docsy 受影响 #2243](https://github.com/google/docsy/issues/2243) · [Hugo #13588](https://github.com/gohugoio/hugo/issues/13588) —— 重写对下游主题的实际冲击
- [Hugo fingerprint / PurgeCSS 缺陷 #11268](https://github.com/gohugoio/hugo/issues/11268) —— asset hash 未覆盖全部 transform
- [Halo 主题开发指南](https://docs.halo.run/developer-guide/theme/prepare) · [Halo theme-starter settings.yaml](https://github.com/halo-dev/theme-starter/blob/main/settings.yaml) —— `requires` 与 settings schema 驱动 Admin UI 的成功先例
