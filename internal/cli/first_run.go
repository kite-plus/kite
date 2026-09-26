package cli

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/kite-plus/kite/internal/api"
	"github.com/kite-plus/kite/internal/setup"
	"github.com/kite-plus/kite/web"
)

// emptyFolder reports whether dir holds nothing but hidden files, such as
// the .git of a fresh clone or what a desktop leaves in every folder it
// opens: a place a new site can be started without burying anything.
func emptyFolder(dir string) (bool, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false, err
	}
	for _, e := range entries {
		if !strings.HasPrefix(e.Name(), ".") {
			return false, nil
		}
	}
	return true, nil
}

// firstRun serves the studio's setup page over an empty folder until the
// site has been described there, and returns once its project is written.
//
// It is `kite init` asked in a browser, for somebody who would rather not
// answer questions in a terminal. Only the way through setup answers: the
// rest of the API says the site is still being created, since there is
// nothing behind it yet, and the address is this machine's alone.
func firstRun(ctx context.Context, cmd *cobra.Command, root, addr string, open bool, log *slog.Logger) error {
	flow := setup.NewSite(setup.Site{BaseURL: defaultBaseURL}, func(site setup.Site) error {
		p := plan{
			Root:        root,
			Title:       site.Title,
			Description: site.Description,
			BaseURL:     site.BaseURL,
			Language:    site.Language,
			Workflow:    true,
		}
		p.fill()
		_, err := create(ctx, p)
		return err
	})
	studio := http.StripPrefix(api.Prefix, api.New(api.Options{
		Site:   func() api.View { return api.View{} },
		Logger: log,
		Setup:  flow,
	}).Handler())

	mux := http.NewServeMux()
	for _, path := range []string{"/setup", "/auth/session", api.OpenAPIPath} {
		mux.Handle(api.Prefix+path, studio)
	}
	mux.HandleFunc(api.Prefix+"/", notYet)
	mux.Handle(web.Path+"/", web.Handler())
	mux.Handle("/", http.RedirectHandler(web.Path+"/", http.StatusFound))

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	server := &http.Server{Handler: mux, ReadHeaderTimeout: 10 * time.Second}
	served := make(chan error, 1)
	go func() { served <- server.Serve(listener) }()

	url := displayURL(addr)
	printf(cmd, "\n  %s\n\n", url)
	printf(cmd, "  there is no site in this folder yet; create one in the browser:\n\n")
	printf(cmd, "    %s%s/\n\n", url, web.Path)
	printf(cmd, "  press ctrl-c to stop\n\n")
	if open {
		go openBrowser(ctx, url+web.Path+"/", log)
	}

	select {
	case <-flow.Created():
	case err := <-served:
		return err
	case <-ctx.Done():
		_ = server.Close()
		return ctx.Err()
	}

	// Shutdown waits for the answer to the request that created the site,
	// then frees the address for the server that will serve it.
	done, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(done); err != nil {
		return err
	}
	if err := <-served; err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	printf(cmd, "  created the site in %s\n", root)
	return nil
}

// notYet answers every other API request while the site is being created.
func notYet(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusServiceUnavailable)
	_ = json.NewEncoder(w).Encode(api.ErrorBody{Error: api.ErrorDetail{
		Code:    api.CodeSetupRequired,
		Message: "there is no site in this folder yet; create it first",
	}})
}
