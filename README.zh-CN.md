<p align="center">
  <img src="docs/assets/logo.svg" alt="" width="84" height="84">
</p>

<h1 align="center">Kite</h1>

<p align="center">
  <strong>现代开源内容发布平台</strong><br>
  One Content, Multiple Destinations.
</p>

<p align="center">
  <a href="https://github.com/kite-plus/kite/actions/workflows/ci.yml"><img
    alt="CI"
    src="https://img.shields.io/github/actions/workflow/status/kite-plus/kite/ci.yml?branch=main&style=flat-square&logo=github&logoColor=white&label=CI&labelColor=1f2328"></a>
  <a href="go.mod"><img
    alt="Go"
    src="https://img.shields.io/github/go-mod/go-version/kite-plus/kite?style=flat-square&logo=go&logoColor=white&label=Go&labelColor=1f2328&color=4A77D6"></a>
  <a href="LICENSE"><img
    alt="License"
    src="https://img.shields.io/github/license/kite-plus/kite?style=flat-square&label=License&labelColor=1f2328&color=4A77D6"></a>
  <a href="docs/design/"><img
    alt="设计文档"
    src="https://img.shields.io/badge/design-docs-4A77D6?style=flat-square&labelColor=1f2328"></a>
</p>

<p align="center">
  <a href="README.md">English</a> · 简体中文
</p>

---

Kite 负责管理你的内容。至于部署到哪里，那是内容的一个属性，而不是另一个产品。

> **状态：早期开发中。** 写作、后台与 Git 发布已经端到端跑通；动态模式和插件
> 还没开始 —— 见 [路线图](#路线图)。

<p align="center">
  <img src="docs/assets/screenshot.png" alt="用 Kite 构建的站点，深浅两种配色" width="880">
</p>

## 为什么做这个

内容发布工具长期分成两个阵营，而且一旦选定就很难反悔。

**静态站点生成器**（Hugo、Hexo）给了你 Markdown、Git 和廉价托管，但没有真正的内容管理：写文章就是编辑文件，没有媒体库，没有分类浏览，也没有一个"发布"按钮。

**内容管理系统**（WordPress、Halo）把这些都给了你，但它们默认存在一台服务器和一个数据库。Git、Markdown 和静态托管塞进去总是别扭。

Kite 拒绝这道选择题。同一份内容、同一个后台、同一套主题，无论站点是构建成静态文件、由数据库驱动，还是作为 API 被消费，都照常工作。**换部署方式从此是改一行配置，而不是一次迁移。**

## 上手

```bash
go install github.com/kite-plus/kite/cmd/kite@latest
```

```bash
kite init blog && cd blog
kite new post "你好，Kite"
kite run
```

`kite run` 会启动站点、打开浏览器，保存即刷新，后台在 `/admin/`。想要文件而不是
一个服务时：

```bash
kite build
```

`public/` 下会得到一个完整站点：文章页、列表页、分页、分类页与标签页、404、`sitemap.xml` 和 `rss.xml`。

**已经有为别的生成器写的内容？直接指给 Kite 就行。**

```bash
kite doctor --fix-ids   # 给还没有 id 的文件补上
kite build --verify     # 构建两次，逐字节比对
```

`draft: true`、`date`、`lastmod`、`tags`、`categories` 都能原样读懂，不需要先改写任何文件。

### 命令

| | |
|---|---|
| `kite init [dir]` | 初始化项目 |
| `kite new <kind> <title>` | 新建内容 |
| `kite build` | 构建静态站点到 `public/` |
| `kite run` | 带后台启动站点、打开浏览器，保存即刷新 |
| `kite serve` | 启动站点，每个请求都从文件渲染 |
| `kite index` | 刷新派生索引 |
| `kite list` | 从索引里查询内容 |
| `kite doctor` | 体检，并修复可以安全修复的问题 |
| `kite publish` | 提交内容，需要时推送到远端 |
| `kite auth` | 管理守卫后台的账号 |
| `kite openapi` | 打印 API 描述文档 |
| `kite version` | 打印版本、commit 和构建时间 |

所有命令都支持 `--json`，不必把输出当作自然语言去解析。

## 后台

`kite run` 会在 `/admin/` 打开后台。它是一个编译进二进制的 React 应用，没有东西
要装，也没有东西需要和服务端对版本。

| | |
|---|---|
| **仪表盘** | 已发布多少、还有多少草稿、有哪些没提交，以及按月的发布趋势 |
| **内容** | 文章和页面，通过索引而不是文件系统来过滤和搜索 |
| **编辑器** | Markdown 实时预览、front matter 表单、分类标签、slug、字数，图片直接拖进 bundle |
| **分类法** | 标签和分类在全部内容里的真实分布 |
| **主题** | 当前主题在 `theme.yaml` 里声明的设置项，渲染成表单 |
| **设置** | 标题、描述、baseURL 和语言 |

加载之后又在磁盘上变过的内容，会被拒绝写入而不是覆盖，后台会明说这件事。后台
界面支持 English 和简体中文，按浏览器语言选择。

### 登录

在 localhost 上，没有设置密码的项目是敞开的 —— 这台机器上没有别人需要挡。换成
任何别的地址，后台就必须有账号：没有账号却要把它放到别人能访问的地址上时，服务
器会直接拒绝启动，而不是打一行警告了事。

```bash
kite auth set-password                          # 输入两次，不回显
kite serve --admin --write --addr 0.0.0.0:1717
```

`kite auth status` 会说明当前项目是否需要密码，`kite auth remove` 则把账号去掉。

账号以 argon2id 哈希的形式存放在 `.kite/secrets/account.json`，永远不会被提交，
也必须在部署时保留下来，账号才会跟着留下来。容器可以改用环境变量提供账号：
`KITE_ADMIN_USER` 配 `KITE_ADMIN_PASSWORD`，或者用 `KITE_ADMIN_PASSWORD_HASH`
以免明文密码出现在进程列表里；环境变量优先于文件。

### API

后台做的每一件事，都走 `/api/v1` 下的同一套 HTTP API，而这套 API 由二进制自己
描述：

```bash
kite openapi > openapi.json
```

`--admin` 提供这套 API，`--write` 允许它改动项目；不加 `--write` 时同一套 API
是只读的。后台的类型化客户端由这份描述生成，并在 CI 里校验，所以它编译时依赖的
类型不可能描述一个服务端并不提供的 API。

## 主题

Kite 自带一套主题，编译进二进制：为个人写作准备的安静衬线排版，深浅两色，不主动
引入任何 Web 字体 —— 除非你自己指定，否则页面不向第三方请求任何东西。

主题在 `theme.yaml` 里声明自己的设置项，后台把它们渲染成表单 —— 一个选项是一处
声明，而不是一个文档问题。主题和站点的模板都放在 `layouts/` 下，同名相对路径以
站点的为准，所以替换单个模板不需要 fork 整套主题。

主题契约尚未冻结；它会在 M5、也就是有了第二套按它写出来的主题之后再冻结。

## 配置

`kite.yaml` 放在项目根目录。除 `site` 外全部可选，下面写的就是默认值。

```yaml
site:
  title: My Site
  baseURL: https://example.com
  language: en

content:
  store: file          # 内容存在哪里
  dir: content

theme:
  name: default
  settings:            # 主题在 theme.yaml 里声明的那些
    accent: "#7d5c3c"

markdown:
  highlightTheme: github

build:
  output: public
  urlStyle: directory  # 或 extension，产出 /posts/hello.html
  pageSize: 10
  sitemap: true
  feed: true
  feedLimit: 20

publish:
  publisher: git
  branch: main
```

少数几个键可以用环境变量覆盖，供产出依赖运行环境的构建使用：`KITE_SITE_TITLE`、
`KITE_SITE_BASEURL`、`KITE_SITE_LANGUAGE`、`KITE_THEME`、`KITE_BUILD_OUTPUT`、
`KITE_BUILD_URLSTYLE` 和 `KITE_BUILD_PAGESIZE`。

## 部署

`kite init` 会写好一个 GitHub Pages 工作流，构建时带 `--verify` —— 跑第二次会得到
不同产物的站点，会在这里失败，而不是被发布出去。在 **Settings → Pages → Source →
GitHub Actions** 打开 Pages，之后推送到 `main` 即部署。

从本机发布则走 Git：

```bash
kite publish content/posts/hello --push
```

它只提交你给出的那些路径，别的一概不动：你暂存的东西还在暂存区，其余改动留在原
地。`--all` 会发布 Kite 管理范围内所有未提交的改动，`--dry-run` 则只报告会发生
什么然后停下。

## 设计

三条决策决定了其余的一切。

**Markdown 文件是唯一的真相源。** 静态模式下，数据库里不存在任何无法从文件推导出来的东西。`.kite/` 下的索引是缓存：删掉、重建，得到的行完全一样。

**是编辑你的文件，不是重写它。** 保存一篇内容时，只有真正变化的 key 会被改写。key 的顺序、注释、`[a, b]` 这样的行内列表全都原样保留 —— 改个标题，`git diff` 就只有一行。

**存储与运行时相互独立。** 内容存在哪里、以什么方式交付，是两个分开的选择，它们的每一种组合都成立。

完整的推理过程（包括哪些东西是**刻意没做**的）在 [docs/design](docs/design/)。

| 文档 | 内容 |
|---|---|
| [architecture.md](docs/design/architecture.md) | 内容模型、存储、构建引擎、发布器、路线图 |
| [theme-system.md](docs/design/theme-system.md) | 模板查找顺序、数据契约、`theme.yaml` |
| [plugin-system.md](docs/design/plugin-system.md) | WebAssembly 运行时、Host ABI、Capability |

> 设计文档为中文撰写，专有名词保留英文。

## 从源码构建

需要 Go 1.26 或更新版本：

```bash
make build      # ./bin/kite
make check      # 格式化、vet、分层规则、go.mod 整洁性、linter、测试
make web        # 后台界面，会被嵌入二进制
make web-gen    # 用这次构建自己的描述重新生成 API 客户端
```

`make web` 需要 Node 和 pnpm，两者版本都被精确钉死 —— 见 `web/.nvmrc` 和
`web/package.json`；其余目标两者都不需要。没有跑过它的二进制照样能用，只是访问
后台时会明说后台没有构建。

二进制是自包含的：默认主题和 SQLite 驱动都编译在内，不依赖 cgo，任何平台都能交叉编译出全部发布目标。

## 发布产物

Release 的二进制是可重现的：同一个 commit，用 `go.mod` 里钉死的工具链构建，在任何机器上都编译出相同的字节。

```bash
GOTOOLCHAIN=$(awk '/^toolchain /{print $2}' go.mod) goreleaser build --snapshot --clean
```

下载后请对照 release 附带的 `checksums.txt` 校验。

## 路线图

| 里程碑 | 交付内容 | |
|---|---|---|
| M0 | `kite build`：内容模型、索引、Markdown、主题、静态产出 | 已完成 |
| M1 | `kite serve`：按请求渲染，文件监听与热重载 | 已完成 |
| M2 | 只读后台，能打开现有仓库 | 已完成 |
| M3 | 可写后台：编辑器、媒体、冲突处理 | 已完成 |
| M4 | Git 发布器 —— **v1.0** | 已完成 |
| M5 | 公开主题契约 | |
| M6 | `kite.lock` 与 `kitew` wrapper | |
| M7 | 基于 SQLite 的动态模式 | |
| M8 | WebAssembly 插件 | |

## 参与贡献

`scripts/check-imports.sh` 里的分层规则由 CI 强制执行：领域核心不得 import 存储、渲染或运行时包。提交信息遵循 [Conventional Commits](https://www.conventionalcommits.org/zh-hans/v1.0.0/)。

提 PR 之前请先跑：

```bash
make check
```

如果动过后台，`make web-check` 做类型检查，`make web` 构建 CI 会拿来比对的产物。

## 许可证

[Apache License 2.0](LICENSE)。
