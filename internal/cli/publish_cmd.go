package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/kite-plus/kite/internal/publish"
	"github.com/kite-plus/kite/internal/site"
)

func newPublishCmd() *cobra.Command {
	var (
		message string
		push    bool
		dryRun  bool
		force   bool
		all     bool
	)

	cmd := &cobra.Command{
		Use:   "publish [path...]",
		Short: "Commit content, and push it when asked to",
		Long: "Commits exactly the paths given and nothing else: what you have\n" +
			"staged stays staged, and every other change stays where it is.\n\n" +
			"With no paths, --all publishes everything Kite can see is\n" +
			"uncommitted under content, static and kite.yaml.",
		RunE: func(cmd *cobra.Command, args []string) error {
			wd, err := os.Getwd()
			if err != nil {
				return err
			}
			s, err := site.Open(cmd.Context(), wd)
			if err != nil {
				return err
			}
			defer func() { _ = s.Close() }()

			publisher := s.Publisher()
			if publisher == nil {
				return fmt.Errorf("publish: this project has no publisher configured")
			}

			paths := args
			if all {
				state, err := publisher.State(cmd.Context())
				if err != nil {
					return err
				}
				paths = append(paths, state.Dirty...)
			}
			if len(paths) == 0 {
				return fmt.Errorf("publish: name what to publish, or pass --all")
			}

			plan, err := publisher.Preflight(cmd.Context(), publish.Request{
				Paths:   paths,
				Message: message,
				Push:    push,
				Force:   force,
			})
			if err != nil {
				return err
			}

			if jsonOut(cmd) && dryRun {
				return writeJSON(cmd.OutOrStdout(), plan)
			}
			describe(cmd, plan)

			if len(plan.Problems) > 0 {
				return fmt.Errorf("publish: refused")
			}
			if dryRun {
				return nil
			}
			if len(plan.Warnings) > 0 && !force {
				return fmt.Errorf("publish: pass --force to go ahead despite the warnings above")
			}

			result, err := publisher.Apply(cmd.Context(), plan)
			// A push can fail after its commit succeeded, so what did happen
			// is reported before the failure that followed it.
			if result != nil {
				report(cmd, result)
			}
			if err != nil {
				return err
			}
			if jsonOut(cmd) {
				return writeJSON(cmd.OutOrStdout(), result)
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&message, "message", "m", "", "commit message")
	cmd.Flags().BoolVar(&push, "push", false, "push the commit to the remote")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "report what would happen and stop")
	cmd.Flags().BoolVar(&force, "force", false, "go ahead despite warnings")
	cmd.Flags().BoolVar(&all, "all", false, "publish everything uncommitted that Kite manages")
	return cmd
}

// describe prints a plan the way a person reads it: what will happen, then
// what stands in the way.
func describe(cmd *cobra.Command, plan *publish.Plan) {
	if jsonOut(cmd) {
		return
	}
	for _, p := range plan.Paths {
		printf(cmd, "  %s\n", p)
	}
	if len(plan.Paths) > 0 {
		printf(cmd, "\n  %q on %s\n", plan.Message, plan.Branch)
		if plan.Push {
			printf(cmd, "  pushing to %s\n", plan.Remote)
		}
	}
	for _, p := range plan.Warnings {
		printf(cmd, "\n  warning: %s\n", p.Detail)
		if p.Fix != "" {
			printf(cmd, "           %s\n", p.Fix)
		}
	}
	for _, p := range plan.Problems {
		printf(cmd, "\n  %s\n", p.Detail)
		if p.Fix != "" {
			printf(cmd, "  %s\n", p.Fix)
		}
	}
}

func report(cmd *cobra.Command, result *publish.Result) {
	if jsonOut(cmd) || result.Commit == "" {
		return
	}
	printf(cmd, "\n  committed %s\n", result.Commit[:min(8, len(result.Commit))])
	if result.Pushed {
		printf(cmd, "  pushed\n")
	}
}
