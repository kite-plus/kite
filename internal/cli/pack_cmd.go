package cli

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/spf13/cobra"

	"github.com/kite-plus/kite/internal/archive"
	"github.com/kite-plus/kite/internal/pack"
	"github.com/kite-plus/kite/internal/plugin"
	"github.com/kite-plus/kite/internal/render/theme"
)

// besideTheManifest matches the files beside a manifest that a package
// carries whatever its kind: its license and notices, and its readme.
var besideTheManifest = regexp.MustCompile(`(?i)^(licen[cs]e|copying|notice|readme)(\..+)?$`)

// packed is what kite theme pack and kite plugin pack report.
type packed struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Path    string `json:"path"`
	Files   int    `json:"files"`
	Size    int    `json:"size"`
	SHA256  string `json:"sha256"`
}

func newThemePackCmd() *cobra.Command {
	var output string
	cmd := &cobra.Command{
		Use:   "pack [dir]",
		Short: "Pack a theme as the zip archive a release carries",
		Long: "Packs the theme in dir, or in the current directory, as <name>-<version>.zip:\n" +
			"theme.yaml, layouts, static, assets, i18n, the screenshot, and the license\n" +
			"and readme, under one folder named after the theme. The same files always\n" +
			"pack to the same bytes, and the archive is checked the way an install\n" +
			"checks it before it is written.",
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := dirArg(args)
			th, err := theme.Load(os.DirFS(dir))
			if err != nil {
				return err
			}
			m := th.Manifest
			if m.Version == "" {
				return fmt.Errorf("theme %s: give it a version in %s before packing it", m.Name, theme.ManifestName)
			}
			names := []string{theme.ManifestName}
			if shot, ok := th.ScreenshotPath(); ok {
				names = append(names, shot)
			}
			files, err := packageFiles(dir, names, []string{theme.LayoutsDir, "static", "assets", pack.Dir}, themePackage)
			if err != nil {
				return err
			}
			return writePackage(cmd, dir, output, m.Name, m.Version, files, themePackage, func(fsys fs.FS) error {
				_, err := theme.Load(fsys)
				return err
			})
		},
	}
	cmd.Flags().StringVarP(&output, "output", "o", "", "where to write the archive (default dist/<name>-<version>.zip in dir)")
	return cmd
}

func newPluginPackCmd() *cobra.Command {
	var output string
	cmd := &cobra.Command{
		Use:   "pack [dir]",
		Short: "Pack a plugin as the zip archive a release carries",
		Long: "Packs the plugin in dir, or in the current directory, as <id>-<version>.zip:\n" +
			"plugin.yaml, plugin.wasm, assets, i18n, and the license and readme, under\n" +
			"one folder named after the plugin. The same files always pack to the same\n" +
			"bytes, and the archive is checked the way an install checks it before it\n" +
			"is written.",
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := dirArg(args)
			manifest, err := plugin.ReadManifest(os.DirFS(dir))
			if err != nil {
				return err
			}
			loaded, err := plugin.Load(os.DirFS(dir), manifest.ID)
			if err != nil {
				return err
			}
			m := loaded.Manifest
			if m.Version == "" {
				return fmt.Errorf("plugin %s: give it a version in %s before packing it", m.ID, plugin.ManifestName)
			}
			names := []string{plugin.ManifestName}
			if _, err := os.Stat(filepath.Join(dir, plugin.WasmName)); err == nil {
				names = append(names, plugin.WasmName)
			}
			files, err := packageFiles(dir, names, []string{plugin.AssetsDir, pack.Dir}, pluginPackage)
			if err != nil {
				return err
			}
			return writePackage(cmd, dir, output, m.ID, m.Version, files, pluginPackage, func(fsys fs.FS) error {
				p, err := plugin.Load(fsys, m.ID)
				if err != nil {
					return err
				}
				return p.Check(cmd.Context(), "")
			})
		},
	}
	cmd.Flags().StringVarP(&output, "output", "o", "", "where to write the archive (default dist/<id>-<version>.zip in dir)")
	return cmd
}

func dirArg(args []string) string {
	if len(args) == 1 {
		return args[0]
	}
	return "."
}

// packageFiles reads what a package is made of: the named files, every file
// in the named folders, and the license and readme beside the manifest.
// Hidden files are left out, and anything but a plain file is refused, as an
// install refuses it.
func packageFiles(dir string, names, dirs []string, kind packageKind) (map[string][]byte, error) {
	files := make(map[string][]byte)
	var total int64
	add := func(rel string) error {
		data, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(rel)))
		if err != nil {
			return err
		}
		if total += int64(len(data)); total > kind.maxSize || len(files) == kind.maxFiles {
			return fmt.Errorf("a %s may take at most %d MB in %d files", kind.name, kind.maxSize>>20, kind.maxFiles)
		}
		files[rel] = data
		return nil
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		if e.Type().IsRegular() && besideTheManifest.MatchString(e.Name()) {
			names = append(names, e.Name())
		}
	}
	for _, name := range names {
		if err := add(name); err != nil {
			return nil, err
		}
	}
	for _, sub := range dirs {
		root := filepath.Join(dir, sub)
		if info, err := os.Stat(root); errors.Is(err, fs.ErrNotExist) {
			continue
		} else if err != nil {
			return nil, err
		} else if !info.IsDir() {
			return nil, fmt.Errorf("%s is not a folder", filepath.Join(dir, sub))
		}
		err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if strings.HasPrefix(d.Name(), ".") {
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			if d.IsDir() {
				return nil
			}
			if !d.Type().IsRegular() {
				return fmt.Errorf("%s is not a plain file, and an install would refuse it", p)
			}
			rel, err := filepath.Rel(dir, p)
			if err != nil {
				return err
			}
			return add(filepath.ToSlash(rel))
		})
		if err != nil {
			return nil, err
		}
	}
	return files, nil
}

// writePackage packs files under name, checks the archive as an install
// would, and writes it where output says.
func writePackage(cmd *cobra.Command, dir, output, name, version string, files map[string][]byte,
	kind packageKind, load func(fs.FS) error) error {
	var buf bytes.Buffer
	if err := archive.Pack(&buf, name, files); err != nil {
		return err
	}
	if buf.Len() > kind.maxArchive {
		return fmt.Errorf("a %s archive may be at most %d MB", kind.name, kind.maxArchive>>20)
	}
	unpacked, problem := archive.Unpack(buf.Bytes(), kind.name, kind.manifest, kind.maxSize, kind.maxFiles)
	if problem != "" {
		return errors.New(problem)
	}
	if err := load(archive.FS(unpacked)); err != nil {
		return fmt.Errorf("the packed %s does not load: %w", kind.name, err)
	}

	if output == "" {
		output = filepath.Join(dir, "dist", name+"-"+version+".zip")
	}
	if err := os.MkdirAll(filepath.Dir(output), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(output, buf.Bytes(), 0o644); err != nil {
		return err
	}
	sum := sha256.Sum256(buf.Bytes())
	report := packed{
		Name: name, Version: version, Path: output,
		Files: len(files), Size: buf.Len(), SHA256: hex.EncodeToString(sum[:]),
	}
	if jsonOut(cmd) {
		return writeJSON(cmd.OutOrStdout(), report)
	}
	printf(cmd, "packed %s %s in %s: %d files, %d bytes\nsha256 %s\n",
		name, version, output, report.Files, report.Size, report.SHA256)
	return nil
}
