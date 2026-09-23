package cli

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/kite-plus/kite/internal/config"
	"github.com/kite-plus/kite/internal/project"
	"github.com/kite-plus/kite/internal/render/theme"
	"github.com/kite-plus/kite/internal/themecheck"
	"github.com/kite-plus/kite/themes"
)

func newThemeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "theme",
		Short: "Work with themes",
	}
	cmd.AddCommand(newThemeVerifyCmd())
	return cmd
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
				printf(cmd, "%s (%s): %d pages identical built and served\n", report.Theme, where, report.Compared)
			} else {
				printf(cmd, "%s (%s): %d of %d pages differ between build and serve\n",
					report.Theme, where, len(report.Differ), report.Compared)
				for _, d := range report.Differ {
					printf(cmd, "  %s\n    %s\n", d.URL, d.Detail)
				}
			}
			if !report.OK() {
				return fmt.Errorf("theme: %s does not draw the same pages in both runtimes", report.Theme)
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
