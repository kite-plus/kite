package plugin

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"runtime"
	"strings"
	"sync"
	"time"

	extism "github.com/extism/go-sdk"
	"github.com/tetratelabs/wazero"

	"github.com/kite-plus/kite/internal/hook"
)

// memoryPages caps a module's memory at 64 MiB, in pages of 64 KiB.
const memoryPages = 1024

// How long a module may take: over one page, and over the whole site in
// build_complete. Variables for the tests.
var (
	pageLimit = 10 * time.Second
	siteLimit = 2 * time.Minute
)

// module is a compiled plugin.wasm, with instances of it kept for reuse. An
// instance takes one call at a time, and a build renders pages on every core,
// so each call takes an instance of its own.
type module struct {
	compiled *extism.CompiledPlugin
	idle     chan *extism.Plugin
}

// modules holds the modules this process has compiled, by the sha256 of their
// bytes. A site is loaded again whenever its settings change, and compiling a
// module takes far longer than anything else about loading a plugin.
//
// An author rebuilding a plugin while previewing makes a new module each
// time, so only the latest few are kept. One that is dropped is not closed:
// a site loaded before may still be rendering with it, and it goes when that
// site does.
var (
	modulesMu    sync.Mutex
	modules      = map[[sha256.Size]byte]*module{}
	compiledSums [][sha256.Size]byte
)

const keptModules = 16

// compileModule returns wasm compiled, compiling it at most once a process.
// cacheDir, when not empty, keeps the machine code between runs.
func compileModule(ctx context.Context, wasm []byte, cacheDir string) (*module, error) {
	sum := sha256.Sum256(wasm)
	modulesMu.Lock()
	defer modulesMu.Unlock()
	if m, ok := modules[sum]; ok {
		return m, nil
	}

	config := wazero.NewRuntimeConfig().
		WithCloseOnContextDone(true).
		WithMemoryLimitPages(memoryPages)
	if cacheDir != "" {
		// Without the cache a module is only compiled again, so a directory
		// that cannot be made is no reason to refuse it.
		if cache, err := wazero.NewCompilationCacheWithDir(cacheDir); err == nil {
			config = config.WithCompilationCache(cache)
		}
	}
	// No allowed hosts and no allowed paths: a module reaches no network and
	// no files.
	compiled, err := extism.NewCompiledPlugin(ctx,
		extism.Manifest{Wasm: []extism.Wasm{extism.WasmData{Data: wasm}}},
		extism.PluginConfig{RuntimeConfig: config, EnableWasi: true},
		nil)
	if err != nil {
		return nil, err
	}
	m := &module{compiled: compiled, idle: make(chan *extism.Plugin, runtime.GOMAXPROCS(0))}
	modules[sum] = m
	if compiledSums = append(compiledSums, sum); len(compiledSums) > keptModules {
		delete(modules, compiledSums[0])
		compiledSums = compiledSums[1:]
	}
	return m, nil
}

// instance takes an idle instance, or makes one.
func (m *module) instance(ctx context.Context) (*extism.Plugin, error) {
	select {
	case p := <-m.idle:
		return p, nil
	default:
	}
	return m.fresh(ctx)
}

// fresh makes an instance no call has run in.
func (m *module) fresh(ctx context.Context) (*extism.Plugin, error) {
	// A new module config gives an instance a clock that does not tell the
	// time, random numbers that are the same in every instance, and nowhere
	// to write.
	return m.compiled.Instance(ctx, extism.PluginInstanceConfig{ModuleConfig: wazero.NewModuleConfig()})
}

// release keeps an instance for a later call, or closes it when as many are
// kept as there are cores to call them from.
func (m *module) release(p *extism.Plugin) {
	select {
	case m.idle <- p:
	default:
		_ = p.Close(context.Background())
	}
}

// call runs an export over input within limit and returns its output.
//
// A call alone runs in a fresh instance, so that nothing an earlier call did
// can change what it answers. Transforms run once a page, where making an
// instance each time would cost more than the call, so they share instances
// and are asked to keep nothing between calls.
func (m *module) call(ctx context.Context, name string, input []byte, limit time.Duration, alone bool) ([]byte, error) {
	take := m.instance
	if alone {
		take = m.fresh
	}
	p, err := take(ctx)
	if err != nil {
		return nil, err
	}
	run, cancel := context.WithTimeout(ctx, limit)
	defer cancel()
	code, out, err := p.CallWithContext(run, name, input)
	if err == nil && code != 0 {
		err = fmt.Errorf("it returned %d", code)
	}
	// A trap or a timeout can leave an instance half way through a call, so
	// one that failed is not used again.
	if err != nil || alone {
		_ = p.Close(context.Background())
	} else {
		m.release(p)
	}
	switch {
	case err == nil:
		return out, nil
	case ctx.Err() == nil && errors.Is(run.Err(), context.DeadlineExceeded):
		return nil, fmt.Errorf("%s took longer than %s", name, limit)
	default:
		return nil, fmt.Errorf("%s: %w", name, err)
	}
}

// exports checks that the module exports every hook the plugin declares, so
// that a misspelled one refuses the plugin when it loads rather than failing
// a build.
func (m *module) exports(ctx context.Context, hooks []string) error {
	p, err := m.instance(ctx)
	if err != nil {
		return err
	}
	defer m.release(p)
	for _, name := range hooks {
		if !p.FunctionExists(name) {
			return fmt.Errorf("the plugin declares %s, which %s does not export", name, WasmName)
		}
	}
	return nil
}

// Check compiles the plugin's module and checks that it exports every hook
// the plugin declares. A plugin that declares none passes.
func (p *Plugin) Check(ctx context.Context, cacheDir string) error {
	_, err := p.module(ctx, cacheDir)
	return err
}

// module compiles the plugin's module and checks its exports.
func (p *Plugin) module(ctx context.Context, cacheDir string) (*module, error) {
	if len(p.Manifest.Hooks) == 0 {
		return nil, nil
	}
	m, err := compileModule(ctx, p.Wasm, cacheDir)
	if err != nil {
		return nil, fmt.Errorf("plugin %s: %s: %w", p.Manifest.ID, WasmName, err)
	}
	if err := m.exports(ctx, p.Manifest.Hooks); err != nil {
		return nil, fmt.Errorf("plugin %s: %w", p.Manifest.ID, err)
	}
	return m, nil
}

// Hooks returns the hooks that run the plugin's module during a build, given
// the settings a site gave it, or none when the plugin declares none.
func (p *Plugin) Hooks(ctx context.Context, settings map[string]any, site Site, links Links, cacheDir string) ([]hook.Hook, error) {
	m, err := p.module(ctx, cacheDir)
	if m == nil || err != nil {
		return nil, err
	}
	resolved := p.Settings(settings)
	encoded, err := json.Marshal(resolved)
	if err != nil {
		return nil, fmt.Errorf("plugin %s: settings: %w", p.Manifest.ID, err)
	}

	// What the output depends on: the module, the manifest, and the settings
	// and site it runs with.
	key, err := json.Marshal([]any{sha256.Sum256(p.Wasm), p.Manifest, resolved, site})
	if err != nil {
		return nil, err
	}
	sum := sha256.Sum256(key)

	r := &runner{
		Base: hook.Base{
			HookName:    "plugin " + p.Manifest.ID,
			HookPhase:   hook.PhaseBuild,
			HookVersion: fmt.Sprintf("%x", sum),
		},
		id:       p.Manifest.ID,
		module:   m,
		settings: encoded,
		site:     site,
		links:    links,
	}
	hooks := make([]hook.Hook, 0, len(p.Manifest.Hooks))
	for _, name := range p.Manifest.Hooks {
		switch name {
		case HookTransformMarkdown:
			hooks = append(hooks, markdownHook{r})
		case HookTransformHTML:
			hooks = append(hooks, htmlHook{r})
		case HookBuildComplete:
			hooks = append(hooks, completeHook{r})
		}
	}
	return hooks, nil
}

// runner calls one plugin's module for the hooks it declares.
type runner struct {
	hook.Base
	id       string
	module   *module
	settings json.RawMessage
	site     Site
	links    Links
}

// The input of each hook, and the output it answers with. A transform that
// answers with nothing, or leaves its field out, keeps its document as it was.
type (
	markdownInput struct {
		Settings json.RawMessage `json:"settings"`
		Site     Site            `json:"site"`
		Page     Page            `json:"page"`
		Markdown string          `json:"markdown"`
	}
	markdownOutput struct {
		Markdown *string `json:"markdown"`
	}

	htmlInput struct {
		Settings json.RawMessage `json:"settings"`
		Site     Site            `json:"site"`
		Page     Page            `json:"page"`
		HTML     string          `json:"html"`
	}
	htmlOutput struct {
		HTML *string `json:"html"`
	}

	completeInput struct {
		Settings json.RawMessage `json:"settings"`
		Site     Site            `json:"site"`
		Pages    []BuiltPage     `json:"pages"`
	}
	completeOutput struct {
		Files []struct {
			Path    string `json:"path"`
			Content string `json:"content"`
		} `json:"files"`
	}
)

// BuiltPage is what build_complete is told about each page of the site.
type BuiltPage struct {
	Page
	Excerpt string `json:"excerpt,omitempty"`
	// Text is the plain text of the item's body, a block to a line.
	Text string `json:"text,omitempty"`
}

// run calls one export with input as JSON and decodes its answer into output,
// reporting whether there was one. build_complete runs once a build, alone.
func (r *runner) run(ctx context.Context, name string, input, output any, limit time.Duration) (bool, error) {
	data, err := json.Marshal(input)
	if err != nil {
		return false, err
	}
	out, err := r.module.call(ctx, name, data, limit, name == HookBuildComplete)
	if err != nil || len(out) == 0 {
		return false, err
	}
	if err := json.Unmarshal(out, output); err != nil {
		return false, fmt.Errorf("%s answered with something other than JSON: %w", name, err)
	}
	return true, nil
}

type markdownHook struct{ *runner }

// TransformMarkdown hands an item's source to the module to rewrite. The
// source is rendered for the item's own page, which is the page it is told
// about.
func (h markdownHook) TransformMarkdown(ctx context.Context, doc *hook.MarkdownDoc) error {
	var url string
	if h.links.Item != nil {
		url = h.links.Item(doc.Item)
	}
	in := markdownInput{
		Settings: h.settings,
		Site:     h.site,
		Page:     pageOf(url, "single", doc.Item, h.links),
		Markdown: doc.Source,
	}
	var out markdownOutput
	answered, err := h.run(ctx, HookTransformMarkdown, in, &out, pageLimit)
	if answered && out.Markdown != nil {
		doc.Source = *out.Markdown
	}
	return err
}

type htmlHook struct{ *runner }

// TransformHTML hands a rendered page to the module to rewrite.
func (h htmlHook) TransformHTML(ctx context.Context, doc *hook.HTMLDoc) error {
	in := htmlInput{
		Settings: h.settings,
		Site:     h.site,
		Page:     pageOf(doc.URL, doc.Kind, doc.Item, h.links),
		HTML:     doc.HTML,
	}
	var out htmlOutput
	answered, err := h.run(ctx, HookTransformHTML, in, &out, pageLimit)
	if answered && out.HTML != nil {
		doc.HTML = *out.HTML
	}
	return err
}

type completeHook struct{ *runner }

// BuildComplete hands the module every page of the site but the error page,
// and writes the files it answers with under plugins/<id>/, the one place in
// the output that is the plugin's own.
func (h completeHook) BuildComplete(ctx context.Context, build *hook.BuildInfo) error {
	in := completeInput{Settings: h.settings, Site: h.site, Pages: []BuiltPage{}}
	for _, info := range build.Pages {
		if !info.Indexable {
			continue
		}
		page := pageOf(info.URL, info.Kind, info.Item, h.links)
		if page.Title == "" {
			page.Title = info.Title
		}
		in.Pages = append(in.Pages, BuiltPage{Page: page, Excerpt: info.Excerpt, Text: info.Text})
	}
	var out completeOutput
	if _, err := h.run(ctx, HookBuildComplete, in, &out, siteLimit); err != nil {
		return err
	}
	for _, f := range out.Files {
		to, err := h.within(f.Path)
		if err != nil {
			return err
		}
		if err := build.Emit(to, []byte(f.Content)); err != nil {
			return err
		}
	}
	return nil
}

// within is where a file a module writes goes in the output, under the
// plugin's own directory, or an error for a path that would leave it.
func (r *runner) within(name string) (string, error) {
	clean := path.Clean(name)
	if name == "" || strings.Contains(name, `\`) || path.IsAbs(clean) ||
		clean == "." || clean == ".." || strings.HasPrefix(clean, "../") {
		return "", fmt.Errorf("%s wrote %q, which is not a path under %s/%s/", HookBuildComplete, name, Dir, r.id)
	}
	return path.Join(Dir, r.id, clean), nil
}
