<p align="center">
  <img src="docs/assets/logo.svg" alt="Kite" width="84" height="84">
</p>

<h1 align="center">Kite</h1>

<p align="center">
  用 Markdown 写作，在浏览器里管理内容，把站点发布到你想去的地方。
</p>

<p align="center">
  <a href="https://www.kite.plus">官网</a> ·
  <a href="README.md">English</a> · 简体中文
</p>

<p align="center">
  <img src="docs/assets/screenshot.png" alt="Kite 站点的浅色与深色外观" width="880">
</p>

Kite 是一个开源内容发布平台，兼顾 CMS 的写作体验与静态站点生成器的可迁移性。它是一个用 Go 编写的单文件程序，管理后台直接内置其中。

- **浏览器里的后台**：可视化编辑器直接读写 Markdown，一键切换源码，支持图片上传、标签、分类和草稿。
- **文件始终属于你**：内容就是磁盘上的 Markdown 文件。保存时只改写真正变化的部分，key 的顺序和注释原样保留，改个标题，`git diff` 只有一行。
- **发布方式由你选**：导出静态页面放到任意托管平台，在自己的服务器上运行站点，或者通过 Git 提交并推送。
- **无需额外安装**：后台、支持深浅色的默认主题和 SQLite 驱动都已编译进这一个程序。
- **按名字装主题和插件**：评论、统计、站内搜索、公式与图表都有官方插件，另有文档站和个人站主题。后台的应用中心，或者终端里的 `kite theme add vane`，按名字安装并保持更新。插件可以往页面里加代码，也可以在构建时于沙箱中运行 WebAssembly。

> 项目仍在早期开发中：0.1 是第一个发布版本，版本之间仍可能有变化。

## 工作方式

<p align="center">
  <img src="docs/assets/workflow.svg" width="100%" alt="一篇 Markdown 文章经过 kite 产生三种输出：kite build 把静态 HTML 写入 public/，kite run 在 localhost:1717 提供站点和后台，kite publish 通过 Git 提交并推送">
</p>

Markdown 文件是唯一的真相源。后台直接编辑这些文件，Kite 在 `.kite/` 下维护的索引只是缓存：删掉、重建，得到的数据完全一样。同一份文件，用 `kite build` 生成静态页面，用 `kite run` 运行站点，用 `kite publish` 提交到 Git。

## 安装并启动

### 下载（推荐）

从[最新版本](https://github.com/kite-plus/kite/releases/latest)下载对应系统的压缩包，解压后把 `kite` 放到 `PATH` 里的某个目录。然后在一个空文件夹里启动：

```bash
mkdir blog
cd blog
kite run
```

浏览器会打开一个建站页面，填上站点名称和你的名字（它会作为文章的作者）就进入了[管理后台](http://localhost:1717/admin/)。想在终端里回答这些问题，可以改用 `kite init`。后续只需在 `blog` 目录运行 `kite run`。

从其他博客程序迁移过来？Hugo 站点的内容原地就能打开；`kite import hexo ../old-blog blog` 会把 Hexo 站点的内容连同旧地址一起导入一个新站点。详见[从 Hugo 迁移](docs/reference.zh-CN.md#从-hugo-迁移)和[从 Hexo 迁移](docs/reference.zh-CN.md#从-hexo-迁移)。

在 macOS 上，下载的程序第一次运行会被系统拦下，运行 `xattr -d com.apple.quarantine kite` 即可放行。每个压缩包都可以用版本附带的 `checksums.txt` 校验。

### Docker

在服务器上运行站点，安装 Docker 后执行：

```bash
docker run -d --name kite --restart unless-stopped -p 127.0.0.1:1717:1717 -v kite-data:/data ghcr.io/kite-plus/kite:latest
```

打开[管理后台](http://localhost:1717/admin/)，按提示设置站点和管理员账号，即可开始写作。站点地址是 [localhost:1717](http://localhost:1717)。

内容和账号保存在 `kite-data` 数据卷中。停止或重新启动：

```bash
docker stop kite
docker start kite
```

<details>
<summary>从源码构建</summary>

需要 Git、Make、Go 1.26.4+、Node.js 22.19.0 和 pnpm 10.11.1（构建会使用项目指定的 Go 工具链）。

```bash
git clone https://github.com/kite-plus/kite.git
cd kite
make web
make install
```

将 Go 的二进制安装目录（默认 `~/go/bin`）加入 `PATH`，然后按上面的方式启动。想自己构建 Docker 镜像，在克隆的目录里运行 `docker build -t kite .`。

</details>

## 写作与使用

1. 在后台新建文章或页面，用可视化编辑器写作，也可以切换到 Markdown 源码。
2. 拖入图片，设置分类、标签和文章地址；写作期间保留为草稿，准备好后取消草稿状态并保存。
3. 在「设置」里修改站点名称、网址和语言，在「主题」里调整外观。

本机使用 `kite run` 时会显示草稿，方便预览；正常运行站点和静态构建会排除草稿。

## 发布站点

**导出静态页面：** 在后台打开「部署」，把网站导出成 zip，再把里面的文件上传到任何静态托管服务。命令行里运行 `kite build`，会把同样的文件写到 `public/`。Docker 用户运行：

```bash
docker exec kite kite build
docker cp kite:/data/public ./public
```

**部署到服务器：** 查看[部署说明](docs/reference.zh-CN.md#部署)，设置自己的站点域名并通过 HTTPS 对外提供访问。

**通过 Git 发布：** 本机站点配置好 Git 远程仓库后，运行 `kite publish --all --push` 提交并推送内容；自动上线还需要托管平台的部署工作流。

## 路线图

- **已完成**：静态构建、实时预览服务、浏览器后台、Git 发布（M0–M4），以及 WebAssembly 插件的第一版（M8），作为 0.1 发布；主题契约 `kite/v1`，在 0.1.4 冻结（M5）；[应用中心](https://github.com/kite-plus/kite/issues/16)，在后台和命令行按名字安装、更新主题和插件，在 0.1.5 发布，0.1.6 起索引带签名。
- **接下来**：在 `kite.lock` 里锁定 Kite 版本并配上 `kitew` wrapper（M6），以及基于 SQLite 的动态模式（M7）。

完整的里程碑列表见[详细使用说明](docs/reference.zh-CN.md#路线图)，逐项进度见[路线图与实现现状](docs/design/roadmap.md)。

## 参与贡献

- **反馈问题或提出想法**：[提交 Issue](https://github.com/kite-plus/kite/issues)，写明版本、环境和复现步骤。
- **提交改动**：先阅读[参与贡献说明](docs/reference.zh-CN.md#参与贡献)，提交 PR 前运行 `make check`。与[设计文档](docs/design/)冲突的改动，先改文档，再改代码。

## 更多

- [详细使用说明](docs/reference.zh-CN.md)：账号、主题、插件、配置、部署与常用开发操作。
- [设计文档](docs/design/)：架构、主题系统与插件系统。
- [Apache License 2.0](LICENSE)
