package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/kite-plus/kite/internal/auth"
)

func newAuthCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Manage the account that guards the studio",
		Long: "A project with an account asks for a password before the studio or\n" +
			"the API will answer. A project without one is open, and a server that\n" +
			"would put an open studio on a public address refuses to start.",
		Args: cobra.NoArgs,
	}
	cmd.AddCommand(newAuthStatusCmd(), newAuthSetPasswordCmd(), newAuthRemoveCmd())
	return cmd
}

type authReport struct {
	Configured bool   `json:"configured"`
	User       string `json:"user,omitempty"`
	Source     string `json:"source,omitempty"`
}

func newAuthStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Report whether the studio asks for a password",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			p, err := openProject()
			if err != nil {
				return err
			}

			account, err := auth.Open(p.Root)
			if err != nil && !errors.Is(err, auth.ErrNoAccount) {
				return err
			}

			report := authReport{Configured: account != nil}
			if account != nil {
				report.User, report.Source = account.User(), account.Source()
			}
			if jsonOut(cmd) {
				return writeJSON(cmd.OutOrStdout(), report)
			}

			if !report.Configured {
				printf(cmd, "no account: the studio is open to anyone who can reach it\n\n")
				printf(cmd, "run 'kite auth set-password' before serving on anything but localhost\n")
				return nil
			}
			printf(cmd, "user    %s\n", report.User)
			printf(cmd, "from    %s\n", report.Source)
			return nil
		},
	}
}

func newAuthSetPasswordCmd() *cobra.Command {
	var user string
	var stdin bool

	cmd := &cobra.Command{
		Use:   "set-password",
		Short: "Create or change the studio account",
		Long: "The password is hashed with argon2id and written to " + auth.File + ",\n" +
			"which is ignored by git and has to survive a deployment for the\n" +
			"account to. Changing a password signs out every browser.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			p, err := openProject()
			if err != nil {
				return err
			}

			// An existing account keeps its name unless one is given, so
			// changing a password cannot quietly rename the account.
			if user == "" {
				user = "admin"
				if existing, err := auth.Load(p.Root); err == nil {
					user = existing.User()
				}
			}

			password, err := readPassword(cmd, stdin)
			if err != nil {
				return err
			}
			if _, err := auth.SetPassword(p.Root, user, password); err != nil {
				return err
			}

			printf(cmd, "account %s written to %s\n", user, auth.File)
			if _, err := auth.FromEnv(); err == nil {
				printf(cmd, "\nnote: KITE_ADMIN_PASSWORD is set here and takes precedence\n"+
					"      over the stored account while it is\n")
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&user, "user", "", "the name to sign in with (default admin)")
	cmd.Flags().BoolVar(&stdin, "stdin", false, "read the password from standard input instead of asking")
	return cmd
}

func newAuthRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "remove",
		Short: "Delete the stored account, leaving the studio open",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			p, err := openProject()
			if err != nil {
				return err
			}
			if err := auth.Remove(p.Root); err != nil {
				if errors.Is(err, auth.ErrNoAccount) {
					return errors.New("there is no stored account to remove")
				}
				return err
			}
			printf(cmd, "account removed; the studio is open again\n")
			printf(cmd, "a server on anything but localhost will now refuse to start\n")
			return nil
		},
	}
}

// readPassword asks for a password twice without echoing it.
//
// A piped password is read as it is: a deployment script has no terminal to
// type into, and asking it to fake one is how passwords end up in shell
// history instead.
func readPassword(cmd *cobra.Command, stdin bool) (string, error) {
	fd := int(os.Stdin.Fd())
	if stdin || !term.IsTerminal(fd) {
		data, err := io.ReadAll(cmd.InOrStdin())
		if err != nil {
			return "", err
		}
		return strings.TrimRight(string(data), "\r\n"), nil
	}

	first, err := ask(cmd, fd, "new password: ")
	if err != nil {
		return "", err
	}
	second, err := ask(cmd, fd, "again: ")
	if err != nil {
		return "", err
	}
	if first != second {
		return "", errors.New("the two passwords do not match")
	}
	return first, nil
}

func ask(cmd *cobra.Command, fd int, prompt string) (string, error) {
	printf(cmd, "%s", prompt)
	// The typed password never reaches the terminal, so it cannot be left on
	// screen or scrolled back to.
	typed, err := term.ReadPassword(fd)
	printf(cmd, "\n")
	if err != nil {
		return "", fmt.Errorf("could not read the password: %w", err)
	}
	return string(typed), nil
}
