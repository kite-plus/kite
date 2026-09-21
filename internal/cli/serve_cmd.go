package cli

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/kite-plus/kite/internal/api"
	"github.com/kite-plus/kite/internal/serve"
	"github.com/kite-plus/kite/internal/site"
	"github.com/kite-plus/kite/web"
)

func newServeCmd() *cobra.Command {
	return serveCommand(commandShape{
		use:   "serve",
		short: "Serve the site over HTTP, rendering each request from the files",
		long: "Pages are rendered on demand from the same content, the same\n" +
			"templates and the same resolver a build uses, so what is served is\n" +
			"what would be published.",
		defaultWatch:  true,
		defaultReload: true,
		open:          false,
	})
}

func newRunCmd() *cobra.Command {
	return serveCommand(commandShape{
		use:   "run",
		short: "Start writing: serve the site and open it in a browser",
		long: "The shortest path from a checkout to a page on screen. Equivalent\n" +
			"to serve with drafts included and a browser opened.",
		defaultWatch:  true,
		defaultReload: true,
		defaultDrafts: true,
		defaultAdmin:  true,
		defaultWrite:  true,
		open:          true,
	})
}

type commandShape struct {
	use, short, long string
	defaultWatch     bool
	defaultReload    bool
	defaultDrafts    bool
	defaultAdmin     bool
	defaultWrite     bool
	open             bool
}

func serveCommand(shape commandShape) *cobra.Command {
	var (
		addr   string
		port   int
		watch  bool
		reload bool
		drafts bool
		admin  bool
		write  bool
		open   bool
		quiet  bool
	)

	cmd := &cobra.Command{
		Use:   shape.use,
		Short: shape.short,
		Long:  shape.long,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			wd, err := os.Getwd()
			if err != nil {
				return err
			}

			// Ctrl-C should stop the server rather than kill it mid-response.
			ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
			defer stop()

			s, err := site.Open(ctx, wd)
			if err != nil {
				return err
			}
			defer func() { _ = s.Close() }()

			listenAddr, err := resolveAddr(addr, port)
			if err != nil {
				return err
			}

			level := slog.LevelInfo
			if quiet {
				level = slog.LevelWarn
			}
			log := slog.New(slog.NewTextHandler(cmd.ErrOrStderr(), &slog.HandlerOptions{Level: level}))

			srv, err := serve.New(ctx, s, serve.Options{
				Addr:       listenAddr,
				LiveReload: reload,
				Watch:      watch,
				Drafts:     drafts,
				Admin:      admin,
				Write:      write,
				Logger:     log,
			})
			if err != nil {
				return err
			}

			url := "http://" + listenAddr
			printf(cmd, "\n  %s\n\n", url)
			if drafts {
				printf(cmd, "  drafts included\n")
			}
			if watch {
				printf(cmd, "  watching for changes\n")
			}
			if admin {
				printf(cmd, "  studio at %s%s/\n", url, web.Path)
				printf(cmd, "  api at %s%s\n", url, api.Prefix)
				if !write {
					printf(cmd, "  read only\n")
				}
			}
			printf(cmd, "  press ctrl-c to stop\n\n")

			if open {
				go openBrowser(ctx, url, log)
			}
			return srv.ListenAndServe(ctx)
		},
	}

	cmd.Flags().StringVar(&addr, "addr", "", "address to listen on (default 127.0.0.1)")
	cmd.Flags().IntVarP(&port, "port", "p", 1717, "port to listen on, or 0 to pick a free one")
	cmd.Flags().BoolVar(&watch, "watch", shape.defaultWatch, "reload when the project changes")
	cmd.Flags().BoolVar(&reload, "live-reload", shape.defaultReload, "refresh open pages after a change")
	cmd.Flags().BoolVar(&drafts, "drafts", shape.defaultDrafts, "include unpublished content")
	cmd.Flags().BoolVar(&admin, "admin", shape.defaultAdmin, "serve the read model API")
	cmd.Flags().BoolVar(&write, "write", shape.defaultWrite, "let the API change the project")
	cmd.Flags().BoolVar(&open, "open", shape.open, "open the site in a browser")
	cmd.Flags().BoolVarP(&quiet, "quiet", "q", false, "only log warnings and errors")
	return cmd
}

// resolveAddr turns the address flags into something to listen on, asking the
// operating system for a free port when one was not chosen.
func resolveAddr(addr string, port int) (string, error) {
	if addr != "" {
		if _, _, err := net.SplitHostPort(addr); err != nil {
			return "", fmt.Errorf("serve: invalid --addr %q: %w", addr, err)
		}
		return addr, nil
	}
	if port == 0 {
		l, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			return "", err
		}
		chosen := l.Addr().String()
		// Closing before the server binds leaves a small race, which is the
		// price of letting the OS choose and is fine for a local preview.
		_ = l.Close()
		return chosen, nil
	}
	return fmt.Sprintf("127.0.0.1:%d", port), nil
}

// openBrowser waits for the server to answer before opening a window, so the
// first thing the author sees is the site rather than a connection error.
func openBrowser(ctx context.Context, url string, log *slog.Logger) {
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if ctx.Err() != nil {
			return
		}
		conn, err := net.DialTimeout("tcp", trimScheme(url), 200*time.Millisecond)
		if err == nil {
			_ = conn.Close()
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.CommandContext(ctx, "open", url)
	case "windows":
		cmd = exec.CommandContext(ctx, "rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.CommandContext(ctx, "xdg-open", url)
	}
	if err := cmd.Start(); err != nil {
		log.Warn("could not open a browser", "url", url, "err", err)
	}
}

func trimScheme(url string) string {
	after, _ := strings.CutPrefix(url, "http://")
	return after
}
