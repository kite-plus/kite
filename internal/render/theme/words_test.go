package theme_test

import (
	"testing"
	"testing/fstest"

	"github.com/kite-plus/kite/internal/pack"
	"github.com/kite-plus/kite/internal/render/theme"
)

func packs(t *testing.T, files map[string]string) pack.Packs {
	t.Helper()
	fsys := fstest.MapFS{}
	for name, body := range files {
		fsys["i18n/"+name] = file(body)
	}
	p, err := pack.Load(fsys, "test")
	if err != nil {
		t.Fatal(err)
	}
	return p
}

// sayIn draws src with the words of a theme and a site in a language.
func sayIn(t *testing.T, lang, src string, data any) string {
	t.Helper()
	th := packs(t, map[string]string{
		"en.yaml": `
read_more: Read more
posts: {one: "{{ .Count }} post", other: "{{ .Count }} posts"}
of: "{{ .Count }} of {{ .Total }}"
only_english: Only in English
quote: 'Say "hi" & go'
theme: {title: Paper}
`,
		"zh-CN.yaml": `
read_more: 阅读全文
posts: "{{ .Count }} 篇"
of: "第 {{ .Count }} 篇，共 {{ .Total }} 篇"
`,
		"ru.yaml": `
posts: {one: "{{ .Count }} запись", few: "{{ .Count }} записи", many: "{{ .Count }} записей"}
`,
	})
	site := packs(t, map[string]string{"zh.yaml": "read_more: 继续读\n"})
	e := theme.NewEngine(theme.Options{
		Sources: []theme.Source{{Name: "theme", FS: fstest.MapFS{"single.html": file(src)}}},
		Words:   theme.NewWords(lang, site, th),
	})
	out, err := e.Render(theme.Target{Kind: "single"}, data)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	return string(out)
}

// A theme's pages say their words in the site's language, as its packs and
// the site's own write them; a word the language lacks is said in English,
// and one nobody has is its key.
func TestTSaysAThemesWordsInTheSitesLanguage(t *testing.T) {
	for _, c := range []struct{ lang, src, want string }{
		{"en", `{{ T "read_more" }}`, "Read more"},
		{"zh-CN", `{{ T "read_more" }}`, "继续读"},
		{"zh-TW", `{{ T "read_more" }}`, "继续读"},
		{"zh-CN", `{{ T "only_english" }}`, "Only in English"},
		{"fr", `{{ T "read_more" }}`, "Read more"},
		{"en", `{{ T "missing.key" }}`, "missing.key"},
		{"en", `{{ T "theme.title" }}`, "Paper"},
		{"en", `{{ i18n.T "read_more" }}|{{ i18n.Has "read_more" }}|{{ i18n.Has "missing" }}`, "Read more|true|false"},
	} {
		if got := sayIn(t, c.lang, c.src, nil); got != c.want {
			t.Errorf("%s %s = %q, want %q", c.lang, c.src, got, c.want)
		}
	}
}

// A count chooses a word's plural form by the language's rule, and it and
// anything else given are put in the word, as "8 of 13" needs.
func TestTPutsCountsInTheirPluralForm(t *testing.T) {
	for _, c := range []struct{ lang, src, want string }{
		{"en", `{{ T "posts" 1 }}; {{ T "posts" 8 }}; {{ T "posts" 0 }}`, "1 post; 8 posts; 0 posts"},
		{"zh-CN", `{{ T "posts" 1 }}; {{ T "posts" 8 }}`, "1 篇; 8 篇"},
		{"ru", `{{ T "posts" 1 }}; {{ T "posts" 3 }}; {{ T "posts" 5 }}; {{ T "posts" 21 }}; {{ T "posts" 12 }}`,
			"1 запись; 3 записи; 5 записей; 21 запись; 12 записей"},
		{"en", `{{ T "of" (dict "Count" 8 "Total" 13) }}`, "8 of 13"},
		{"zh-CN", `{{ T "of" (dict "Count" 8 "Total" 13) }}`, "第 8 篇，共 13 篇"},
		{"en", `{{ T "posts" .N }}`, "3 posts"},
	} {
		if got := sayIn(t, c.lang, c.src, map[string]any{"N": int64(3)}); got != c.want {
			t.Errorf("%s %s = %q, want %q", c.lang, c.src, got, c.want)
		}
	}
}

// A word is text, escaped once for wherever it lands, so it can go into an
// attribute or a script; a script gets its words as a JSON object.
func TestAWordIsEscapedWhereItLands(t *testing.T) {
	got := sayIn(t, "en", `<a title="{{ T "quote" }}">{{ T "quote" }}</a><script>const words = {{ i18n.Words "read_more" "posts" }};</script>`, nil)
	want := `<a title="Say &#34;hi&#34; &amp; go">Say &#34;hi&#34; &amp; go</a>` +
		`<script>const words = {"posts":"{{ .Count }} posts","read_more":"Read more"};</script>`
	if got != want {
		t.Errorf("got  %s\nwant %s", got, want)
	}
}

// An engine given no words says every key as it is, as a theme drawn with
// no packs at all does.
func TestWithoutWordsTSaysTheKey(t *testing.T) {
	got, err := render(t, `{{ T "read_more" 3 }}`, nil)
	if err != nil || got != "read_more" {
		t.Errorf("got %q, %v", got, err)
	}
}
