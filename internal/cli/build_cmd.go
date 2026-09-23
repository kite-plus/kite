package cli

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"time"

	"github.com/spf13/cobra"

	"github.com/kite-plus/kite/internal/site"
)

type buildReport struct {
	Targets  int      `json:"targets"`
	Rendered int      `json:"rendered"`
	Skipped  int      `json:"skipped"`
	Extra    int      `json:"extra"`
	Files    int      `json:"files"`
	Output   string   `json:"output"`
	Took     string   `json:"took"`
	Verified bool     `json:"verified,omitempty"`
	Diverged []string `json:"diverged,omitempty"`

	// NextDue is when a scheduled post falls due and the site has to be
	// built again to publish it. The deploy workflow reads it.
	NextDue string `json:"next_due,omitempty"`
}

func newBuildCmd() *cobra.Command {
	var (
		out    string
		drafts bool
		verify bool
	)
	cmd := &cobra.Command{
		Use:   "build",
		Short: "Render the site into static files",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			wd, err := os.Getwd()
			if err != nil {
				return err
			}
			s, err := site.Open(cmd.Context(), wd)
			if err != nil {
				return err
			}
			defer func() { _ = s.Close() }()

			outDir := out
			if outDir == "" {
				outDir = filepath.Join(s.Project.Root, s.Config.Build.Output)
			}

			// One instant for both runs of --verify: a scheduled post falling
			// due between them is not a reproducibility failure.
			opts := site.BuildOptions{OutDir: outDir, Drafts: drafts, Now: time.Now()}
			stats, files, err := s.Build(cmd.Context(), opts)
			if err != nil {
				return err
			}

			report := buildReport{
				Targets:  stats.Targets,
				Rendered: stats.Rendered,
				Skipped:  stats.Skipped,
				Extra:    stats.Extra,
				Files:    len(files),
				Output:   outDir,
				Took:     stats.Duration.Round(100000).String(),
			}
			if !stats.NextDue.IsZero() {
				report.NextDue = stats.NextDue.UTC().Format(time.RFC3339)
			}

			if verify {
				diverged, err := verifyBuild(cmd, s, opts)
				if err != nil {
					return err
				}
				report.Verified = true
				report.Diverged = diverged
			}

			if jsonOut(cmd) {
				if err := writeJSON(cmd.OutOrStdout(), report); err != nil {
					return err
				}
			} else {
				printBuild(cmd, report)
			}
			if len(report.Diverged) > 0 {
				return fmt.Errorf("build is not reproducible: %d file(s) differ between two runs", len(report.Diverged))
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&out, "output", "o", "", "write the site here instead of the configured directory")
	cmd.Flags().BoolVar(&drafts, "drafts", false, "include unpublished content")
	cmd.Flags().BoolVar(&verify, "verify", false, "build twice and compare, proving the output is reproducible")
	return cmd
}

func printBuild(cmd *cobra.Command, r buildReport) {
	printf(cmd, "%d target(s), %d rendered, %d extra file(s) (%s)\n", r.Targets, r.Rendered, r.Extra, r.Took)
	printf(cmd, "%d file(s) written to %s\n", r.Files, r.Output)
	if r.NextDue != "" {
		printf(cmd, "next scheduled post is due %s; build again then to publish it\n", r.NextDue)
	}
	if !r.Verified {
		return
	}
	if len(r.Diverged) == 0 {
		printf(cmd, "\nverified: two builds produced byte-identical output\n")
		return
	}
	printf(cmd, "\nNOT reproducible, %d file(s) differ:\n", len(r.Diverged))
	for _, f := range r.Diverged {
		printf(cmd, "  %s\n", f)
	}
}

// verifyBuild renders the site a second time into a scratch directory and
// compares every byte.
//
// Reproducibility is not a nicety here: the build records what each output
// depended on so that it can later skip unchanged work, and a build whose
// output varies between runs would make those records, and any cache built on
// them, meaningless. Checking it in CI is the only way to notice the day some
// hidden input creeps in.
func verifyBuild(cmd *cobra.Command, s *site.Site, opts site.BuildOptions) ([]string, error) {
	scratch, err := os.MkdirTemp("", "kite-verify-*")
	if err != nil {
		return nil, err
	}
	defer func() { _ = os.RemoveAll(scratch) }()

	again := opts
	again.OutDir = filepath.Join(scratch, "public")
	if _, _, err := s.Build(cmd.Context(), again); err != nil {
		return nil, err
	}

	first, err := hashTree(opts.OutDir)
	if err != nil {
		return nil, err
	}
	other, err := hashTree(again.OutDir)
	if err != nil {
		return nil, err
	}

	var diverged []string
	for path, sum := range first {
		if other[path] != sum {
			diverged = append(diverged, path)
		}
	}
	for path := range other {
		if _, ok := first[path]; !ok {
			diverged = append(diverged, path)
		}
	}
	slices.Sort(diverged)
	return slices.Compact(diverged), nil
}

func hashTree(root string) (map[string]string, error) {
	out := make(map[string]string)
	err := filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}
		sum := sha256.Sum256(data)
		out[filepath.ToSlash(rel)] = hex.EncodeToString(sum[:])
		return nil
	})
	return out, err
}
