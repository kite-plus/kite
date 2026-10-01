package cli

import (
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/spf13/cobra"

	"github.com/kite-plus/kite/internal/archive"
	"github.com/kite-plus/kite/internal/config"
	"github.com/kite-plus/kite/internal/content"
	"github.com/kite-plus/kite/internal/project"
	"github.com/kite-plus/kite/internal/render/theme"
	"github.com/kite-plus/kite/internal/site"
	"github.com/kite-plus/kite/internal/themecheck"
	"github.com/kite-plus/kite/themes"
)

func newThemeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "theme",
		Short: "Work with themes",
		Long: "A site's themes live in themes/<name>, beside the one built into Kite,\n" +
			"and theme.name in kite.yaml chooses the one in use. These commands make\n" +
			"the same changes the studio's theme page does.",
		Args: cobra.NoArgs,
	}
	cmd.AddCommand(
		newThemeListCmd(),
		newThemeAddCmd(),
		newThemeRemoveCmd(),
		newThemeUseCmd(),
		newThemeVerifyCmd(),
		newThemePackCmd(),
		newThemeNewCmd(),
	)
	return cmd
}

type themeRow struct {
	Name    string `json:"name"`
	Title   string `json:"title,omitempty"`
	Version string `json:"version,omitempty"`
	Builtin bool   `json:"builtin"`
	Active  bool   `json:"active"`
	Problem string `json:"problem,omitempty"`
}

func newThemeListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List the themes a site can use and which one it uses",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			p, cfg, err := openWithConfig()
			if err != nil {
				return err
			}
			var rows []themeRow
			for _, one := range site.Themes(p.Root) {
				row := themeRow{
					Name:    one.Name,
					Builtin: one.Builtin,
					Active:  inUse(cfg, one),
					Problem: one.Problem,
				}
				if m := one.Manifest; m != nil {
					row.Title, row.Version = m.Title, m.Version
				}
				rows = append(rows, row)
			}
			if jsonOut(cmd) {
				return writeJSON(cmd.OutOrStdout(), rows)
			}
			for _, row := range rows {
				mark := " "
				if row.Active {
					mark = "*"
				}
				title := row.Title
				if row.Builtin {
					title += " (built in)"
				}
				printf(cmd, "%s  %-20s %-10s %s\n", mark, row.Name, row.Version, strings.TrimSpace(title))
				if row.Problem != "" {
					printf(cmd, "     %s\n", row.Problem)
				}
			}
			return nil
		},
	}
}

// inUse reports whether a theme is the one a site is built with. A directory
// named default is never used: the name chooses the built-in theme.
func inUse(cfg *config.Config, one site.Installed) bool {
	return one.Name == cfg.Theme.Name && (one.Builtin || one.Name != site.BuiltinTheme)
}

func newThemeAddCmd() *cobra.Command {
	var replace bool
	cmd := &cobra.Command{
		Use:   "add <archive.zip|dir>",
		Short: "Install a theme from a zip archive or a directory",
		Long: "Installs a theme into themes/<name>, checked the way a site checks a\n" +
			"theme it loads. A theme's release carries a zip archive that holds only\n" +
			"the theme. The site keeps its theme until 'kite theme use' switches it.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			p, _, err := openWithConfig()
			if err != nil {
				return err
			}
			files, err := readPackage(args[0], themePackage)
			if err != nil {
				return err
			}
			added, err := theme.Load(archive.FS(files))
			if err != nil {
				return err
			}
			if !added.Manifest.SupportsStatic() {
				return errors.New("the theme says it cannot be built into a static site")
			}
			name := added.Manifest.Name
			switch {
			case name == site.BuiltinTheme:
				return fmt.Errorf("the name %s belongs to the theme built into Kite; the theme needs a name of its own", name)
			case !content.ValidThemeName(name):
				return fmt.Errorf("the theme is named %q, and a name has to be usable as a directory: letters, digits, dots, - and _", name)
			}
			exists := slices.ContainsFunc(site.Themes(p.Root), func(one site.Installed) bool {
				return one.Name == name && !one.Builtin
			})
			if exists && !replace {
				return fmt.Errorf("theme %s is installed already; add --replace to replace it", name)
			}
			verb := "install"
			if exists {
				verb = "replace"
			}
			if _, err := p.Writer().Apply(cmd.Context(), content.ChangeSet{
				Ops:     []content.Op{content.PutTheme{Name: name, Files: files, Replace: exists}},
				Message: strings.TrimSpace(fmt.Sprintf("theme: %s %s %s", verb, name, added.Manifest.Version)),
			}); err != nil {
				return err
			}
			printf(cmd, "installed %s %s in %s\n", name, added.Manifest.Version, filepath.Join(site.ThemesDir, name))
			printf(cmd, "run 'kite theme use %s' to build the site with it\n", name)
			return nil
		},
	}
	cmd.Flags().BoolVar(&replace, "replace", false, "replace an installed theme of the same name")
	return cmd
}

func newThemeRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "remove <name>",
		Short: "Delete an installed theme that is not in use",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			p, cfg, err := openWithConfig()
			if err != nil {
				return err
			}
			name := args[0]
			one, ok := installedTheme(p.Root, name)
			switch {
			case !ok:
				return fmt.Errorf("theme %s is not installed", name)
			case one.Builtin:
				return errors.New("the theme built into Kite cannot be removed")
			case inUse(cfg, one):
				return fmt.Errorf("theme %s is in use; run 'kite theme use default' or another first", name)
			}
			if _, err := p.Writer().Apply(cmd.Context(), content.ChangeSet{
				Ops:     []content.Op{content.DeleteTheme{Name: name}},
				Message: "theme: remove " + name,
			}); err != nil {
				return err
			}
			printf(cmd, "removed %s; the settings under theme.settings stay in %s\n",
				filepath.Join(site.ThemesDir, name), config.Name)
			return nil
		},
	}
}

func newThemeUseCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "use <name>",
		Short: "Build the site with an installed theme",
		Long: "Sets theme.name in kite.yaml. Settings stay under theme.settings, which\n" +
			"every theme shares, so switching back finds them where they were.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			p, cfg, err := openWithConfig()
			if err != nil {
				return err
			}
			name := args[0]
			one, ok := installedTheme(p.Root, name)
			switch {
			case !ok:
				return fmt.Errorf("theme %s is not installed; 'kite theme list' shows the ones that are", name)
			case one.Theme == nil:
				return fmt.Errorf("theme %s cannot be used: %s", name, one.Problem)
			case inUse(cfg, one):
				printf(cmd, "%s is in use already\n", name)
				return nil
			}
			if _, err := p.Writer().Apply(cmd.Context(), content.ChangeSet{
				Ops:     []content.Op{content.PutSettings{Values: map[string]any{"theme.name": name}}},
				Message: "theme: use " + name,
			}); err != nil {
				return err
			}
			printf(cmd, "the site is built with %s now\n", name)
			return nil
		},
	}
}

// installedTheme finds a theme by name the way theme.name does: the built-in
// one first.
func installedTheme(root, name string) (site.Installed, bool) {
	for _, one := range site.Themes(root) {
		if one.Name == name {
			return one, true
		}
	}
	return site.Installed{}, false
}

func newThemeVerifyCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "verify [dir]",
		Short: "Prove a theme draws the same pages built and served",
		Long: "Builds a small site that uses every kind of page with the theme, asks\n" +
			"a server for the same pages, and compares every byte. A theme that\n" +
			"passes publishes exactly what `kite run` previewed.\n\n" +
			"With no directory it checks the theme of the project it is run in, or\n" +
			"the theme built into Kite outside of one.",
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			th, where, err := themeToVerify(args)
			if err != nil {
				return err
			}
			report, err := themecheck.Check(cmd.Context(), th)
			if err != nil {
				return fmt.Errorf("theme: %s: %w", where, err)
			}

			if jsonOut(cmd) {
				if err := writeJSON(cmd.OutOrStdout(), report); err != nil {
					return err
				}
			} else if report.OK() {
				printf(cmd, "%s (%s): %d files identical built and served\n", report.Theme, where, report.Compared)
				if len(report.Loads) > 0 {
					printf(cmd, "  pages load from %s\n", strings.Join(report.Loads, ", "))
				}
			} else {
				if len(report.Differ) > 0 {
					printf(cmd, "%s (%s): %d of %d files differ between build and serve\n",
						report.Theme, where, len(report.Differ), report.Compared)
					for _, d := range report.Differ {
						printf(cmd, "  %s\n    %s\n", d.URL, d.Detail)
					}
				}
				if len(report.Outside) > 0 {
					printf(cmd, "%s (%s): %d link(s) leave a site published under a path; use url.For or url.Rel\n",
						report.Theme, where, len(report.Outside))
					for _, d := range report.Outside {
						printf(cmd, "  %s\n    %s\n", d.URL, d.Detail)
					}
				}
			}
			switch {
			case len(report.Differ) > 0:
				return fmt.Errorf("theme: %s does not draw the same pages in both runtimes", report.Theme)
			case !report.OK():
				return fmt.Errorf("theme: %s links outside the site", report.Theme)
			}
			return nil
		},
	}
}

// themeToVerify finds the theme a verify was asked about, and says where it
// came from.
func themeToVerify(args []string) (fs.FS, string, error) {
	if len(args) == 1 {
		dir := args[0]
		if _, err := os.Stat(filepath.Join(dir, theme.ManifestName)); err != nil {
			return nil, "", fmt.Errorf("theme: %s has no %s, so it is not a theme", dir, theme.ManifestName)
		}
		return os.DirFS(dir), dir, nil
	}

	wd, err := os.Getwd()
	if err != nil {
		return nil, "", err
	}
	root, err := project.FindRoot(wd)
	if errors.Is(err, project.ErrNotFound) {
		return themes.Default(), "built in", nil
	}
	if err != nil {
		return nil, "", err
	}
	cfg, err := config.Load(root)
	if err != nil {
		return nil, "", err
	}
	name := cfg.Theme.Name
	if name == "" || name == "default" {
		return themes.Default(), "built in", nil
	}
	dir := filepath.Join(root, "themes", name)
	return os.DirFS(dir), filepath.Join("themes", name), nil
}

func newThemeNewCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "new <name>",
		Short: "Start a theme in a new directory",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			switch {
			case name == site.BuiltinTheme:
				return fmt.Errorf("the name %s belongs to the theme built into Kite", name)
			case !content.ValidThemeName(name):
				return fmt.Errorf("%q cannot name a theme: use letters, digits, dots, - and _", name)
			}
			if _, err := os.Stat(name); err == nil {
				return fmt.Errorf("%s already exists", name)
			}
			files := starterTheme(name)
			for _, file := range slices.Sorted(maps.Keys(files)) {
				path := filepath.Join(name, filepath.FromSlash(file))
				if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
					return err
				}
				if err := os.WriteFile(path, []byte(files[file]), 0o644); err != nil {
					return err
				}
			}
			printf(cmd, "created %s\n\n", name)
			printf(cmd, "Next:\n  edit %s and the templates in %s\n  kite theme verify %s\n  kite theme add %s   (in a site)\n",
				filepath.Join(name, theme.ManifestName), filepath.Join(name, theme.LayoutsDir), name, name)
			return nil
		},
	}
}

// starterTheme is the theme a new one starts from: every kind of page, the
// site's main menu, its words in a language pack, and light and dark by the
// data-theme convention.
func starterTheme(name string) map[string]string {
	return map[string]string{
		theme.ManifestName: fmt.Sprintf(`name: %s
title: %s
version: 0.1.0
apiVersion: %s
description: What this theme is for.

menus:
  - name: main
    label: Header
    description: The links across the top of every page.

settings:
  - key: accent
    type: color
    label: Accent color
    default: "#2f6fdb"
`, name, name, theme.APIVersion),

		"layouts/baseof.html": `<!DOCTYPE html>
<html lang="{{ .Site.Language }}">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>{{ with .Page }}{{ with .Title }}{{ . }} · {{ end }}{{ end }}{{ .Site.Title }}</title>
  {{- with .Site.Description }}
  <meta name="description" content="{{ . }}">
  {{- end }}
  <link rel="stylesheet" href="{{ url.Rel "style.css" }}">
  <link rel="alternate" type="application/rss+xml" title="{{ .Site.Title }}" href="{{ url.Rel "rss.xml" }}">
  <style>:root { --accent: {{ default "#2f6fdb" .Site.ThemeSettings.accent }}; }</style>
  {{ .Site.HeadHTML }}
</head>
<body>
  <header>
    <a class="site" href="{{ url.For "home" }}">{{ .Site.Title }}</a>
    <nav>
      {{- $here := "" }}{{ with .Page }}{{ $here = .RelPermalink }}{{ end }}
      {{- range $item := .Site.Menus.main }}{{ with $item.URL }}
      <a href="{{ . }}"{{ if eq . $here }} aria-current="page"{{ end }}>{{ $item.Name }}</a>
      {{- end }}{{ else }}
      <a href="{{ url.For "list" "post" }}">{{ T "posts" }}</a>
      {{- end }}
    </nav>
  </header>
  <main>{{ block "main" . }}{{ end }}</main>
  <footer>{{ .Site.Title }}</footer>
  {{ .Site.FooterHTML }}
</body>
</html>
`,

		// The home page, each list and each term's page fall back to this.
		"layouts/list.html": `{{ define "main" }}
  {{- with .Page }}{{ with .Title }}<h1>{{ . }}</h1>{{ end }}{{ end }}
  <ul class="posts">
    {{- range .Pages }}
    <li>
      <a href="{{ .RelPermalink }}">{{ .Title }}</a>
      {{- if not (time.IsZero .PublishDate) }}
      <time datetime="{{ time.Format "2006-01-02" .PublishDate }}">{{ time.Format "2006-01-02" .PublishDate }}</time>
      {{- end }}
    </li>
    {{- else }}
    <li>{{ T "empty" }}</li>
    {{- end }}
  </ul>
  {{- with .Paginator }}{{ if gt .TotalPages 1 }}
  <nav class="pages">
    {{ if .HasPrev }}<a rel="prev" href="{{ .PrevURL }}">{{ T "newer" }}</a>{{ end }}
    <span>{{ .PageNumber }} / {{ .TotalPages }}</span>
    {{ if .HasNext }}<a rel="next" href="{{ .NextURL }}">{{ T "older" }}</a>{{ end }}
  </nav>
  {{- end }}{{ end }}
{{ end }}
`,

		"layouts/single.html": `{{ define "main" }}
  {{- with .Page }}
  <article>
    <h1>{{ .Title }}</h1>
    {{- if not (time.IsZero .PublishDate) }}
    <p class="meta"><time datetime="{{ time.Format "2006-01-02" .PublishDate }}">{{ time.Format "January 2, 2006" .PublishDate }}</time></p>
    {{- end }}
    {{ .Content }}
    {{- with .Terms "tags" }}
    <p class="meta">{{ range . }}<a href="{{ .RelPermalink }}">#{{ .Name }}</a> {{ end }}</p>
    {{- end }}
  </article>
  {{- end }}
{{ end }}
`,

		"layouts/taxonomy.html": `{{ define "main" }}
  {{- with .Page }}{{ with .Title }}<h1>{{ . }}</h1>{{ end }}{{ end }}
  <ul class="posts">
    {{- range .Terms }}
    <li><a href="{{ .RelPermalink }}">{{ .Name }}</a> <span>{{ .Count }}</span></li>
    {{- end }}
  </ul>
{{ end }}
`,

		"layouts/404.html": `{{ define "main" }}
  <h1>{{ T "not_found" }}</h1>
  <p><a href="{{ url.For "home" }}">{{ T "home" }}</a></p>
{{ end }}
`,

		"i18n/en.yaml": `# The words the theme's pages say, read with T. A pack of another
# language, as zh-CN.yaml, says them in that language.
posts: Posts
empty: Nothing here yet.
newer: Newer
older: Older
not_found: Page not found
home: Back to the home page
`,

		"static/style.css": `:root {
  color-scheme: light dark;
  --ink: #1d1d1f;
  --paper: #ffffff;
  --muted: #6e6e73;
}

/* The page follows the reader's system, and a page that sets data-theme on
   <html> to light or dark chooses for itself. */
@media (prefers-color-scheme: dark) {
  :root:not([data-theme="light"]) { --ink: #f5f5f7; --paper: #161617; --muted: #a1a1a6; }
}
:root[data-theme="dark"] { --ink: #f5f5f7; --paper: #161617; --muted: #a1a1a6; }

body {
  margin: 0 auto;
  max-width: 42rem;
  padding: 2rem 1rem;
  font: 17px/1.7 system-ui, sans-serif;
  color: var(--ink);
  background: var(--paper);
}
a { color: var(--accent); }
header { display: flex; flex-wrap: wrap; gap: 1.5rem; align-items: baseline; margin-bottom: 3rem; }
header .site { color: inherit; font-weight: 700; text-decoration: none; }
nav { display: flex; flex-wrap: wrap; gap: 1rem; }
nav a[aria-current] { font-weight: 600; text-decoration: none; }
.posts { list-style: none; padding: 0; }
.posts li { display: flex; justify-content: space-between; gap: 1rem; padding: .3rem 0; }
time, .meta, .posts span, footer { color: var(--muted); font-size: .9em; }
img { max-width: 100%; height: auto; }
footer { margin-top: 4rem; }
`,
	}
}
