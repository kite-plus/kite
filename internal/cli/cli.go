// Package cli implements the kite command line interface.
package cli

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/spf13/cobra"
)

// Execute runs the root command.
func Execute() error {
	return newRootCmd().Execute()
}

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "kite",
		Short: "A modern open-source publishing platform",
		Long: "Kite manages your content. Where it is deployed is a property of\n" +
			"the content, not a different product.",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.PersistentFlags().BoolP("json", "j", false, "emit machine readable output")

	root.AddCommand(
		newVersionCmd(),
		newInitCmd(),
		newNewCmd(),
		newDoctorCmd(),
		newIndexCmd(),
		newListCmd(),
		newBuildCmd(),
		newServeCmd(),
		newPublishCmd(),
		newAuthCmd(),
		newOpenAPICmd(),
		newRunCmd(),
		newThemeCmd(),
		newPluginCmd(),
	)
	return root
}

// jsonOut reports whether the command should emit JSON, so that every command
// is scriptable without a second code path for humans.
func jsonOut(cmd *cobra.Command) bool {
	v, _ := cmd.Flags().GetBool("json")
	return v
}

func writeJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func printf(cmd *cobra.Command, format string, args ...any) {
	// Failing to write to stdout is not something a command can react to.
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), format, args...)
}
