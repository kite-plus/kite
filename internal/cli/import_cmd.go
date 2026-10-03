package cli

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/kite-plus/kite/internal/config"
	"github.com/kite-plus/kite/internal/importer/hexo"
	"github.com/kite-plus/kite/internal/project"
)

func newImportCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "import",
		Short: "Bring another site's content into a Kite project",
	}
	cmd.AddCommand(newImportHexoCmd())
	return cmd
}

func newImportHexoCmd() *cobra.Command {
	var workflow, ping bool
	cmd := &cobra.Command{
		Use:   "hexo <hexo-site> [dir]",
		Short: "Import the content of a Hexo site",
		Long: "Reads the posts, drafts, pages and files of a Hexo site into a Kite\n" +
			"project, and leaves the Hexo site as it is. Every address Hexo\n" +
			"published a post or page at keeps leading to it.\n\n" +
			"An empty or missing dir becomes a new project named and addressed as\n" +
			"the Hexo site is; a Kite project gets the content added to its own.",
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			from, err := filepath.Abs(args[0])
			if err != nil {
				return err
			}
			dir := "."
			if len(args) == 2 {
				dir = args[1]
			}
			root, err := filepath.Abs(dir)
			if err != nil {
				return err
			}
			if within(root, from) {
				return fmt.Errorf("import into a folder outside the Hexo site: kite import hexo %s <dir>", args[0])
			}
			cfg, err := hexo.ReadConfig(from)
			if err != nil {
				return err
			}

			created := false
			if _, err := os.Stat(filepath.Join(root, project.ConfigName)); errors.Is(err, fs.ErrNotExist) {
				if !emptyDir(root) {
					return fmt.Errorf("%s holds files and is not a Kite project; import into an empty folder", root)
				}
				if _, err := create(cmd.Context(), planFromHexo(root, cfg, workflow, ping)); err != nil {
					return err
				}
				created = true
			}

			p, err := project.Open(root)
			if err != nil {
				return err
			}
			report, err := hexo.Import(cmd.Context(), from, p)
			if err != nil {
				return err
			}
			if jsonOut(cmd) {
				return writeJSON(cmd.OutOrStdout(), map[string]any{"root": root, "created": created, "report": report})
			}
			reportImport(cmd, from, root, report)
			return nil
		},
	}
	cmd.Flags().BoolVar(&workflow, "workflow", true, "write a GitHub Pages deploy workflow into a new project")
	cmd.Flags().BoolVar(&ping, "ping", true, "have that workflow tell Explore after each deploy")
	return cmd
}

// planFromHexo is the new project a Hexo site becomes, named, addressed and
// dated as the Hexo site is where its settings make sense to Kite.
func planFromHexo(root string, cfg hexo.Config, workflow, ping bool) plan {
	p := plan{
		Root:        root,
		Title:       cfg.Title,
		BaseURL:     cfg.URL,
		Author:      cfg.Author,
		Description: cfg.Description,
		Workflow:    workflow,
		Ping:        ping,
	}
	if config.WellFormedLanguage(cfg.Language) {
		p.Language = cfg.Language
	}
	if _, err := time.LoadLocation(cfg.Timezone); err == nil {
		p.Timezone = cfg.Timezone
	}
	p.fill()
	return p
}

// within reports whether path is dir or inside it.
func within(path, dir string) bool {
	rel, err := filepath.Rel(dir, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// emptyDir reports whether a folder is missing or holds nothing but hidden
// files, such as a fresh repository's .git.
func emptyDir(dir string) bool {
	entries, err := os.ReadDir(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return true
	}
	for _, e := range entries {
		if !strings.HasPrefix(e.Name(), ".") {
			return false
		}
	}
	return err == nil
}

func reportImport(cmd *cobra.Command, from, root string, r hexo.Report) {
	printf(cmd, "\nImported the Hexo site at %s into %s\n\n", from, root)
	printf(cmd, "  posts    %d", r.Posts)
	if r.Drafts > 0 {
		printf(cmd, ", and %d draft(s)", r.Drafts)
	}
	printf(cmd, "\n  pages    %d\n  files    %d\n", r.Pages, r.Files)
	if r.Aliases > 0 {
		printf(cmd, "  aliases  %d old address(es) lead on to where each item is now\n", r.Aliases)
	}

	if len(r.Tags) > 0 {
		printf(cmd, "\nKite does not read Hexo tags such as {%% note %%}, and shows them as\n")
		printf(cmd, "written until they are rewritten as markdown. These items hold some:\n")
		for _, loc := range r.Tags {
			printf(cmd, "  %s\n", loc)
		}
	}
	if len(r.Skipped) > 0 {
		printf(cmd, "\nNot imported:\n")
		for _, s := range r.Skipped {
			printf(cmd, "  %s: %s\n", s.Path, s.Reason)
		}
	}

	printf(cmd, "\nNext:\n")
	if rel := relativeTo(root); rel != "" {
		printf(cmd, "  cd %s\n", rel)
	}
	printf(cmd, "  kite run\n")
}
