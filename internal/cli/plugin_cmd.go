package cli

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/spf13/cobra"

	"github.com/kite-plus/kite/internal/archive"
	"github.com/kite-plus/kite/internal/config"
	"github.com/kite-plus/kite/internal/content"
	"github.com/kite-plus/kite/internal/plugin"
	"github.com/kite-plus/kite/internal/project"
	"github.com/kite-plus/kite/internal/render/theme"
)

func newPluginCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "plugin",
		Short: "Work with plugins",
		Long: "A plugin lives in plugins/<id> and runs once it is enabled, which\n" +
			"lists it under plugins.enabled in kite.yaml. These commands make the\n" +
			"same changes the studio's plugin page does.",
		Args: cobra.NoArgs,
	}
	cmd.AddCommand(
		newPluginListCmd(),
		newPluginAddCmd(),
		newPluginRemoveCmd(),
		newPluginSwitchCmd(true),
		newPluginSwitchCmd(false),
		newPluginVerifyCmd(),
		newPluginPackCmd(),
		newPluginNewCmd(),
	)
	return cmd
}

type pluginRow struct {
	ID      string `json:"id"`
	Name    string `json:"name,omitempty"`
	Version string `json:"version,omitempty"`
	Enabled bool   `json:"enabled"`
	Problem string `json:"problem,omitempty"`
}

func newPluginListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List the plugins in plugins/ and which are enabled",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			p, cfg, err := openWithConfig()
			if err != nil {
				return err
			}
			ids, err := plugin.Installed(p.Root)
			if err != nil {
				return err
			}
			rows := make([]pluginRow, 0, len(ids))
			for _, id := range ids {
				row := pluginRow{ID: id, Enabled: slices.Contains(cfg.Plugins.Enabled, id)}
				if loaded, err := plugin.Open(p.Root, id); err == nil {
					row.Name, row.Version = loaded.Manifest.Name, loaded.Manifest.Version
				} else {
					row.Problem = err.Error()
				}
				rows = append(rows, row)
			}
			// Enabled but not installed is worth a line of its own: the next
			// build will refuse it.
			for _, id := range cfg.Plugins.Enabled {
				if !slices.Contains(ids, id) {
					rows = append(rows, pluginRow{ID: id, Enabled: true, Problem: "enabled, but not installed in " + plugin.Dir})
				}
			}
			if jsonOut(cmd) {
				return writeJSON(cmd.OutOrStdout(), rows)
			}
			if len(rows) == 0 {
				printf(cmd, "no plugins installed\n")
				return nil
			}
			for _, row := range rows {
				state := "off"
				if row.Enabled {
					state = "on "
				}
				printf(cmd, "%s  %-20s %-10s %s\n", state, row.ID, row.Version, row.Name)
				if row.Problem != "" {
					printf(cmd, "     %s\n", row.Problem)
				}
			}
			return nil
		},
	}
}

func newPluginAddCmd() *cobra.Command {
	var replace bool
	cmd := &cobra.Command{
		Use:   "add <name[@version]|archive.zip|dir>",
		Short: "Install a plugin by name from the index, or from a zip archive or a directory, turned off",
		Long: "Installs a plugin into plugins/<id>, turned off, and says what it does\n" +
			"to a site. A name that is not a file or a directory is looked up in the\n" +
			"index, which 'kite apps search' looks through: the newest version that\n" +
			"works with this Kite is installed, or the one after @, and kite.lock\n" +
			"records where it came from and what it was installed to do.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			p, cfg, err := openWithConfig()
			if err != nil {
				return err
			}
			var added *plugin.Plugin
			var files map[string][]byte
			var origin *content.Origin
			if id, want, ok := byName(args[0]); ok {
				got, err := fetchByName(cmd, p.Root, cfg, "plugin", id, want)
				if err != nil {
					return err
				}
				added, files, origin = got.plugin, got.files, got.origin
			} else {
				if files, err = readPackage(args[0], pluginPackage); err != nil {
					return err
				}
				if added, err = checkPlugin(files); err != nil {
					return err
				}
			}
			id := added.Manifest.ID
			installed, err := plugin.Installed(p.Root)
			if err != nil {
				return err
			}
			exists := slices.Contains(installed, id)
			if exists && !replace {
				return fmt.Errorf("plugin %s is installed already; add --replace to replace it, "+
					"or run 'kite apps update %s' to update it from the index", id, id)
			}
			if _, err := p.Writer().Apply(cmd.Context(), content.ChangeSet{
				Ops:     []content.Op{content.PutPlugin{ID: id, Files: files, Replace: exists, Origin: origin}},
				Message: fmt.Sprintf("plugin: add %s %s", id, added.Manifest.Version),
			}); err != nil {
				return err
			}
			printf(cmd, "installed %s %s%s in %s\n", id, added.Manifest.Version, fromIndex(origin),
				filepath.Join(plugin.Dir, id))
			printf(cmd, "  %s\n", describeGrant(grantOf(added)))
			printf(cmd, "it is off: run 'kite plugin enable %s' to turn it on\n", id)
			return nil
		},
	}
	cmd.Flags().BoolVar(&replace, "replace", false, "replace an installed plugin of the same id")
	return cmd
}

// checkPlugin checks a plugin's files the way a site checks a plugin it
// loads.
func checkPlugin(files map[string][]byte) (*plugin.Plugin, error) {
	manifest, err := plugin.ReadManifest(archive.FS(files))
	if err != nil {
		return nil, err
	}
	return plugin.Load(archive.FS(files), manifest.ID)
}

// packageKind is what a package of one kind may hold when it is installed
// from a zip archive or a directory.
type packageKind struct {
	name       string
	manifest   string
	maxArchive int
	maxSize    int64
	maxFiles   int
}

var (
	pluginPackage = packageKind{"plugin", plugin.ManifestName, plugin.MaxArchive, plugin.MaxSize, plugin.MaxFiles}
	themePackage  = packageKind{"theme", theme.ManifestName, theme.MaxArchive, theme.MaxSize, theme.MaxFiles}
)

// readPackage reads a plugin's or a theme's files from a zip archive or a
// directory.
func readPackage(path string, kind packageKind) (map[string][]byte, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		if len(data) > kind.maxArchive {
			return nil, fmt.Errorf("a %s archive may be at most %d MB", kind.name, kind.maxArchive>>20)
		}
		files, problem := archive.Unpack(data, kind.name, kind.manifest, kind.maxSize, kind.maxFiles)
		if problem != "" {
			return nil, errors.New(problem)
		}
		return files, nil
	}

	files := make(map[string][]byte)
	var total int64
	err = filepath.WalkDir(path, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		name := d.Name()
		if d.IsDir() {
			// What version control and editors keep beside a package is not
			// part of it.
			if p != path && strings.HasPrefix(name, ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasPrefix(name, ".") || !d.Type().IsRegular() {
			return nil
		}
		rel, err := filepath.Rel(path, p)
		if err != nil {
			return err
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		if total += int64(len(data)); total > kind.maxSize || len(files) == kind.maxFiles {
			return fmt.Errorf("a %s may take at most %d MB in %d files", kind.name, kind.maxSize>>20, kind.maxFiles)
		}
		files[filepath.ToSlash(rel)] = data
		return nil
	})
	return files, err
}

func newPluginRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "remove <id>",
		Short: "Delete an installed plugin that is turned off",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			p, cfg, err := openWithConfig()
			if err != nil {
				return err
			}
			id := args[0]
			if slices.Contains(cfg.Plugins.Enabled, id) {
				return fmt.Errorf("plugin %s is enabled; run 'kite plugin disable %s' first", id, id)
			}
			if _, err := p.Writer().Apply(cmd.Context(), content.ChangeSet{
				Ops:     []content.Op{content.DeletePlugin{ID: id}},
				Message: "plugin: remove " + id,
			}); err != nil {
				if errors.Is(err, content.ErrNotFound) {
					return fmt.Errorf("plugin %s is not installed", id)
				}
				return err
			}
			printf(cmd, "removed %s; its settings stay in %s\n", filepath.Join(plugin.Dir, id), config.Name)
			return nil
		},
	}
}

func newPluginSwitchCmd(on bool) *cobra.Command {
	use, short := "disable <id>", "Stop running a plugin"
	if on {
		use, short = "enable <id>", "Run a plugin, after the ones already enabled"
	}
	return &cobra.Command{
		Use:   use,
		Short: short,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			p, cfg, err := openWithConfig()
			if err != nil {
				return err
			}
			id := args[0]
			enabled := slices.DeleteFunc(slices.Clone(cfg.Plugins.Enabled), func(one string) bool { return one == id })
			if on {
				// Refused here rather than left for the next build to find.
				one, err := plugin.Open(p.Root, id)
				if err != nil {
					return err
				}
				if err := one.Check(cmd.Context(), filepath.Join(p.Root, filepath.FromSlash(plugin.CacheDir))); err != nil {
					return err
				}
				if slices.Contains(cfg.Plugins.Enabled, id) {
					enabled = cfg.Plugins.Enabled
				} else {
					enabled = append(enabled, id)
				}
			}
			if slices.Equal(enabled, cfg.Plugins.Enabled) {
				printf(cmd, "%s is %s already\n", id, map[bool]string{true: "enabled", false: "disabled"}[on])
				return nil
			}
			verb := map[bool]string{true: "enable", false: "disable"}[on]
			if _, err := p.Writer().Apply(cmd.Context(), content.ChangeSet{
				Ops:     []content.Op{content.PutSettings{Values: map[string]any{"plugins.enabled": enabled}}},
				Message: "plugin: " + verb + " " + id,
			}); err != nil {
				return err
			}
			printf(cmd, "%sd %s\n", verb, id)
			return nil
		},
	}
}

func newPluginVerifyCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "verify [dir]",
		Short: "Check that a plugin loads, as a site would load it",
		Long: "Reads the plugin in dir, or in the current directory, and checks its\n" +
			"manifest, its templates, its settings and its module the way a site\n" +
			"checks them when it loads the plugin.",
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := "."
			if len(args) == 1 {
				dir = args[0]
			}
			manifest, err := plugin.ReadManifest(os.DirFS(dir))
			if err != nil {
				return err
			}
			loaded, err := plugin.Load(os.DirFS(dir), manifest.ID)
			if err != nil {
				return err
			}
			if err := loaded.Check(cmd.Context(), ""); err != nil {
				return err
			}
			m := loaded.Manifest
			if jsonOut(cmd) {
				return writeJSON(cmd.OutOrStdout(), pluginCheck{
					ID:         m.ID,
					Version:    m.Version,
					APIVersion: m.APIVersion,
					Requires:   m.Requires,
					License:    m.License,
					Inject:     len(m.Inject),
					Loads:      nonNil(loaded.Hosts()),
					Hooks:      nonNil(m.Hooks),
					Settings:   len(m.Settings),
				})
			}
			printf(cmd, "%s %s loads\n", m.ID, m.Version)
			if len(m.Inject) > 0 {
				printf(cmd, "  injects %d piece(s) of code\n", len(m.Inject))
			}
			if hosts := loaded.Hosts(); len(hosts) > 0 {
				printf(cmd, "  loads from %s\n", strings.Join(hosts, ", "))
			}
			if len(m.Hooks) > 0 {
				printf(cmd, "  %s exports %s\n", plugin.WasmName, strings.Join(m.Hooks, ", "))
			}
			if len(m.Settings) > 0 {
				printf(cmd, "  %d setting(s)\n", len(m.Settings))
			}
			return nil
		},
	}
}

// pluginCheck is what kite plugin verify reports with --json: what an index
// of plugins shows a site owner before one is installed.
type pluginCheck struct {
	ID         string   `json:"id"`
	Version    string   `json:"version"`
	APIVersion string   `json:"api_version"`
	Requires   string   `json:"requires,omitempty"`
	License    string   `json:"license,omitempty"`
	Inject     int      `json:"inject"`
	Loads      []string `json:"loads"`
	Hooks      []string `json:"hooks"`
	Settings   int      `json:"settings"`
}

func nonNil(list []string) []string {
	if list == nil {
		return []string{}
	}
	return list
}

func newPluginNewCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "new <id>",
		Short: "Start a plugin in a new directory",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]
			if !config.ValidPluginID(id) {
				return fmt.Errorf("%q is not a plugin id: use lowercase words joined by -", id)
			}
			if _, err := os.Stat(id); err == nil {
				return fmt.Errorf("%s already exists", id)
			}
			files := map[string]string{
				plugin.ManifestName:                        starterPlugin(id),
				filepath.Join(plugin.AssetsDir, id+".css"): "." + id + " { font-style: italic; }\n",
			}
			for name, body := range files {
				path := filepath.Join(id, name)
				if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
					return err
				}
				if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
					return err
				}
			}
			printf(cmd, "created %s\n\n", id)
			printf(cmd, "Next:\n  edit %s\n  kite plugin verify %s\n  kite plugin add %s   (in a site)\n",
				filepath.Join(id, plugin.ManifestName), id, id)
			return nil
		},
	}
}

// starterPlugin is the manifest a new plugin starts from: one stylesheet on
// every page and one line under every post, with a setting to say it.
func starterPlugin(id string) string {
	return fmt.Sprintf(`id: %s
name: %s
version: 0.1.0
apiVersion: %s
description: What this plugin adds to a site.

inject:
  - at: head
    html: <link rel="stylesheet" href="{{ asset "%s.css" }}">
  - at: body
    pages: [single]
    kinds: [post]
    html: <p class="%s">{{ .Settings.message }}</p>

settings:
  - key: message
    type: string
    label: Message
    default: Thanks for reading.
`, id, id, plugin.APIVersion, id, id)
}

// openWithConfig opens the project the command runs in, with its kite.yaml.
func openWithConfig() (*project.Project, *config.Config, error) {
	p, err := openProject()
	if err != nil {
		return nil, nil, err
	}
	cfg, err := config.Load(p.Root)
	if err != nil {
		return nil, nil, err
	}
	return p, cfg, nil
}
