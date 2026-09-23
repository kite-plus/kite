# Kite reference

[Back to README](../README.md)

## The studio

`kite run` opens the studio at `/admin/`. It is a React application compiled
into the binary, so there is nothing to install and nothing to keep in sync
with the server.

| | |
|---|---|
| **Dashboard** | what is published, what is still a draft, what is uncommitted, and a publishing trend by month |
| **Content** | posts and pages, filtered and searched through the index rather than the filesystem |
| **Editor** | A visual editor that reads and writes Markdown, with the source one click away, a live preview, front matter as a form, terms, slug, word count, and files dropped straight into the bundle |
| **Taxonomies** | tags and categories as they actually exist across the content |
| **Theme** | the settings the active theme declares in its `theme.yaml`, rendered as a form |
| **Settings** | title, description, base URL and language |

An item that changed on disk since it was loaded is refused rather than
overwritten, and the studio says so. Editing is available in English and
简体中文, chosen from the browser.

### Signing in

On localhost a project with no password is open, because there is nobody else
on the machine to keep out. Anywhere else the studio needs an account, and a
server that would put an unguarded one on a reachable address does not come up
open.

It comes up in **setup** instead. Nothing but the installer answers — not the
content, not the drafts, not the settings, not even the site's own title — so
a server nobody has configured hands nothing out.

```bash
kite serve --admin --write --addr 0.0.0.0:1717
```
```
  Finish installing this site in a browser:

    http://localhost:1717/admin/setup
```

The installer itself is open, and deliberately so: until it has been finished
there is no account, so there is nobody a request could be checked against,
and the first browser to reach the form is the one that gets the account.
Finish it, or skip it entirely by giving the server an account before it is
reachable:

```bash
kite auth set-password                          # asked for twice, never echoed
```

`kite auth status` reports whether a project asks for a password, and
`kite auth remove` takes the account away again.

The account is stored in `.kite/secrets/account.json` as an argon2id hash. It
is never committed, and it has to survive a deployment for the account to. A
container can supply one from the environment instead — `KITE_ADMIN_USER` with
`KITE_ADMIN_PASSWORD`, or `KITE_ADMIN_PASSWORD_HASH` to keep a plaintext
password out of the process list — and the environment wins over the file.

### The API

Everything the studio does, it does over one HTTP API under `/api/v1`, which
the binary describes itself:

```bash
kite openapi > openapi.json
```

`--admin` serves it and `--write` allows it to change the project; without
`--write` the same API is read-only. The studio's typed client is generated
from that description and checked in CI, so the types it compiles against
cannot describe an API the server does not serve.

## Themes

Kite ships with one theme, compiled into the binary: quiet serif typography for
personal writing, light and dark, and no web fonts unless you ask for them, so
a page asks nothing of a third party.

A theme declares its own settings in `theme.yaml`, and the studio renders them
as a form — an option is a declaration rather than a documentation problem.
Templates live under `layouts/` in a theme and in a site alike, and the same
relative path in the site wins, so a single template can be replaced without
forking the theme.

A theme can be checked against the contract it is written to:

```bash
kite theme verify ./themes/paper
```

It builds a small site that uses every kind of page with the theme, asks a
server for the same pages, and compares every byte. A theme that passes
publishes exactly what `kite run` previewed; one that fails is shown the first
line that differs on each page. With no directory it checks the project's own
theme, or the built-in one outside a project.

The theme contract is not frozen yet; it freezes at M5, after a second theme
has been written against it.

## Configuration

`kite.yaml` sits at the root of a project. Everything except `site` is
optional, and the values below are the defaults.

```yaml
site:
  title: My Site
  baseURL: https://example.com
  language: en

content:
  store: file          # where content lives
  dir: content

theme:
  name: default
  settings:            # whatever the theme declares in theme.yaml
    accent: "#7d5c3c"

markdown:
  highlightTheme: github

build:
  output: public
  urlStyle: directory  # or extension, for /posts/hello.html
  pageSize: 10
  sitemap: true
  feed: true
  feedLimit: 20

publish:
  publisher: git
  branch: main
```

A few keys can be overridden from the environment, for a build whose output
depends on where it runs: `KITE_SITE_TITLE`, `KITE_SITE_BASEURL`,
`KITE_SITE_LANGUAGE`, `KITE_THEME`, `KITE_BUILD_OUTPUT`,
`KITE_BUILD_URLSTYLE` and `KITE_BUILD_PAGESIZE`.

## Deploying

A site can be built into files and hosted anywhere, or run as a server that
manages itself. It is the same content either way, so this is a decision you
can change your mind about.

### Static, to GitHub Pages

`kite init` writes a GitHub Pages workflow that builds with `--verify`, so a
site that would deploy differently on a second run fails before it is
published. Turn Pages on under **Settings → Pages → Source → GitHub Actions**
and a push to `main` deploys.

A second workflow, `scheduled.yml`, publishes posts scheduled for later. Each
build records when the next scheduled post falls due, and once an hour the
workflow checks that time and deploys only if it has passed, so a post goes
live within the hour after its time and an hour with nothing due costs one
short job. In a private repository each check is billed as a minute of
Actions time; change its `cron` line to check less often.

GitHub turns scheduled workflows off in a public repository with no commits
for 60 days. Turn it back on under the **Actions** tab. Deploying on push is
a separate workflow for exactly this reason, and keeps working either way.

The studio follows a publish from the commit through the push to the
deployment. For a public repository on GitHub that deploys to Pages, it asks
GitHub's API, anonymously and read-only, whether the pushed commit is live,
and links to the site once it is. Other hosts do not report deployments, and
the studio says so instead of waiting.

`kite build` prints when the next scheduled post falls due. On any other host
that is when the site has to be built again, because a static site only shows
a scheduled post once it has been built after the post's time.

Publishing from a machine instead goes through Git:

```bash
kite publish content/posts/hello --push
```

It commits exactly the paths given and nothing else: what you have staged stays
staged, and every other change stays where it is. `--all` publishes everything
uncommitted that Kite manages, and `--dry-run` reports what would happen and
stops.

A push is never forced. When the remote has commits the branch does not, the
commit stays where it is and the refusal lists them. If none of them change
what was published, `kite publish --push --rebase`, or the button the studio
shows, puts the commit on top of them and pushes, without touching anything
else in the working tree. If they changed the same files, the remote's side is
shown and settling it is left to you. `kite publish --push` on its own pushes
whatever is already committed, for a push that failed the first time.

### With Docker

```bash
docker build -t ghcr.io/kite-plus/kite:latest .
docker compose up -d
docker compose logs kite      # it prints where to finish installing
```

Then open `http://localhost:1717/admin/` and finish the installation in the
browser. [`docker-compose.yaml`](../docker-compose.yaml) is the whole of the
configuration.

The image is `ghcr.io/kite-plus/kite`, built for amd64 and arm64. It holds the
binary, the admin and git, runs as an unprivileged user, and keeps nothing of
its own: the site lives in a volume at `/data`, and an empty one becomes a new
project on the first start. Everything else is an ordinary `kite` command:

```bash
docker compose run --rm kite build
docker compose run --rm kite publish --all --push
docker compose run --rm kite auth set-password
```

`KITE_SITE_BASEURL` is the one setting worth giving it up front, because it is
the address that ends up in feeds and sitemaps and it is not the container's.

### On a server, without Docker

```bash
kite serve --admin --write --addr 0.0.0.0:1717
```

The first start prints a link and waits for a browser, exactly as the
container does — see [Signing in](#signing-in).

Kite terminates no TLS of its own, so put it behind something that does. A
password crossing the network in the clear is not protected by the fact that
it was hashed at the other end.

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
[docs/design](design/).

| Document | Contents |
|---|---|
| [architecture.md](design/architecture.md) | Content model, storage, build engine, publisher, roadmap |
| [theme-system.md](design/theme-system.md) | Template lookup, data contract, `theme.yaml` |
| [plugin-system.md](design/plugin-system.md) | WebAssembly runtime, host ABI, capabilities |

> The design documents are written in Chinese; terms of art stay in English.

## Build from source

Go 1.26 or newer:

```bash
make build      # ./bin/kite
make check      # format, vet, layering rules, tidiness, linter, tests
make web        # the admin, which is embedded into the binary
make web-gen    # regenerate the API client from this build's own description
make docker     # the container image, which compiles both of those itself
make perf       # time a 2000-post site against the design's latency targets
```

`make web` needs Node and pnpm, both pinned exactly — the versions live in
`web/.nvmrc` and `web/package.json`. The rest of the build needs neither. A
binary built without it works and says the admin is missing rather than failing
to link.

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
| M3 | Editing admin: editor, media, conflict handling | done |
| M4 | Git publisher — **v1.0** | done, wrapping up before the tag |
| M5 | Public theme contract | |
| M6 | `kite.lock` and the `kitew` wrapper | |
| M7 | Dynamic mode backed by SQLite | |
| M8 | WebAssembly plugins | |

The [roadmap](design/roadmap.md) (in Chinese) records what has been verified as done, what remains before v1.0 is tagged, and the plan after it.

## Contributing

The layering rule in `scripts/check-imports.sh` is enforced in CI: the domain
core may not import storage, rendering or runtime packages. Commits follow
[Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/).

Before opening a pull request:

```bash
make check
```

If you touched the admin, `make web-check` type checks it and `make web` builds
the bundle CI compares against.

