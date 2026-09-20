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

> **状态：早期开发中。** 静态链路已端到端跑通；后台界面和动态运行时尚未开始 —— 见 [路线图](#路线图)。

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
| `kite init` | 初始化项目 |
| `kite new <kind> <title>` | 新建内容 |
| `kite build` | 构建静态站点到 `public/` |
| `kite index` | 刷新派生索引 |
| `kite list` | 从索引里查询内容 |
| `kite doctor` | 体检，并修复可以安全修复的问题 |

所有命令都支持 `--json`，不必把输出当作自然语言去解析。

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
make check      # 格式化、vet、分层规则、linter、测试
```

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
| M1 | `kite serve`：按请求渲染，文件监听与热重载 | |
| M2 | 只读后台，能打开现有仓库 | |
| M3 | 可写后台：编辑器、媒体、冲突处理 | |
| M4 | Git 发布器 —— **v1.0** | |
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

## 许可证

[Apache License 2.0](LICENSE)。
