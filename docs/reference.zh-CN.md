# Kite 使用与开发参考

[返回 README](../README.zh-CN.md)

## 后台

`kite run` 会在 `/admin/` 打开后台。它是一个编译进二进制的 React 应用，没有东西
要装，也没有东西需要和服务端对版本。在空文件夹里运行时，它会先打开一个建站页面，
填好站点名称、你的名字、地址和语言就在这里建好站点，问的就是 `kite init` 在终端里
问的那些。你的名字就是站点的作者。

| | |
|---|---|
| **仪表盘** | 已发布多少、还有多少草稿、有哪些没提交，以及按月的发布趋势 |
| **内容** | 文章和页面，通过索引而不是文件系统来过滤和搜索 |
| **编辑器** | 可视化编辑，读写的都是 Markdown，一键切到源码；实时预览、front matter 表单、分类标签、slug、字数，图片直接拖进 bundle，“附件”里列出 bundle 的文件和正文用到了哪些，可以复制链接、原地替换或删除；草稿边写边自动保存，还没保存的内容先留在浏览器里 |
| **分类法** | 标签和分类在全部内容里的真实分布 |
| **主题** | 项目里的全部主题，启用前都能在整站上预览；上传 zip 安装主题；在实时预览旁边调整当前主题的设置 |
| **部署** | 把网站导出成 zip，上传到任何地方；或者查看推送到 GitHub Pages 进行到了哪一步 |
| **设置** | 站点的标题、描述、地址、语言、关键词、时区、搜索引擎收录和全站附加代码；后台自己的语言和配色；你的账户：名称、头像、密码和登录会话 |

加载之后又在磁盘上变过的内容，会被拒绝写入而不是覆盖，后台会明说这件事。后台
界面支持 English 和简体中文，按浏览器语言选择。

在后台添加的照片，保存之前会去掉拍摄位置，因为存下的文件都会发布：JPEG、PNG、WebP
和 AVIF 里 EXIF 的 GPS 信息和 XMP 里标明位置的部分，还有一些相机藏在 JPEG 里的预览图
上的同样信息。照片怎么转正、用什么拍的、什么时候拍的都保留，像素不动；去掉了位置时，
后台会提示。用 git 添加的文件原样发布。

标签和分类按站点列出它们的方式计数：Go、go 和 GO 是同一个标签，名称取写得最多的
那种，卡片上会注明它还有哪些写法。给标签改名，会把新名称写进每篇带着它的文章，不管
原来怎么写，所以把 Go 改名为 Go 就能统一写法；删除时每种写法都会去掉。在编辑器里
换一种写法输入已有的标签，会直接选中它，而不是新建一个。

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

这些也可以在后台的 **设置 → 账户** 里完成。在 localhost 上敞开的后台可以在那里
设置密码，设置后立刻生效，不用重启。已经有密码的后台，确认当前密码后可以修改
用户名和密码，其他浏览器随之退出登录，当前浏览器保持登录；也可以什么都不改，只让
其他浏览器退出；在 localhost 上还能把密码移除。来自环境变量的账号，要在设置环境
变量的地方修改。

同一个页面里还有你的名称、邮箱和头像。名称就是站点的作者 `site.author`，主题会把它
显示在你发布的内容旁边，所以它写在 `kite.yaml` 里，和其他设置一样要发布；后台也用它
称呼你。邮箱和头像和账号放在一起，存在 `.kite/secrets/profile.json` 和
`.kite/secrets/avatar`，永远不会发布。

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
是只读的，后台会在站点名旁标出“只读”，只提供浏览和预览。后台的类型化客户端由这份描述生成，并在 CI 里校验，所以它编译时依赖的
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
    preview: home          # 编辑这个分区时后台预览哪个页面
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
`image`、`url`、`date`、`code`、`group`、`repeat` 和 `section`。后台把主题的分区列在
表单旁边，分区的 `preview` 指定编辑它时预览哪个页面：`home`（首页）、`post`（最新的
文章）、`page`（一个独立页面）或 `posts`（文章列表）。设置值存在 `kite.yaml` 的
`theme.settings` 下，恢复成默认值的设置会从中删除。模板按字段声明的类型读取每个值，
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

同一套语言包里 `theme` 以外的键，是主题页面上的词，模板用 `T` 读：`{{ T "read_more" }}`。
词取自站点语言对应的语言包，站点自己的 `i18n/<语言>.yaml` 盖过主题的，所以站点不用替换模板
就能改主题的用词；这门语言里没有的词用英文的，都没有就是键本身。词里可以放模板给的值，
带数量时按复数规则选形式：

```yaml
# i18n/en.yaml
posts:
  one: "{{ .Count }} post"
  other: "{{ .Count }} posts"
of: "{{ .Count }} of {{ .Total }}"
```

`{{ T "posts" 8 }}` 是 `8 posts`，`{{ T "of" (dict "Count" 8 "Total" 13) }}` 是
`8 of 13`。词是文字，落在哪里就在哪里转义，所以能放进属性里。`i18n.Has "key"` 说明有没有
这个词，`{{ i18n.Words "copy" "copied" }}` 把几个词作为一个 JSON 对象交给脚本。内置主题
带英文和中文的词；用其他语言写的站点，加一个自己的语言包就能翻译它。

主题目录里放 `screenshot.png`、`.jpg` 或 `.webp` 作为截图，也可以在 `theme.yaml` 里用
`screenshot:` 指定其他文件。

页面里的代码按类名而不是颜色高亮，主题的样式表可以为浅色和深色各带一套配色。一个代码块写出来是
`<pre class="chroma" data-lang="go">`，`data-lang` 是作者标注的语言，主题可以用它给代码块加上标题。

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

站点的菜单写在 `kite.yaml` 里，不属于哪一个主题，换了主题还在。站内地址从站点根写起，
发布时放在站点的路径下：

```yaml
menus:
  main:
    - name: 归档
      url: /posts/
    - name: 关于
      url: /about/
    - name: 别处            # 只用来归拢下级链接的一项
      children:
        - {name: 代码, url: "https://github.com/someone"}
```

主题在 `theme.yaml` 里声明它画哪些菜单、每个画几层，后台的“菜单”页据此列出要填的菜单，
页面和文章可以按标题搜索添加：

```yaml
menus:
  - name: main
    label: Header
    description: 每一页顶部的链接。
    depth: 1                 # 2 表示链接可以展开下级菜单
```

模板用 `{{ range .Site.Menus.main }}` 画它：每个链接有 `.Name`、已经带上站点路径的
`.URL`、`.Children`，以及 `.Params`，即站点给它的其他东西，比如图标。站点没写的菜单是空的。
内置主题在页头画 `main`，站点写这个菜单之前，显示它自己的链接。

主题也可以在命令行里管理。命令检查主题的方式和后台一样，写下的改动也一样，之后照常发布：

```bash
kite theme list                  # 站点能用的主题，* 标出正在用的
kite theme add vane-0.2.0.zip    # 发布附带的压缩包，或一个目录；--replace 替换同名的
kite theme use vane              # 写 theme.name
kite theme remove paper          # 正在用的不能删
kite theme new paper             # 生成一个起步的主题目录
```

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

写法不同但地址相同的词条是同一个词条：Go、go 和 GO 都是 `/tags/go/`，Web Dev 和
web-dev 都是 `/tags/web-dev/`。它的页面列出带着其中任何一种写法的文章，名称取写得最多
的那种；各种写法一样多时，按字符顺序取第一个，Go 排在 go 前面。分类页的 `.Terms` 只把
它算一次，文章自己的 `.Terms` 按这篇文章的写法显示。只由短横线、斜杠或空格组成的词条
没有自己的页面。

以 bundle 保存的页面，它的文件在 `.Resources` 里，按 bundle 里的名字取：
`.Resources.Get "cover.jpg"`、`.Resources.Match "images/*"` 或 `.Resources.ByType "image"`。
其中的图片可以做成另一张：更小、裁过或者换格式，主题就这样给列表配小封面，或者给图片写
`srcset`：

```html
{{ with .Resources.Get "river.jpg" }}
  {{ $small := img.Fit "800x800" . }}
  {{ $card := . | img.Fill "600x400" | img.Format "webp" | img.Quality 80 }}
  <img src="{{ $small.RelPermalink }}" width="{{ $small.Width }}" height="{{ $small.Height }}">
{{ end }}
```

`img.Resize "800x"` 缩放到这个尺寸，缺的一边按比例；`img.Fit` 只缩小，放进这个框；
`img.Fill "600x400 top"` 先按比例裁、再缩放到正好这个尺寸，锚点指定保留哪一部分；
`img.Crop` 只裁不缩；`img.Format` 写成 `webp`、`jpeg`、`png` 或 `gif`；`img.Quality` 是 WebP
或 JPEG 的质量，不写就是 75。照片先按 EXIF 转正，做出的图不带任何 EXIF，也就不会说出拍摄
地点。JPEG、PNG、GIF 和 WebP 都能读写，用的是 Kite 自己的代码，在每台机器上做出同样的字节，
所以在笔记本上和在 CI 里构建的站点发布的是同样的文件。WebP 是有损压缩，保留透明；写成 JPEG
时透明的部分铺成白色。每张图只做一次：模板第一次问它的地址或尺寸时才做，发布在
源文件旁边，名字是 `river_<key>.jpg`，并保存在 `.kite/cache/images/`，之后的构建和
`kite serve` 直接用。

正文里的图片，站点或主题有 `layouts/_markup/render-image.html` 时由它来画，和 Hugo 一样；
手机拍的照片就这样缩小后再发布：

```html
{{- with .Page.Resources.Get .Destination -}}
  {{- with img.Fit "1600x1600" . -}}
  <img src="{{ .RelPermalink }}" width="{{ .Width }}" height="{{ .Height }}" alt="{{ $.Text }}">
  {{- end -}}
{{- else -}}
  <img src="{{ .Src }}" alt="{{ .Text }}"{{ with .Title }} title="{{ . }}"{{ end }}>
{{- end -}}
```

`.Destination` 是正文里写的图片地址，写的是 bundle 里的文件时，正好是 `.Resources.Get` 要的名字；
`.Src` 是没有这个模板时页面引用它的地址，正文从站点根开始写的，前面带上站点的路径。
`.Text` 是图片的替代文字，`.Title` 是标题。

文章的封面是 front matter 里的 `cover`，模板从 `.Params.cover` 读到的就是作者写的原文；
`.Images` 按出现顺序列出正文里的图片，也是原文。列表里的页面同样带着这两样。主题解析它们的方式，
和浏览器解析正文里的图片一样：完整地址原样用，从站点根开始写的用 `url.Rel`，其余的相对于页面地址。
没写封面时，主题可以改用正文第一张图；写了 `cover: false` 就不显示封面，编辑器里的「不用封面」写的就是它。

主题可以按列表的种类规定分页，用在设计需要的地方：首页不是一页页翻的文章列表，或者归档页
要列出所有文章。

```yaml
pagination:
  home: 0    # 全部显示在一页
  list: 0
  term: 20   # 每页 20 条
```

没写到的种类按站点的 `build.pageSize` 分页，站点的 `build.pagination` 可以替换其中任何
一种。`.Paginator` 描述的是页面最终的分页；全部显示在一页的列表是第 1 页、共 1 页，
`PageSize` 是条目的数目。

主题契约尚未冻结；它会在 M5、也就是有了第二套按它写出来的主题之后再冻结。

## 短代码

短代码按名字调用一个模板，把 Markdown 没有写法的东西放进页面，比如视频、相册或提示框。
语法和 Hugo 相同，所以从 Hugo 迁过来的内容，只要有了对应的模板就照常可用：

```markdown
{{< figure src="river.jpg" caption="上游" >}}

{{< note title="注意" >}}
一对标签中间的 **Markdown** 也会渲染。
{{< /note >}}

按 {{< kbd Enter >}} 继续。
```

`{{% %}}` 的读法与此相同。单独占一行的标签是一个块，不会被包进段落，一对这样的标签
包住的是中间的块；写在一行文字当中的标签是这行里的一个词，一对这样的标签包住的是中间的
文字。参数按顺序给，如 `{{< kbd Enter >}}`，或者按名字给，如
`{{< figure src="river.jpg" >}}`，不能混用。带引号的值是文字；不带引号的值读得出
`true`、`false` 或数字时，就是那个值。代码里的标签原样显示；`{{</* figure */>}}`
在任何地方都显示成它注释掉的那个标签，写介绍短代码的文章时就这样写。

模板是 `layouts/_shortcodes/<名字>.html`，放在站点或主题里，站点的优先；名字里可以
带目录，`docs/note` 对应 `layouts/_shortcodes/docs/note.html`。模板拿到的是这一次调用：

```html
<!-- layouts/_shortcodes/note.html -->
<aside class="note">
  {{ with .Get "title" }}<strong>{{ . }}</strong>{{ end }}
  {{ .Inner }}
</aside>
```

- `.Get` 按位置（`.Get 0`）或按名字（`.Get "src"`）取一个参数，调用里没有给时什么也
  不返回。`.Params` 是全部参数，`.IsNamedParams` 说明参数是按哪种方式给的。
- `.Inner` 是一对标签包住的内容，按 Markdown 渲染好；`.RawInner` 是它的原文，给把它
  当成别的东西来读的短代码用，比如图表。
- `.Parent` 是这次调用所在的外层短代码，`.Ordinal` 是它在外层里排第几个，从 0 数起。
- `.Page` 是正文里写了这次调用的页面，`.Site` 是站点。页面的正文还在绘制当中，所以
  `.Page.Content` 是空的。模板能调用页面模板能调用的所有 partial。

只有模板显示出来的内容才算数：模板没有用到的 `.Inner`，其中的字不计入页面的字数、
正文和摘要，所以留空的 `layouts/_shortcodes/private.html` 能让它包住的内容不出现在
网站上。没有人定义的短代码会让构建停下，并指出所在的文件和行，而不是把标签原样印出来；
写作时后台的预览也会同样提示。可视化编辑器无法保留短代码，所以用到短代码的内容会以
Markdown 源码打开。

## 插件

插件为网站加上主题之外的功能：评论、统计、搜索、公式。插件放在 `plugins/` 下，每个
一个目录，列进 `kite.yaml` 的 `plugins.enabled` 后才会运行，列表的顺序就是运行顺序。
后台的「插件」页可以上传 zip 安装插件、开关、修改设置、删除；开启之前会说明插件往页面里
加什么、它的代码会从哪些网站加载内容。命令行也能做同样的事：

```bash
kite plugin add search-0.1.0.zip   # 也可以是一个目录
kite plugin enable search
kite plugin list
kite plugin disable search
kite plugin remove search
```

官方插件和默认主题以外的主题一样，各自放在独立的仓库里：

| 插件 | 作用 |
|---|---|
| [analytics](https://github.com/kite-plus/plugin-analytics) | 用百度统计、Google Analytics、Umami 或 Plausible 统计访问量 |
| [comments](https://github.com/kite-plus/plugin-comments) | 在文章下放评论区，支持 Giscus、Waline 和 Twikoo |
| [math](https://github.com/kite-plus/plugin-math) | 用 KaTeX 排版 TeX 公式，把 mermaid 代码块画成图表 |
| [search](https://github.com/kite-plus/plugin-search) | 在读者的浏览器里搜索，索引在构建时生成 |

### 编写插件

```bash
kite plugin new greet     # 生成一个起步目录
kite plugin verify greet  # 按站点加载插件的标准检查
```

插件是一个带 `plugin.yaml` 的目录。`assets/` 里的文件随网站发布到 `plugins/<id>/`
下；`i18n/` 里的语言包放在 `plugin` 键下，翻译后台对插件的描述，和主题的语言包一样。

```yaml
id: greet                  # 与目录名相同
name: Greet
version: 0.1.0
apiVersion: kite/plugin/v1
requires: ">=0.1.0 <2.0.0" # 适用的 Kite 版本
description: A line under every post.
hosts: [cdn.example.com]   # 插件自带脚本会从哪些网站加载内容

inject:
  - at: head               # 放在 </head> 之前；body 放在 </body> 之前
    html: <link rel="stylesheet" href="{{ asset "greet.css" }}">
  - at: body
    pages: [single]        # home、single、list、taxonomy、term、notFound
    kinds: [post]          # 只放进这些内容类型的单页
    when: {style: plain}   # 设置为这些值时才放；写成列表表示其中任意一个
    skip: {greet: false}   # front matter 为这些值的页面不放
    html: <p class="greet">{{ .Settings.message }}</p>

settings:                  # 字段和主题的设置完全相同
  - {key: message, type: string, label: Message, default: Thanks for reading.}
  - key: style
    type: select
    default: plain
    options: [{value: plain, label: Plain}, {value: bold, label: Bold}]
```

`html` 是 Go 的 `html/template`，可以读 `.Settings`、`.Site`（`Title`、
`Description`、`BaseURL`、`Language`）和 `.Page`（`URL`、`Permalink`、`Kind`、
`Title`，单页上还有条目的 `ID`、`Type`、`Params`、`Taxonomies`、`PublishedAt`）。
`asset` 给出插件自己某个文件的地址。写进脚本里的设置会变成 JavaScript 的值，所以
`{{ .Settings }}` 可以把全部设置交给脚本。`skip` 按作者习惯的写法读开关：`false`、
`no`、`off` 都表示关闭。

设置保存在 `kite.yaml` 的 `plugins.settings.<id>` 下，关闭插件时仍然保留。

### 构建期钩子

插件还可以带一个 `plugin.wasm`，也就是 WebAssembly 模块，在 `hooks` 里列出它在构建
网站时运行的函数：

| 钩子 | 何时运行 | 收到 | 返回 |
|---|---|---|---|
| `transform_markdown` | 页面的 Markdown 渲染之前 | `markdown` | `{"markdown": ...}` |
| `transform_html` | 每个渲染好的页面 | `html` | `{"html": ...}` |
| `build_complete` | 整站构建完成后 | `pages`，每页带纯文本 `text` | `{"files": [{"path": ..., "content": ...}]}` |

每个钩子收到的 JSON 里还有 `settings`、`site`，除 `build_complete` 外还有
`page`，字段与模板里的相同，只是写成 snake_case；返回 JSON，或什么都不返回，表示页面
保持原样。`build_complete` 写出的文件只能放在输出目录的 `plugins/<id>/` 下。预览运行
同样的钩子，所以预览看到的就是构建发布的结果。

模块通过 [Extism](https://extism.org) 运行，任何有 Extism PDK 的语言都能写；官方
插件用的是 Go 1.24 及以上：

```go
//go:build wasip1

package main

import (
	"strings"

	"github.com/extism/go-pdk"
)

func main() {}

//go:wasmexport transform_html
func transformHTML() int32 {
	var in struct {
		HTML string `json:"html"`
	}
	if err := pdk.InputJSON(&in); err != nil {
		pdk.SetError(err)
		return 1
	}
	out := strings.Replace(in.HTML, "</body>", "<p>Built with Kite.</p></body>", 1)
	_ = pdk.OutputJSON(map[string]string{"html": out})
	return 0
}
```

```bash
GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared -o plugin.wasm .
```

模块访问不了网络和文件，拿到的时钟不是真实时间，随机数每次运行都一样，所以它的输出只取决
于收到的输入。模块有 64 MiB 内存，每个页面限时 10 秒，`build_complete` 限时 2 分钟；
出错或超时会让构建失败，并指明是哪个插件。页面在所有 CPU 核上同时渲染，transform 钩子
会在任意一个空闲的模块实例里运行，因此不能在两次调用之间保存状态；`build_complete` 每次
构建都在全新的实例里运行。模块在插件开启或站点第一次加载它时编译一次，编译结果缓存在
`.kite/cache/wasm` 下。

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
  pagination: {}       # 按列表的种类，如 {home: 0, term: 20}
  sitemap: true
  feed: true
  feedLimit: 20
  feedAliases: []      # 订阅另外还写到哪些文件，例如 index.xml

publish:
  publisher: git
  branch: main

plugins:
  enabled: []          # 启用的插件，按运行顺序排列
  settings:            # 各插件在 plugin.yaml 里声明的设置
    search: {full_text: true}

menus:                 # 主题画的链接，按菜单分；见“主题”一节
  main:
    - {name: 关于, url: /about/}
```

`timezone` 决定日期落在哪一天。不设置时，日期按写入时的时区显示，后台写入的是 UTC，
所以在上海刚过零点发布的文章会显示成前一天。站点的关键词、作者、`noindex` 和自定义代码
由主题写进每个页面，模板里对应 `.Site.Keywords`、`.Site.Author`、`.Site.NoIndex`、
`.Site.HeadHTML` 和 `.Site.FooterHTML`，默认主题都写了。单个页面可以在 front matter 里
用 `keywords` 写自己的关键词。这些设置，连同每页文章数和订阅文章数，都可以在后台的
设置 → 站点里修改。

列表每页显示 `pageSize` 条，除非主题在 `theme.yaml` 里给这种列表另定了分页。
`build.pagination` 替站点规定，盖过主题：`home` 是首页，`list` 是某一种内容的列表，
如 `/posts/`，`term` 是某个标签或分类的页面。数目写 0 就是全部显示在一页，归档页可以这样
列出所有文章，而不用把每个标签的页面也拉得一样长。

少数几个键可以用环境变量覆盖，供产出依赖运行环境的构建使用：`KITE_SITE_TITLE`、
`KITE_SITE_BASEURL`、`KITE_SITE_LANGUAGE`、`KITE_THEME`、`KITE_BUILD_OUTPUT`、
`KITE_BUILD_URLSTYLE` 和 `KITE_BUILD_PAGESIZE`。

`build.output`（或者代替它的 `KITE_BUILD_OUTPUT`）是 `kite build` 写出站点的目录。
相对路径从项目根目录算起，不能跑到项目外面；绝对路径按原样使用，和 `kite build --output`
一样，要把站点构建到项目以外就写绝对路径。每次构建都会把这个目录整个换掉，
所以它只能用来放站点。不管用哪种方式指定，下面这些 Kite 都会拒绝：文件；
项目本身、所用的主题或插件，或者包含它们的目录；`content`、`static`、`layouts`、
`themes`、`plugins`、`.kite`、`.git`，包含它们的目录，以及它们里面的目录。
已经有文件的目录，只有是这个项目的构建写出的（Kite 记在 `.kite/outputs` 里），或者里面有
本站的 `sitemap.xml` 或 `rss.xml`（旧版 Kite 构建出的目录就是这样），才会被换掉；其他的，比如误填的
个人文件夹，会被拒绝并原样保留。`kite init` 写出的部署工作流上传的是 `public`，改了
`build.output`，工作流里的 `path` 也要跟着改。

## 内容类型

站点有文章和页面，还可以在 `kite.yaml` 里声明自己的种类，放两者都不是的内容，比如作品集里
的项目、书单里的书：

```yaml
content:
  types:
    - kind: project
      label: 项目
      dir: projects          # 放在 content/projects/，列表页在 /projects/
      route: /projects/:slug # 默认就是目录加 slug
      layout: bundle         # 每条一个文件夹；single 是每条一个文件
      order: weight          # 或 date，新的在前，这是默认
      feed: false            # 写 true 就和文章一样进订阅
      taxonomies: [stack]
      fields:
        - {key: repo, type: url, label: 仓库地址}
        - {key: status, type: select, label: 状态,
           options: [{value: active, label: 进行中}, {value: done, label: 已完成}]}
```

只有 `kind` 必须写，其余的默认值就是上面写的。声明之后，每一条都有自己的地址、在列表里的
位置、种类要求时的订阅条目，以及后台里的表单：后台在文章和页面旁边列出这个种类，按主题设置
的画法画出它的字段。模板用 `.Params.repo` 读字段。站点或主题有 `layouts/project/single.html`
和 `layouts/project/list.html` 时用它们画条目和列表，没有就用 `single.html` 和 `list.html`。

`order: weight` 按 front matter 里的 `weight` 从小到大排列和阅读，文档就这样读，`.Prev` 和
`.Next` 也按这个顺序；没有 weight 或者是 0 的排在后面，按标题。种类的名字用小写字母、数字、
`-` 和 `_`，条目放在 `content/` 下属于它自己的一个文件夹里，名字和文件夹都不能和别的种类或
分类轴相同。`kite serve` 运行时声明或去掉的种类，不用重启就会生效。

## 从 Hugo 迁移

Hugo 站点的内容原地就能打开。在站点目录里运行 `kite init .`，它只添加 `kite.yaml`，
不动 `content/`；再给每一篇内容分配 Kite 用来识别它的 ID，Hugo 不写这个：

```bash
kite doctor --fix-ids
```

文章可以是单个文件 `content/posts/hello.md`，Hugo 站点大多这样写；也可以是 bundle
`content/posts/hello/index.md`，图片放在旁边，后台新建的文章就是这种。bundle 里的
子目录原样随它发布，放在 `images/` 里的图片照样显示；子目录里有自己的 `index.md`
时，它是另一篇文章。两种不论放在 `content/posts/` 的哪一层，地址都是
`/posts/<slug>/`；Hugo 的 `_index.md` 不读。拖到单文件文章里的图片存进站点自己的
`static/uploads/`。

文章以外的栏目，比如 `content/projects/`，给它声明一个内容类型就能读到，见[内容类型](#内容类型)。

slug 里可以带路径：slug 是 `projects/tideline` 的页面发布在 `/projects/tideline/`，
和 Hugo 里放在文件夹中的页面一样。地址和另一个页面相同的内容，比如叫 `posts` 的页面，
会让构建停下并指出是哪个。

Hugo 写的 front matter 键，Kite 当作自己的来读：`date` 是发布时间，`lastmod` 是最后
修改时间，`draft: true` 是草稿；没有 `description` 时，`summary` 就是摘要，列表里显示
的是你写的摘要，而不是正文的开头。保存一篇内容时，Kite 会在旁边写上自己的键，并用
它们替换 `draft` 和 `summary`。

摘要也可以在正文里截断，Hugo 和 Hexo 都这样写：单独一行 `<!--more-->`，它前面的文字
就是摘要，不论多长；页面上这一行什么也不显示。

内容里用到的每个短代码，只要站点或主题里有对应的模板，就照常可用，见[短代码](#短代码)。
Hugo 内置的短代码，比如 `figure` 和 `youtube`，Kite 没有内置：构建遇到时会指出名字和
所在的行，给它写一个模板只要几行。

标题的锚点按标题文字生成，和 Hugo、GitHub 的规则一样，所以指向某一节的链接迁移后仍然
有效：`## 近况` 的地址是 `#近况`，`## Getting Started` 是 `#getting-started`。

改过的地址用 `aliases` 保持可用，和 Hugo 一样：

```yaml
aliases: [/2019/05/trip/, /travel/trip.html, old-trip]
```

每个别名是从站点根开始的路径；不以 `/` 开头时，相对于这篇内容自己的地址所在的目录。
每个别名处都会发布一个页面，把浏览器立即带到这篇内容，并告诉搜索引擎该收录哪个地址。
它是一个页面，不是托管平台的跳转规则，所以在 GitHub Pages 上也能用，`kite serve` 也
返回同样的页面。别名和另一个页面的地址相同时，构建会停下并指出是哪个。

Kite 的订阅地址是 `rss.xml`，Hugo 的是 `index.xml`，每个栏目还各有一个。订阅器不会
跟着跳转页走，所以要留住在旧地址订阅的读者，就让同一份订阅也写到那里：

```yaml
build:
  feedAliases: [index.xml, posts/index.xml]
```

## 从 Hexo 迁移

Hexo 站点的内容组织方式不同，所以是导入，而不是原地打开：

```bash
kite import hexo ../old-blog blog
```

Hexo 站点只读不改。目标文件夹为空或不存在时，会按 Hexo 的 `_config.yml` 新建一个站点，
标题、地址、语言、作者和时区都照搬；目标已经是 Kite 站点时，内容加进去，和已有的 slug
重复时加上编号。

- 文章和草稿都成为文章，各自是一个 bundle，放着它资源文件夹里的文件。页面成为页面，
  `source/` 里的其他文件成为站点的静态文件。以 `_` 开头、Hexo 不发布的文件夹不导入。
- `date` 和 `updated` 按站点的时区读取，多层分类展开成一层，`published: false` 成为
  草稿；没有 `description` 时 `excerpt` 就是摘要；没写 `cover` 时，Hexo 主题用的图片，
  如 `thumbnail`、`index_img`，就是封面。
- Hexo 按站点的 `permalink` 规则（或文章自己的 `permalink`）发布过的每个地址都成为
  别名，指向旧站点的链接继续有效。
- `{% asset_img %}`、`{% asset_link %}` 和 `{% asset_path %}` 转成 Markdown。其他
  Hexo 标签，如 `{% note %}`，原样保留、显示为文字，导入结束时会列出含有它们的内容。
  `<!-- more -->` 照旧截断摘要。

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
| M4 | Git 发布器 —— 第一个发布版本 **0.1** | 已完成 |
| M5 | 公开主题契约 | |
| M6 | `kite.lock` 与 `kitew` wrapper | |
| M7 | 基于 SQLite 的动态模式 | |
| M8 | WebAssembly 插件 | 第一版完成：注入代码和构建期钩子 |

[路线图与实现现状](design/roadmap.md)记录了逐项核实过的完成情况，以及之后的计划。

## 参与贡献

`scripts/check-imports.sh` 里的分层规则由 CI 强制执行：领域核心不得 import 存储、渲染或运行时包。提交信息遵循 [Conventional Commits](https://www.conventionalcommits.org/zh-hans/v1.0.0/)。

提 PR 之前请先跑：

```bash
make check
```

如果动过后台，`make web-check` 做类型检查，`make web` 构建 CI 会拿来比对的产物。

