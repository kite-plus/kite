package cli

import (
	"fmt"
	"net/http"

	"github.com/spf13/cobra"

	"github.com/kite-plus/kite/internal/project"
	"github.com/kite-plus/kite/internal/publish/ping"
)

type pingReport struct {
	Site   string       `json:"site,omitempty"`
	Feed   string       `json:"feed,omitempty"`
	Method string       `json:"method,omitempty"`
	Pings  []pingResult `json:"pings"`
}

type pingResult struct {
	Endpoint string `json:"endpoint"`
	OK       bool   `json:"ok"`
	// Message is what the service said, or why the ping failed.
	Message string `json:"message,omitempty"`
}

func newPingCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "ping",
		Short: "Tell update services that the site has changed",
		Long: "Sends each update service publish.ping lists in kite.yaml the XML-RPC\n" +
			"ping blog engines send once something is published, naming the site, its\n" +
			"address and its feed. The deploy workflow kite init writes runs it once\n" +
			"the site is deployed. KITE_SITE_BASEURL stands in for site.baseURL, as it\n" +
			"does for kite build. It fails if any service could not be told.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			_, cfg, err := openWithConfig()
			if err != nil {
				return err
			}
			report := pingReport{Pings: []pingResult{}}
			if len(cfg.Publish.Ping) == 0 {
				if jsonOut(cmd) {
					return writeJSON(cmd.OutOrStdout(), report)
				}
				printf(cmd, "publish.ping in %s lists no update service; nothing to ping\n", project.ConfigName)
				return nil
			}
			if cfg.Site.BaseURL == "" {
				return fmt.Errorf("site.baseURL is empty, and a ping names the site by its address; "+
					"set it in %s or with KITE_SITE_BASEURL", project.ConfigName)
			}

			site := ping.Site{Title: cfg.Site.Title, URL: cfg.Site.BaseURL}
			if cfg.Build.Feed {
				site.Feed = cfg.Site.BaseURL + "/rss.xml"
			}
			report.Site, report.Feed, report.Method = site.URL, site.Feed, site.Method()

			failed := 0
			for _, endpoint := range cfg.Publish.Ping {
				message, err := ping.Send(cmd.Context(), http.DefaultClient, endpoint, site)
				result := pingResult{Endpoint: endpoint, OK: err == nil, Message: message}
				if err != nil {
					failed++
					result.Message = err.Error()
				}
				report.Pings = append(report.Pings, result)
				switch {
				case jsonOut(cmd):
				case err != nil:
					_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "could not ping %s: %v\n", endpoint, err)
				case message != "":
					printf(cmd, "pinged %s: %s\n", endpoint, message)
				default:
					printf(cmd, "pinged %s\n", endpoint)
				}
			}
			if jsonOut(cmd) {
				if err := writeJSON(cmd.OutOrStdout(), report); err != nil {
					return err
				}
			}
			if failed > 0 {
				return fmt.Errorf("%d of %d ping(s) failed", failed, len(report.Pings))
			}
			return nil
		},
	}
}
