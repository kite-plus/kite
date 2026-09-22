package cli

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"
	"gopkg.in/yaml.v3"

	"github.com/kite-plus/kite/internal/auth"
	"github.com/kite-plus/kite/internal/config"
	"github.com/kite-plus/kite/internal/content"
	"github.com/kite-plus/kite/internal/project"
	"github.com/kite-plus/kite/internal/store/file"
)

// plan is what a new project will be made of.
//
// It is filled in from flags, then from answers, then from defaults, and
// nothing is written until it is complete. An installation that stopped
// halfway because of a typo in the last question would leave a directory that
// is neither a project nor empty.
type plan struct {
	Root     string
	Title    string
	BaseURL  string
	Language string

	// Workflow writes the GitHub Pages deploy workflow.
	Workflow bool
	// Git runs git init, which the workflow is useless without.
	Git bool
	// Password is the studio account to create, or "" for a project that
	// will only ever be edited on this machine.
	Password string
}

// Defaults are what a project gets when nobody is asked.
//
// The language is not read from the environment here, deliberately: the
// answer to `kite init --yes` should be the same project on every machine,
// and a locale is exactly the kind of hidden input that makes it not be. The
// wizard offers the local one as a default, where a person can see it.
const (
	defaultTitle    = "My Site"
	defaultBaseURL  = "http://localhost:1717"
	defaultLanguage = config.DefaultLanguage
)

func newInitCmd() *cobra.Command {
	var (
		p        plan
		assumeOK bool
	)

	cmd := &cobra.Command{
		Use:   "init [dir]",
		Short: "Create a new Kite project",
		Long: "Asks what the site is called, where it will live and how it gets\n" +
			"deployed, then writes a project that is ready to serve.\n\n" +
			"Every answer has a flag, and --yes takes the defaults without asking,\n" +
			"so the same command also works in a script with no terminal.",
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := "."
			if len(args) == 1 {
				dir = args[0]
			}
			root, err := filepath.Abs(dir)
			if err != nil {
				return err
			}
			p.Root = root

			// Checked before anything is asked. Ten questions and then "that
			// directory is already a project" is a waste of the answers.
			if _, err := os.Stat(filepath.Join(root, project.ConfigName)); err == nil {
				return fmt.Errorf("%s already exists in %s", project.ConfigName, root)
			}

			if interactive(cmd, assumeOK) {
				if err := interview(cmd, &p); err != nil {
					return err
				}
			}
			p.fill()

			created, err := create(cmd.Context(), p)
			if err != nil {
				return err
			}
			if jsonOut(cmd) {
				return writeJSON(cmd.OutOrStdout(), map[string]any{"root": root, "created": created})
			}
			reportInit(cmd, p, created)
			return nil
		},
	}

	cmd.Flags().StringVar(&p.Title, "title", "", "what the site is called (default "+defaultTitle+")")
	cmd.Flags().StringVar(&p.BaseURL, "base-url", "", "where the site will be published (default "+defaultBaseURL+")")
	cmd.Flags().StringVar(&p.Language, "language", "", "the language it is written in (default "+defaultLanguage+")")
	cmd.Flags().BoolVar(&p.Workflow, "workflow", true, "write a GitHub Pages deploy workflow")
	cmd.Flags().BoolVar(&p.Git, "git", false, "run git init in the new project")
	cmd.Flags().BoolVarP(&assumeOK, "yes", "y", false, "take the defaults without asking")
	return cmd
}

// fill replaces anything still unanswered with a default.
func (p *plan) fill() {
	if strings.TrimSpace(p.Title) == "" {
		p.Title = defaultTitle
	}
	if strings.TrimSpace(p.BaseURL) == "" {
		p.BaseURL = defaultBaseURL
	}
	if strings.TrimSpace(p.Language) == "" {
		p.Language = defaultLanguage
	}
}

// interactive reports whether there is a person to ask.
//
// A pipe is not a person: `kite init` inside a Dockerfile or a CI job has to
// finish on its own rather than wait forever for an answer nobody is there to
// give.
func interactive(cmd *cobra.Command, assumeOK bool) bool {
	if assumeOK {
		return false
	}
	in, ok := cmd.InOrStdin().(*os.File)
	return ok && term.IsTerminal(int(in.Fd()))
}

// interview asks for everything the flags did not already answer.
func interview(cmd *cobra.Command, p *plan) error {
	lines := bufio.NewReader(cmd.InOrStdin())

	printf(cmd, "\nSetting up a Kite project in %s\n\n", p.Root)

	var err error
	if p.Title == "" {
		if p.Title, err = askLine(cmd, lines, "What is this site called?", defaultTitle); err != nil {
			return err
		}
	}
	if p.BaseURL == "" {
		printf(cmd, "\nThe address the site will be published at. It ends up in feeds,\n")
		printf(cmd, "sitemaps and canonical links, and can be changed later.\n")
		if p.BaseURL, err = askLine(cmd, lines, "Where will it live?", defaultBaseURL); err != nil {
			return err
		}
	}
	if p.Language == "" {
		for {
			p.Language, err = askLine(cmd, lines, "\nWhat language is it written in?", localLanguage())
			if err != nil {
				return err
			}
			if config.WellFormedLanguage(p.Language) {
				break
			}
			printf(cmd, "  %q is not a language tag; try something like en or zh-CN\n", p.Language)
		}
	}

	// Only offered where it would work. A deploy workflow in a directory that
	// is not going to be a repository is a file nobody will ever run.
	if !cmd.Flags().Changed("workflow") {
		printf(cmd, "\nKite can write a GitHub Actions workflow that builds this site and\n")
		printf(cmd, "publishes it to GitHub Pages on every push.\n")
		if p.Workflow, err = askYesNo(cmd, lines, "Write it?", true); err != nil {
			return err
		}
	}
	if !cmd.Flags().Changed("git") && !inRepository(cmd.Context(), p.Root) && haveGit() {
		if p.Git, err = askYesNo(cmd, lines, "\nStart a git repository here?", true); err != nil {
			return err
		}
	}

	printf(cmd, "\nThe studio is open on this machine and needs a password anywhere\n")
	printf(cmd, "else. You can set one now, or later with 'kite auth set-password'.\n")
	wanted, err := askYesNo(cmd, lines, "Set a password now?", false)
	if err != nil {
		return err
	}
	if wanted {
		if p.Password, err = readPassword(cmd, false); err != nil {
			return err
		}
	}
	return nil
}

// create writes the project described by a plan and returns what it made.
func create(ctx context.Context, p plan) ([]string, error) {
	types := content.DefaultRegistry()
	dirs := []string{"static", "layouts", "themes"}
	for _, t := range types.Types() {
		dirs = append(dirs, filepath.Join(file.ContentDir, t.Dir))
	}
	for _, d := range dirs {
		if err := os.MkdirAll(filepath.Join(p.Root, d), 0o755); err != nil {
			return nil, err
		}
	}
	if err := os.WriteFile(filepath.Join(p.Root, project.ConfigName), []byte(starterConfig(p)), 0o644); err != nil {
		return nil, err
	}
	created := append([]string{project.ConfigName}, dirs...)

	// Everything under .kite is derived and must never be committed.
	ignore := filepath.Join(p.Root, ".gitignore")
	if _, err := os.Stat(ignore); errors.Is(err, os.ErrNotExist) {
		if err := os.WriteFile(ignore, []byte("/.kite/\n/public/\n"), 0o644); err != nil {
			return nil, err
		}
		created = append(created, ".gitignore")
	}

	if p.Workflow {
		// Written now rather than offered later, so that the first push
		// already has somewhere to go.
		workflow, err := writeWorkflow(p.Root, "main")
		if err != nil {
			return nil, err
		}
		if workflow != "" {
			created = append(created, workflow)
		}
	}
	if p.Git {
		if err := gitInit(ctx, p.Root); err != nil {
			return nil, err
		}
		created = append(created, ".git")
	}
	if p.Password != "" {
		if _, err := auth.SetPassword(p.Root, "admin", p.Password); err != nil {
			return nil, err
		}
		created = append(created, auth.File)
	}
	return created, nil
}

// starterConfig is the kite.yaml a new project gets.
//
// It names only what was asked about. Every other key has a default that this
// version of kite already applies, and writing them out would freeze today's
// defaults into every project created today.
func starterConfig(p plan) string {
	return fmt.Sprintf(`site:
  title: %s
  baseURL: %s
  language: %s

content:
  store: file
  dir: content

build:
  output: public
`, yamlScalar(p.Title), yamlScalar(p.BaseURL), yamlScalar(p.Language))
}

// yamlScalar quotes a value the way YAML needs it quoted, if it does.
//
// A site called "Notes: a journal" is an ordinary thing to want and an
// unparseable configuration file if it is written out as it stands.
func yamlScalar(v string) string {
	v = strings.Join(strings.Fields(v), " ")
	out, err := yaml.Marshal(v)
	if err != nil {
		// Marshaling a string cannot fail; if it somehow did, a quoted value
		// is still the safer thing to write.
		return fmt.Sprintf("%q", v)
	}
	return strings.TrimRight(string(out), "\n")
}

func reportInit(cmd *cobra.Command, p plan, created []string) {
	printf(cmd, "\nInitialized a Kite project in %s\n", p.Root)
	printf(cmd, "\n  title     %s\n", p.Title)
	printf(cmd, "  address   %s\n", p.BaseURL)
	printf(cmd, "  language  %s\n", p.Language)
	if p.Password != "" {
		printf(cmd, "  account   admin (%s)\n", auth.File)
	}

	printf(cmd, "\nNext:\n")
	if rel := relativeTo(p.Root); rel != "" {
		printf(cmd, "  cd %s\n", rel)
	}
	printf(cmd, "  kite new post \"My first post\"\n")
	printf(cmd, "  kite run\n")

	if slices.Contains(created, WorkflowPath) {
		printf(cmd, "\n%s will build and deploy this site on every push to main.\n", WorkflowPath)
		printf(cmd, "Turn on Pages first: Settings -> Pages -> Source -> GitHub Actions.\n")
	}
}

// relativeTo is the path to print in a "cd" line, or "" when the project was
// created where the shell already is.
//
// A relative path that climbs out of the working directory is longer and
// harder to read than the absolute one it is a detour to, so it is not used.
func relativeTo(root string) string {
	wd, err := os.Getwd()
	if err != nil {
		return root
	}
	rel, err := filepath.Rel(wd, root)
	switch {
	case err != nil, strings.HasPrefix(rel, ".."):
		return root
	case rel == ".":
		return ""
	}
	return rel
}

// readLine prints a question with what pressing Enter would mean, and returns
// what was typed -- empty when that is what Enter was.
func readLine(cmd *cobra.Command, in *bufio.Reader, question, hint string) (string, error) {
	printf(cmd, "%s [%s] ", question, hint)
	line, err := in.ReadString('\n')
	// A last answer with no newline after it is still an answer; only an EOF
	// with nothing before it means there is nobody there.
	if err != nil && (!errors.Is(err, io.EOF) || line == "") {
		return "", fmt.Errorf("could not read the answer: %w", err)
	}
	return strings.TrimSpace(line), nil
}

// askLine asks a question and returns the answer, or the default for an empty
// one.
func askLine(cmd *cobra.Command, in *bufio.Reader, question, fallback string) (string, error) {
	answer, err := readLine(cmd, in, question, fallback)
	if err != nil {
		return "", err
	}
	if answer == "" {
		return fallback, nil
	}
	return answer, nil
}

// askYesNo asks a question that has two answers.
func askYesNo(cmd *cobra.Command, in *bufio.Reader, question string, fallback bool) (bool, error) {
	hint := "y/N"
	if fallback {
		hint = "Y/n"
	}
	for {
		answer, err := readLine(cmd, in, question, hint)
		if err != nil {
			return false, err
		}
		switch strings.ToLower(answer) {
		case "":
			return fallback, nil
		case "y", "yes":
			return true, nil
		case "n", "no":
			return false, nil
		}
		printf(cmd, "  answer y or n\n")
	}
}

// localLanguage is the language tag this machine is set to, for the wizard to
// offer. It is only ever a default somebody can see and overtype.
func localLanguage() string {
	for _, key := range []string{"LC_ALL", "LC_MESSAGES", "LANG"} {
		value, _, _ := strings.Cut(os.Getenv(key), ".")
		value = strings.ReplaceAll(strings.TrimSpace(value), "_", "-")
		if value == "" || value == "C" || value == "POSIX" {
			continue
		}
		if config.WellFormedLanguage(value) {
			return value
		}
	}
	return defaultLanguage
}

func haveGit() bool {
	_, err := exec.LookPath("git")
	return err == nil
}

// inRepository reports whether the new project would land inside a repository
// that already exists, where a second one nested in it is almost never what
// anybody meant.
func inRepository(ctx context.Context, root string) bool {
	if !haveGit() {
		return false
	}
	// The directory may not exist yet, so the question is asked of the
	// nearest parent that does.
	dir := root
	for {
		if _, err := os.Stat(dir); err == nil {
			break
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return false
		}
		dir = parent
	}

	cmd := exec.CommandContext(ctx, "git", "-C", dir, "rev-parse", "--is-inside-work-tree")
	out, err := cmd.Output()
	return err == nil && strings.TrimSpace(string(out)) == "true"
}

func gitInit(ctx context.Context, root string) error {
	cmd := exec.CommandContext(ctx, "git", "-C", root, "init", "-b", "main")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git init: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}
