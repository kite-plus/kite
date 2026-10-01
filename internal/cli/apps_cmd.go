package cli

import (
	"bufio"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/spf13/cobra"

	"github.com/kite-plus/kite/internal/apps"
	"github.com/kite-plus/kite/internal/archive"
	"github.com/kite-plus/kite/internal/buildinfo"
	"github.com/kite-plus/kite/internal/config"
	"github.com/kite-plus/kite/internal/content"
	"github.com/kite-plus/kite/internal/lock"
	"github.com/kite-plus/kite/internal/plugin"
	"github.com/kite-plus/kite/internal/project"
	"github.com/kite-plus/kite/internal/render/theme"
	"github.com/kite-plus/kite/internal/site"
)

func newAppsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "apps",
		Short: "Find themes and plugins in the index, and keep them up to date",
		Long: "The index lists the themes and plugins Kite installs by name, and what\n" +
			"each version needs and loads. 'kite theme add <name>' and 'kite plugin\n" +
			"add <name>' install from it, and kite.lock records what came from where;\n" +
			"these commands look through it and update what a site installed.",
		Args: cobra.NoArgs,
	}
	cmd.AddCommand(newAppsSearchCmd(), newAppsOutdatedCmd(), newAppsUpdateCmd())
	return cmd
}

// indexName is how a package is asked for by name: an id from the index,
// perhaps with the version wanted, as vane or vane@1.0.1.
var indexName = regexp.MustCompile(`^([a-z0-9]+(?:-[a-z0-9]+)*)(?:@(v?[0-9][0-9A-Za-z.+-]*))?$`)

// byName reads an argument to add as a package in the index, with the
// version wanted, when it is not a file or a directory and reads as a name.
func byName(arg string) (id, version string, ok bool) {
	if _, err := os.Stat(arg); !errors.Is(err, fs.ErrNotExist) {
		return "", "", false
	}
	m := indexName.FindStringSubmatch(arg)
	if m == nil {
		return "", "", false
	}
	return m[1], m[2], true
}

// appsClient reads the index a site installs from: the one apps.index or
// KITE_APPS_URL names, or Kite's own. What it fetches is kept in the site's
// .kite/cache/apps, or outside a site in the user's cache.
func appsClient(root string, cfg *config.Config) *apps.Client {
	urls := apps.Indexes
	switch {
	case cfg != nil && cfg.Apps.Index != "":
		urls = []string{cfg.Apps.Index}
	case os.Getenv("KITE_APPS_URL") != "":
		urls = []string{os.Getenv("KITE_APPS_URL")}
	}
	cache := ""
	if root != "" {
		cache = filepath.Join(root, ".kite", "cache", "apps")
	} else if dir, err := os.UserCacheDir(); err == nil {
		cache = filepath.Join(dir, "kite", "apps")
	}
	return apps.NewClient(urls, cache)
}

// readIndex reads the index, and says so when it is the copy kept from an
// earlier fetch because none of its addresses answered.
func readIndex(cmd *cobra.Command, c *apps.Client, refresh bool) (*apps.Index, error) {
	ix, _, err := readIndexAt(cmd, c, refresh)
	return ix, err
}

// readIndexAt is readIndex, with when the index was fetched.
func readIndexAt(cmd *cobra.Command, c *apps.Client, refresh bool) (*apps.Index, time.Time, error) {
	ix, got, err := c.Index(cmd.Context(), refresh)
	if err != nil {
		return nil, time.Time{}, err
	}
	if got.Offline {
		why, _, _ := strings.Cut(got.Err.Error(), "\n")
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "the index could not be fetched (%s);\nusing the copy fetched %s ago\n",
			why, ago(c.Now().Sub(got.At)))
	}
	return ix, got.At, nil
}

func ago(d time.Duration) string {
	switch {
	case d < time.Minute:
		return "less than a minute"
	case d < time.Hour:
		return plural(int(d/time.Minute), "minute")
	case d < 48*time.Hour:
		return plural(int(d/time.Hour), "hour")
	}
	return plural(int(d/(24*time.Hour)), "day")
}

func plural(n int, unit string) string {
	if n == 1 {
		return "1 " + unit
	}
	return fmt.Sprintf("%d %ss", n, unit)
}

// fetched is a release from the index, downloaded and checked the way an
// install checks what it installs.
type fetched struct {
	app    *apps.App
	rel    *apps.Release
	files  map[string][]byte
	theme  *theme.Theme
	plugin *plugin.Plugin
	origin *content.Origin
}

// fetchByName finds a theme or a plugin, kind, in the index, and fetches the
// version wanted, or else the newest that works with this Kite.
func fetchByName(cmd *cobra.Command, root string, cfg *config.Config, kind, id, want string) (*fetched, error) {
	client := appsClient(root, cfg)
	asked := client.Now()
	ix, fetchedAt, err := readIndexAt(cmd, client, false)
	if err != nil {
		return nil, err
	}
	app := ix.Find(kind, id)
	// A package or version released since the copy kept was fetched is
	// worth asking the index for again.
	if (app == nil || (want != "" && app.Release(want) == nil)) && fetchedAt.Before(asked) {
		if ix, err = readIndex(cmd, client, true); err != nil {
			return nil, err
		}
		app = ix.Find(kind, id)
	}
	if err := ix.Usable(); err != nil {
		return nil, err
	}
	if app == nil {
		return nil, fmt.Errorf("the index has no %s named %s; 'kite apps search %s' looks for one", kind, id, id)
	}
	rel, err := app.Pick(want, buildinfo.Version)
	if err != nil {
		return nil, err
	}
	return fetchRelease(cmd, client, app, rel)
}

// fetchRelease downloads a release and checks that it is the package the
// index says it is.
func fetchRelease(cmd *cobra.Command, client *apps.Client, app *apps.App, rel *apps.Release) (*fetched, error) {
	data, err := client.Archive(cmd.Context(), rel)
	if err != nil {
		return nil, err
	}
	kind := themePackage
	if app.Kind == "plugin" {
		kind = pluginPackage
	}
	files, problem := archive.Unpack(data, kind.name, kind.manifest, kind.maxSize, kind.maxFiles)
	if problem != "" {
		return nil, fmt.Errorf("%s %s %s: %s", app.Kind, app.ID, rel.Version, problem)
	}
	got := &fetched{app: app, rel: rel, files: files, origin: &content.Origin{
		Version:  rel.Version,
		Source:   client.Source(),
		Resolved: rel.Archive.URLs[0],
		Checksum: lock.Checksum(rel.Archive.SHA256),
	}}
	var name, version string
	if app.Kind == "plugin" {
		if got.plugin, err = checkPlugin(files); err != nil {
			return nil, err
		}
		name, version = got.plugin.Manifest.ID, got.plugin.Manifest.Version
		g := grantOf(got.plugin)
		got.origin.Granted = &g
	} else {
		if got.theme, err = checkTheme(files); err != nil {
			return nil, err
		}
		name, version = got.theme.Manifest.Name, got.theme.Manifest.Version
	}
	if name != app.ID || version != rel.Version {
		return nil, fmt.Errorf("the archive of %s %s %s holds %s %s instead", app.Kind, app.ID, rel.Version, name, version)
	}
	return got, nil
}

// grantOf is what a plugin does to a site, which its owner is told before
// it runs.
func grantOf(pl *plugin.Plugin) content.Grant {
	return content.Grant{
		Inject: len(pl.Manifest.Inject),
		Loads:  nonNil(pl.Hosts()),
		Hooks:  nonNil(slices.Clone(pl.Manifest.Hooks)),
	}
}

func describeGrant(g content.Grant) string {
	loads := "its code loads nothing from other sites"
	if len(g.Loads) > 0 {
		loads = "its code loads from " + strings.Join(g.Loads, ", ")
	}
	hooks := "it runs nothing while the site is built"
	if len(g.Hooks) > 0 {
		hooks = "it runs " + strings.Join(g.Hooks, ", ") + " while the site is built"
	}
	return fmt.Sprintf("it injects %d piece(s) of code; %s; %s", g.Inject, loads, hooks)
}

// installed is a theme or a plugin a site has in themes/ or plugins/.
type installed struct {
	kind, name, version, homepage string
	// dir is where it is, relative to the project root.
	dir string
}

func installedPackages(root string) ([]installed, error) {
	var out []installed
	for _, one := range site.Themes(root) {
		if one.Builtin {
			continue
		}
		pkg := installed{kind: "theme", name: one.Name, dir: filepath.Join(site.ThemesDir, one.Name)}
		if m := one.Manifest; m != nil {
			pkg.version, pkg.homepage = m.Version, m.Homepage
		}
		out = append(out, pkg)
	}
	ids, err := plugin.Installed(root)
	if err != nil {
		return nil, err
	}
	for _, id := range ids {
		pkg := installed{kind: "plugin", name: id, dir: filepath.Join(plugin.Dir, id)}
		if m, err := plugin.ReadManifest(os.DirFS(filepath.Join(root, pkg.dir))); err == nil {
			pkg.version, pkg.homepage = m.Version, m.Homepage
		}
		out = append(out, pkg)
	}
	return out, nil
}

// tracked is an installed package the index is where updates come from:
// kite.lock records it as installed from the index, or it was installed
// some other way and is the project the index lists under its name.
type tracked struct {
	installed
	app   *apps.App
	entry *lock.Entry
}

func trackedPackages(root string, ix *apps.Index, lf *lock.File, source string) ([]tracked, error) {
	pkgs, err := installedPackages(root)
	if err != nil {
		return nil, err
	}
	var out []tracked
	for _, pkg := range pkgs {
		app := ix.Find(pkg.kind, pkg.name)
		if app == nil {
			continue
		}
		t := tracked{installed: pkg, app: app}
		if e, ok := lf.Get(pkg.kind, pkg.name); ok {
			if e.Source != source {
				continue
			}
			t.entry = &e
		} else if !app.ProjectOf(pkg.homepage) {
			continue
		}
		out = append(out, t)
	}
	return out, nil
}

// optionalProject is the project a command runs in, or none outside of one.
func optionalProject() (*project.Project, *config.Config, error) {
	wd, err := os.Getwd()
	if err != nil {
		return nil, nil, err
	}
	p, err := project.Open(wd)
	if errors.Is(err, project.ErrNotFound) {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, err
	}
	cfg, err := config.Load(p.Root)
	if err != nil {
		return nil, nil, err
	}
	return p, cfg, nil
}

type appRow struct {
	Kind        string `json:"kind"`
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Official    bool   `json:"official"`
	Version     string `json:"version,omitempty"`
	Installed   string `json:"installed,omitempty"`
	Problem     string `json:"problem,omitempty"`
}

func newAppsSearchCmd() *cobra.Command {
	var refresh bool
	cmd := &cobra.Command{
		Use:   "search [words...]",
		Short: "Look through the themes and plugins in the index",
		Long: "Lists the themes and plugins whose id, title, description or tags hold\n" +
			"every word given, or all of them, with the newest version that works\n" +
			"with this Kite.",
		RunE: func(cmd *cobra.Command, args []string) error {
			p, cfg, err := optionalProject()
			if err != nil {
				return err
			}
			root, lang := "", "en"
			if p != nil {
				root, lang = p.Root, cfg.Site.Language
			}
			client := appsClient(root, cfg)
			ix, err := readIndex(cmd, client, refresh)
			if err != nil {
				return err
			}
			// Only a package the index is where it came from counts as
			// installed, not another of the same name.
			have := map[string]string{}
			if root != "" {
				lf, err := lock.Read(root)
				if err != nil {
					return err
				}
				pkgs, err := trackedPackages(root, ix, lf, client.Source())
				if err != nil {
					return err
				}
				for _, t := range pkgs {
					have[t.kind+"/"+t.name] = t.version
				}
			}
			rows := []appRow{}
			for _, a := range ix.Apps {
				if a.Delisted != "" || !a.Matches(args) {
					continue
				}
				row := appRow{
					Kind:        a.Kind,
					ID:          a.ID,
					Title:       apps.Text(a.Title, lang),
					Description: apps.Text(a.Description, lang),
					Official:    a.Official,
					Installed:   have[a.Kind+"/"+a.ID],
				}
				if rel, err := a.Pick("", buildinfo.Version); err == nil {
					row.Version = rel.Version
				} else {
					row.Problem = err.Error()
				}
				rows = append(rows, row)
			}
			if jsonOut(cmd) {
				return writeJSON(cmd.OutOrStdout(), rows)
			}
			if err := ix.Usable(); err != nil {
				printf(cmd, "%s\n\n", err)
			}
			if len(rows) == 0 {
				printf(cmd, "nothing in the index matches\n")
				return nil
			}
			for _, row := range rows {
				notes := []string{}
				if !row.Official {
					notes = append(notes, "community")
				}
				if row.Installed != "" {
					notes = append(notes, "installed "+row.Installed)
				}
				line := fmt.Sprintf("%-7s %-16s %-8s %s", row.Kind, row.ID, row.Version, row.Title)
				if len(notes) > 0 {
					line += "  (" + strings.Join(notes, ", ") + ")"
				}
				printf(cmd, "%s\n", line)
				if row.Description != "" {
					printf(cmd, "        %s\n", shorten(row.Description, 96))
				}
				if row.Problem != "" {
					printf(cmd, "        %s\n", row.Problem)
				}
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&refresh, "refresh", false, "fetch the index even if the copy kept is less than an hour old")
	return cmd
}

// shorten cuts text to at most n characters, marking the cut.
func shorten(text string, n int) string {
	if utf8.RuneCountInString(text) <= n {
		return text
	}
	return string([]rune(text)[:n-1]) + "…"
}

type outdatedRow struct {
	Kind      string `json:"kind"`
	Name      string `json:"name"`
	Installed string `json:"installed"`
	Latest    string `json:"latest,omitempty"`
	// Locked says kite.lock records the package as installed from the index.
	Locked   bool   `json:"locked"`
	Yanked   bool   `json:"yanked,omitempty"`
	Delisted string `json:"delisted,omitempty"`
	Problem  string `json:"problem,omitempty"`
}

// outdated compares what a site has with the index: the rows of the
// packages with a newer version, a yanked one or none listed any more.
func outdated(pkgs []tracked) []outdatedRow {
	rows := []outdatedRow{}
	for _, t := range pkgs {
		row := outdatedRow{Kind: t.kind, Name: t.name, Installed: t.version, Locked: t.entry != nil, Delisted: t.app.Delisted}
		if rel := t.app.Release(t.version); rel != nil {
			row.Yanked = rel.Yanked
		}
		if t.app.Delisted == "" {
			rel, err := t.app.Pick("", buildinfo.Version)
			switch {
			case err != nil:
				row.Problem = err.Error()
			case theme.CompareVersions(rel.Version, t.version) > 0:
				row.Latest = rel.Version
			}
		}
		if row.Latest != "" || row.Yanked || row.Delisted != "" {
			rows = append(rows, row)
		}
	}
	return rows
}

func newAppsOutdatedCmd() *cobra.Command {
	var refresh bool
	cmd := &cobra.Command{
		Use:   "outdated",
		Short: "List the themes and plugins with a newer version in the index",
		Long: "Compares the themes and plugins a site installed from the index with\n" +
			"what it lists now: a newer version that works with this Kite, a version\n" +
			"that was yanked, a package no longer listed. One installed from an\n" +
			"archive counts when its homepage is the repository the index names.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			p, cfg, err := openWithConfig()
			if err != nil {
				return err
			}
			client := appsClient(p.Root, cfg)
			ix, err := readIndex(cmd, client, refresh)
			if err != nil {
				return err
			}
			if err := ix.Usable(); err != nil {
				return err
			}
			lf, err := lock.Read(p.Root)
			if err != nil {
				return err
			}
			pkgs, err := trackedPackages(p.Root, ix, lf, client.Source())
			if err != nil {
				return err
			}
			rows := outdated(pkgs)
			if jsonOut(cmd) {
				return writeJSON(cmd.OutOrStdout(), rows)
			}
			if len(rows) == 0 {
				printf(cmd, "everything installed from the index is up to date\n")
				return nil
			}
			for _, row := range rows {
				line := fmt.Sprintf("%-7s %-16s %-8s", row.Kind, row.Name, row.Installed)
				if row.Latest != "" {
					line += " -> " + row.Latest
				}
				var notes []string
				if row.Yanked {
					notes = append(notes, "this version is yanked")
				}
				if row.Delisted != "" {
					notes = append(notes, "no longer listed: "+row.Delisted)
				}
				if !row.Locked {
					notes = append(notes, "installed from an archive")
				}
				if len(notes) > 0 {
					line += "  (" + strings.Join(notes, "; ") + ")"
				}
				printf(cmd, "%s\n", strings.TrimRight(line, " "))
				if row.Problem != "" {
					printf(cmd, "        %s\n", row.Problem)
				}
			}
			printf(cmd, "\nrun 'kite apps update' to update them\n")
			return nil
		},
	}
	cmd.Flags().BoolVar(&refresh, "refresh", false, "fetch the index even if the copy kept is less than an hour old")
	return cmd
}

func newAppsUpdateCmd() *cobra.Command {
	var force, yes, refresh bool
	cmd := &cobra.Command{
		Use:   "update [name[@version]...]",
		Short: "Update themes and plugins installed from the index",
		Long: "Updates each theme or plugin named, or every one 'kite apps outdated'\n" +
			"lists, to the newest version that works with this Kite; name@version\n" +
			"installs that version instead. theme/<name> or plugin/<name> tells a\n" +
			"theme from a plugin of the same name.\n\n" +
			"A package whose files changed since it was installed is left alone,\n" +
			"since the update would replace the changes, unless --force. A plugin\n" +
			"whose new version does more than its owner agreed to, loads from\n" +
			"another site, runs another hook or injects more code, is updated once\n" +
			"that is agreed to, or with --yes.",
		RunE: func(cmd *cobra.Command, args []string) error {
			p, cfg, err := openWithConfig()
			if err != nil {
				return err
			}
			client := appsClient(p.Root, cfg)
			ix, err := readIndex(cmd, client, refresh)
			if err != nil {
				return err
			}
			if err := ix.Usable(); err != nil {
				return err
			}
			lf, err := lock.Read(p.Root)
			if err != nil {
				return err
			}
			pkgs, err := trackedPackages(p.Root, ix, lf, client.Source())
			if err != nil {
				return err
			}
			targets, err := updateTargets(pkgs, args, p.Root, client.Source())
			if err != nil {
				return err
			}
			if len(targets) == 0 {
				printf(cmd, "everything installed from the index is up to date\n")
				return nil
			}
			u := updater{cmd: cmd, p: p, client: client, force: force, yes: yes, answers: bufio.NewReader(cmd.InOrStdin())}
			failed := 0
			for _, t := range targets {
				if err := u.update(t); err != nil {
					failed++
					_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "%s %s: %v\n", t.kind, t.name, err)
				}
			}
			if failed > 0 {
				return fmt.Errorf("%d of %d update(s) failed", failed, len(targets))
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "update a package whose files changed since it was installed")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "agree to what a plugin's new version does without being asked")
	cmd.Flags().BoolVar(&refresh, "refresh", false, "fetch the index even if the copy kept is less than an hour old")
	return cmd
}

// target is a package to update, and the version wanted, or "" for the
// newest that works with this Kite.
type target struct {
	tracked
	want string
}

func updateTargets(pkgs []tracked, args []string, root, source string) ([]target, error) {
	var out []target
	if len(args) == 0 {
		for _, row := range outdated(pkgs) {
			if row.Latest == "" {
				continue
			}
			i := slices.IndexFunc(pkgs, func(t tracked) bool { return t.kind == row.Kind && t.name == row.Name })
			out = append(out, target{tracked: pkgs[i]})
		}
		return out, nil
	}
	for _, arg := range args {
		kind, spec, scoped := strings.Cut(arg, "/")
		if !scoped {
			kind, spec = "", arg
		}
		name, want, _ := strings.Cut(spec, "@")
		var found []tracked
		for _, t := range pkgs {
			if t.name == name && (kind == "" || t.kind == kind) {
				found = append(found, t)
			}
		}
		switch {
		case len(found) == 1:
			out = append(out, target{tracked: found[0], want: want})
		case len(found) > 1:
			return nil, fmt.Errorf("a theme and a plugin are both named %s; say theme/%s or plugin/%s", name, name, name)
		default:
			return nil, notTracked(root, source, kind, name)
		}
	}
	return out, nil
}

// notTracked says why a package named for an update has nothing to update
// from in the index in use, source.
func notTracked(root, source, kind, name string) error {
	pkgs, err := installedPackages(root)
	if err != nil {
		return err
	}
	i := slices.IndexFunc(pkgs, func(pkg installed) bool { return pkg.name == name && (kind == "" || pkg.kind == kind) })
	if i < 0 {
		return fmt.Errorf("%s is not installed", name)
	}
	if lf, err := lock.Read(root); err == nil {
		if e, ok := lf.Get(pkgs[i].kind, name); ok && e.Source != source {
			return fmt.Errorf("%s was installed from %s, and the index in use is %s", name, e.Source, source)
		}
	}
	return fmt.Errorf("%s was not installed from the index, and the index lists no package of its homepage under that name; "+
		"'kite %s add %s --replace' installs it from the index", name, pkgs[i].kind, name)
}

type updater struct {
	cmd     *cobra.Command
	p       *project.Project
	client  *apps.Client
	force   bool
	yes     bool
	answers *bufio.Reader
}

func (u *updater) update(t target) error {
	rel, err := t.app.Pick(t.want, buildinfo.Version)
	if err != nil {
		return err
	}
	if rel.Version == t.version && t.entry != nil {
		printf(u.cmd, "%s %s is at %s already\n", t.kind, t.name, t.version)
		return nil
	}
	got, err := fetchRelease(u.cmd, u.client, t.app, rel)
	if err != nil {
		return err
	}
	if !u.force {
		why, err := u.changed(t)
		if err != nil {
			return err
		}
		if why != "" {
			return fmt.Errorf("%s; updating would replace that, so it is left at %s. Run with --force to update it anyway",
				why, t.version)
		}
	}
	if got.plugin != nil {
		ok, err := u.agree(t, got)
		if err != nil || !ok {
			return err
		}
	}
	var op content.Op = content.PutTheme{Name: t.name, Files: got.files, Replace: true, Origin: got.origin}
	if t.kind == "plugin" {
		if err := u.checkEnabled(t, got); err != nil {
			return err
		}
		op = content.PutPlugin{ID: t.name, Files: got.files, Replace: true, Origin: got.origin}
	}
	if _, err := u.p.Writer().Apply(u.cmd.Context(), content.ChangeSet{
		Ops:     []content.Op{op},
		Message: fmt.Sprintf("%s: update %s %s to %s", t.kind, t.name, t.version, rel.Version),
	}); err != nil {
		return err
	}
	if rel.Version == t.version {
		printf(u.cmd, "%s %s %s is recorded as installed from the index now\n", t.kind, t.name, t.version)
	} else {
		printf(u.cmd, "updated %s %s from %s to %s\n", t.kind, t.name, t.version, rel.Version)
	}
	return nil
}

// changed says how an installed package's files are not what was installed,
// or "" when they are: for one installed from the index, the files kite.lock
// recorded; for one installed from an archive, the same version's archive in
// the index. A package Kite has nothing to compare with counts as changed,
// since nothing says it is not.
func (u *updater) changed(t target) (string, error) {
	dir := filepath.Join(u.p.Root, t.dir)
	tree, err := lock.TreeOf(dir)
	if err != nil {
		return "", err
	}
	if t.entry != nil {
		if tree == t.entry.Tree {
			return "", nil
		}
		return t.dir + " has changed since " + t.version + " was installed", nil
	}
	rel := t.app.Release(t.version)
	if rel == nil {
		return t.dir + " was installed from an archive of a version the index does not list, " + t.version, nil
	}
	same, err := fetchRelease(u.cmd, u.client, t.app, rel)
	if err != nil {
		return "", err
	}
	if lock.Tree(same.files) == tree {
		return "", nil
	}
	return t.dir + " differs from " + t.version + " in the index", nil
}

// agree asks for what a plugin's new version does beyond what its owner
// agreed to, and reports whether it was agreed to.
func (u *updater) agree(t target, got *fetched) (bool, error) {
	var granted content.Grant
	if t.entry != nil && t.entry.Granted != nil {
		g := t.entry.Granted
		granted = content.Grant{Inject: g.Inject, Loads: g.Loads, Hooks: g.Hooks}
	} else if pl, err := plugin.Open(u.p.Root, t.name); err == nil {
		granted = grantOf(pl)
	}
	g := *got.origin.Granted
	more := lock.Grant{Inject: g.Inject, Loads: g.Loads, Hooks: g.Hooks}.Exceeds(
		lock.Grant{Inject: granted.Inject, Loads: granted.Loads, Hooks: granted.Hooks})
	if len(more) == 0 || u.yes {
		return true, nil
	}
	ask := fmt.Sprintf("plugin %s %s %s", t.name, got.rel.Version, strings.Join(more, ", "))
	if !interactive(u.cmd, false) {
		return false, fmt.Errorf("%s; run with --yes to agree to that", ask)
	}
	printf(u.cmd, "%s.\n", ask)
	ok, err := askYesNo(u.cmd, u.answers, "Update it?", false)
	if err == nil && !ok {
		printf(u.cmd, "left %s at %s\n", t.name, t.version)
	}
	return ok, err
}

// checkEnabled checks a new version of an enabled plugin the way enabling
// it does, so that an update never leaves the next build to fail on it.
func (u *updater) checkEnabled(t target, got *fetched) error {
	cfg, err := config.Load(u.p.Root)
	if err != nil {
		return err
	}
	if !slices.Contains(cfg.Plugins.Enabled, t.name) {
		return nil
	}
	return got.plugin.Check(u.cmd.Context(), filepath.Join(u.p.Root, filepath.FromSlash(plugin.CacheDir)))
}

// lockProblems compares kite.lock with what is installed, for kite doctor.
func lockProblems(root string) []string {
	lf, err := lock.Read(root)
	if err != nil {
		return []string{err.Error()}
	}
	var out []string
	for _, kind := range []string{"theme", "plugin"} {
		section, dir := lf.Themes, site.ThemesDir
		if kind == "plugin" {
			section, dir = lf.Plugins, plugin.Dir
		}
		for _, name := range slices.Sorted(maps.Keys(section)) {
			e := section[name]
			at := filepath.Join(dir, name)
			if _, err := os.Stat(filepath.Join(root, at)); errors.Is(err, fs.ErrNotExist) {
				out = append(out, fmt.Sprintf("%s records %s %s, which is not installed", lock.Name, kind, name))
				continue
			}
			tree, err := lock.TreeOf(filepath.Join(root, at))
			if err != nil {
				out = append(out, fmt.Sprintf("%s: %v", at, err))
				continue
			}
			if tree != e.Tree {
				out = append(out, fmt.Sprintf("%s has changed since %s %s was installed from the index; "+
					"'kite apps update' leaves it alone unless --force", at, name, e.Version))
			}
		}
	}
	return out
}
