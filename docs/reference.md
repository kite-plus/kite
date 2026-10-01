# Kite reference

[Back to README](../README.md)

## The studio

`kite run` opens the studio at `/admin/`. It is a React application compiled
into the binary, so there is nothing to install and nothing to keep in sync
with the server. Run in an empty folder, it opens on a page that creates the
site there first: its name, your own name, its address and language, which
is what `kite init` asks in a terminal. Your name is the site's author.

| | |
|---|---|
| **Dashboard** | what is published, what is still a draft, what is uncommitted, and a publishing trend by month |
| **Content** | posts and pages, filtered and searched through the index rather than the filesystem |
| **Editor** | A visual editor that reads and writes Markdown, with the source one click away, a live preview, front matter as a form, terms, slug, word count, and files dropped straight into the bundle, where a Files chip lists them with what the text uses, to copy the link to, replace in place or delete; a draft saves itself as you write, and what is not saved yet stays in the browser until it is |
| **Taxonomies** | tags and categories as they actually exist across the content |
| **Theme** | every theme the project has, each previewed on the whole site before it is used; a theme installed from its zip archive; the active theme's settings edited beside a live preview |
| **Deploy** | the site exported as a zip to upload anywhere, or how far a push to GitHub Pages has got |
| **Settings** | the site's title, description, address, language, keywords, time zone, search engine visibility and code added to every page; the studio's own language and colors; your account: name, picture, password and sessions |

An item that changed on disk since it was loaded is refused rather than
overwritten, and the studio says so. Editing is available in English and
简体中文, chosen from the browser.

A photo added in the studio loses where it was taken before it is stored,
since what is stored is published: the GPS of its EXIF and whatever of its
XMP gives the place, in a JPEG, PNG, WebP or AVIF, and in the previews some
cameras keep inside a JPEG. How to turn it, what took it and when stay, its
pixels are not touched, and the studio says when it has taken a place out.
A file added with git is published as it is.

Tags and categories are counted the way the site lists them, so Go, go and
GO are one tag, called the way most posts write it, and its card says how
else it is written. Renaming a tag writes the new name on every post that
carries it, however each wrote it, so renaming Go to Go writes it one way;
removing one takes off every way of writing it. The editor offers a tag in
use when it is typed another way, rather than starting a new one.

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

The same can be done in the studio, under **Settings → Account**. A studio
open on localhost can be given a password there, and asks for it from that
moment, without a restart. A guarded one changes its user name and password
once the current password is confirmed, which signs out every other browser
and keeps this one; it can also sign the others out without changing
anything, and on localhost take the password away again. An account that
comes from the environment is changed where it is set.

The same page holds your name, an email address and a picture. The name is
the site's author, `site.author`, which a theme shows with what you publish,
so it is kept in `kite.yaml` and published like any setting; the studio calls
you by it too. The email and the picture are kept beside the account, in
`.kite/secrets/profile.json` and `.kite/secrets/avatar`, and never published.

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
`--write` the same API is read-only, and the studio says so beside the site's
name and offers only what can be read and previewed. The studio's typed
client is generated from that description and checked in CI, so the types it
compiles against cannot describe an API the server does not serve.

## Themes

Kite ships with one theme, compiled into the binary: quiet serif typography for
personal writing, light and dark, and no web fonts unless you ask for them, so
a page asks nothing of a third party.

Other themes live in `themes/`, one folder each, and `theme.name` in
`kite.yaml` chooses one by its folder's name; `default` is always the built-in
one. The studio lists them all, says why one cannot be used, and installs a
theme from a zip archive holding `theme.yaml` at its top or in the one folder
it contains. An installed theme is checked the way a site checks one it loads,
and one of the same name is replaced only when you confirm it. Before a theme
is used it can be tried on the whole site: the preview is drawn with it and
with the settings as you edit them, its links lead through the preview, and
nothing is written until you save. What a switch writes, `kite.yaml` and the
theme's folder, is published from the settings screen like any item.

A theme declares its own settings in `theme.yaml`, and the studio renders them
as a form — an option is a declaration rather than a documentation problem.
Templates live under `layouts/` in a theme and in a site alike, and the same
relative path in the site wins, so a single template can be replaced without
forking the theme.

```yaml
settings:
  - key: look
    type: section          # a heading in the form; its fields are stored beside it
    label: Look
    preview: home          # the page the studio previews while it is edited
    fields:
      - key: accent
        type: color
        label: Accent color
        default: "#7d5c3c"
        options:           # on a color, colors to suggest rather than a limit
          - {value: "#7d5c3c", label: Umber}
      - {key: favicon, type: image, label: Site icon}
  - key: nav
    type: repeat           # a list of entries, each with these fields
    label: Extra links
    fields:
      - {key: label, type: string, label: Label}
      - {key: url, type: url, label: Address}
```

The types are `string`, `text`, `number`, `boolean`, `color`, `select`,
`multiselect`, `image`, `url`, `date`, `code`, `group`, `repeat` and
`section`. The studio lists a theme's sections beside the form, and a
section's `preview` names the page shown while it is edited: `home`, `post`
(the newest post), `page` or `posts` (the list of posts). Values are stored
under `theme.settings` in `kite.yaml`, and a setting put back to its default
is removed from it. A template reads each one
as the type its field declares, `.Site.ThemeSettings.accent`, and falls back
to the default for a value it cannot read as one; a `repeat` also reads text
written one `Label | /path/` a line, so a theme can turn a text setting into a
list without losing what sites filled in.

What a theme says about itself in the studio is translated by language packs
in its `i18n/` folder, one file per language, under the key `theme`; anything
a pack leaves out is shown as `theme.yaml` writes it:

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

The same packs hold the words a theme's pages say, outside `theme`, and a
template says one with `T`: `{{ T "read_more" }}`. The word comes from the
pack for the site's language, the site's own `i18n/<lang>.yaml` over the
theme's, so a site says a theme's words its own way without replacing a
template; a word the language lacks is said in English, and one nobody has is
its key. A word can hold what a template puts in it, and a count chooses its
plural form:

```yaml
# i18n/en.yaml
posts:
  one: "{{ .Count }} post"
  other: "{{ .Count }} posts"
of: "{{ .Count }} of {{ .Total }}"
```

`{{ T "posts" 8 }}` says `8 posts`, and `{{ T "of" (dict "Count" 8 "Total" 13) }}`
says `8 of 13`. A word is text, escaped wherever it lands, so it can go in an
attribute. `i18n.Has "key"` says whether there is a word for a key, and
`{{ i18n.Words "copy" "copied" }}` hands several to a script as a JSON object.
The built-in theme says its words in English and Chinese; a site written in
another language translates it with a pack of its own.

A theme can show itself with `screenshot.png`, `.jpg` or `.webp` in its
folder, or name another file with `screenshot:` in `theme.yaml`.

Code in a page is highlighted with classes rather than colors, so a theme's
stylesheet can carry one palette for light and one for dark. A fenced block is
written as `<pre class="chroma" data-lang="go">`, where `data-lang` is the
language the author gave it, for a theme that labels its code blocks.

A theme can also offer templates for an author to choose page by page, such
as a page of links, by declaring them in `theme.yaml`:

```yaml
layouts:
  - name: links
    label: Links
    description: A list of links drawn as cards.
    types: [page]        # left out, every type is offered it
```

An item chooses one in its front matter, as Hugo writes it: `layout: links`,
or from the Template menu in the editor, which lists what the active theme
offers that kind of item and previews the page with it. It is then drawn with `layouts/page/links.html`, or `layouts/links.html`. A
theme cannot declare a layout it has no template for, and a page naming a
layout the active theme lacks keeps its type's own template.

A site's menus live in `kite.yaml`, apart from any theme, so another theme
finds them where they were. An address within the site is written from the
site's root and published under its base path:

```yaml
menus:
  main:
    - name: Archive
      url: /posts/
    - name: About
      url: /about/
    - name: Elsewhere        # a link that only heads others
      children:
        - {name: Code, url: "https://github.com/someone"}
```

A theme names the menus it draws in `theme.yaml`, and how many levels of each,
and the studio's Menus page offers those to fill in, with pages and posts
picked by their titles:

```yaml
menus:
  - name: main
    label: Header
    description: The links across the top of every page.
    depth: 1                 # 2 for links that open a submenu
```

A template draws one with `{{ range .Site.Menus.main }}`: each link has
`.Name`, `.URL`, already under the base path, `.Children` and `.Params`,
whatever else the site gives it, such as an icon. A menu the site has not
written is empty. The built-in theme draws `main` in its header, and its own
links until the site writes one.

Themes can be handled from the command line too. The commands check a theme
the way the studio does and write the same changes, which are then published
like any other:

```bash
kite theme list                  # what the site can use; * marks the one in use
kite theme add vane-0.2.0.zip    # a release's archive, or a folder; --replace over one
kite theme use vane              # sets theme.name
kite theme remove paper          # not the one in use
kite theme new paper             # a folder to start a theme from
```

A theme can be checked against the contract it is written to:

```bash
kite theme verify ./themes/paper
```

It builds a small site that uses every kind of page with the theme, asks a
server for every file the build wrote, the feed and the sitemap included, and
compares every byte. A theme that passes publishes exactly what `kite run`
previewed; one that fails is shown the first line that differs in each file.
With no directory it checks the project's own theme, or the built-in one
outside a project.

The contract a theme is written to, `apiVersion: kite/v1`, is frozen: what a
template can call, every method and function with its signature, is listed in
[theme-system.md](design/theme-system.md) sections 6 and 7 and is only ever
added to, never renamed, removed or changed. A theme that uses something added
later names the Kite it needs with `requires`.

The small site is published under a path, as a GitHub Pages project site is,
so a link written from the root of the host, such as `/rss.xml`, is reported
as well. A template links to Kite's own pages with `url.For "home"`,
`url.For "list" "post"`, `url.For "taxonomy" "tags"` or
`url.For "term" "tags" "Go"`, and to any other path of the site with
`url.Rel "rss.xml"`; both carry the path.

Terms written differently that share an address are one term: Go, go and GO
are all `/tags/go/`, and Web Dev and web-dev are both `/tags/web-dev/`. The
term's page lists the posts carrying it in any of these ways, and is called
the way most of them write it, or, when as many write it each way, the first
in character order, Go before go. A taxonomy page's `.Terms` count such a term
once, and a post's own `.Terms` show it as that post writes it. A term of
nothing but dashes, slashes or spaces has no page.

A page kept as a bundle has its files in `.Resources`, by their names in it:
`.Resources.Get "cover.jpg"`, `.Resources.Match "images/*"` or
`.Resources.ByType "image"`. A picture among them can be made into another,
smaller, cropped or in another format, which is how a theme gives a list a
small cover or a picture a `srcset`:

```html
{{ with .Resources.Get "river.jpg" }}
  {{ $small := img.Fit "800x800" . }}
  {{ $card := . | img.Fill "600x400" | img.Format "webp" | img.Quality 80 }}
  <img src="{{ $small.RelPermalink }}" width="{{ $small.Width }}" height="{{ $small.Height }}">
{{ end }}
```

`img.Resize "800x"` scales to a size, keeping the ratio where a side is left
out; `img.Fit` scales down to fit inside one; `img.Fill "600x400 top"` crops
to its ratio and scales to it, keeping the part an anchor names; `img.Crop`
cuts without scaling; `img.Format` writes `webp`, `jpeg`, `png` or `gif`,
and `img.Quality` a WebP's or a JPEG's quality, 75 unless asked. A photo is
turned upright as its EXIF says first, and what is made carries none of it,
so it never says where a photo was taken. Kite reads and writes JPEG, PNG,
GIF and WebP with code of its own, the same on every machine, so a site built
on a laptop and on a CI runner publishes the same bytes. A WebP is lossy and
keeps what is transparent; a picture written as a JPEG is put on white.
A picture is made once, the first time a template asks where it is or how
large, published beside its source as `river_<key>.jpg`, and kept in
`.kite/cache/images/`, which a later build and `kite serve` use.

The pictures a post's text shows are drawn by `layouts/_markup/render-image.html`
when the site or its theme has one, as in Hugo, which is how a phone's photos
go out smaller than they were taken:

```html
{{- with .Page.Resources.Get .Destination -}}
  {{- with img.Fit "1600x1600" . -}}
  <img src="{{ .RelPermalink }}" width="{{ .Width }}" height="{{ .Height }}" alt="{{ $.Text }}">
  {{- end -}}
{{- else -}}
  <img src="{{ .Src }}" alt="{{ .Text }}"{{ with .Title }} title="{{ . }}"{{ end }}>
{{- end -}}
```

`.Destination` is the picture's address as the text writes it, which names a
file of the bundle as `.Resources.Get` takes it; `.Src` is where a page shows
it from without the template, under the site's path when the text names it
from the site's root. `.Text` is its alternative text and `.Title` its title.

A post's cover is the `cover` in its front matter, which a template reads as
written in `.Params.cover`, and `.Images` lists the pictures its text shows,
in order and as written; a listed page carries both. A theme resolves either
as a browser resolves the text's own pictures: a full address as it is, one
from the site's root with `url.Rel`, and any other from the page's address.
It may fall back to the first picture when no cover is named, and shows none
for `cover: false`, which is what the editor's No cover writes.

A theme says how its listings page, by kind, where its design needs it: a home
page that is not a list of posts to page through, or an archive that lists
every post.

```yaml
pagination:
  home: 0    # every item on one page
  list: 0
  term: 20   # 20 a page
```

A kind it leaves out pages by the site's `build.pageSize`, and a site's
`build.pagination` stands in for any of them. `.Paginator` describes whatever
a page ends up with; on a listing with every item on one page it is page 1 of
1, and its `PageSize` is the number of items.

## Shortcodes

A shortcode puts into a page what markdown has no syntax for, such as a video,
a gallery or a note, by calling a template by name. The syntax is Hugo's, so
content moved from a Hugo site keeps working once the templates are there:

```markdown
{{< figure src="river.jpg" caption="Upstream" >}}

{{< note title="Heads up" >}}
Markdown **inside** a pair of tags is rendered too.
{{< /note >}}

Press {{< kbd Enter >}} to go on.
```

`{{% %}}` is read the same way. A tag on a line of its own is a block, which
no paragraph wraps, and a pair of them encloses the blocks between; a tag
inside a line is a word of it, and a pair there encloses the words between.
Parameters are given in order, `{{< kbd Enter >}}`, or by name,
`{{< figure src="river.jpg" >}}`, not both. A quoted value is text, and a
word written without quotes that reads as `true`, `false` or a number is that
value. A tag in code is shown as written, and `{{</* figure */>}}` shows the
tag it comments out anywhere, which is how to write about shortcodes.

The template is `layouts/_shortcodes/<name>.html`, in the site or in its
theme, the site's first; a name may hold folders, as `docs/note` does for
`layouts/_shortcodes/docs/note.html`. It receives the call:

```html
<!-- layouts/_shortcodes/note.html -->
<aside class="note">
  {{ with .Get "title" }}<strong>{{ . }}</strong>{{ end }}
  {{ .Inner }}
</aside>
```

- `.Get` is a parameter by position, `.Get 0`, or by name, `.Get "src"`, and
  nothing when the call does not give it. `.Params` is all of them, and
  `.IsNamedParams` says which way they were given.
- `.Inner` is what a pair encloses, rendered as markdown, and `.RawInner` is
  it as written, for a shortcode that reads it as something else, such as a
  diagram.
- `.Parent` is the shortcode a call sits inside, and `.Ordinal` counts the
  calls before it there, from 0.
- `.Page` is the page whose text makes the call, and `.Site` the site. The
  page's own content is still being drawn, so `.Page.Content` is empty. The
  template can call the partials a page can.

Only what a template shows counts: the words of an `.Inner` it leaves out are
not the page's words, text or excerpt, so an empty
`layouts/_shortcodes/private.html` keeps what it encloses off the site. A
shortcode nobody defines stops the build and names its file and line rather
than printing the tag, and the studio's preview says the same while you write.
The visual editor cannot keep shortcodes, so an item that calls one opens as
markdown.

## Plugins

A plugin adds to a site what its theme does not: comments, analytics, search,
math. It lives in `plugins/`, one folder each, and runs once it is listed
under `plugins.enabled` in `kite.yaml`, in the order it runs in. The studio's
Plugins screen installs one from a zip archive, turns it on and off, edits its
settings and removes it, and says before a plugin is turned on what it adds to
pages and which other sites its code loads from. The command line does the
same:

```bash
kite plugin add search-0.1.0.zip   # or a folder
kite plugin enable search
kite plugin list
kite plugin disable search
kite plugin remove search
```

The official plugins live in their own repositories, like themes other than
the default one:

| Plugin | Does |
|---|---|
| [analytics](https://github.com/kite-plus/plugin-analytics) | Counts visits with Baidu Tongji, Google Analytics, Umami or Plausible |
| [comments](https://github.com/kite-plus/plugin-comments) | A comment thread under posts, with Giscus, Waline or Twikoo |
| [math](https://github.com/kite-plus/plugin-math) | TeX math with KaTeX, and mermaid code blocks drawn as diagrams |
| [search](https://github.com/kite-plus/plugin-search) | Search in the reader's browser, over an index written at build time |

### Writing a plugin

```bash
kite plugin new greet     # a folder to start from
kite plugin verify greet  # checked the way a site checks it
```

A plugin is a folder with a `plugin.yaml`. Files under its `assets/` are
published with the site at `plugins/<id>/`, and language packs in `i18n/`,
under the key `plugin`, translate what the studio says about it, as a theme's
do.

```yaml
id: greet                  # the folder's name
name: Greet
version: 0.1.0
apiVersion: kite/plugin/v1
requires: ">=0.1.0 <2.0.0" # the Kite versions it works with
description: A line under every post.
hosts: [cdn.example.com]   # other sites its own scripts load from

inject:
  - at: head               # before </head>; body is before </body>
    html: <link rel="stylesheet" href="{{ asset "greet.css" }}">
  - at: body
    pages: [single]        # home, single, list, taxonomy, term, notFound
    kinds: [post]          # the content kinds of single pages it goes on
    when: {style: plain}   # settings it waits for; a list is any of them
    skip: {greet: false}   # front matter that leaves a page out
    html: <p class="greet">{{ .Settings.message }}</p>

settings:                  # the same fields as a theme's
  - {key: message, type: string, label: Message, default: Thanks for reading.}
  - key: style
    type: select
    default: plain
    options: [{value: plain, label: Plain}, {value: bold, label: Bold}]
```

`html` is a Go `html/template`, given `.Settings`, `.Site` with its `Title`,
`Description`, `BaseURL` and `Language`, and `.Page` with its `URL`,
`Permalink`, `Kind` and `Title`, and on a single page its item's `ID`, `Type`,
`Params`, `Taxonomies` and `PublishedAt`. `asset` gives the address of one of
the plugin's files. A setting written into a script becomes a JavaScript
value, so `{{ .Settings }}` hands a script all of them. `skip` reads a switch
the way authors write it: `false`, `no` and `off` all turn a page's code off.

Settings are stored under `plugins.settings.<id>` in `kite.yaml`, and stay
there when a plugin is turned off.

### Build hooks

A plugin can also bring `plugin.wasm`, a WebAssembly module whose functions
run while a site is built, and name them under `hooks`:

| Hook | Runs | Is handed | Answers with |
|---|---|---|---|
| `transform_markdown` | before a page's markdown is rendered | `markdown` | `{"markdown": ...}` |
| `transform_html` | on each rendered page | `html` | `{"html": ...}` |
| `build_complete` | once the site is built | `pages`, each with its plain `text` | `{"files": [{"path": ..., "content": ...}]}` |

Each is handed JSON with `settings`, `site` and, but for `build_complete`,
`page`, named as in templates but in snake case, and answers with JSON or with
nothing, which leaves the page as it was. The files `build_complete` writes go
under `plugins/<id>/` in the output and nowhere else. A preview runs the same
hooks, so it shows what a build publishes.

The module runs through [Extism](https://extism.org), so any language with an
Extism PDK can write one; the official plugins use Go 1.24 or later:

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

A module reaches no network and no files, its clock does not tell the time and
its random numbers are the same on every run, so what it answers depends on
what it is handed alone. It has 64 MiB of memory, ten seconds a page and two
minutes for `build_complete`; one that fails, or takes longer, fails the build
and names the plugin. Pages are rendered on every core at once, so a transform
runs in whichever instance of the module is free and must keep nothing between
calls; `build_complete` runs in a fresh one each build. A module is compiled
once, when its plugin is turned on or a site first loads it, and kept compiled
under `.kite/cache/wasm`.

## Configuration

`kite.yaml` sits at the root of a project. Everything except `site` is
optional, and the values below are the defaults.

```yaml
site:
  title: My Site
  description: ""      # search results and feeds; themes often show it too
  baseURL: https://example.com
  language: en
  author: ""
  keywords: []         # a list, or one line separated by commas
  timezone: ""         # an IANA zone such as Asia/Shanghai
  noindex: false       # true asks search engines to leave the site out
  headHTML: ""         # added before </head> on every page
  footerHTML: ""       # added before </body> on every page

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
  pagination: {}       # by kind of listing, as {home: 0, term: 20}
  sitemap: true
  feed: true
  feedLimit: 20
  feedAliases: []      # more files the feed is written to, such as index.xml

publish:
  publisher: git
  branch: main

plugins:
  enabled: []          # the plugins that run, in the order they run in
  settings:            # whatever each plugin declares in plugin.yaml
    search: {full_text: true}

menus:                 # the links themes draw, by menu; see Themes
  main:
    - {name: About, url: /about/}
```

`timezone` decides which day a date falls on. Left empty, a date is shown in
the zone it was written in, which for what the studio writes is UTC, so a post
published just after midnight in Shanghai would read as the day before. The
site's keywords, author and `noindex`, and its own code, reach every page
through the theme, which writes them into the page as `.Site.Keywords`,
`.Site.Author`, `.Site.NoIndex`, `.Site.HeadHTML` and `.Site.FooterHTML`; the
default theme does. A page can give its own keywords with `keywords` in its
front matter. The studio edits all of these, and the page size and feed limit,
under Settings → Site.

A listing shows `pageSize` items a page unless its theme says otherwise for
that kind of listing in `theme.yaml`. `build.pagination` says it for the site,
over the theme: `home` for the home page, `list` for the list of a kind of
item, such as `/posts/`, and `term` for the page of a tag or a category. A
size of 0 puts every item on one page, which is how an archive lists every
post without making every tag's page as long.

A few keys can be overridden from the environment, for a build whose output
depends on where it runs: `KITE_SITE_TITLE`, `KITE_SITE_BASEURL`,
`KITE_SITE_LANGUAGE`, `KITE_THEME`, `KITE_BUILD_OUTPUT`,
`KITE_BUILD_URLSTYLE` and `KITE_BUILD_PAGESIZE`.

`build.output`, or `KITE_BUILD_OUTPUT` in its place, is the directory
`kite build` writes the site to. A relative path is taken from the project
root and has to stay inside it. An absolute one is used as it stands, as with
`kite build --output`, which is how a site is built outside the project. A
build replaces the directory whole, so it has to be one of its own, and Kite
refuses one that is not, however it is given: a file, a directory that is or
holds the project, its theme or one of its plugins, and one that is, holds or
lies inside `content`, `static`, `layouts`, `themes`, `plugins`, `.kite` or
`.git`. A directory that already holds files is replaced only when a build of
the project wrote it, which Kite records in `.kite/outputs`, or when it holds
this site's `sitemap.xml` or `rss.xml`, as one built by an older Kite does;
anything else, such as a home folder named by mistake, is refused and left as
it is. The deploy workflow `kite init` writes uploads `public`; if
`build.output` changes, change the workflow's `path` with it.

## Content types

A site has posts and pages, and can declare kinds of its own in `kite.yaml`,
for what is neither, such as a portfolio's projects or a reading list's books:

```yaml
content:
  types:
    - kind: project
      label: Project
      dir: projects          # content/projects/, and the listing at /projects/
      route: /projects/:slug # the default: the dir and the slug
      layout: bundle         # a folder per item, or single, a file per item
      order: weight          # or date, newest first, which is the default
      feed: false            # true puts the items in the feed, as posts are
      taxonomies: [stack]
      fields:
        - {key: repo, type: url, label: Repository}
        - {key: status, type: select, label: Status,
           options: [{value: active, label: Active}, {value: done, label: Done}]}
```

Only `kind` is needed; the rest has the defaults shown. Each item then has an
address, a place in its listing, a feed entry when the kind asks for one, and
a form in the studio, which lists the kind beside posts and pages and draws
its fields the way the theme's settings are drawn. A template reads a field
as `.Params.repo`. Items are drawn with `layouts/project/single.html` and
listed with `layouts/project/list.html` when the site or the theme has them,
and with `single.html` and `list.html` otherwise.

`order: weight` lists and reads the items by the `weight` in their front
matter, smallest first, as documentation is read, and `.Prev` and `.Next`
follow that order; items with no weight, or 0, come after, by title. A kind
is named by lowercase letters, digits, `-` and `_`, keeps its items in one
folder of its own under `content/`, and cannot share its name or folder with
another kind or a taxonomy. `kite serve` picks up a kind declared or taken
out while it runs.

## Moving from Hugo

A Hugo site's content opens where it is. Run `kite init .` in the site's
folder, which adds `kite.yaml` and leaves `content/` alone, then give every
item the id Kite keeps it by, which Hugo does not write:

```bash
kite doctor --fix-ids
```

A post can be a single file, `content/posts/hello.md`, as Hugo sites mostly
write them, or a bundle, `content/posts/hello/index.md` with its pictures
beside it, which is what the studio creates. A bundle's folders are published
with it as they are, so a picture kept in `images/` shows where the text
links it; a folder with an `index.md` of its own is another post. Either is
published at `/posts/<slug>/` wherever in `content/posts/` it sits, and
Hugo's `_index.md` is left out. A picture dropped on a post kept as a single
file goes among the site's own files, in `static/uploads/`.

A section other than posts, such as `content/projects/`, is read once a kind
of content is declared for it; see [Content types](#content-types).

A slug may hold a path: a page whose slug is `projects/tideline` is published
at `/projects/tideline/`, as pages kept in folders were in Hugo. An item whose
address is already another page's, such as a page called `posts`, stops the
build and says which.

Kite reads the front matter keys Hugo writes as its own: `date` is when an
item was published, `lastmod` when it last changed, `draft: true` makes it a
draft, and `summary` is its description when it has none, so a list shows the
summary you wrote rather than the opening of the text. Saving an item writes
Kite's own keys beside them, and those replace `draft` and `summary`.

A summary can also be ended in the text, as Hugo and Hexo both allow: a line
holding only `<!--more-->` makes the prose before it the excerpt, however
long, and the page shows nothing where it stands.

Shortcodes keep working once the site or its theme has a template for each
one the content calls; see [Shortcodes](#shortcodes). Hugo's built-in ones,
such as `figure` and `youtube`, are not built into Kite: a build that meets
one names it and its line, and a template for it takes a few lines.

Headings are anchored by their text the way Hugo and GitHub anchor them, so
a link into a section keeps working after the move: `## 近况` is reached at
`#近况`, and `## Getting Started` at `#getting-started`.

An address that changed keeps working through `aliases`, as in Hugo:

```yaml
aliases: [/2019/05/trip/, /travel/trip.html, old-trip]
```

Each alias is a path from the site's root, or, without a leading slash, one
beside the item's own address, and a page is published there that sends a
browser on to the item at once and tells search engines which address to
keep. It is a page rather than a redirect of the host's, so it works on
GitHub Pages too, and `kite serve` answers with the same page. An alias that
is another page's address stops the build and says which.

Kite's feed is `rss.xml`, and Hugo's was `index.xml`, with one more per
section. Feed readers do not follow a page that redirects, so to keep the
people subscribed at the old addresses, have the same feed written there too:

```yaml
build:
  feedAliases: [index.xml, posts/index.xml]
```

## Moving from Hexo

A Hexo site keeps its content another way, so it is imported rather than
opened:

```bash
kite import hexo ../old-blog blog
```

The Hexo site is read and left as it is. An empty or missing folder becomes
a new project with the title, address, language, author and time zone the
Hexo site's `_config.yml` gives; a Kite project gets the content added to its
own, and a slug it already uses gets a number.

- Posts and drafts become posts, each a bundle holding the files of its
  asset folder. Pages become pages, and every other file in `source/`
  becomes one of the site's static files. Folders whose names start with
  `_`, which Hexo does not publish, are left out.
- `date` and `updated` are read in the site's time zone, nested categories
  are flattened, `published: false` makes a draft, an `excerpt` is the
  description when there is none, and a picture a Hexo theme names, such as
  `thumbnail` or `index_img`, is the cover when none is named.
- Every address Hexo published a post or page at, by the site's `permalink`
  pattern or a post's own, becomes an alias, so links to the old site keep
  working.
- `{% asset_img %}`, `{% asset_link %}` and `{% asset_path %}` become
  markdown. Other Hexo tags, such as `{% note %}`, stay as written and show
  as text; the import lists the items that hold them. `<!-- more -->` ends
  an excerpt as it did.

## Deploying

A site can be built into files and hosted anywhere, or run as a server that
manages itself. It is the same content either way, so this is a decision you
can change your mind about.

### Static, to GitHub Pages

`kite init` writes a GitHub Pages workflow that builds with `--verify`, so a
site that would deploy differently on a second run fails before it is
published. Turn Pages on under **Settings → Pages → Source → GitHub Actions**
and a push to `main` deploys.

Until it has a domain of its own, a repository's site lives at
`https://<owner>.github.io/<repository>/`. Give that address as `baseURL`:
every link Kite makes carries the path, and `kite serve` previews the site
under it.

A second workflow, `scheduled.yml`, publishes posts scheduled for later. A
post dated in the future waits for its date whether its status is `scheduled`
or `published`, as in Hugo and Jekyll, so a site moved from either keeps its
future posts back. Each build records when the next scheduled post falls due,
and once an hour the workflow checks that time and deploys only if it has
passed, so a post goes live within the hour after its time and an hour with
nothing due costs one short job. In a private repository each check is billed
as a minute of Actions time; change its `cron` line to check less often.

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

The repository's commit hooks run as they would for any commit. When one
refuses, the studio shows what it said and offers to publish without the
hooks; `kite publish --no-verify` does the same from a terminal.

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
so changing a title produces a one-line diff. Front matter can be YAML between
`---` lines or TOML between `+++` lines, as Hugo writes it; a TOML file stays
TOML.

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
make e2e        # drive the admin in Chromium against a binary built with it
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
| M4 | Git publisher — the first release, **0.1** | done |
| M5 | Public theme contract | done: `kite/v1` frozen in 0.1.4 |
| M6 | `kite.lock` and the `kitew` wrapper | |
| M7 | Dynamic mode backed by SQLite | |
| M8 | WebAssembly plugins | first version done: injected code and build hooks |

The [roadmap](design/roadmap.md) (in Chinese) records what has been verified as done and the plan after it.

## Contributing

The layering rule in `scripts/check-imports.sh` is enforced in CI: the domain
core may not import storage, rendering or runtime packages. Commits follow
[Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/).

Before opening a pull request:

```bash
make check
```

If you touched the admin, `make web-check` type checks it and `make web` builds
the bundle CI compares against. `make e2e` runs its browser tests, each on a
throwaway site in a temporary directory; the first run downloads Chromium.

