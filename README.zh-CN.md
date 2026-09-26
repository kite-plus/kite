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

> 项目仍在早期开发中，目前请从源码安装，包含完整后台的步骤如下。

## 工作方式

<p align="center">
  <img src="docs/assets/workflow.svg" width="100%" alt="一篇 Markdown 文章经过 kite 产生三种输出：kite build 把静态 HTML 写入 public/，kite run 在 localhost:1717 提供站点和后台，kite publish 通过 Git 提交并推送">
</p>

Markdown 文件是唯一的真相源。后台直接编辑这些文件，Kite 在 `.kite/` 下维护的索引只是缓存：删掉、重建，得到的数据完全一样。同一份文件，用 `kite build` 生成静态页面，用 `kite run` 运行站点，用 `kite publish` 提交到 Git。

## 安装并启动

### Docker（推荐）

安装 Git 和 Docker 后，运行：

```bash
git clone https://github.com/kite-plus/kite.git
cd kite
docker build -t kite .
docker run -d --name kite --restart unless-stopped -p 127.0.0.1:1717:1717 -v kite-data:/data kite
```

打开 [管理后台](http://localhost:1717/admin/)，按提示设置站点和管理员账号，即可开始写作。站点地址是 [localhost:1717](http://localhost:1717)。首次构建需要下载依赖，请稍等。

内容和账号保存在 `kite-data` 数据卷中。停止或重新启动：

```bash
docker stop kite
docker start kite
```

<details>
<summary>不用 Docker：安装到本机</summary>

需要 Git、Make、Go 1.26.4+、Node.js 22.19.0 和 pnpm 10.11.1（构建会使用项目指定的 Go 工具链）。

```bash
git clone https://github.com/kite-plus/kite.git
cd kite
make web
make install
```

将 Go 的二进制安装目录（默认 `~/go/bin`）加入 `PATH`，然后在一个空文件夹里启动：

```bash
mkdir blog
cd blog
kite run
```

浏览器会打开一个建站页面，填上站点名称就进入了[管理后台](http://localhost:1717/admin/)。想在终端里回答这些问题，可以改用 `kite init`。后续只需在 `blog` 目录运行 `kite run`。

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

- **已完成**：静态构建、实时预览服务、浏览器后台、Git 发布（M0–M4），打 v1.0 标签前还有少量收尾。
- **接下来**：公开主题契约、`kite.lock` 与 `kitew` wrapper、基于 SQLite 的动态模式、WebAssembly 插件（M5–M8）。

完整的里程碑列表见[详细使用说明](docs/reference.zh-CN.md#路线图)，逐项进度见[路线图与实现现状](docs/design/roadmap.md)。

## 参与贡献

- **反馈问题或提出想法**：[提交 Issue](https://github.com/kite-plus/kite/issues)，写明版本、环境和复现步骤。
- **提交改动**：先阅读[参与贡献说明](docs/reference.zh-CN.md#参与贡献)，提交 PR 前运行 `make check`。与[设计文档](docs/design/)冲突的改动，先改文档，再改代码。

## 更多

- [详细使用说明](docs/reference.zh-CN.md)：账号、主题、配置、部署与常用开发操作。
- [设计文档](docs/design/)：架构、主题系统与插件系统。
- [Apache License 2.0](LICENSE)
