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
| **主题** | 项目里的全部主题，启用前都能在整站上预览；上传 zip 安装主题；在实时预览旁边调整当前主题的设置 |
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

其他主题放在 `themes/` 下，每套一个目录，`kite.yaml` 里的 `theme.name` 用目录名
选择它；`default` 永远指内置主题。后台会列出全部主题，说明哪些无法使用以及原因；也可以
上传 zip 压缩包安装主题，`theme.yaml` 放在最外层或压缩包里唯一的文件夹中。安装时按站点
加载主题的标准检查，同名主题只有在你确认后才会被替换。启用之前可以先在整站上试用：预览
用这套主题和正在编辑的设置绘制，页面里的链接都留在预览中，保存之前什么都不会写入。切换
写下的 `kite.yaml` 和主题目录，在设置页里像内容一样发布。

主题在 `theme.yaml` 里声明自己的设置项，后台把它们渲染成表单 —— 一个选项是一处
声明，而不是一个文档问题。主题和站点的模板都放在 `layouts/` 下，同名相对路径以
站点的为准，所以替换单个模板不需要 fork 整套主题。

```yaml
settings:
  - key: look
    type: section          # 表单里的一个分组标题；其中的字段仍存在同一层
    label: Look
    fields:
      - key: accent
        type: color
        label: Accent color
        default: "#7d5c3c"
        options:           # 颜色字段的 options 是推荐色，不是限制
          - {value: "#7d5c3c", label: Umber}
      - {key: favicon, type: image, label: Site icon}
  - key: nav
    type: repeat           # 由若干项组成的列表，每项有下面这些字段
    label: Extra links
    fields:
      - {key: label, type: string, label: Label}
      - {key: url, type: url, label: Address}
```

字段类型有 `string`、`text`、`number`、`boolean`、`color`、`select`、`multiselect`、
`image`、`url`、`date`、`code`、`group`、`repeat` 和 `section`。设置值存在 `kite.yaml`
的 `theme.settings` 下，恢复成默认值的设置会从中删除。模板按字段声明的类型读取每个值，
写作 `.Site.ThemeSettings.accent`，读不成该类型的值就用默认值；`repeat` 还能读取每行
一条 `名称 | /路径/` 的文本，所以主题把文本设置改成列表时，已经填好的站点不会丢内容。

主题在后台里的说明文字由主题 `i18n/` 目录下的语言包翻译，每种语言一个文件，放在
`theme` 键下；语言包里没有的部分按 `theme.yaml` 的原文显示：

```yaml
# i18n/zh-CN.yaml
theme:
  title: 纸
  settings:
    accent: {label: 强调色, options: {"#7d5c3c": 赭石}}
    nav:
      label: 额外链接
      fields: {url: {label: 地址}}
  layouts:
    links: {label: 友链}
```

主题目录里放 `screenshot.png`、`.jpg` 或 `.webp` 作为截图，也可以在 `theme.yaml` 里用
`screenshot:` 指定其他文件。

主题还可以提供让作者按页面选用的模板，比如友链页，在 `theme.yaml` 里声明：

```yaml
layouts:
  - name: links
    label: Links
    description: 把一组链接排成卡片。
    types: [page]        # 不写则所有类型都可以选
```

内容在 front matter 里用 `layout: links` 选用，和 Hugo 的写法一样；也可以在
编辑器的“模板”下拉框里选，它列出当前主题为这类内容提供的模板，预览随之切换。页面随后用
`layouts/page/links.html` 渲染，没有的话用 `layouts/links.html`。主题不能声明
没有模板文件的布局；页面选了当前主题没有的模板时，退回它所属类型的默认模板。

主题可以对照它所依据的契约检查：

```bash
kite theme verify ./themes/paper
```

它用这套主题构建一个用到每种页面的小站点，再向服务器请求构建写出的每个文件，包括
RSS 和 sitemap，逐字节比较。通过检查的主题，发布出去的就是 `kite run` 预览时看到的；
没通过的，会指出每个文件第一处不同的行。不给目录时，检查当前项目在用的主题，在项目之外则检查内置主题。

这个小站点发布在一个路径下，就像 GitHub Pages 的项目站点那样，所以从域名根开始写的链接，
比如 `/rss.xml`，也会被报告出来。模板链接到 Kite 自己的页面用 `url.For "home"`、
`url.For "list" "post"`、`url.For "taxonomy" "tags"` 或 `url.For "term" "tags" "Go"`，
链接到站点的其他路径用 `url.Rel "rss.xml"`，两者都会带上这段路径。

主题契约尚未冻结；它会在 M5、也就是有了第二套按它写出来的主题之后再冻结。

## 配置

`kite.yaml` 放在项目根目录。除 `site` 外全部可选，下面写的就是默认值。

```yaml
site:
  title: My Site
  description: ""      # 用于搜索结果和订阅，主题也常把它显示出来
  baseURL: https://example.com
  language: en
  author: ""
  keywords: []         # 列表，或者用逗号隔开写成一行
  timezone: ""         # IANA 时区，例如 Asia/Shanghai
  noindex: false       # 设为 true 时要求搜索引擎不要收录
  headHTML: ""         # 插到每个页面的 </head> 之前
  footerHTML: ""       # 插到每个页面的 </body> 之前

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

`timezone` 决定日期落在哪一天。不设置时，日期按写入时的时区显示，后台写入的是 UTC，
所以在上海刚过零点发布的文章会显示成前一天。站点的关键词、作者、`noindex` 和自定义代码
由主题写进每个页面，模板里对应 `.Site.Keywords`、`.Site.Author`、`.Site.NoIndex`、
`.Site.HeadHTML` 和 `.Site.FooterHTML`，默认主题都写了。单个页面可以在 front matter 里
用 `keywords` 写自己的关键词。这些设置，连同每页文章数和订阅文章数，都可以在后台的
设置 → 站点里修改。

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

在有自己的域名之前，仓库的站点位于 `https://<owner>.github.io/<仓库名>/`。把这个地址
填为 `baseURL`：Kite 生成的每个链接都会带上这段路径，`kite serve` 也会在这个路径下预览。

定时文章由另一个工作流 `scheduled.yml` 发布。日期在未来的文章，状态是 `scheduled`
还是 `published` 都一样，要等到那个时间才公开，这和 Hugo、Jekyll 的做法相同，
从它们迁过来的站点里排在未来的文章不会提前上线。每次构建都会记下下一篇定时文章的
时间，这个工作流每小时检查一次，时间已过才部署，所以定时文章会在设定时间之后的
一小时内上线，没有文章到点的那一小时只跑一个很短的检查。私有仓库里每次检查按
1 分钟的 Actions 时长计费，想少查几次，改它的 `cron` 一行即可。

公开仓库 60 天没有提交时，GitHub 会关掉定时工作流，需要到 **Actions** 页面重新打开。
推送时的部署正是因此单独放在另一个工作流里，不受影响。

后台会跟踪一次发布从提交、推送到部署的全过程。对于部署到 Pages 的 GitHub 公开仓库，
它会匿名、只读地调用 GitHub API，查询推送的那个提交是否已经上线，上线后给出站点
链接。其他托管平台不回报部署状态，后台会直接说明，而不是一直等待。

`kite build` 会显示下一篇定时文章的时间。换用别的托管平台时，那就是需要重新构建的
时间：静态站点只有在文章的时间之后构建过，才会出现这篇定时文章。

从本机发布则走 Git：

```bash
kite publish content/posts/hello --push
```

它只提交你给出的那些路径，别的一概不动：你暂存的东西还在暂存区，其余改动留在原
地。`--all` 会发布 Kite 管理范围内所有未提交的改动，`--dry-run` 则只报告会发生
什么然后停下。

推送从不强制。远端有这个分支没有的提交时，提交留在本地，并列出远端的那些提交。
如果它们都没有改到这次发布的文件，`kite publish --push --rebase`（或后台显示的按钮）
会把你的提交接在它们后面再推送，工作区里的其他改动一概不碰；如果改到了同样的文件，
会显示远端那一侧的改动，怎么合由你决定。单独运行 `kite publish --push` 会推送已经
提交的内容，用于第一次没推送成功的情况。

仓库里的提交 hook 会像任何一次提交那样运行。hook 拒绝时，后台会显示它给出的理由，
并提供「跳过 hooks 发布」；在终端里用 `kite publish --no-verify` 效果相同。

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

**是编辑你的文件，不是重写它。** 保存一篇内容时，只有真正变化的 key 会被改写。key 的顺序、注释、`[a, b]` 这样的行内列表全都原样保留 —— 改个标题，`git diff` 就只有一行。front matter 可以是写在 `---` 之间的 YAML，也可以是 Hugo 那样写在 `+++` 之间的 TOML；TOML 文件保存后仍是 TOML。

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
make perf       # 用 2000 篇的站点对照设计里的时延目标计时
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

