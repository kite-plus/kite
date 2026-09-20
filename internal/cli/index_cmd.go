package cli

import (
	"time"

	"github.com/spf13/cobra"

	"github.com/kite-plus/kite/internal/content"
	"github.com/kite-plus/kite/internal/index"
	"github.com/kite-plus/kite/internal/reader"
)

type indexReport struct {
	Scanned int    `json:"scanned"`
	Indexed int    `json:"indexed"`
	Skipped int    `json:"skipped"`
	Removed int    `json:"removed"`
	Items   int    `json:"items"`
	Took    string `json:"took"`
}

func newIndexCmd() *cobra.Command {
	var rebuild bool
	cmd := &cobra.Command{
		Use:   "index",
		Short: "Update the derived content index",
		Long: "The index is a cache derived from the markdown files. Deleting it\n" +
			"and running this command again must reproduce it exactly.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			p, err := openProject()
			if err != nil {
				return err
			}
			ix, err := index.Open(p.Root, p.Types)
			if err != nil {
				return err
			}
			defer ix.Close()

			run := ix.Reconcile
			if rebuild {
				run = ix.Rebuild
			}
			stats, err := run(cmd.Context())
			if err != nil {
				return err
			}

			items, err := ix.Count(cmd.Context())
			if err != nil {
				return err
			}
			report := indexReport{
				Scanned: stats.Scanned,
				Indexed: stats.Indexed,
				Skipped: stats.Skipped,
				Removed: stats.Removed,
				Items:   items,
				Took:    stats.Duration.Round(time.Millisecond / 10).String(),
			}
			if jsonOut(cmd) {
				return writeJSON(cmd.OutOrStdout(), report)
			}
			printf(cmd, "scanned %d, indexed %d, skipped %d, removed %d (%s)\n",
				report.Scanned, report.Indexed, report.Skipped, report.Removed, report.Took)
			printf(cmd, "%d item(s) in the index\n", items)
			return nil
		},
	}
	cmd.Flags().BoolVar(&rebuild, "rebuild", false, "discard the index and rebuild it from scratch")
	return cmd
}

func newListCmd() *cobra.Command {
	var (
		kind   string
		tag    string
		limit  int
		cursor string
	)
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List content from the index",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			p, err := openProject()
			if err != nil {
				return err
			}
			ix, err := index.Open(p.Root, p.Types)
			if err != nil {
				return err
			}
			defer ix.Close()
			if _, err := ix.Reconcile(cmd.Context()); err != nil {
				return err
			}

			q := content.Query{Limit: limit, Cursor: cursor}
			if kind != "" {
				q.Kinds = []content.Kind{content.Kind(kind)}
			}
			if tag != "" {
				q.TermsAny = map[string][]string{"tag": {tag}}
			}

			page, err := reader.New(ix.DB()).Query(cmd.Context(), q)
			if err != nil {
				return err
			}
			if jsonOut(cmd) {
				return writeJSON(cmd.OutOrStdout(), page)
			}
			for _, s := range page.Items {
				date := "          "
				if s.PublishedAt != nil {
					date = s.PublishedAt.Format("2006-01-02")
				}
				printf(cmd, "%s  %-8s %-9s %s\n", date, s.Kind, s.Status, s.Title)
			}
			if page.HasMore {
				printf(cmd, "\nmore results: --cursor %s\n", page.NextCursor)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&kind, "kind", "", "only list this content kind")
	cmd.Flags().StringVar(&tag, "tag", "", "only list items carrying this tag")
	cmd.Flags().IntVar(&limit, "limit", 20, "maximum number of items")
	cmd.Flags().StringVar(&cursor, "cursor", "", "continue from a previous page")
	return cmd
}
