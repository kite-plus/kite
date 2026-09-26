package plugin_test

import (
	"bytes"
	"context"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/kite-plus/kite/internal/content"
	"github.com/kite-plus/kite/internal/hook"
	"github.com/kite-plus/kite/internal/plugin"
)

const comments = `id: comments
name: Comments
version: 1.0.0
apiVersion: kite/plugin/v1
description: Discussion under every post.
inject:
  - at: head
    html: <link rel="stylesheet" href="{{ asset "comments.css" }}">
  - at: body
    pages: [single]
    kinds: [post]
    when: {provider: giscus}
    skip: {comments: false}
    html: |
      <script src="https://giscus.app/client.js" data-repo="{{ .Settings.repo }}" data-term="{{ .Page.ID }}" async></script>
  - at: body
    pages: [single]
    when: {provider: [waline, twikoo]}
    html: |
      <script>window.discuss = {server: {{ .Settings.server }}, page: {{ .Page.Permalink }}};</script>
settings:
  - key: provider
    type: select
    default: giscus
    options:
      - {value: giscus, label: Giscus}
      - {value: waline, label: Waline}
      - {value: twikoo, label: Twikoo}
  - key: repo
    type: string
  - key: server
    type: url
`

func tree(files map[string]string) fstest.MapFS {
	out := fstest.MapFS{}
	for name, body := range files {
		out[name] = &fstest.MapFile{Data: []byte(body)}
	}
	return out
}

func load(t *testing.T, files map[string]string) *plugin.Plugin {
	t.Helper()
	p, err := plugin.Load(tree(files), "comments")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	return p
}

func TestAPluginLoadsFromItsDirectory(t *testing.T) {
	p := load(t, map[string]string{
		"plugin.yaml":            comments,
		"assets/comments.css":    ".comments {}",
		"i18n/zh-CN.yaml":        "plugin:\n  title: 评论\n  settings:\n    provider:\n      label: 评论服务\n      options: {twikoo: Twikoo 评论}\n",
		"README.md":              "not published",
		"assets/nested/icon.svg": "<svg/>",
	})
	if p.Assets == nil {
		t.Fatal("the assets directory was not found")
	}
	if p.Wasm != nil {
		t.Error("a plugin with no hooks was given a module")
	}
	if got := p.Hosts(); !slices.Equal(got, []string{"giscus.app"}) {
		t.Errorf("hosts = %q, want only giscus.app", got)
	}

	zh := p.Localized("zh-CN")
	if zh.Name != "评论" || zh.Settings[0].Label != "评论服务" || zh.Settings[0].Options[2].Label != "Twikoo 评论" {
		t.Errorf("localized = %q / %q / %q", zh.Name, zh.Settings[0].Label, zh.Settings[0].Options[2].Label)
	}
	if en := p.Localized("fr"); en.Name != "Comments" {
		t.Errorf("a language with no pack reads %q, want the manifest's own words", en.Name)
	}
}

func TestAPluginThatCannotWorkIsRefused(t *testing.T) {
	valid := "id: comments\nname: C\nversion: 1.0.0\napiVersion: kite/plugin/v1\n"
	inject := "inject:\n  - at: head\n    html: <meta>\n"
	for _, tc := range []struct{ name, manifest, want string }{
		{"no id", "name: C\nversion: 1\napiVersion: kite/plugin/v1\n" + inject, "id is required"},
		{"a bad id", strings.Replace(valid, "id: comments", "id: Comments_2", 1) + inject, "lowercase"},
		{"another directory", strings.Replace(valid, "id: comments", "id: other", 1) + inject, "have to match"},
		{"another contract", strings.Replace(valid, "kite/plugin/v1", "kite/plugin/v9", 1) + inject, "apiVersion"},
		{"nothing to do", valid, "neither injects"},
		{"a bad place", valid + "inject:\n  - at: footer\n    html: <p>\n", "want head or body"},
		{"no html", valid + "inject:\n  - at: body\n    html: ' '\n", "html is empty"},
		{"an unknown page", valid + "inject:\n  - at: body\n    pages: [archive]\n    html: <p>\n", "kind of page"},
		{"an unknown setting", valid + "inject:\n  - at: body\n    when: {mode: x}\n    html: <p>\n", "not a setting"},
		{"a broken template", valid + "inject:\n  - at: body\n    html: '{{ .Settings.x '\n", "inject 1"},
		{"an unknown hook", valid + "hooks: [transform_css]\n", "not a hook"},
		{"hooks with no module", valid + "hooks: [transform_html]\n", "plugin.wasm"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := plugin.Load(tree(map[string]string{"plugin.yaml": tc.manifest}), "comments")
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Errorf("Load = %v, want an error about %q", err, tc.want)
			}
		})
	}
}

// inject runs a plugin's injector over one page.
func inject(t *testing.T, p *plugin.Plugin, settings map[string]any, doc hook.HTMLDoc) string {
	t.Helper()
	in, err := p.Injector(settings, plugin.Site{Title: "Notes", BaseURL: "https://example.com/blog"}, plugin.Links{
		Asset:    func(name string) string { return "/blog/plugins/comments/" + name },
		Absolute: func(url string) string { return "https://example.com" + url },
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := in.TransformHTML(context.Background(), &doc); err != nil {
		t.Fatal(err)
	}
	return doc.HTML
}

const page = "<html><head><title>x</title></head><body><main>text</main></body></html>"

func TestInjectedCodeGoesWhereItBelongsOnThePagesItCovers(t *testing.T) {
	p := load(t, map[string]string{"plugin.yaml": comments})
	post := &content.Content{ID: "01J8KQ2P3R4S5T6V7W8X9YZ000", Kind: "post", Title: "Hello"}
	about := &content.Content{ID: "01J8KQ2P3R4S5T6V7W8X9YZ900", Kind: "page", Title: "About"}

	got := inject(t, p, map[string]any{"repo": "me/notes"},
		hook.HTMLDoc{Item: post, URL: "/blog/posts/hello/", Kind: "single", HTML: page})
	if !strings.Contains(got, `<link rel="stylesheet" href="/blog/plugins/comments/comments.css">`+"\n</head>") {
		t.Errorf("the stylesheet is not at the end of the head:\n%s", got)
	}
	if !strings.Contains(got, `data-repo="me/notes" data-term="01J8KQ2P3R4S5T6V7W8X9YZ000" async></script>`) ||
		!strings.HasSuffix(got, "</script>\n\n</body></html>") {
		t.Errorf("the giscus script is not at the end of the body:\n%s", got)
	}

	// Not a post: the head goes in, the thread does not.
	got = inject(t, p, nil, hook.HTMLDoc{Item: about, URL: "/blog/about/", Kind: "single", HTML: page})
	if strings.Contains(got, "giscus") || !strings.Contains(got, "comments.css") {
		t.Errorf("a page that is not a post:\n%s", got)
	}
	// A listing has no thread either.
	if got := inject(t, p, nil, hook.HTMLDoc{URL: "/blog/", Kind: "home", HTML: page}); strings.Contains(got, "<script") {
		t.Errorf("the home page got a thread:\n%s", got)
	}

	// Another provider picks the other rule, and a setting in a script is
	// written as a value there rather than as code.
	got = inject(t, p, map[string]any{"provider": "waline", "server": `https://c.example.com/"};alert(1);//`},
		hook.HTMLDoc{Item: post, URL: "/blog/posts/hello/", Kind: "single", HTML: page})
	if strings.Contains(got, "giscus.app") {
		t.Errorf("the giscus rule ran for waline:\n%s", got)
	}
	if !strings.Contains(got, `{server: "https://c.example.com/\"};alert(1);//", page: "https://example.com/blog/posts/hello/"}`) {
		t.Errorf("the settings were not written as values:\n%s", got)
	}
}

// A post turns a plugin's code off in its front matter, however its author
// writes the switch.
func TestAPageSkipsCodeItsFrontMatterTurnsOff(t *testing.T) {
	p := load(t, map[string]string{"plugin.yaml": comments})
	for _, off := range []any{false, "false", "no", "Off"} {
		post := &content.Content{ID: "01J8KQ2P3R4S5T6V7W8X9YZ000", Kind: "post", Meta: map[string]any{"comments": off}}
		got := inject(t, p, nil, hook.HTMLDoc{Item: post, URL: "/posts/hello/", Kind: "single", HTML: page})
		if strings.Contains(got, "giscus") {
			t.Errorf("comments: %v still got the thread", off)
		}
	}
	for _, on := range []any{true, "yes", 1} {
		post := &content.Content{ID: "01J8KQ2P3R4S5T6V7W8X9YZ000", Kind: "post", Meta: map[string]any{"comments": on}}
		got := inject(t, p, nil, hook.HTMLDoc{Item: post, URL: "/posts/hello/", Kind: "single", HTML: page})
		if !strings.Contains(got, "giscus") {
			t.Errorf("comments: %v lost the thread", on)
		}
	}
}

func TestCodeStillGoesInWhenAPageLeavesOutItsTags(t *testing.T) {
	p := load(t, map[string]string{"plugin.yaml": comments})
	post := &content.Content{ID: "01J8KQ2P3R4S5T6V7W8X9YZ000", Kind: "post"}
	got := inject(t, p, nil, hook.HTMLDoc{Item: post, Kind: "single", HTML: "<title>x</title><body><p>text</p>"})
	if !strings.HasPrefix(got, "<title>x</title><link") {
		t.Errorf("the head code is not before the body:\n%s", got)
	}
	if !strings.HasSuffix(got, "</script>\n\n") {
		t.Errorf("the body code is not at the end:\n%s", got)
	}
}

// A page the plugin touches has to be built again when anything that decides
// what it writes changes, or the site keeps the old code.
func TestTheCacheKeyFollowsTheSettings(t *testing.T) {
	p := load(t, map[string]string{"plugin.yaml": comments})
	key := func(settings map[string]any) []byte {
		in, err := p.Injector(settings, plugin.Site{}, plugin.Links{Asset: func(string) string { return "" }})
		if err != nil {
			t.Fatal(err)
		}
		return in.CacheKey()
	}
	a, b := key(map[string]any{"repo": "me/a"}), key(map[string]any{"repo": "me/b"})
	if bytes.Equal(a, b) {
		t.Error("two settings share a cache key")
	}
	if !bytes.Equal(a, key(map[string]any{"repo": "me/a"})) {
		t.Error("the same settings give two cache keys")
	}
}
