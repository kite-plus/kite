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
	"time"

	"github.com/kite-plus/kite/internal/build"
	"github.com/kite-plus/kite/internal/buildinfo"
	"github.com/kite-plus/kite/internal/config"
	"github.com/kite-plus/kite/internal/hook"
	"github.com/kite-plus/kite/internal/hook/builtin"
	"github.com/kite-plus/kite/internal/index"
	"github.com/kite-plus/kite/internal/project"
	"github.com/kite-plus/kite/internal/reader"
	"github.com/kite-plus/kite/internal/render"
	"github.com/kite-plus/kite/internal/render/markdown"
	"github.com/kite-plus/kite/internal/render/theme"
	kurl "github.com/kite-plus/kite/internal/render/url"
	"github.com/kite-plus/kite/themes"
)

// Site is an opened project with every layer assembled.
type Site struct {
	Project  *project.Project
	Config   *config.Config
	Index    *index.Index
	Reader   *reader.Reader
	Resolver *kurl.Resolver
	Theme    *theme.Theme
	Engine   *theme.Engine
	Markdown *markdown.Renderer
	Hooks    *hook.Bus
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
	if _, err := ix.Reconcile(ctx); err != nil {
		_ = ix.Close()
		return nil, err
	}

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
		_ = ix.Close()
		return nil, err
	}

	th, sources, err := loadTheme(p.Root, cfg.Theme.Name)
	if err != nil {
		_ = ix.Close()
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
		Engine:   theme.NewEngine(theme.Options{Sources: sources}),
		Markdown: markdown.New(markdown.Options{
			UnsafeHTML:     cfg.Markdown.UnsafeHTML,
			Typographer:    cfg.Markdown.Typographer,
			HardWraps:      cfg.Markdown.HardWraps,
			HighlightTheme: cfg.Markdown.HighlightTheme,
		}),
		Hooks: bus,
	}, nil
}

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

// Build renders the whole site to disk.
func (s *Site) Build(ctx context.Context, opts BuildOptions) (build.Stats, []string, error) {
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

	builder, err := build.New(build.Options{
		Site: render.SiteInfo{
			Title:         s.Config.Site.Title,
			Description:   s.Config.Site.Description,
			BaseURL:       s.Config.Site.BaseURL,
			Language:      s.Config.Site.Language,
			Params:        s.Config.Site.Params,
			ThemeSettings: s.ThemeSettings(),
			Version:       buildinfo.Version,
		},
		Reader:        s.Reader,
		Resolver:      s.Resolver,
		Engine:        s.Engine,
		Markdown:      s.Markdown,
		Hooks:         s.Hooks,
		Types:         s.Project.Types,
		Emitter:       emitter,
		PageSize:      s.Config.Build.PageSize,
		IncludeDrafts: opts.Drafts,
		Now:           opts.Now,
	})
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

// ThemeSettings merges the theme's declared defaults with the site's overrides,
// so a template can read a setting the site never mentioned.
func (s *Site) ThemeSettings() map[string]any {
	out := s.Theme.Manifest.DefaultSettings()
	if out == nil {
		out = make(map[string]any)
	}
	for k, v := range s.Config.Theme.Settings {
		out[k] = v
	}
	return out
}
