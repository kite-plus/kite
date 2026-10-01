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
	"github.com/kite-plus/kite/internal/buildinfo"
	"github.com/kite-plus/kite/internal/config"
	"github.com/kite-plus/kite/internal/content"
	"github.com/kite-plus/kite/internal/lock"
	"github.com/kite-plus/kite/internal/plugin"
	"github.com/kite-plus/kite/internal/project"
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

// appsClient is the client of the index a site, root, installs from, or for
// no site, root "", the one KITE_APPS_URL names or Kite's own.
func appsClient(root string, cfg *config.Config) *apps.Client {
	index := ""
	if cfg != nil {
		index = cfg.Apps.Index
	}
	return apps.ClientFor(root, index)
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

// fetchByName finds a theme or a plugin, kind, in the index, and fetches the
// version wanted, or else the newest that works with this Kite.
func fetchByName(cmd *cobra.Command, root string, cfg *config.Config, kind, id, want string) (*apps.Package, error) {
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
	return client.Fetch(cmd.Context(), app, rel)
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

func installedPackages(root string) ([]apps.Installed, error) {
	var out []apps.Installed
	for _, one := range site.Themes(root) {
		if one.Builtin {
			continue
		}
		pkg := apps.Installed{Kind: "theme", Name: one.Name}
		if m := one.Manifest; m != nil {
			pkg.Version, pkg.Homepage = m.Version, m.Homepage
		}
		out = append(out, pkg)
	}
	ids, err := plugin.Installed(root)
	if err != nil {
		return nil, err
	}
	for _, id := range ids {
		pkg := apps.Installed{Kind: "plugin", Name: id}
		if pl, err := plugin.Open(root, id); err == nil {
			g := apps.GrantOf(pl)
			pkg.Version, pkg.Homepage, pkg.Grant = pl.Manifest.Version, pl.Manifest.Homepage, &g
		} else if m, err := plugin.ReadManifest(os.DirFS(filepath.Join(root, apps.Dir("plugin", id)))); err == nil {
			pkg.Version, pkg.Homepage = m.Version, m.Homepage
		}
		out = append(out, pkg)
	}
	return out, nil
}

// trackedPackages are the packages a site, root, has whose updates come from
// the index the client reads.
func trackedPackages(root string, ix *apps.Index, client *apps.Client) ([]apps.Tracked, error) {
	lf, err := lock.Read(root)
	if err != nil {
		return nil, err
	}
	pkgs, err := installedPackages(root)
	if err != nil {
		return nil, err
	}
	return apps.Track(pkgs, ix, lf, client.Source()), nil
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
				pkgs, err := trackedPackages(root, ix, client)
				if err != nil {
					return err
				}
				for _, t := range pkgs {
					have[t.Kind+"/"+t.Name] = t.Version
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
func outdated(pkgs []apps.Tracked) []outdatedRow {
	rows := []outdatedRow{}
	for _, t := range pkgs {
		st := t.Status(buildinfo.Version)
		if st.Latest == "" && !st.Yanked && st.Delisted == "" {
			continue
		}
		rows = append(rows, outdatedRow{
			Kind: t.Kind, Name: t.Name, Installed: t.Version, Locked: t.Entry != nil,
			Latest: st.Latest, Yanked: st.Yanked, Delisted: st.Delisted, Problem: st.Problem,
		})
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
			pkgs, err := trackedPackages(p.Root, ix, client)
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
			pkgs, err := trackedPackages(p.Root, ix, client)
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
					_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "%s %s: %v\n", t.Kind, t.Name, err)
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
	apps.Tracked
	want string
}

func updateTargets(pkgs []apps.Tracked, args []string, root, source string) ([]target, error) {
	var out []target
	if len(args) == 0 {
		for _, t := range pkgs {
			if t.Status(buildinfo.Version).Latest != "" {
				out = append(out, target{Tracked: t})
			}
		}
		return out, nil
	}
	for _, arg := range args {
		kind, spec, scoped := strings.Cut(arg, "/")
		if !scoped {
			kind, spec = "", arg
		}
		name, want, _ := strings.Cut(spec, "@")
		var found []apps.Tracked
		for _, t := range pkgs {
			if t.Name == name && (kind == "" || t.Kind == kind) {
				found = append(found, t)
			}
		}
		switch {
		case len(found) == 1:
			out = append(out, target{Tracked: found[0], want: want})
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
	i := slices.IndexFunc(pkgs, func(pkg apps.Installed) bool { return pkg.Name == name && (kind == "" || pkg.Kind == kind) })
	if i < 0 {
		return fmt.Errorf("%s is not installed", name)
	}
	if lf, err := lock.Read(root); err == nil {
		if e, ok := lf.Get(pkgs[i].Kind, name); ok && e.Source != source {
			return fmt.Errorf("%s was installed from %s, and the index in use is %s", name, e.Source, source)
		}
	}
	return fmt.Errorf("%s was not installed from the index, and the index lists no package of its homepage under that name; "+
		"'kite %s add %s --replace' installs it from the index", name, pkgs[i].Kind, name)
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
	rel, err := t.App.Pick(t.want, buildinfo.Version)
	if err != nil {
		return err
	}
	if rel.Version == t.Version && t.Entry != nil {
		printf(u.cmd, "%s %s is at %s already\n", t.Kind, t.Name, t.Version)
		return nil
	}
	next, err := u.client.Fetch(u.cmd.Context(), t.App, rel)
	if err != nil {
		return err
	}
	if !u.force {
		tree, err := lock.TreeOf(filepath.Join(u.p.Root, apps.Dir(t.Kind, t.Name)))
		if err != nil {
			return err
		}
		why, err := u.client.Changed(u.cmd.Context(), t.Tracked, tree)
		if err != nil {
			return err
		}
		if why != "" {
			return fmt.Errorf("%s; updating would replace that, so it is left at %s. Run with --force to update it anyway",
				why, t.Version)
		}
	}
	if next.Plugin != nil {
		ok, err := u.agree(t, next)
		if err != nil || !ok {
			return err
		}
		if err := u.checkEnabled(t, next); err != nil {
			return err
		}
	}
	if _, err := u.p.Writer().Apply(u.cmd.Context(), content.ChangeSet{
		Ops:     []content.Op{next.Op(true)},
		Message: fmt.Sprintf("%s: update %s %s to %s", t.Kind, t.Name, t.Version, rel.Version),
	}); err != nil {
		return err
	}
	if rel.Version == t.Version {
		printf(u.cmd, "%s %s %s is recorded as installed from the index now\n", t.Kind, t.Name, t.Version)
	} else {
		printf(u.cmd, "updated %s %s from %s to %s\n", t.Kind, t.Name, t.Version, rel.Version)
	}
	return nil
}

// agree asks for what a plugin's new version does beyond what its owner
// agreed to, and reports whether it was agreed to.
func (u *updater) agree(t target, next *apps.Package) (bool, error) {
	more := apps.Exceeds(*next.Origin.Granted, t.Granted())
	if len(more) == 0 || u.yes {
		return true, nil
	}
	ask := fmt.Sprintf("plugin %s %s %s", t.Name, next.Release.Version, strings.Join(more, ", "))
	if !interactive(u.cmd, false) {
		return false, fmt.Errorf("%s; run with --yes to agree to that", ask)
	}
	printf(u.cmd, "%s.\n", ask)
	ok, err := askYesNo(u.cmd, u.answers, "Update it?", false)
	if err == nil && !ok {
		printf(u.cmd, "left %s at %s\n", t.Name, t.Version)
	}
	return ok, err
}

// checkEnabled checks a new version of an enabled plugin the way enabling
// it does, so that an update never leaves the next build to fail on it.
func (u *updater) checkEnabled(t target, next *apps.Package) error {
	cfg, err := config.Load(u.p.Root)
	if err != nil {
		return err
	}
	if !slices.Contains(cfg.Plugins.Enabled, t.Name) {
		return nil
	}
	return next.Plugin.Check(u.cmd.Context(), filepath.Join(u.p.Root, filepath.FromSlash(plugin.CacheDir)))
}

// lockProblems compares kite.lock with what is installed, for kite doctor.
func lockProblems(root string) []string {
	lf, err := lock.Read(root)
	if err != nil {
		return []string{err.Error()}
	}
	var out []string
	for _, kind := range []string{"theme", "plugin"} {
		section := lf.Themes
		if kind == "plugin" {
			section = lf.Plugins
		}
		for _, name := range slices.Sorted(maps.Keys(section)) {
			e := section[name]
			at := apps.Dir(kind, name)
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
