<p align="center">
  <img src="docs/assets/logo.svg" alt="" width="84" height="84">
</p>

<h1 align="center">Kite</h1>

<p align="center">
  <strong>A modern open-source publishing platform.</strong><br>
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
    alt="Design docs"
    src="https://img.shields.io/badge/design-docs-4A77D6?style=flat-square&labelColor=1f2328"></a>
</p>

<p align="center">
  English · <a href="README.zh-CN.md">简体中文</a>
</p>

---

Kite manages your content. Where it is deployed is a property of the content,
not a different product.

> **Status: early development.** The static path works end to end. The admin
> interface and the dynamic runtime are not built yet — see [Roadmap](#roadmap).

<p align="center">
  <img src="docs/assets/screenshot.png" alt="A site built with Kite, in light and dark" width="880">
</p>

## Why

Publishing tools split into two camps, and picking one is hard to undo.

**Static site generators** give you Markdown, Git and cheap hosting, but no real
content management: you edit files, and there is no media library, no category
browser, no publish button.

**Content management systems** give you all of that, but they assume a server
and a database. Git, Markdown and static hosting fit awkwardly at best.

Kite refuses the choice. The same content, the same admin and the same themes
work whether a site is built into static files, served from a database, or
consumed through an API. Deployment becomes a setting rather than a migration.

## Try it

```bash
go install github.com/kite-plus/kite/cmd/kite@latest
```

```bash
kite init blog && cd blog
kite new post "Hello, Kite"
kite build
```

That produces a complete site in `public/`: pages, listings, pagination,
taxonomy and term pages, a 404, `sitemap.xml` and `rss.xml`.

Already have content written for another generator? Point Kite at it.

```bash
kite doctor --fix-ids   # adopt files that have no id yet
kite build --verify     # build twice, compare every byte
```

`draft: true`, `date`, `lastmod`, `tags` and `categories` are understood as they
are; nothing needs rewriting first.

### Commands

| | |
|---|---|
| `kite init` | create a project |
| `kite new <kind> <title>` | create content |
| `kite build` | render the site into `public/` |
| `kite run` | serve the site, open it, reload on every save |
| `kite serve` | serve the site, rendering each request from the files |
| `kite index` | refresh the derived index |
| `kite list` | query content from the index |
| `kite doctor` | check the project, and repair what is safe to repair |
| `kite openapi` | print the description of the read model API |

Every command takes `--json`, so none of them have to be parsed as prose.

## Design

Three decisions shape everything else.

**Markdown files are the source of truth.** In static mode nothing is stored in
a database that is not derived from the files. The index under `.kite/` is a
cache: delete it, rebuild, and the same rows come back.

**Your files are edited, not rewritten.** Saving a document rewrites only the
keys that changed. Key order, comments and flow-style lists survive untouched,
so changing a title produces a one-line diff.

**Store and runtime are independent.** Where content lives and how it is
delivered are separate choices, and every combination of them is legal.

The full reasoning, including the parts deliberately left unbuilt, is in
[docs/design](docs/design/).

| Document | Contents |
|---|---|
| [architecture.md](docs/design/architecture.md) | Content model, storage, build engine, publisher, roadmap |
| [theme-system.md](docs/design/theme-system.md) | Template lookup, data contract, `theme.yaml` |
| [plugin-system.md](docs/design/plugin-system.md) | WebAssembly runtime, host ABI, capabilities |

## Build from source

Go 1.26 or newer:

```bash
make build      # ./bin/kite
make check      # format, vet, layering rules, linter, tests
make web        # the admin, which is embedded into the binary
```

`make web` needs Node and pnpm; the rest does not. A binary built without it
works and says the admin is missing rather than failing to link.

The binary is self-contained. The default theme and the SQLite driver are
compiled in, nothing needs cgo, and every release target cross-compiles from any
host.

## Releases

Release binaries are reproducible: a given commit, built with the toolchain
pinned in `go.mod`, compiles to the same bytes anywhere.

```bash
GOTOOLCHAIN=$(awk '/^toolchain /{print $2}' go.mod) goreleaser build --snapshot --clean
```

Verify a download against the `checksums.txt` published with the release.

## Roadmap

| Milestone | Delivers | |
|---|---|---|
| M0 | `kite build`: content model, index, markdown, themes, static output | done |
| M1 | `kite serve`: render per request, watch and reload | done |
| M2 | Read-only admin over an existing repository | done |
| M3 | Editing admin: editor, media, conflict handling | |
| M4 | Git publisher — **v1.0** | |
| M5 | Public theme contract | |
| M6 | `kite.lock` and the `kitew` wrapper | |
| M7 | Dynamic mode backed by SQLite | |
| M8 | WebAssembly plugins | |

## Contributing

The layering rule in `scripts/check-imports.sh` is enforced in CI: the domain
core may not import storage, rendering or runtime packages. Commits follow
[Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/).

Before opening a pull request:

```bash
make check
```

## License

[Apache License 2.0](LICENSE).
