# Kite 主题系统设计

> 状态：契约 **`kite/v1` 已于 2026-10-01 冻结**，只增不改（§11.2） · 最近更新：2026-10-01
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
├── theme.yaml              # 元数据 + 兼容范围 + 设置、布局、菜单的声明
├── layouts/                # 模板；与站点 layouts/ 同名同结构
│   ├── baseof.html         # 基础骨架
│   ├── home.html           # 没有时用 list.html
│   ├── single.html
│   ├── list.html
│   ├── taxonomy.html       # 某个分类轴的所有 term 列表（如「所有标签」）；没有时用 list.html
│   ├── term.html           # 某个 term 下的内容列表（如「标签 go」）；没有时用 list.html
│   ├── 404.html
│   ├── post/               # 按 ContentType 覆盖
│   │   └── single.html
│   ├── _partials/          # 下划线前缀 = 不可路由
│   │   ├── header.html
│   │   └── post-card.html
│   ├── _shortcodes/        # 正文里按名字调用的短代码（§6.8）
│   └── _markup/            # 正文节点的 render hook，如 render-image.html
├── assets/                 # 原样发布到站点的 /assets/（§10）
├── static/                 # 原样发布到站点根
├── i18n/                   # 语言包：后台用的 theme 键，页面上的词（§12）
│   ├── en.yaml
│   └── zh-CN.yaml
└── screenshot.png          # 1280×800，后台主题列表用；也可以是 .jpg 或 .webp
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

加一个**语言后缀**。查找已经支持它；v1 的站点只有一种语言，页面还没有语言后缀可查（§12）。

输出格式这一维目前只有 `html`：RSS 和 sitemap 由 Kite 的钩子生成，不经过主题模板。

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

必须输出完整的候选链、每一条的命中/未命中、以及最终选中的文件。（这条命令还没有实现；它是工具，不属于契约，随时可以加。模板找不到时的报错已经列出完整的候选链。）**"为什么我的模板没生效"是主题开发最高频的问题，把它变成一条命令就能自答，收益远大于成本。**

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

模板拿到的 `.` 永远是一个 `Context`：

```go
type Context interface {
    Site() Site
    Page() Page              // 当前页面；列表类页面也有（代表列表页自身）
    Pages() []Page           // 列表、分类法、term 页本页的条目；single 页为空
    Paginator() Paginator    // 需要分页时非 nil
    Terms() []Term           // 分类法页列出的 term
    Request() Request        // 静态构建时为 nil —— 见 §9.2
}
```

> 为方便书写，模板里 `.Title` 是 `.Page.Title` 的简写（在 single 上下文中）。这个简写规则本身也是契约的一部分。

**`kite/v1` 冻结的就是这里和 §6.3–§6.9 列出的方法，一个不多、一个不少。** 每一项连同签名记录在 `internal/render/theme/testdata/kite-v1.txt`，`TestTheContractOnlyGrows` 在任何一项被删掉、改名或换了签名时失败（§11.2）。设计阶段设想过、v1 没有的方法，列在各小节末尾的「以后可以加」里：加进来是只增，不破坏已有主题。

### 6.3 `Site`

```go
type Site interface {
    Title() string
    Description() string
    BaseURL() string
    Language() string                       // "zh-CN"
    Params() map[string]any                 // kite.yaml 的 site.params
    ThemeSettings() map[string]any          // theme.yaml settings 的当前值，按字段类型读好
    Taxonomies() []string                   // ["tags", "categories"]
    Menus() map[string][]MenuItem           // kite.yaml 的 menus，见 §6.9
    Author() string                         // site.author
    Keywords() []string                     // site.keywords，给搜索引擎
    NoIndex() bool                          // site.noindex，要求搜索引擎不收录
    HeadHTML() template.HTML                // site.headHTML，原样写到 </head> 前
    FooterHTML() template.HTML              // site.footerHTML，原样写到 </body> 前
    BuildTime() time.Time                   // 冻结的构建时间戳
    Version() string                        // 正在运行的 Kite 的版本
    IsBuild() bool                          // 静态构建
    IsServe() bool                          // kite serve / kite run
}
```

**`Site.BuildTime()` 是模板获取时间的唯一途径。** funcmap 里**没有** `time.Now`——见 [§7.1](#71-命名空间设计-已冻结)。

**日期按站点的时区给出。** `site.timezone` 设置后，`BuildTime` 和页面的 `Date`、`PublishDate`、`Lastmod` 都换算到这个时区，模板直接格式化即可；不设置时，日期保持写入时的时区。时区属于站点而不是主题：同一篇文章落在哪一天，不应随换主题而变。

**日期可能是零值。** `PublishDate` 在没有发布时间时退回创建时间；手写的文章可能两者都没写，这时 `Date`、`PublishDate`、`Lastmod` 都是零值。主题先用 `time.IsZero` 判断：零值不显示，也不按它的年份（公元 1 年）归档。

**站点的关键词、作者、`noindex` 和自定义代码由主题写进页面。** 它们跟着站点保存，换主题不丢；主题负责把它们放到 `<head>` 和 `</body>` 前。自动注入更多 SEO 标签（如 Open Graph）留给插件（[plugin-system.md](plugin-system.md)）。

以后可以加：`Languages()`（多语言站点的语言，见 §12）、`Pages()`（全站内容，见 §14 第 4 条）、`Taxonomy(name)`（一个分类法的全部 term）。

### 6.4 `Page`

```go
type Page interface {
    ID() string                  // ULID，跨重命名稳定
    Kind() Kind                  // home/single/list/taxonomy/term/404
    Type() string                // ContentType
    Title() string
    Slug() string
    Description() string

    Permalink() string           // 绝对 URL
    RelPermalink() string        // 站内 URL，带着站点的路径
    Aliases() []string           // 条目以前的地址，Kite 在那里放一个跳转页

    Content() template.HTML      // 渲染后的正文（惰性）
    Excerpt() string             // 摘要：作者写的，或正文开头的一段
    TableOfContents() []Heading  // 正文的标题，每个有 Level、ID、Text
    WordCount() int              // 中日韩文字每字算一个词
    ReadingTime() time.Duration
    Images() []string            // 正文里的图片，按出现顺序，保持原文写法

    Date() time.Time
    PublishDate() time.Time
    Lastmod() time.Time
    Draft() bool

    Params() map[string]any      // front matter 的 meta 字段，保持 YAML/TOML 里的类型
    Terms(taxonomy string) []Term
    Resources() ResourceList     // page bundle 里的文件，见 §6.7

    Prev() Page                  // 同一种条目里更早的一篇，没有是 nil
    Next() Page                  // 更新的一篇
}
```

列表里的页（`.Pages` 的每一项）和前后篇也是 `Page`，带着列表需要的一切：标题、地址、日期、摘要、字数、`Params`、`Terms`、`Images`、`Resources`；只有 `Content`、`TableOfContents` 这些要渲染正文的方法是空的。

以后可以加：`Layout()`、`Summary()` / `Truncated()` / `Plain()`（v1 用 `Excerpt`）、`Parent()` / `Ancestors()` / `Children()`（分节的站点）、`Translations()` 和 `Language()`（多语言，见 §12）。

### 6.5 `Paginator`

```go
type Paginator interface {
    PageNumber() int
    TotalPages() int
    TotalItems() int
    PageSize() int
    HasPrev() bool
    HasNext() bool
    PrevURL() string
    NextURL() string
    FirstURL() string
    LastURL() string
    URL(n int) string     // 第 n 页的 URL；URL 模式是配置，这个方法是契约
}
```

本页的条目是 `.Pages`，不在 `Paginator` 上。**URL 的生成规则（`/page/2/` 还是 `?page=2`）是站点配置**，但模板可见的 API 固定。这样 Build 模式展开成静态页、Serve 模式走查询参数，主题**完全不需要知道**。

**每种列表怎么分页由主题声明**（`theme.yaml` 的 `pagination`，见 [§8.1](#81-完整-schema)），站点的 `build.pagination` 按种类盖过它，都没写的按 `build.pageSize`。数目是 0 的列表只有一页，放全部条目，`PageSize` 等于条目数；每一页仍是一个独立的构建目标。

### 6.6 `Term`

```go
type Term interface {
    Taxonomy() string             // "tags"
    Name() string                 // 写法不同但地址相同的词条是同一个，名字取写得最多的写法
    Slug() string
    Count() int
    Permalink() string
    RelPermalink() string
    Page() Page                   // 可能为 nil —— 见下方决策
}
```

分类法页（`taxonomy.html`）的 term 在 `.Terms` 里；一篇文章的 term 用 `.Page.Terms "tags"`，按这篇文章的写法给出。

以后可以加：`Taxonomy` 对象（`Hierarchical`、`Terms`、`Get`）、`Term.Pages()`、`Term.Parent()` / `Children()`（有层级的分类）。

### 决策：**Term 是「可以带自己内容文件的页面」** `[已冻结]`

`content/tags/go/_index.md` 存在时，`Term.Page()` 返回它，主题可以拿到标签的描述、封面、自定义 meta；不存在时返回 nil，`{{ with .Page }}` 优雅降级。

> **这件事必须在契约冻结前决定。** 将来补"标签可以有描述和封面"会改变每一个 taxonomy / term 模板的数据形状，属于破坏性变更。

### 6.7 `Resource`

```go
type Resource interface {
    Name() string                 // bundle 里的路径，如 images/01.jpg
    RelPermalink() string
    Permalink() string
    MediaType() string            // image/jpeg
    Content() (string, error)
    Data() (map[string]any, error)  // 图片的 Width、Height
}

type ResourceList []Resource
func (ResourceList) Get(name string) Resource        // 没有是 nil
func (ResourceList) Match(pattern string) ResourceList
func (ResourceList) ByType(kind string) ResourceList // image / video / audio / text / application
```

图片另有 `Width()`、`Height()`（按 EXIF 方向转正后的尺寸，返回 `(int, error)`）。`img.*` 做出的图是 `Image`，方法和上面相同，每一个都返回 `(值, error)`：图只在模板第一次问它的地址或尺寸时才真正去做，做图的错误从这里报出来。

**冻结 `Resource` 接口，不冻结 transform 列表。** 这样以后加 AVIF、加 CSS bundling、换 minifier 都不是破坏性变更。

- `.Resources` 按名字排序列出 bundle 发布的每个文件（`.md` 和隐藏文件除外，另一篇内容的 bundle 目录除外），`Name` 是 bundle 里的路径，如 `images/01.jpg`。`.Resources.Get "cover.jpg"` 按名字取一个，没有是 nil；`.Resources.Match "*.jpg"` 按通配符取，不分大小写，`*` 不跨 `/`，`**` 跨；`.Resources.ByType "image"` 按媒体类型的大类取。单页、列表里的页、前后篇和短代码的 `.Page` 都有。
- `img.Resize "800x"`（缺一边就按比例）、`img.Fit "1200x1200"`（只缩小，放进这个框）、`img.Fill "600x400 top"`（先按比例裁、再缩放到正好这个尺寸，锚点可选）、`img.Crop "600x400"`（只裁不缩）、`img.Format "webp"`（webp / jpeg / png / gif）、`img.Quality 80`（WebP 和 JPEG 的质量，默认 75）。它们接一张图、返回一张待做的图，可以串起来；模板第一次问它的地址或尺寸时才真正去做，所以中间步骤不会多做。JPEG、PNG、GIF 和 WebP 都能读写，全是纯 Go：WebP 用 `github.com/deepteams/webp`（不查找系统里的 libwebp；不用 `golang.org/x/image/webp`，它把 WebP 的黑和白读成 16 和 235 的灰），缩放用 `x/image/draw` 的 Catmull-Rom。固定输入做出的字节由测试核对 hash，CI 在 amd64 和 arm64 上都跑，所以不同机器做出同样的图。WebP 是有损的，保留透明；写成 JPEG 时透明部分铺白。没指定格式就保持源文件的格式。手机照片先按 EXIF 方向转正；做出的图不带源文件的 EXIF，拍摄地点之类不会随图发布。
- 做出的图发布在源文件旁边，名字是源文件名加上 16 位 key：`river_<key>.jpg`。key 是源文件内容、整条做法和做图代码版本的 hash，符合 §10.1 的要求。做出的图缓存在 `.kite/cache/images/`，下一次构建和 serve 直接用；serve 按文件名里的 key 找到它，所以 build 和 serve 给出的是同一份字节，`kite theme verify` 的夹具也有一张。

以后可以加：`img.Filter`、AVIF 输出。

### 6.8 `Shortcode`

正文用 Hugo 的语法按名字调用模板：`{{< figure src="a.jpg" >}}`，或者 `{{< note >}}…{{< /note >}}` 包住一段 Markdown；`{{% %}}` 的读法相同。模板是 `layouts/_shortcodes/<name>.html`，查找顺序和其他模板一样，站点的优先于主题的；名字可以带目录（`docs/note`）。模板拿到的 `.` 是这一次调用：

```go
type Shortcode interface {
    Name() string
    Get(key any) any        // .Get 0 按位置，.Get "src" 按名字；没给是 nil
    Params() any            // 按位置给是 []any，按名字给是 map[string]any
    IsNamedParams() bool
    Inner() template.HTML   // 一对标签包住的内容，按 Markdown 渲染好
    RawInner() string       // 同一段内容的原文
    Ordinal() int           // 在同一个外层里是第几个调用，从 0 数
    Parent() Shortcode      // 外层的调用；在正文顶层是 nil
    Page() Page             // 写了这次调用的页面；正文还在绘制，Content 为空
    Site() Site
}
```

读法：

- 单独占一行的标签是块，不被段落包住，一对这样的标签包住中间的块；写在一行文字里的标签是这行的一个词，一对这样的标签包住中间的文字。开始标签单独一行、结束标签在最后一段的行尾，也是一对块。一个自闭合的标签单独一行夹在段落中间时，仍然是段落里的词，和 CommonMark 对单独一行 HTML 标签的处理一样。
- 参数要么全按位置、要么全按名字给。带引号的值是字符串；不带引号、读得出 `true`、`false` 或数字的，是 bool、int 或 float64。
- 代码里的标签原样显示；`{{</* name */>}}` 在任何地方都显示成它注释掉的标签。原样的 HTML 里的标签照常绘制。标签也可以写在链接和图片的地址里（`[x]({{< ref "a" >}})`），它画出来的文字就是地址。
- **只有模板显示出来的内容才算数。** 模板没有调用 `.Inner` 时，里面的字不计入页面的 `WordCount`、摘要和交给钩子的正文（比如搜索索引），里面的标题也不进 `TableOfContents`。索引不知道模板显示什么，所以调用了短代码的条目，列表里的摘要和字数由构建按页面的算法重新算出。
- 没有模板的短代码、写错的标签都让构建停下，并给出文件和行号，而不是把标签原样发布；后台的预览同样报出这个错误。

参数没有 schema、不做校验，见 [§11.3](#113-明确不做的东西)。

**render hook**：正文里的图片由 `layouts/_markup/render-image.html` 画（站点的优先于主题的），没有这个模板时按 Markdown 原样输出。模板拿到的 `.` 是 `BodyImage`：`Destination`（正文里写的地址，写 bundle 里的文件时就是 `.Resources.Get` 要的名字）、`Src`（没有模板时页面引用它的地址，从站点根写的带上站点路径）、`Text`（替代文字）、`Title`、`Page`、`Site`。配合 [§6.7](#67-resource) 的 `img.*`，正文里的照片可以缩小后再发布。链接、标题、代码块的 hook 还没有。`kite theme verify` 的夹具站点自带两个短代码，所以正文里的短代码也在 build 与 serve 的逐字节比较之内。

### 6.9 `MenuItem`（菜单）

```go
type MenuItem interface {
    Name() string
    URL() string              // 站内地址已经带上站点的路径；完整网址原样
    Params() map[string]any   // 站点给这个链接的其他东西，比如图标
    Children() []MenuItem
}
```

菜单属于站点，不属于主题：写在 `kite.yaml` 的 `menus` 下，按名字分（`main`、`footer`……），换主题不丢。主题在 `theme.yaml` 的 `menus` 里声明它画哪些菜单、每个画几层（§8.1），后台的“菜单”页据此列出要填的菜单。模板用 `{{ range .Site.Menus.main }}` 画；站点没写的菜单是空的，主题可以用 `{{ with .Site.Menus.main }}…{{ else }}…{{ end }}` 退回自己的链接，默认主题就是这样。菜单的名字只能是小写字母、数字和 `_`，所以总能写成 `.Site.Menus.<名字>`。

`URL` 由 Kite 按 `url.Rel` 的规则算好：站内地址从站点根写起（`/about/`），发布在站点的路径下，主题不用再处理；只用来归拢下级链接的一项没有 `URL`。判断当前页用 `eq .URL $.Page.RelPermalink`，两边都带着站点的路径。

以后可以加：按页面引用的菜单项（`page:`，页面改地址时菜单跟着变）、`Page.IsMenuCurrent` 一类的判断。

---

## 7. 函数命名空间

### 7.1 命名空间设计 `[已冻结]`

**所有函数必须挂在命名空间下**，只有模板机制本身必需的几个例外。

> **为什么**：Hugo 有约 200 个扁平的全局函数名，这是**永久的兼容性负债**——它永远不能再新增一个叫 `title` 或 `where` 的函数，也不能把 `title` 的语义改掉。命名空间让这个问题从一开始就不存在。`[EV]`

### 7.2 `kite/v1` 的函数

下面是 v1 的全部函数，每一个的签名都记录在契约清单里（§11.2）。

**顶层例外**（模板机制必需，无法命名空间化）：

```
dict  slice  default  partial  partialCached
safeHTML  safeURL  safeCSS  safeJS  safeHTMLAttr
T                              # i18n 的简写，高频到必须顶层
```

`default` 的参数顺序和 Hugo 一样，默认值在前：`default "#7d5c3c" .Site.ThemeSettings.accent`。`partialCached` 在 v1 就是 `partial`，接受并忽略 Hugo 写在数据之后的变体参数（§14 第 2 条）。

**`str.*`** —— 字符串

```
str.Title  str.Upper  str.Lower  str.Trim  str.TrimPrefix  str.TrimSuffix
str.Replace  str.Split  str.Join  str.Truncate  str.Repeat
str.HasPrefix  str.HasSuffix  str.Contains  str.CountWords
```

**`collections.*`** —— 集合（别名 `coll.*`）

```
coll.First  coll.Last  coll.After  coll.Reverse  coll.Uniq  coll.Sort  coll.Len  coll.In
```

**`time.*`** —— 时间

```
time.Format  time.Parse  time.AsTime  time.Unix  time.Year  time.Minutes  time.IsZero
```

front matter 里的值到模板里仍是原来的类型，在单页和列表里一样：日期（无论写在哪一层，YAML 和 TOML 都一样）是 `time.Time`，整数是 `int`，小数是 `float64`。`time.AsTime` 把时间原样返回，把用 front matter 常见写法写成文字的日期（`2026-03-04`、`2026-03-04 09:30`、RFC 3339）读成时间，其余的值报错。`time.Minutes` 把 `.ReadingTime` 这样的时长取整成分钟，`time.IsZero` 判断一个日期是不是没写（§6.3）。

> **`time.Now` 不存在。** 模板获取当前时间的唯一途径是 `.Site.BuildTime`。
> 这不是疏漏，是[构建纯度](architecture.md#142-现在必须做对的四件事回填--重写)的强制执行点——`time.Now()` 是一个隐藏输入，会永久毒化增量构建的可缓存性。

**`url.*`** —— 链接

```
url.For        # ← 主题生成站内链接的唯一正确方式
url.Rel  url.Abs  url.Join  url.Query  url.Escape  url.Anchorize
```

> **主题里禁止手工拼接站内链接。** Static 模式（目录索引 / `.html` 后缀）与 Dynamic 模式（路由）的 URL 形态不同，只有 `url.For` / `.Permalink` 能保证两边一致。这是 [§9 跨 Runtime 一致性](#9-跨-runtime-一致性)的第一道防线。

语义：

- `url.For "home"`、`url.For "list" KIND`、`url.For "taxonomy" NAME`、`url.For "term" NAME TERM`：Kite 规划出的页面的链接，按站点配置的路由、URL 风格和 `baseURL` 的路径生成。名字或参数个数不对时模板报错。
- `url.Rel PATH`：站点里任意路径的链接，如 `"rss.xml"`，或作者在主题设置里填的 `/about/`，前面加上 `baseURL` 的路径。开头有没有 `/` 都一样，这一点和 Hugo 的 `relURL` 不同：交给 Kite 的路径一律是站内路径。完整网址、`//` 开头的地址、只有 `#` 或 `?` 的引用原样返回。
- `url.Abs PATH`：`url.Rel` 的结果再用 `baseURL` 补成绝对地址。
- `url.Join`、`url.Query`、`url.Escape`：拼路径的各段、给查询参数的值转义、给路径的一段转义。
- `url.Anchorize TEXT`：按标题锚点的规则把文字变成 id。标题的 id 从标题的文字生成，规则和 GitHub 相同：转小写，保留任何文字的字母、数字以及 `-`、`_`，空格变成 `-`，其余标点去掉，所以「近况」的 id 就是 `近况`；重复的依次加 `-1`、`-2`，什么都不剩的叫 `heading`，作者用 `{#id}` 写的 id 原样保留。`.TableOfContents` 里的 `.ID` 就是它。中文 id 写进 `href` 时会被 `html/template` 百分号编码，浏览器跳转时会解码；主题脚本拿链接找标题时要先 `decodeURIComponent`。

> 站点部署在子路径下时（GitHub Pages 的项目站点 `user.github.io/repo/`），`.RelPermalink`、分页、term 链接、菜单和上面这些函数给出的链接都以这段路径开头，输出文件的位置不变。`kite theme verify` 的夹具站点就发布在子路径下，并报告从域名根开始写的链接。

**`img.*`** —— 图片处理

```
img.Resize  img.Fit  img.Fill  img.Crop  img.Format  img.Quality
```

语义见 [§6.7](#67-resource)。

**`i18n.*`**

```
i18n.T  i18n.Has  i18n.Words
```

语义见 [§12](#12-i18n)。

**`math.*`**

```
math.Add  math.Sub  math.Mul  math.Div  math.Mod  math.Ceil  math.Floor  math.Round  math.Max  math.Min
math.Int  math.Float
```

`math.*` 和 `collections.*` 对参数类型的约定：

- `math.*` 接受任意整数和浮点数。参数都是整数时结果是 `int`，所以 `range` 里用 `math.Add` 数出来的数能直接和 `8` 比较；有一个是浮点数，结果就是 `float64`。`math.Div 7 2` 是 3，和 Go、Hugo 一样舍去余数，`math.Div 7.0 2` 是 3.5。`math.Mod` 和集合函数的个数参数也接受没有小数部分的浮点数。`math.Int`、`math.Float` 把数字或写成文字的数字（`"8"`）转成整数（舍去小数）或浮点数，没有设置的值转成 0。
- `collections.*` 接受任意切片和数组，如 `.Pages`（`[]Page`）、`str.Split` 的结果（`[]string`），返回的列表和传入的元素类型相同；`str.Join` 同样接受任意列表。`collections.Len` 还能数 map 和字符串（按字符），没有设置的值是 0，其余的值报错，而不是返回 0。`collections.Sort` 按大小排数字、按先后排时间，其余按字符串。

**以后可以加的函数**（设计阶段列过、v1 没有；加进来是只增，已有主题不受影响）：

```
str.Slugify  str.Pad  str.Markdownify  str.Plainify
coll.Where  coll.Shuffle  coll.Union  coll.Intersect  coll.Symdiff  coll.GroupBy  coll.GroupByDate  coll.Apply  coll.Seq
time.Since  time.Duration
url.Parse
img.Filter
asset.Get  asset.CSS  asset.JS  asset.Fingerprint  asset.Minify  asset.Bundle  asset.Inline   # 见 §10
i18n.Lang  i18n.Translate                                                                      # 见 §12
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
homepage: https://github.com/kite-plus/theme-paper
tags: [blog, minimal, dark-mode]
screenshot: screenshot.png

# ── 能力声明 ──
capabilities: [static, dynamic]     # 支持的 Runtime；缺省两者都支持
contentTypes: [post, page]          # 该主题能渲染的内容类型
taxonomies: [tags, categories]      # 该主题会渲染的分类轴

# ── 分页：按列表的种类（home / list / term），每页几条，0 是全部在一页 ──
# 没写的种类按站点的 build.pageSize；站点的 build.pagination 盖过这里。
pagination:
  home: 0                           # 首页不是一页页翻的文章列表
  term: 20

# ── 可选布局：作者按页面选用，front matter 写 layout: links ──
# 查找顺序同 §5：layouts/page/links.html，再 layouts/links.html。
# 声明了却没有模板文件的布局在加载主题时即报错；后台据此列出“模板”下拉框。
layouts:
  - name: links                     # 只能是文件名：小写字母、数字、- 与 _
    label: Links
    description: A list of links drawn as cards.
    types: [page]                   # 缺省即对所有内容类型提供

# ── 菜单：主题画哪些站点菜单，每个画几层（§6.9）──
# 名字只能是小写字母、数字和 _；depth 缺省是 1，最多 3。
menus:
  - name: main
    label: Header
    description: The links across the top of every page.
    depth: 1

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

  - key: links
    type: repeat
    label: 友情链接
    fields:
      - { key: text, type: string, label: 文字 }
      - { key: url,  type: url,    label: 链接 }
```

v1 读的就是上面这些键；其余的键被忽略，所以以后加一个键不会让旧的 Kite 拒绝新主题。主题的文字（`title`、`description`、设置、布局和菜单的说明）可以由 `i18n/<lang>.yaml` 的 `theme` 键翻译，见 §12。

### 8.2 Settings 字段类型（v1 基线） `[已冻结]`

| type | Admin 控件 | 模板取值 |
|---|---|---|
| `string` | 单行输入 | string |
| `text` | 多行输入 | string |
| `number` | 数字输入（`min`/`max`/`step`） | int 或 float64，保持写法 |
| `boolean` | 开关 | bool |
| `color` | 取色器 | string |
| `select` | 下拉（`options`） | string |
| `multiselect` | 多选 | []any，每项是 string |
| `image` | 上传图片或填地址 | string，图片的地址；站内地址用 `url.Rel` |
| `url` | URL 输入（带校验） | string |
| `date` | 日期和时间 | string |
| `code` | 代码编辑器（`language`） | string |
| `group` | 嵌套分组（`fields`） | map[string]any |
| `repeat` | 可增删的重复项（`fields`） | []any，每项是 map[string]any |
| `section` | 表单里的分组标题（`fields`），自己不存值 | —（其中的字段存在同一层） |

通用可选属性：`label` / `help` / `default` / `required` / `placeholder` / `showIf`（条件显示）。`color` 字段的 `options` 是推荐色，不限制取值。

**主题里的取值**：`{{ .Site.ThemeSettings.primary_color }}`

模板读到的是按字段类型解析过的值（`schema.Resolve`）：存储的值能读成声明的类型就用它，读不成就用默认值；`group` 总是一个 map，模板可以直接取 `.social.github`；主题没声明的键原样保留。`repeat` 还能读每行一条、字段按声明顺序用 `|` 分隔的文本，这样主题把一个 `text` 设置改成列表时，已经填好的站点不会丢内容（默认主题的 `nav` 就是这样从 `关于 | /about/` 升级过来的）。

通过 API 写入时按字段严格校验（`Field.Check`），写不进去的值在写文件之前就被拒绝；`null` 删除这个键，让主题的默认值重新生效，并跟随主题以后对默认值的修改。主题只能写自己声明过的键；同一个请求里切换了主题时，按新主题的 schema 校验。

> 同一套 schema 定义与渲染器被 **ContentType.Fields** 与 **Plugin Settings** 复用。一次投入，三处受益——这是"每加一种内容类型就要写一个 Admin 页面"的唯一解药。

### 8.3 校验时机

`apiVersion` 与 `requires` 在**安装时和每次构建时都校验**：

- 未知 `apiVersion` → **硬拒绝**，不是"尽力渲染"
- `requires` 不满足 → 拒绝并给出明确的升级/降级提示

Halo 的 `requires` 是已验证有效的模式 `[EV]`。

### 8.4 安装、切换与预览

- **来源**：内置的 `default`，加上项目 `themes/` 下的每个目录；`theme.name` 用目录名选主题，`default` 永远指内置主题。无法使用的主题也会列出来，并说明原因。
- **分发**：主程序只内置 `default`。其余主题，包括官方主题，各自一个仓库（官方的叫 `kite-plus/theme-<主题名>`），打 tag 发版并附上能直接在后台安装的 zip；站点装的是发布出来的版本，改主题回到主题自己的仓库去改。
- **安装**：后台上传 zip，`theme.yaml` 在最外层或压缩包里唯一的文件夹中。按站点加载主题的标准检查（`apiVersion`、`requires`、布局模板、语言包），主题名必须能当目录名且不能是 `default`；链接、跳出主题目录的路径一律拒绝，解压大小按实际字节计。同名主题要确认后才替换，替换时删掉新版本没有的文件；通过符号链接接入的主题不会被写穿。安装和删除都是 ChangeSet 里的操作（`PutTheme` / `DeleteTheme`），发布器因此像对待文章一样暂存 `themes/`。之后的 Git 地址安装复用同一个 `PutTheme`。
- **切换**：写 `theme.name`，写入前确认主题可用。设置统一放在 `theme.settings` 下、各主题共用，新主题只读取自己声明过的键，所以换回原主题时之前的设置还在。
- **预览**：`POST /api/v1/previews` 按主题和草稿设置组装一个变体站点（`site.With`），挂在 `/api/v1/previews/<token>/` 下，按构建的方式规划整站：链接在预览里互相跳转，样式表和图片用的是这套主题自己的。设置变化时原地重绘（`PUT`），30 分钟没人访问就回收，最多保留 8 个。
- **截图**：主题目录里的 `screenshot.png`、`.jpg` 或 `.webp`，或 manifest 里的 `screenshot:`；建议 1280×800。

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

见 [§7.2](#72-kitev1-的函数)。

### 9.4 `kite theme verify` —— **契约的真身**

```bash
kite theme verify ./themes/paper
```

行为：

1. 拿一个内置的 **fixture 站点**（`internal/themecheck/fixture`，发布在 `/blog/` 路径下）：single / list / taxonomy / term / 分页 / 404 / RSS / sitemap、带图片的 page bundle 和 `img.*` 做出的图、正文里的短代码、有 term 页与无 term 页、没有日期的文章、主菜单（含下级链接和站外链接）与另一个菜单，以及不该出现的草稿和定时文章；主题声明的每个布局各有一页用它
2. 在 **build 模式**下渲染全站
3. 在 **serve 模式**下逐 URL 请求 build 写出的每个文件
4. **逐字节 diff**，并报告从域名根开始写、漏掉站点路径的链接

> **那个测试才是契约本身，文档不是。** 文档会过时、会被误读；一个跑在 CI 里的黄金文件测试不会。
>
> `kite theme verify` **从 M0 就存在**（即使那时只有一套内置主题），不能等到 M5 才补。

`kite theme verify` 管的是「同一套主题在两种 Runtime 下画得一样」；「契约本身不被改坏」由契约清单管（§11.2）。多语言站点到来时夹具要加上第二种语言；有了废弃告警后再加 `--strict`，把告警升级为失败。

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
| `img.Resize` / `Fit` / `Fill` / `Crop` | 图片缩放、裁切 | ✅ |
| `img.Format` / `Quality` | WebP / JPEG / PNG / GIF 和质量；AVIF 暂不做 | ✅ |
| `asset.Get` | 从 `assets/` 取资源 | 以后 |
| `asset.Fingerprint` | 内容 hash 进文件名 | 以后 |
| `asset.Minify` | CSS/JS/HTML 压缩 | 以后 |
| `asset.Inline` | 内联进 HTML | 以后 |
| `asset.Bundle` | 多文件合并 | 以后 |
| SCSS / PostCSS | —— | **不做**，见 [§11.3](#113-明确不做的东西) |

**v1 没有 `asset.*`。** 主题的 `assets/` 原样发布到站点的 `/assets/`，`static/` 原样发布到站点根，模板用 `url.Rel "assets/main.css"` 链接它们。目前唯一的 transform 是 `img.*`，它做出的图的名字已经按上面的规则由源文件、整条做法和做图代码的版本算出（§6.7）；`asset.*` 以后加入时同样遵守 10.1，并且是只增。

**v1 的主题用纯 CSS。** 需要构建步骤的主题可以自己在发布前编译好，产物放进 `assets/` 或 `static/`。

### 10.3 带 fingerprint 的产物可以 `immutable`

`Cache-Control: public, max-age=31536000, immutable` —— 这是 fingerprint 的全部意义。`img.*` 做出的图已经是这样的名字；`asset.Fingerprint` 到来之后，样式表和脚本也一样。

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
| **M5** | **写第二套主题**（已完成） | 用第一套主题的契约去写一套风格完全不同的主题。**每一处别扭都是契约缺陷的证据**。实际写了两套，各在自己的仓库：文档站主题风标（`kite-plus/theme-vane`，原名司南）和个人站主题年鉴（`kite-plus/theme-almanac`）。写的时候发现的缺口都在冻结前补上了：列表页的 `Params` 和字数、图片处理、`T` 和语言包、按列表分页、短代码、render hook、声明内容类型、菜单 |
| **M5 末** | **冻结 `kite/v1`**（2026-10-01） | 契约的每一项连同签名记录在 `internal/render/theme/testdata/kite-v1.txt`，`TestTheContractOnlyGrows` 保证它只增不改；§6、§7 按实现逐项写明，设计阶段设想过、v1 没有的标为「以后可以加」；主题作者文档在 [reference.md 的 Themes](../reference.md#themes)；风标随后发布 1.0 |
| M5 之后 | 只增不改 | 新增方法/函数可以；改名/改语义要走 `kite/v2` |

新增一项时，同时在契约清单里加一行、在 §6 或 §7 写明它，并让用到它的主题把 `requires` 提到带这一项的 Kite 版本；契约测试在清单和实现不一致时失败，所以不会有人不经意地加上或删掉一项。

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

v1 只发一种语言，**冻结前定死的三件事都已定下**：

1. **Catalog 格式**（已实现）：go-i18n 风格的 YAML，放 `i18n/<lang>.yaml`，主题和站点各有一份，站点的盖过主题的。
2. **函数与上下文**：v1 有 `T "key" count`、`i18n.T`、`i18n.Has`、`i18n.Words` 和 `.Site.Language`。多语言站点到来时再加 `.Site.Languages`、`.Page.Language`、`.Page.Translations`，以及查找顺序里的语言后缀（`single.zh.html`，查找本身已经支持，§5.2）。这些名字现在就保留，加进来是只增。
3. **多语言 URL 策略**（2026-10-01 定）：**路径前缀，默认语言不加前缀。** 默认语言的页面就是现在的地址（`/posts/hello/`），其他语言在自己的前缀下（`/en/posts/hello/`）。所以现在的单语言站点以后加上第二种语言，已有页面的地址一个都不变。以后可以加配置改成子域（`en.example.com`），也是只增。

> **第 3 条最容易被忽略，代价也最大**：它影响**每一个页面的 `.Permalink`**。等有了多语言站点再决定，等于要求所有已存在的站点改 URL。选「默认语言不加前缀」，正是为了让现在发布的地址以后不用改。

```yaml
# i18n/zh-CN.yaml
read_more: 阅读全文
posts_count:
  one: "{{.Count}} 篇文章"
  other: "{{.Count}} 篇文章"
```

**主题在后台里的文字已经用上这套目录**：`theme` 键下放 `theme.yaml` 里给后台看的文字——`title`、`description`、`settings.<key>.label` / `help` / `placeholder` / `options.<value>`、`group` 和 `repeat` 的 `settings.<key>.fields.<子键>...`、`layouts.<name>.label` / `description`、`menus.<name>.label` / `description`；`section` 里的字段与 section 同层，直接用自己的键。API 按请求的 `Accept-Language` 选最接近的语言包（完全匹配 → 基础语言 → 同一语言的其他地区），语言包里没有的回退到 `theme.yaml` 原文。

**`theme` 以外的键是页面上的词，模板用 `T` 读**（已实现，一个站点一种语言）：

- `T "read_more"` 按站点的 `site.language` 选语言包，选法同上。站点自己的 `i18n/<lang>.yaml` 盖过主题的同名键，站点不用覆盖模板就能改主题的用词。语言包里没有的词用英文包里的，英文包也没有就是键本身。
- 要放进词里的值作为第二个参数：一个数字就是词里的 `.Count`，如 `T "posts" 8`；一个 map 原样交给词，如 `T "of" (dict "Count" 8 "Total" 13)` 配 `"{{ .Count }} of {{ .Total }}"`。带 `.Count` 时按这门语言的复数规则选 `one` / `few` / `many` / `other`，词缺那一种就用 `other`：英文 1 是 `one`；中文、日文、韩文等没有复数，总是 `other`；法文 0 和 1 是 `one`；俄文、乌克兰文、波兰文、捷克文、斯洛伐克文按各自的 CLDR 规则；其余语言按英文。
- `T` 返回的是文字，不是 HTML：落在哪里（正文、属性、脚本）就按那里的规则转义一次，所以能放进 `title=""` 或 `printf`。要带标记的词拆成几个词，把标记写在模板里。
- `i18n.Has "key"` 判断站点语言或英文里有没有这个词，给「有译名用译名，没有用原名」这类写法用。`i18n.Words "copy" "copied"` 返回这几个词的 map，写进 `<script>` 里就是一个 JSON 对象，交给主题的脚本；词里的占位符原样保留，由脚本自己替换。

默认主题就这样说它的词：`i18n/en.yaml` 和 `i18n/zh-CN.yaml`。页面标题里 Kite 用英文起的名字（如列表页 Posts），由语言包里的 `title.<kind>.<type>`（如 `title.list.post`）改写，没有就用 Kite 的名字。

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
| 契约靠文档维护 → 实现漂移无人发现 | —— | `kite theme verify` 黄金文件测试；契约清单 `testdata/kite-v1.txt` 和 `TestTheContractOnlyGrows` |
| 主题继承 → 覆盖语义爆炸 | 多个 CMS | 不做 |
| Term 不能带内容 → 后来补描述/封面是破坏性变更 | —— | Term 从一开始就可以有 `_index.md` |

---

## 14. 冻结前的问题及结论

这些没有在 M0 决定，M5 冻结前（2026-10-01）都有了结论：

1. **Shortcode 的语法与能力边界** —— 用 Hugo 的 `{{< >}}` 语法，`{{% %}}` 同样读取，从 Hugo 迁来的内容不用改写（[§6.8](#68-shortcode)）。分工：短代码是作者在正文里显式调用的模板，由站点或主题提供；插件的节点匹配器处理 Markdown 自己的节点，比如语言是 mermaid 的代码块。
2. **`partialCached` 的缓存键** —— v1 的 `partialCached` 就是 `partial`：接受并忽略 Hugo 写在数据之后的变体参数，不缓存。构建已经按目标并行渲染，serve 每个请求只画一页，缓存还没有实测的需要；真要缓存时，键从依赖记录推导，模板的写法不变。
3. **主题能否声明自己需要的 Hook / 插件** —— v1 不能。以后加的话只能是**建议**（后台在切换主题时提示），不能是硬依赖：主题在没有任何插件时也要能用。
4. **`.Site.Pages` 在 Serve 模式下的语义** —— v1 没有 `.Site.Pages`。列表、分类法和 term 页通过本页的 `.Pages`、`.Paginator`、`.Terms` 拿到条目，serve 只需规划这一页；首页也是列表。以后加 `.Site.Pages` 时，它是惰性的 PageList，对不带上限的遍历发出告警。
5. **暗色模式** —— 用约定，不加 API：
   - `<html>` 上的 `data-theme="dark"` 或 `"light"` 表示页面选定了深浅；没有这个属性时跟随系统的 `prefers-color-scheme`。
   - 读者的选择存在 `localStorage` 的 `kite-theme` 里（`light` 或 `dark`，没有就是跟随系统），由首屏绘制之前的脚本读出来写到 `data-theme` 上，避免闪一下浅色。
   - 想给站点设默认深浅的主题，用一个叫 `color_scheme` 的 `select` 设置，取值 `auto`、`light`、`dark`。
   - 插件按同样的顺序判断深浅：先看 `data-theme`，再看系统。官方的评论、公式和图表插件就是这样，并且在 `data-theme` 变化时重画。
   - 默认主题、风标和 `kite theme new` 生成的样式表都照此实现。

## 证据来源

- [Hugo 模板系统重写 PR #13541](https://github.com/gohugoio/hugo/pull/13541) —— `_default/` 移除、`layouts/partials` → `_partials`、`index.html` → `home.html` 等破坏性迁移
- [Hugo 新模板系统概览](https://gohugo.io/templates/new-templatesystem-overview/)
- [Hugo 旧查找顺序文档](https://gohugo.io/templates/lookup-order/) —— 六个互相作用的维度
- [Docsy 受影响 #2243](https://github.com/google/docsy/issues/2243) · [Hugo #13588](https://github.com/gohugoio/hugo/issues/13588) —— 重写对下游主题的实际冲击
- [Hugo fingerprint / PurgeCSS 缺陷 #11268](https://github.com/gohugoio/hugo/issues/11268) —— asset hash 未覆盖全部 transform
- [Halo 主题开发指南](https://docs.halo.run/developer-guide/theme/prepare) · [Halo theme-starter settings.yaml](https://github.com/halo-dev/theme-starter/blob/main/settings.yaml) —— `requires` 与 settings schema 驱动 Admin UI 的成功先例
