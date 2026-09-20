<p align="center">
  <img src="docs/assets/logo.svg" alt="" width="88" height="88">
</p>

<h1 align="center">Kite</h1>

<p align="center">
  <strong>A modern open-source publishing platform.</strong><br>
  One Content, Multiple Destinations.
</p>

<p align="center">
  <a href="https://github.com/kite-plus/kite/actions/workflows/ci.yml"><img
    src="https://github.com/kite-plus/kite/actions/workflows/ci.yml/badge.svg"
    alt="CI"></a>
  <a href="LICENSE"><img
    src="https://img.shields.io/badge/license-Apache--2.0-blue.svg"
    alt="Apache-2.0"></a>
  <img src="https://img.shields.io/badge/go-1.26-00ADD8.svg" alt="Go 1.26">
</p>

---

Kite manages your content. Where it is deployed is a property of the content,
not a different product.

> Status: early development. The static path works end to end; the admin
> interface and the dynamic runtime are not built yet. See [Roadmap](#roadmap).

---

## Why

Publishing tools split into two camps, and picking one is hard to undo.

**Static site generators** give you Markdown, Git and cheap hosting, but no
real content management: you edit files, and there is no media library, no
category browser, no publish button.

**Content management systems** give you all of that, but they assume a server
and a database. Git, Markdown and static hosting fit awkwardly at best.

Kite refuses the choice. The same content, the same admin and the same themes
work whether the site is built into static files, served from a database, or
consumed through an API. Deployment becomes a setting rather than a migration.

## What works today

```
kite init                 # create a project
kite new post "Hello"     # create content
kite doctor --fix-ids     # adopt content written for another generator
kite index                # refresh the derived index
kite list --tag go        # query it
kite build --verify       # render the site, twice, and compare every byte
```

The build produces pages, listings, pagination, taxonomy and term pages, a
404 page, `sitemap.xml` and `rss.xml`.

Content written for Hugo or Hexo is read as it is: `draft: true`, `date`,
`lastmod`, `tags` and `categories` are all understood.

## Design

Three decisions shape everything else.

**Markdown files are the source of truth.** In static mode nothing is stored
in a database that is not derived from the files. The index under `.kite/` is
a cache: delete it and rebuild, and you get the same rows back.

**Your files are edited, not rewritten.** Saving a document rewrites only the
keys that changed. Key order, comments and flow-style lists survive untouched,
so changing a title produces a one-line diff.

**Store and runtime are independent.** Where content lives and how it is
delivered are separate choices, and all of their combinations are legal.

The full reasoning, including the parts deliberately left unbuilt, is in
[docs/design](docs/design/):

| Document | Contents |
|---|---|
| [architecture.md](docs/design/architecture.md) | Content model, storage, build engine, publisher, roadmap |
| [theme-system.md](docs/design/theme-system.md) | Template lookup, data contract, `theme.yaml` |
| [plugin-system.md](docs/design/plugin-system.md) | WebAssembly runtime, host ABI, capabilities |

## Build from source

Go 1.26 or newer:

```
make build      # ./bin/kite
make check      # format, vet, layering rules, tests
```

The binary is self-contained: the admin assets, the default theme and the
SQLite driver are all compiled in, and nothing needs cgo.

## Roadmap

| Milestone | Delivers |
|---|---|
| M0 | `kite build`: content model, index, markdown, themes, static output |
| M1 | `kite serve`: render per request, watch and reload |
| M2 | Read-only admin over an existing repository |
| M3 | Editing admin: editor, media, conflict handling |
| M4 | Git publisher — **v1.0** |
| M5 | Public theme contract |
| M6 | `kite.lock` and the `kitew` wrapper |
| M7 | Dynamic mode backed by SQLite |
| M8 | WebAssembly plugins |

M0 is complete.

## Releases

Release binaries are reproducible: a given commit, built with the toolchain
pinned in `go.mod`, compiles to the same bytes anywhere.

```
GOTOOLCHAIN=$(awk '/^toolchain /{print $2}' go.mod) goreleaser build --snapshot --clean
```

Verify a download against the `checksums.txt` published with the release.

## License

[Apache License 2.0](LICENSE).

## Contributing

The layering rule in `scripts/check-imports.sh` is enforced in CI: the domain
core may not import storage, rendering or runtime packages. Commits follow
[Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/).

Before opening a pull request:

```
make check
```
