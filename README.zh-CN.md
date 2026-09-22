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

Kite 是一个开源内容发布平台，内置可视化 Markdown 编辑器、图片上传、标签分类和深浅色主题。内容保存在自己的文件中，可以直接运行站点，也可以导出静态页面。

> 项目仍在早期开发中，目前请从源码安装，包含完整后台的步骤如下。

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

将 Go 的二进制安装目录（默认 `~/go/bin`）加入 `PATH`，然后创建站点：

```bash
kite init blog
cd blog
kite new post "你好，Kite"
kite run
```

打开 [管理后台](http://localhost:1717/admin/) 开始编辑。后续只需在 `blog` 目录运行 `kite run`。

</details>

## 写作与使用

1. 在后台新建文章或页面，用可视化编辑器写作，也可以切换到 Markdown 源码。
2. 拖入图片，设置分类、标签和文章地址；写作期间保留为草稿，准备好后取消草稿状态并保存。
3. 在「设置」里修改站点名称、网址和语言，在「主题」里调整外观。

本机使用 `kite run` 时会显示草稿，方便预览；正常运行站点和静态构建会排除草稿。

## 发布站点

**导出静态页面：** 在本机站点目录运行 `kite build`，将生成的 `public/` 目录上传到静态托管服务即可。Docker 用户运行：

```bash
docker exec kite kite build
docker cp kite:/data/public ./public
```

**部署到服务器：** 查看[部署说明](docs/reference.zh-CN.md#部署)，设置自己的站点域名并通过 HTTPS 对外提供访问。

**通过 Git 发布：** 本机站点配置好 Git 远程仓库后，运行 `kite publish --all --push` 提交并推送内容；自动上线还需要托管平台的部署工作流。

## 更多

- [详细使用说明](docs/reference.zh-CN.md)：账号、主题、配置、部署与常用开发操作。
- [反馈问题](https://github.com/kite-plus/kite/issues) · [架构与路线图](docs/design/)
- [Apache License 2.0](LICENSE)
