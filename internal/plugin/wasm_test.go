package plugin_test

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"sync"
	"testing"
	"testing/fstest"
	"time"

	"github.com/kite-plus/kite/internal/content"
	"github.com/kite-plus/kite/internal/hook"
	"github.com/kite-plus/kite/internal/plugin"
	"github.com/kite-plus/kite/internal/plugin/plugintest"
)

// loadGuest loads testguest as a plugin declaring hooks.
func loadGuest(t *testing.T, hooks ...string) *plugin.Plugin {
	t.Helper()
	fsys := tree(map[string]string{"plugin.yaml": plugintest.Manifest("guest", hooks...)})
	fsys[plugin.WasmName] = &fstest.MapFile{Data: plugintest.Wasm(t)}
	p, err := plugin.Load(fsys, "guest")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	return p
}

var guestLinks = plugin.Links{
	Absolute: func(url string) string { return "https://example.com" + url },
	Item:     func(item *content.Content) string { return "/posts/" + item.Slug + "/" },
}

// guestBus puts the guest's hooks on a bus, with settings.
func guestBus(t *testing.T, p *plugin.Plugin, settings map[string]any) *hook.Bus {
	t.Helper()
	hooks, err := p.Hooks(t.Context(), settings, plugin.Site{Title: "Notes"}, guestLinks, t.TempDir())
	if err != nil {
		t.Fatalf("Hooks: %v", err)
	}
	bus := hook.NewBus()
	for _, h := range hooks {
		bus.Register(h, hook.DefaultPriority)
	}
	return bus
}

// complete runs build_complete over pages and returns what it wrote.
func complete(t *testing.T, bus *hook.Bus, pages []hook.PageInfo) (map[string]string, error) {
	t.Helper()
	files := map[string]string{}
	err := bus.BuildComplete(t.Context(), &hook.BuildInfo{
		Pages: pages,
		Emit: func(path string, data []byte) error {
			files[path] = string(data)
			return nil
		},
	})
	return files, err
}

var hello = &content.Content{ID: "01J8KQ2P3R4S5T6V7W8X9YZ000", Kind: "post", Slug: "hello", Title: "Hello"}

func TestAModuleRewritesSourcesAndPages(t *testing.T) {
	p := loadGuest(t, plugin.HookTransformMarkdown, plugin.HookTransformHTML)
	bus := guestBus(t, p, map[string]any{"sign": "seen"})

	doc := hook.MarkdownDoc{Item: hello, Source: "Look, :kite:!"}
	if err := bus.TransformMarkdown(t.Context(), &doc); err != nil {
		t.Fatal(err)
	}
	if want := "Look, a kite at /posts/hello/!"; doc.Source != want {
		t.Errorf("markdown = %q, want %q", doc.Source, want)
	}
	untouched := hook.MarkdownDoc{Item: hello, Source: "Nothing to see."}
	if err := bus.TransformMarkdown(t.Context(), &untouched); err != nil || untouched.Source != "Nothing to see." {
		t.Errorf("a module that answers with nothing changed the source to %q (%v)", untouched.Source, err)
	}

	page := hook.HTMLDoc{Item: hello, URL: "/posts/hello/", Kind: "single", HTML: "<html><head></head><body></body></html>"}
	if err := bus.TransformHTML(t.Context(), &page); err != nil {
		t.Fatal(err)
	}
	if want := `<meta name="signed" content="seen single /posts/hello/"></head>`; !strings.Contains(page.HTML, want) {
		t.Errorf("html = %q, want it to hold %q", page.HTML, want)
	}
}

// Pages are rendered on every core, and every one of them may call the module
// at once.
func TestAModuleServesPagesRenderedAtOnce(t *testing.T) {
	p := loadGuest(t, plugin.HookTransformHTML)
	bus := guestBus(t, p, nil)

	var wg sync.WaitGroup
	errs := make([]error, 32)
	for i := range errs {
		wg.Go(func() {
			url := fmt.Sprintf("/posts/%d/", i)
			doc := hook.HTMLDoc{URL: url, Kind: "single", HTML: "<head></head>"}
			if errs[i] = bus.TransformHTML(context.Background(), &doc); errs[i] == nil && !strings.Contains(doc.HTML, url) {
				errs[i] = fmt.Errorf("page %d came back as %q", i, doc.HTML)
			}
		})
	}
	wg.Wait()
	for _, err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestAModuleWritesFilesOfItsOwnWhenTheBuildIsDone(t *testing.T) {
	p := loadGuest(t, plugin.HookBuildComplete)
	bus := guestBus(t, p, nil)
	pages := []hook.PageInfo{
		{Item: hello, URL: "/posts/hello/", Kind: "single", Title: "Hello", Text: "Cherry trees\nThey bloom.", Indexable: true},
		{URL: "/", Kind: "home", Title: "Notes", Indexable: true},
		{URL: "/404.html", Kind: "notFound", Indexable: false},
	}
	files, err := complete(t, bus, pages)
	if err != nil {
		t.Fatal(err)
	}
	var index []map[string]string
	if err := json.Unmarshal([]byte(files["plugins/guest/index.json"]), &index); err != nil {
		t.Fatalf("index.json: %v in %q", err, files)
	}
	if len(index) != 1 || index[0]["url"] != "/posts/hello/" || index[0]["text"] != "Cherry trees\nThey bloom." ||
		index[0]["title"] != "Hello" || index[0]["type"] != "post" {
		t.Errorf("index = %v, want the one post with its text", index)
	}

	// The clock does not tell the time and chance is the same every build,
	// so building the same site twice writes the same files.
	again, err := complete(t, bus, pages)
	if err != nil {
		t.Fatal(err)
	}
	if files["plugins/guest/chance.txt"] != again["plugins/guest/chance.txt"] {
		t.Errorf("two builds differ: %q and %q", files["plugins/guest/chance.txt"], again["plugins/guest/chance.txt"])
	}
}

func TestAModuleCannotWriteOutsideItsOwnDirectory(t *testing.T) {
	p := loadGuest(t, plugin.HookBuildComplete)
	bus := guestBus(t, p, map[string]any{"mode": "escape"})
	files, err := complete(t, bus, nil)
	if err == nil || !strings.Contains(err.Error(), "../../outside.json") {
		t.Fatalf("err = %v, want the escaping path refused", err)
	}
	if len(files) != 0 {
		t.Errorf("wrote %v", files)
	}
}

func TestAModuleThatMisbehavesFailsTheBuildNamingThePlugin(t *testing.T) {
	p := loadGuest(t, plugin.HookTransformMarkdown)
	misbehave := func(mode string) error {
		bus := guestBus(t, p, map[string]any{"mode": mode})
		doc := hook.MarkdownDoc{Item: hello, Source: "text"}
		return bus.TransformMarkdown(t.Context(), &doc)
	}
	for mode, want := range map[string]string{
		"fail":  "refused on purpose",
		"fetch": "not allowed",
	} {
		if err := misbehave(mode); err == nil || !strings.Contains(err.Error(), want) || !strings.Contains(err.Error(), "plugin guest") {
			t.Errorf("%s: err = %v, want one naming the plugin and saying %q", mode, err, want)
		}
	}

	// Only the module that never stops gets a short limit: a first call also
	// starts the module's runtime, which a slow runner can take a while over.
	restore := plugin.SetPageLimit(200 * time.Millisecond)
	err := misbehave("spin")
	restore()
	if err == nil || !strings.Contains(err.Error(), "took longer than 200ms") || !strings.Contains(err.Error(), "plugin guest") {
		t.Errorf("spin: err = %v, want one naming the plugin and its time limit", err)
	}

	// A module that failed does not take the next page down with it.
	bus := guestBus(t, p, nil)
	doc := hook.MarkdownDoc{Item: hello, Source: ":kite:"}
	if err := bus.TransformMarkdown(t.Context(), &doc); err != nil || doc.Source != "a kite at /posts/hello/" {
		t.Errorf("after the failures: %q, %v", doc.Source, err)
	}
}

func TestAHookTheModuleDoesNotExportRefusesThePlugin(t *testing.T) {
	p := loadGuest(t, plugin.HookTransformHTML)
	p.Manifest.Hooks = []string{"transform_nothing"}
	if err := p.Check(t.Context(), ""); err == nil || !strings.Contains(err.Error(), "transform_nothing") {
		t.Errorf("Check = %v, want the missing export named", err)
	}

	junk := tree(map[string]string{"plugin.yaml": plugintest.Manifest("guest", plugin.HookTransformHTML), plugin.WasmName: "not a module"})
	p, err := plugin.Load(junk, "guest")
	if err != nil {
		t.Fatal(err)
	}
	if err := p.Check(t.Context(), ""); err == nil || !strings.Contains(err.Error(), plugin.WasmName) {
		t.Errorf("Check = %v, want the module refused", err)
	}
}

func TestTheCacheKeyFollowsTheModuleAndItsSettings(t *testing.T) {
	p := loadGuest(t, plugin.HookTransformHTML)
	key := func(settings map[string]any) string {
		hooks, err := p.Hooks(t.Context(), settings, plugin.Site{}, guestLinks, "")
		if err != nil {
			t.Fatal(err)
		}
		return string(hooks[0].CacheKey())
	}
	before := key(nil)
	if key(map[string]any{"sign": "other"}) == before {
		t.Error("changing a setting kept the cache key")
	}
	// A custom section changes the module's bytes and nothing it does.
	p.Wasm = append(slices.Clone(p.Wasm), 0x00, 0x06, 0x04, 'k', 'i', 't', 'e', 1)
	if key(nil) == before {
		t.Error("changing the module kept the cache key")
	}
}
