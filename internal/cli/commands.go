package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/spf13/cobra"

	"github.com/kite-plus/kite/internal/buildinfo"
	"github.com/kite-plus/kite/internal/content"
	"github.com/kite-plus/kite/internal/project"
	"github.com/kite-plus/kite/internal/store/file"
)

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			info := map[string]any{
				"version":   buildinfo.Version,
				"commit":    buildinfo.Commit,
				"date":      buildinfo.Date,
				"go":        runtime.Version(),
				"platform":  runtime.GOOS + "/" + runtime.GOARCH,
				"themeAPI":  buildinfo.ThemeAPIVersion,
				"pluginABI": buildinfo.PluginABIVersion,
				"buildABI":  buildinfo.BuildABIVersion,
			}
			if jsonOut(cmd) {
				return writeJSON(cmd.OutOrStdout(), info)
			}
			printf(cmd, "kite %s (%s, %s) %s %s\n", buildinfo.Version, buildinfo.Commit,
				buildinfo.Date, runtime.Version(), info["platform"])
			printf(cmd, "theme api %s, plugin abi %d\n", buildinfo.ThemeAPIVersion, buildinfo.PluginABIVersion)
			return nil
		},
	}
}

const starterConfig = `site:
  title: My Site
  baseURL: http://localhost:1717
  language: en

content:
  store: file
  dir: content

build:
  output: public
`

func newInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init [dir]",
		Short: "Create a new Kite project",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := "."
			if len(args) == 1 {
				dir = args[0]
			}
			root, err := filepath.Abs(dir)
			if err != nil {
				return err
			}
			cfg := filepath.Join(root, project.ConfigName)
			if _, err := os.Stat(cfg); err == nil {
				return fmt.Errorf("%s already exists", project.ConfigName)
			}

			types := content.DefaultRegistry()
			dirs := []string{"static", "layouts", "themes"}
			for _, t := range types.Types() {
				dirs = append(dirs, filepath.Join(file.ContentDir, t.Dir))
			}
			for _, d := range dirs {
				if err := os.MkdirAll(filepath.Join(root, d), 0o755); err != nil {
					return err
				}
			}
			if err := os.WriteFile(cfg, []byte(starterConfig), 0o644); err != nil {
				return err
			}
			// Everything under .kite is derived and must never be committed.
			ignore := filepath.Join(root, ".gitignore")
			if _, err := os.Stat(ignore); errors.Is(err, os.ErrNotExist) {
				if err := os.WriteFile(ignore, []byte("/.kite/\n/public/\n"), 0o644); err != nil {
					return err
				}
			}

			// A deploy workflow is written now rather than offered later,
			// so that the first push already has somewhere to go.
			workflow, err := writeWorkflow(root, "main")
			if err != nil {
				return err
			}
			if workflow != "" {
				dirs = append(dirs, workflow)
			}

			if jsonOut(cmd) {
				return writeJSON(cmd.OutOrStdout(), map[string]any{"root": root, "created": dirs})
			}
			printf(cmd, "Initialized a Kite project in %s\n", root)
			printf(cmd, "\nNext:\n  kite new post \"My first post\"\n  kite run\n")
			if workflow != "" {
				printf(cmd, "\n%s will build and deploy this site on every push to main.\n", workflow)
				printf(cmd, "Turn on Pages first: Settings -> Pages -> Source -> GitHub Actions.\n")
			}
			return nil
		},
	}
}

func newNewCmd() *cobra.Command {
	var draft bool
	cmd := &cobra.Command{
		Use:   "new <kind> <title>",
		Short: "Create a content item",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			p, err := openProject()
			if err != nil {
				return err
			}
			t, err := p.TypeOrError(args[0])
			if err != nil {
				return err
			}

			status := content.StatusPublished
			if draft {
				status = content.StatusDraft
			}
			now := time.Now().UTC().Truncate(time.Second)
			item := &content.Content{
				Kind:   t.Kind,
				Title:  args[1],
				Status: status,
				Body:   content.Body{Format: content.FormatMarkdown, Raw: "\n"},
			}
			if status == content.StatusPublished {
				item.PublishedAt = &now
			}

			res, err := p.Writer().Apply(cmd.Context(), content.ChangeSet{
				Ops:     []content.Op{content.PutContent{Content: item}},
				Message: "new: " + item.Title,
			})
			if err != nil {
				return err
			}

			if jsonOut(cmd) {
				return writeJSON(cmd.OutOrStdout(), map[string]any{
					"id":      item.ID,
					"kind":    item.Kind,
					"slug":    item.Slug,
					"locator": item.Locator,
					"written": res.Written,
				})
			}
			for _, w := range res.Written {
				printf(cmd, "%s\n", w)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&draft, "draft", false, "create the item as a draft")
	return cmd
}

type doctorReport struct {
	Root       string   `json:"root"`
	Items      int      `json:"items"`
	MissingIDs []string `json:"missingIds,omitempty"`
	Problems   []string `json:"problems,omitempty"`
	Fixed      []string `json:"fixed,omitempty"`
}

func newDoctorCmd() *cobra.Command {
	var fixIDs bool
	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Check the project for problems",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			p, err := openProject()
			if err != nil {
				return err
			}
			scan, err := p.Scanner().Scan()
			if err != nil {
				// A duplicate ID is reported, never repaired: only the author
				// knows which copy should keep the identity.
				return err
			}

			report := doctorReport{Root: p.Root, Items: len(scan.Entries)}
			for _, pr := range scan.Problems {
				report.Problems = append(report.Problems, pr.Error())
			}

			missing := scan.MissingIDs()
			for _, e := range missing {
				report.MissingIDs = append(report.MissingIDs, e.Path)
			}

			if fixIDs && len(missing) > 0 {
				var cs content.ChangeSet
				for _, e := range missing {
					e.Item.ID = content.NewID()
					cs.Add(content.PutContent{Content: e.Item, IfRevision: e.Item.Revision})
				}
				cs.Message = "doctor: assign content ids"
				res, err := p.Writer().Apply(cmd.Context(), cs)
				if err != nil {
					return err
				}
				report.Fixed = res.Written
				report.MissingIDs = nil
			}

			if jsonOut(cmd) {
				return writeJSON(cmd.OutOrStdout(), report)
			}
			return printDoctor(cmd, report, fixIDs)
		},
	}
	cmd.Flags().BoolVar(&fixIDs, "fix-ids", false, "assign an id to every item that lacks one")
	return cmd
}

func printDoctor(cmd *cobra.Command, r doctorReport, fixed bool) error {
	printf(cmd, "project  %s\n", r.Root)
	printf(cmd, "items    %d\n", r.Items)

	if len(r.Problems) > 0 {
		printf(cmd, "\nunreadable files:\n")
		for _, p := range r.Problems {
			printf(cmd, "  %s\n", p)
		}
	}
	if len(r.Fixed) > 0 {
		printf(cmd, "\nassigned ids to %d file(s):\n", len(r.Fixed))
		for _, f := range r.Fixed {
			printf(cmd, "  %s\n", f)
		}
	}
	if len(r.MissingIDs) > 0 {
		printf(cmd, "\n%d file(s) have no id:\n", len(r.MissingIDs))
		for _, m := range r.MissingIDs {
			printf(cmd, "  %s\n", m)
		}
		printf(cmd, "\nrun 'kite doctor --fix-ids' to assign them\n")
	}
	if len(r.Problems) == 0 && len(r.MissingIDs) == 0 && !fixed {
		printf(cmd, "\nno problems found\n")
	}
	return nil
}

func openProject() (*project.Project, error) {
	wd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	p, err := project.Open(wd)
	if errors.Is(err, project.ErrNotFound) {
		return nil, fmt.Errorf("%w\n\nrun 'kite init' to create one", err)
	}
	return p, err
}
