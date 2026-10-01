package cli

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/kite-plus/kite/internal/buildinfo"
	"github.com/kite-plus/kite/internal/content"
	"github.com/kite-plus/kite/internal/kitew"
	"github.com/kite-plus/kite/internal/lock"
)

func newWrapperCmd() *cobra.Command {
	var version string
	cmd := &cobra.Command{
		Use:   "wrapper",
		Short: "Pin the Kite release the site builds with, and write kitew",
		Long: "Pins a Kite release in kite.lock, the running one unless --version names\n" +
			"another, and writes kitew and kitew.ps1 beside it. They run the pinned\n" +
			"release, downloading it the first time and checking it against the\n" +
			"release's checksums, so that every machine and the deploy build the site\n" +
			"with the same Kite. Commit the three files with the site.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			p, err := openProject()
			if err != nil {
				return err
			}
			v, err := releaseToPin(version)
			if err != nil {
				return err
			}
			ctx, cancel := context.WithTimeout(cmd.Context(), 20*time.Second)
			defer cancel()
			pin, err := kitew.Pin(ctx, http.DefaultClient, v)
			if errors.Is(err, kitew.ErrNoRelease) {
				return err
			}
			if err != nil {
				_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "the checksums of Kite %s could not be fetched: %v\n"+
					"until 'kite wrapper' fetches them, kitew checks downloads against the list the release serves\n", v, err)
			}
			written, err := kitew.Write(p.Root)
			if err != nil {
				return err
			}
			if _, err := p.Writer().Apply(cmd.Context(), content.ChangeSet{
				Ops:     []content.Op{content.PinKite{Version: pin.Version, Checksums: pin.Checksums}},
				Message: "kite: build with " + pin.Version,
			}); err != nil {
				return err
			}

			if jsonOut(cmd) {
				return writeJSON(cmd.OutOrStdout(), map[string]any{
					"version": pin.Version, "checksums": pin.Checksums, "written": written,
				})
			}
			printf(cmd, "pinned Kite %s in %s\n", pin.Version, lock.Name)
			for _, name := range written {
				printf(cmd, "wrote %s\n", name)
			}
			if kitew.Deploy(p.Root) == "self" {
				printf(cmd, "\n%s installs Kite on its own. To build with the pinned release,\n"+
					"run 'sh ./kitew build' in its build step in place of 'kite build'.\n", filepath.ToSlash(WorkflowPath))
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&version, "version", "", "the release to pin, such as 0.1.6; the running one by default")
	return cmd
}

// releaseToPin is the release a --version names, or else the running one.
func releaseToPin(flag string) (string, error) {
	if flag != "" {
		v, ok := kitew.Release(flag)
		if !ok {
			return "", fmt.Errorf("--version %q names no release; give one such as 0.1.6", flag)
		}
		return v, nil
	}
	v, ok := kitew.Release(buildinfo.Version)
	if !ok {
		return "", fmt.Errorf("this Kite, %s, is a build from source, which no release stands for; "+
			"name the release to pin with --version", buildinfo.Version)
	}
	return v, nil
}

// pinNew pins a release in a project being created. Its checksums are
// fetched for a moment only, so that a project made offline is made all the
// same, pinned without them until 'kite wrapper' adds them.
func pinNew(ctx context.Context, root, version string) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	pin, _ := kitew.Pin(ctx, http.DefaultClient, version)
	f, err := lock.Read(root)
	if err != nil {
		return err
	}
	f.Kite = &pin
	data, err := f.Bytes()
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(root, lock.Name), data, 0o644)
}

// pinNotice says so when the project pins another release than the one
// running, which the deploy then builds with. A build from source is no
// release to compare.
func pinNotice(cmd *cobra.Command, root string) {
	running, ok := kitew.Release(buildinfo.Version)
	if !ok {
		return
	}
	f, err := lock.Read(root)
	if err != nil || f.Kite == nil || f.Kite.Version == running {
		return
	}
	_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "note: %s pins Kite %s, which the deploy builds with, and this is %s;\n"+
		"run 'kite wrapper' to build with %s, or ./kitew to run %s\n",
		lock.Name, f.Kite.Version, running, running, f.Kite.Version)
}

// pinProblems are what keeps the release kite.lock pins from being the one
// every build runs. A site that pins none and has no kitew, as one made
// before kitew, has none of them.
func pinProblems(root string) []string {
	f, err := lock.Read(root)
	if err != nil {
		return nil // lockProblems reports it
	}
	scripts := kitew.Installed(root)
	if f.Kite == nil {
		if scripts {
			return []string{"kitew is here, but kite.lock pins no release for it to run; run 'kite wrapper'"}
		}
		return nil
	}
	var out []string
	if !scripts {
		out = append(out, fmt.Sprintf("kite.lock pins Kite %s, but kitew or kitew.ps1 is missing; run 'kite wrapper'", f.Kite.Version))
	}
	if f.Kite.Checksums == "" {
		out = append(out, fmt.Sprintf("kite.lock pins Kite %s without the checksums it was released with; "+
			"run 'kite wrapper' where GitHub can be reached", f.Kite.Version))
	}
	if running, ok := kitew.Release(buildinfo.Version); ok && running != f.Kite.Version {
		out = append(out, fmt.Sprintf("kite.lock pins Kite %s and this is %s; run 'kite wrapper' to build with %s",
			f.Kite.Version, running, running))
	}
	if kitew.Deploy(root) == "self" {
		out = append(out, filepath.ToSlash(WorkflowPath)+" installs Kite on its own rather than running kitew")
	}
	return out
}
