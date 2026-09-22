<p align="center">
  <img src="docs/assets/logo.svg" alt="Kite" width="84" height="84">
</p>

<h1 align="center">Kite</h1>

<p align="center">
  Write in Markdown. Manage content in your browser. Publish on your terms.
</p>

<p align="center">
  <a href="https://www.kite.plus">Website</a> ·
  English · <a href="README.zh-CN.md">简体中文</a>
</p>

<p align="center">
  <img src="docs/assets/screenshot.png" alt="A Kite site in light and dark mode" width="880">
</p>

Kite is an open-source publishing platform with a visual Markdown editor, image uploads, tags, categories, and a light/dark theme. Your content stays in your own files. Run a site directly or export static pages.

> Kite is in early development. Install from source using the steps below, which include the full studio.

## Install and start

### Docker (recommended)

With Git and Docker installed, run:

```bash
git clone https://github.com/kite-plus/kite.git
cd kite
docker build -t kite .
docker run -d --name kite --restart unless-stopped -p 127.0.0.1:1717:1717 -v kite-data:/data kite
```

Open the [studio](http://localhost:1717/admin/) and follow the setup steps to name your site and create an admin account. Your site is at [localhost:1717](http://localhost:1717). The first build downloads dependencies and may take a while.

Content and your account are stored in the `kite-data` volume. To stop or restart:

```bash
docker stop kite
docker start kite
```

<details>
<summary>Without Docker: install locally</summary>

Requires Git, Make, Go 1.26.4+, Node.js 22.19.0, and pnpm 10.11.1. The build uses the Go toolchain pinned by the project.

```bash
git clone https://github.com/kite-plus/kite.git
cd kite
make web
make install
```

Add Go's binary installation directory (usually `~/go/bin`) to your `PATH`, then create a site:

```bash
kite init blog
cd blog
kite new post "Hello, Kite"
kite run
```

Open the [studio](http://localhost:1717/admin/) to start editing. Next time, run `kite run` from the `blog` directory.

</details>

## Write and manage

1. Create a post or page in the studio. Write in the visual editor or switch to Markdown source.
2. Drop in images and set categories, tags, and a URL slug. Keep unfinished work as a draft; clear the draft setting and save when ready.
3. Update your site's name, URL, and language in Settings, and customize its appearance in Theme.

Local `kite run` includes drafts for preview. Normal serving and static builds exclude them.

## Publish your site

**Export static pages:** Run `kite build` in your local site directory, then upload the generated `public/` directory to a static host. With Docker:

```bash
docker exec kite kite build
docker cp kite:/data/public ./public
```

**Run on a server:** Follow the [deployment guide](docs/reference.md#deploying) to configure your site domain and serve it over HTTPS.

**Publish through Git:** With a Git remote configured for your local site, run `kite publish --all --push` to commit and push content. Automatic deployment also needs a hosting workflow.

## More

- [Reference guide](docs/reference.md): accounts, themes, configuration, deployment, and development.
- [Report an issue](https://github.com/kite-plus/kite/issues) · [Architecture and roadmap](docs/design/)
- [Apache License 2.0](LICENSE)
