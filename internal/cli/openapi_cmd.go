package cli

import (
	"github.com/spf13/cobra"

	"github.com/kite-plus/kite/internal/api"
)

func newOpenAPICmd() *cobra.Command {
	return &cobra.Command{
		Use:   "openapi",
		Short: "Print the API description",
		Long: "Writes the OpenAPI document for this build's read model API.\n" +
			"It needs no project and no running server, so a client can be\n" +
			"generated from it during a build.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			doc, err := api.Document()
			if err != nil {
				return err
			}
			printf(cmd, "%s\n", doc)
			return nil
		},
	}
}
