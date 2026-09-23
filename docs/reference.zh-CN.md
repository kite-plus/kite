# Kite 使用与开发参考

[返回 README](../README.zh-CN.md)

## 后台

`kite run` 会在 `/admin/` 打开后台。它是一个编译进二进制的 React 应用，没有东西
要装，也没有东西需要和服务端对版本。

| | |
|---|---|
| **仪表盘** | 已发布多少、还有多少草稿、有哪些没提交，以及按月的发布趋势 |
| **内容** | 文章和页面，通过索引而不是文件系统来过滤和搜索 |
| **编辑器** | 可视化编辑，读写的都是 Markdown，一键切到源码；实时预览、front matter 表单、分类标签、slug、字数，图片直接拖进 bundle |
| **分类法** | 标签和分类在全部内容里的真实分布 |
| **主题** | 当前主题在 `theme.yaml` 里声明的设置项，渲染成表单 |
| **设置** | 标题、描述、baseURL 和语言 |

加载之后又在磁盘上变过的内容，会被拒绝写入而不是覆盖，后台会明说这件事。后台
界面支持 English 和简体中文，按浏览器语言选择。

### 登录

在 localhost 上，没有设置密码的项目是敞开的 —— 这台机器上没有别人需要挡。换成
任何别的地址，后台就必须有账号；没有账号却要放到别人能访问的地址上时，服务器
不会敞着启动。

它会以**安装模式**启动：除了安装页，什么都不会应答 —— 内容不会、草稿不会、设置不会，
连站点自己叫什么都不会，所以一个没装好的服务器什么也交不出去。

```bash
kite serve --admin --write --addr 0.0.0.0:1717
```
```
  Finish installing this site in a browser:

    http://localhost:1717/admin/setup
```

安装页本身是开放的，这是刻意的：装完之前根本没有账号，也就没有任何东西可以拿来
校验请求，最先打开表单的那个浏览器就是拿到账号的人。要么尽快装完，要么在它可达
之前就先给它一个账号，完全跳过安装：

```bash
kite auth set-password                          # 输入两次，不回显
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

站点可以构成静态文件托管在任何地方，也可以作为一个自己管自己的服务跑着。两边的内容
是同一份，所以这是一个可以改主意的决定。

### 静态，发到 GitHub Pages

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

### 用 Docker

```bash
docker build -t ghcr.io/kite-plus/kite:latest .
docker compose up -d
docker compose logs kite      # 它会打印去哪里把安装走完
```

然后打开 `http://localhost:1717/admin/`，在浏览器里把安装走完。配置就只有
[`docker-compose.yaml`](../docker-compose.yaml) 这一份。

镜像是 `ghcr.io/kite-plus/kite`，提供 amd64 和 arm64 两个架构。里面只有二进制、后台
和 git，以非 root 用户运行，自己不存任何东西：站点住在 `/data` 卷里，空卷会在第一次
启动时变成一个新项目。其余都是普通的 `kite` 命令：

```bash
docker compose run --rm kite build
docker compose run --rm kite publish --all --push
docker compose run --rm kite auth set-password
```

值得提前给好的只有 `KITE_SITE_BASEURL`：它是会进入订阅源和站点地图的那个地址，
而那不是容器自己的地址。

### 不用 Docker，直接跑在服务器上

```bash
kite serve --admin --write --addr 0.0.0.0:1717
```

第一次启动会打印一个链接并等浏览器，和容器里一模一样 —— 见[登录](#登录)。

Kite 自己不终结任何 TLS，请把它放在一个能做这件事的东西后面。密码在网络上裸奔，
并不会因为它到了另一头会被哈希而变得安全。

## 设计

三条决策决定了其余的一切。

**Markdown 文件是唯一的真相源。** 静态模式下，数据库里不存在任何无法从文件推导出来的东西。`.kite/` 下的索引是缓存：删掉、重建，得到的行完全一样。

**是编辑你的文件，不是重写它。** 保存一篇内容时，只有真正变化的 key 会被改写。key 的顺序、注释、`[a, b]` 这样的行内列表全都原样保留 —— 改个标题，`git diff` 就只有一行。

**存储与运行时相互独立。** 内容存在哪里、以什么方式交付，是两个分开的选择，它们的每一种组合都成立。

完整的推理过程（包括哪些东西是**刻意没做**的）在 [docs/design](design/)。

| 文档 | 内容 |
|---|---|
| [architecture.md](design/architecture.md) | 内容模型、存储、构建引擎、发布器、路线图 |
| [theme-system.md](design/theme-system.md) | 模板查找顺序、数据契约、`theme.yaml` |
| [plugin-system.md](design/plugin-system.md) | WebAssembly 运行时、Host ABI、Capability |

> 设计文档为中文撰写，专有名词保留英文。

## 从源码构建

需要 Go 1.26 或更新版本：

```bash
make build      # ./bin/kite
make check      # 格式化、vet、分层规则、go.mod 整洁性、linter、测试
make web        # 后台界面，会被嵌入二进制
make web-gen    # 用这次构建自己的描述重新生成 API 客户端
make docker     # 容器镜像，上面两样东西它会自己编译
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
| M4 | Git 发布器 —— **v1.0** | 已完成，打标签前收尾中 |
| M5 | 公开主题契约 | |
| M6 | `kite.lock` 与 `kitew` wrapper | |
| M7 | 基于 SQLite 的动态模式 | |
| M8 | WebAssembly 插件 | |

[路线图与实现现状](design/roadmap.md)记录了逐项核实过的完成情况、打 v1.0 标签前的收尾清单，以及之后的计划。

## 参与贡献

`scripts/check-imports.sh` 里的分层规则由 CI 强制执行：领域核心不得 import 存储、渲染或运行时包。提交信息遵循 [Conventional Commits](https://www.conventionalcommits.org/zh-hans/v1.0.0/)。

提 PR 之前请先跑：

```bash
make check
```

如果动过后台，`make web-check` 做类型检查，`make web` 构建 CI 会拿来比对的产物。

