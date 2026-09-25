// Package site wires a project's pieces into something that can be built or
// served. It is the only place that knows about every layer at once; the
// layers themselves stay unaware of each other.
package site

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kite-plus/kite/internal/build"
	"github.com/kite-plus/kite/internal/buildinfo"
	"github.com/kite-plus/kite/internal/config"
	"github.com/kite-plus/kite/internal/hook"
	"github.com/kite-plus/kite/internal/hook/builtin"
	"github.com/kite-plus/kite/internal/index"
	"github.com/kite-plus/kite/internal/project"
	"github.com/kite-plus/kite/internal/publish"
	gitpub "github.com/kite-plus/kite/internal/publish/git"
	"github.com/kite-plus/kite/internal/reader"
	"github.com/kite-plus/kite/internal/render"
	"github.com/kite-plus/kite/internal/render/markdown"
	"github.com/kite-plus/kite/internal/render/theme"
	kurl "github.com/kite-plus/kite/internal/render/url"
	"github.com/kite-plus/kite/themes"
)

// Site is an opened project with every layer assembled.
type Site struct {
	// Problems lists content the index refused, such as a file with no id.
	//
	// Opening does not fail on them, because how much to tolerate is the
	// caller's decision rather than the project's: a build refuses to publish
	// a site it cannot fully see, while a preview keeps serving the rest and
	// says on the page what it skipped.
	Problems []string

	Project  *project.Project
	Config   *config.Config
	Index    *index.Index
	Reader   *reader.Reader
	Resolver *kurl.Resolver
	Theme    *theme.Theme
	Engine   *theme.Engine
	Markdown *markdown.Renderer
	Hooks    *hook.Bus

	// publisher is made once, because it remembers what the host has said
	// about deployments between one question and the next.
	publisher publish.Publisher
}

// Open loads a project and everything it needs to render.
func Open(ctx context.Context, dir string) (*Site, error) {
	p, err := project.Open(dir)
	if err != nil {
		return nil, err
	}
	cfg, err := config.Load(p.Root)
	if err != nil {
		return nil, err
	}

	ix, err := index.Open(p.Root, p.Types)
	if err != nil {
		return nil, err
	}
	var problems []string
	if _, err := ix.Reconcile(ctx); err != nil {
		problems = strings.Split(err.Error(), "\n")
	}

	s, err := assemble(p, cfg, ix)
	if err != nil {
		_ = ix.Close()
		return nil, err
	}
	s.Problems = problems
	return s, nil
}

// Reconfigure re-reads the configuration and rebuilds everything derived from
// it, returning a new site that shares this one's project and index.
//
// A settings change has to take effect without a restart, or the admin would
// report a title the site is not serving. The index is kept because nothing
// in it depends on configuration, and swapping it would pull the database out
// from under requests already in flight.
func (s *Site) Reconfigure() (*Site, error) {
	cfg, err := config.Load(s.Project.Root)
	if err != nil {
		return nil, err
	}
	out, err := assemble(s.Project, cfg, s.Index)
	if err != nil {
		return nil, err
	}
	out.Problems = s.Problems
	return out, nil
}

// assemble builds the half of a site that comes from its configuration.
func assemble(p *project.Project, cfg *config.Config, ix *index.Index) (*Site, error) {
	resolver, err := kurl.New(kurl.Options{
		BaseURL:        cfg.Site.BaseURL,
		Style:          kurl.Style(cfg.Build.URLStyle),
		PaginationPath: "page",
		TaxonomyRoute:  "/:taxonomy",
		TermRoute:      "/:taxonomy/:term",
		DefaultLocale:  cfg.Site.Language,
		LocalePrefix:   true,
	}, p.Types)
	if err != nil {
		return nil, err
	}

	th, sources, err := loadTheme(p.Root, cfg.Theme.Name)
	if err != nil {
		return nil, err
	}

	bus := hook.NewBus()
	builtin.Register(bus, builtin.Options{
		Sitemap:   cfg.Build.Sitemap,
		Feed:      cfg.Build.Feed,
		FeedLimit: cfg.Build.FeedLimit,
	})

	return &Site{
		Project:  p,
		Config:   cfg,
		Index:    ix,
		Reader:   reader.New(ix.DB()),
		Resolver: resolver,
		Theme:    th,
		Engine:   theme.NewEngine(theme.Options{Sources: sources, Links: resolver}),
		Markdown: markdown.New(markdown.Options{
			UnsafeHTML:     cfg.Markdown.UnsafeHTML,
			Typographer:    cfg.Markdown.Typographer,
			HardWraps:      cfg.Markdown.HardWraps,
			HighlightTheme: cfg.Markdown.HighlightTheme,
		}),
		Hooks:     bus,
		publisher: newPublisher(p, cfg),
	}, nil
}

func newPublisher(p *project.Project, cfg *config.Config) publish.Publisher {
	switch cfg.Publish.Publisher {
	case "", "git":
		return gitpub.New(gitpub.Options{
			Root:    p.Root,
			Branch:  cfg.Publish.Branch,
			Message: cfg.Publish.Message,
		})
	default:
		return nil
	}
}

// Publisher returns the configured publisher, or nil when the project
// publishes nothing.
//
// A project without one is an ordinary way to run: an author may prefer to
// commit and push themselves, and "none" says so rather than leaving a button
// that half works.
func (s *Site) Publisher() publish.Publisher { return s.publisher }

// Close releases the site's resources.
func (s *Site) Close() error { return s.Index.Close() }

// loadTheme resolves the configured theme and the ordered template sources.
//
// The site's own layouts directory comes first so that a site can override any
// single template without copying the theme.
func loadTheme(root, name string) (*theme.Theme, []theme.Source, error) {
	var sources []theme.Source

	siteLayouts := filepath.Join(root, theme.LayoutsDir)
	if info, err := os.Stat(siteLayouts); err == nil && info.IsDir() {
		sources = append(sources, theme.Source{Name: "site", FS: os.DirFS(siteLayouts)})
	}

	themeFS, origin, err := themeFS(root, name)
	if err != nil {
		return nil, nil, err
	}
	th, err := theme.Load(themeFS)
	if err != nil {
		return nil, nil, fmt.Errorf("%s: %w", origin, err)
	}
	sources = append(sources, theme.Source{Name: th.Manifest.Name, FS: th.Layouts})
	return th, sources, nil
}

func themeFS(root, name string) (fs.FS, string, error) {
	if name == "" || name == "default" {
		return themes.Default(), "built-in theme", nil
	}
	dir := filepath.Join(root, "themes", name)
	if info, err := os.Stat(dir); err != nil || !info.IsDir() {
		return nil, "", fmt.Errorf("site: theme %q not found in themes/", name)
	}
	return os.DirFS(dir), "themes/" + name, nil
}

// BuildOptions tunes a build.
type BuildOptions struct {
	OutDir string
	Drafts bool
	Now    time.Time
}

// Builder assembles the build engine without running it, so a server can
// render one target at a time through exactly the path a build would take.
func (s *Site) Builder(opts BuildOptions) (*build.Builder, error) {
	return s.newBuilder(opts, nil)
}

func (s *Site) newBuilder(opts BuildOptions, emitter *build.Emitter) (*build.Builder, error) {
	return build.New(build.Options{
		Site: render.SiteInfo{
			Title:         s.Config.Site.Title,
			Description:   s.Config.Site.Description,
			BaseURL:       s.Config.Site.BaseURL,
			Language:      s.Config.Site.Language,
			Params:        s.Config.Site.Params,
			ThemeSettings: s.ThemeSettings(),
			Version:       buildinfo.Version,
			Build:         emitter != nil,
		},
		Reader:        s.Reader,
		Resolver:      s.Resolver,
		Engine:        s.Engine,
		Markdown:      s.Markdown,
		Hooks:         s.Hooks,
		Types:         s.Project.Types,
		Emitter:       emitter,
		Media:         os.DirFS(s.Project.Root),
		PageSize:      s.Config.Build.PageSize,
		IncludeDrafts: opts.Drafts,
		Now:           opts.Now,
	})
}

// Build renders the whole site to disk.
//
// Content the index could not read stops a build. Publishing a site with a
// page quietly missing is worse than publishing nothing, and the author is at
// a terminal here, where an error is read.
func (s *Site) Build(ctx context.Context, opts BuildOptions) (build.Stats, []string, error) {
	if len(s.Problems) > 0 {
		return build.Stats{}, nil, fmt.Errorf("%s", strings.Join(s.Problems, "\n"))
	}

	outDir := opts.OutDir
	if outDir == "" {
		outDir = filepath.Join(s.Project.Root, s.Config.Build.Output)
	}

	emitter, err := build.NewEmitter(outDir)
	if err != nil {
		return build.Stats{}, nil, err
	}

	// Unprocessed files are copied before rendering so that a template can
	// reference them and a failed render discards them along with everything
	// else.
	if err := emitter.CopyTree(s.Theme.Static, ""); err != nil {
		_ = emitter.Discard()
		return build.Stats{}, nil, err
	}
	if err := emitter.CopyTree(s.Theme.Assets, "assets"); err != nil {
		_ = emitter.Discard()
		return build.Stats{}, nil, err
	}
	staticDir := filepath.Join(s.Project.Root, "static")
	if info, err := os.Stat(staticDir); err == nil && info.IsDir() {
		if err := emitter.CopyTree(os.DirFS(staticDir), ""); err != nil {
			_ = emitter.Discard()
			return build.Stats{}, nil, err
		}
	}

	builder, err := s.newBuilder(opts, emitter)
	if err != nil {
		_ = emitter.Discard()
		return build.Stats{}, nil, err
	}

	stats, err := builder.Run(ctx)
	if err != nil {
		_ = emitter.Discard()
		return stats, nil, err
	}
	return stats, emitter.Files(), nil
}

// ThemeSettings is what templates read as .Site.ThemeSettings: the stored
// settings taken as the types the theme declares them to be, and the theme's
// defaults for whatever the site never set.
func (s *Site) ThemeSettings() map[string]any {
	return s.Theme.Manifest.Settings.Resolve(s.Config.Theme.Settings)
}
