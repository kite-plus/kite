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
		rebase  bool
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
			"uncommitted under content, static and kite.yaml, and --push alone\n" +
			"pushes what is already committed.\n\n" +
			"A remote that has moved on is never overwritten. When nothing it\n" +
			"changed is anything you published, --rebase puts your commit on top\n" +
			"of its commits and pushes.",
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
				if !push {
					return fmt.Errorf("publish: name what to publish, pass --all, or pass --push to push what is committed")
				}
				return pushCommitted(cmd, publisher, rebase)
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
			// Asked for up front, a push refused only because the remote moved
			// on elsewhere is a step rather than a stop.
			if err != nil && rebase && result != nil && result.Remote != nil && result.Remote.Rebase {
				result, err = publisher.Push(cmd.Context(), publish.PushRequest{Rebase: true})
			}
			// A push can fail after its commit succeeded, so what did happen
			// is reported before the failure that followed it.
			if result != nil {
				report(cmd, result)
			}
			if err != nil {
				if result != nil && result.Remote != nil {
					return fmt.Errorf("publish: committed, but not pushed")
				}
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
	cmd.Flags().BoolVar(&rebase, "rebase", false,
		"when the remote has moved on without touching this commit, put it on top and push")
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
	if result.Rebased {
		printf(cmd, "\n  committed %s, on top of the remote's new commits\n", short(result.Commit))
	} else {
		printf(cmd, "\n  committed %s\n", short(result.Commit))
	}
	if result.Pushed {
		printf(cmd, "  pushed\n")
	}
	if result.Remote != nil {
		describeRemote(cmd, result.Remote)
	}
}

// pushCommitted pushes what is already committed, for a publish whose push
// did not go through: publishing the same paths again finds nothing to commit.
func pushCommitted(cmd *cobra.Command, publisher publish.Publisher, rebase bool) error {
	result, err := publisher.Push(cmd.Context(), publish.PushRequest{Rebase: rebase})
	if result != nil && !jsonOut(cmd) {
		if result.Pushed {
			how := ""
			if result.Rebased {
				how = ", on top of the remote's new commits"
			}
			printf(cmd, "\n  pushed %s%s\n", short(result.Commit), how)
		}
		if result.Remote != nil {
			describeRemote(cmd, result.Remote)
		}
	}
	if err != nil {
		if result != nil && result.Remote != nil {
			return fmt.Errorf("publish: not pushed")
		}
		return err
	}
	if jsonOut(cmd) {
		return writeJSON(cmd.OutOrStdout(), result)
	}
	return nil
}

// describeRemote says what a remote that moved on has, and what to do.
func describeRemote(cmd *cobra.Command, r *publish.RemoteChange) {
	printf(cmd, "\n  %s has %d commit(s) this branch does not:\n", r.Upstream, r.Behind)
	shown := min(len(r.Commits), 5)
	for _, c := range r.Commits[:shown] {
		printf(cmd, "    %s %s (%s)\n", short(c.Hash), c.Subject, c.Author)
	}
	if more := r.Behind - shown; more > 0 {
		printf(cmd, "    and %d more\n", more)
	}

	if r.Rebase {
		printf(cmd, "\n  none of them change what you published, so your commit can go on top:\n")
		printf(cmd, "    kite publish --push --rebase\n")
		return
	}
	if r.Blocked != nil {
		printf(cmd, "\n  %s\n", r.Blocked.Detail)
		if r.Blocked.Fix != "" {
			printf(cmd, "  %s\n", r.Blocked.Fix)
		}
	}
	if r.Diff != "" {
		printf(cmd, "\n%s", r.Diff)
	}
}

func short(hash string) string { return hash[:min(8, len(hash))] }
