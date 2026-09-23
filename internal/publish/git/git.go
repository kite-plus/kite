// Package git publishes content by committing and pushing it.
//
// Every operation runs the git binary rather than reimplementing git. The
// repository belongs to the user: it is configured with their credential
// helper, their signing key, their hooks, their .gitattributes filters and
// their LFS setup. A library that reimplements git ignores all of that
// silently, and the first time an author notices is when a pointer file was
// supposed to be written and a 40MB blob was.
package git

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/kite-plus/kite/internal/publish"
)

// timeout bounds one git invocation.
//
// A wedged command inside an HTTP handler is indistinguishable from a hung
// server, and a network operation is the one most likely to wedge.
const (
	timeout     = 30 * time.Second
	pushTimeout = 5 * time.Minute
)

// runner invokes git in a repository.
type runner struct {
	root string

	// env is added to every command's environment, for the scratch index
	// and the literal paths a replay needs.
	env []string
}

// with returns a runner whose commands see more of an environment.
func (r runner) with(env ...string) runner {
	r.env = append(slices.Clone(r.env), env...)
	return r
}

// literal returns a runner that takes every path as spelled. A file name
// with a glob character in it would otherwise be a pattern, and could match
// files nobody named.
func (r runner) literal() runner { return r.with("GIT_LITERAL_PATHSPECS=1") }

// call runs a command with input, for the plumbing that reads its work from
// stdin, and returns what it wrote.
func (r runner) call(ctx context.Context, stdin string, args ...string) (string, error) {
	var stdout, stderr bytes.Buffer

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", r.root}, args...)...)
	cmd.Env = r.environ()
	cmd.Stdin = strings.NewReader(stdin)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return stdout.String(), r.wrap(err, args, &stderr)
	}
	return stdout.String(), nil
}

// ok runs a command and reports only whether it succeeded, for the many git
// questions whose answer is the exit status.
func (r runner) ok(ctx context.Context, args ...string) bool {
	return r.run(ctx, timeout, nil, nil, args...) == nil
}

func (r runner) run(ctx context.Context, limit time.Duration, stdout, stderr *bytes.Buffer, args ...string) error {
	ctx, cancel := context.WithTimeout(ctx, limit)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", r.root}, args...)...)
	cmd.Env = r.environ()
	// Assigned only when there is somewhere to write. A nil *bytes.Buffer in
	// an interface is not nil, and exec would write to it.
	if stdout != nil {
		cmd.Stdout = stdout
	}
	if stderr != nil {
		cmd.Stderr = stderr
	}
	// A git command must never read from the terminal this process happens to
	// have; the one that would is the one asking for a password.
	cmd.Stdin = nil

	return cmd.Run()
}

// environ is the user's environment with every interactive prompt disabled.
//
// The point is not to hide prompts but to turn a hang into an error. A server
// process asked for a password has nobody to ask, and an HTTP request that
// waits forever for one is the worst failure this package can produce: no
// output, no error, nothing to act on.
//
// Everything that makes credentials work -- the credential helper, the ssh
// agent, the signing key -- is left exactly as the user configured it.
func (r runner) environ() []string {
	env := append(os.Environ(),
		// No terminal prompt, and no graphical one either.
		"GIT_TERMINAL_PROMPT=0",
		"SSH_ASKPASS_REQUIRE=never",
		// A path that cannot be executed: git then falls back to the
		// terminal, which the line above has already closed off.
		"GIT_ASKPASS="+filepath.Join(r.root, ".kite", "no-askpass"),
	)
	// Only when the user has not configured ssh themselves, so a custom ssh
	// command keeps working.
	if os.Getenv("GIT_SSH_COMMAND") == "" {
		env = append(env, "GIT_SSH_COMMAND=ssh -o BatchMode=yes")
	}
	return append(env, r.env...)
}

// read runs a command that only inspects the repository.
//
// GIT_OPTIONAL_LOCKS=0 keeps a status check from taking the index lock, so
// reading the state of a repository can never collide with the user's own
// git command running in a terminal beside it.
func (r runner) read(ctx context.Context, args ...string) (string, error) {
	out, err := r.readRaw(ctx, args...)
	return strings.TrimSpace(out), err
}

// readRaw is read without trimming, for output whose leading whitespace is
// data.
//
// A porcelain status line begins with two status characters, and for a file
// that is changed but not staged the first of them is a space. Trimming it
// shifts every field left and quietly renames the first path in the list.
func (r runner) readRaw(ctx context.Context, args ...string) (string, error) {
	var stdout, stderr bytes.Buffer

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", r.root}, args...)...)
	cmd.Env = append(r.environ(), "GIT_OPTIONAL_LOCKS=0")
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return stdout.String(), r.wrap(err, args, &stderr)
	}
	return stdout.String(), nil
}

// wrap turns a failed command into an error carrying git's own message, which
// is almost always more useful than anything this package could say instead.
func (r runner) wrap(err error, args []string, stderr *bytes.Buffer) error {
	detail := strings.TrimSpace(stderr.String())
	if detail == "" {
		detail = err.Error()
	}

	if errors.Is(err, context.DeadlineExceeded) {
		return publish.Problem{
			Code:   publish.CodeGitFailed,
			Detail: "git " + args[0] + " did not finish in time",
			Fix:    "check whether the remote is reachable",
		}
	}
	if isCredentialFailure(detail) {
		return publish.Problem{
			Code:   publish.CodeNoCredentials,
			Detail: detail,
			Fix: "git could not authenticate without asking. Run the same " +
				"push once in a terminal so your credential helper stores it.",
		}
	}
	return publish.Problem{
		Code:   publish.CodeGitFailed,
		Detail: "git " + strings.Join(args, " ") + ": " + detail,
	}
}

// credentialFailures are the ways git reports that it needed to ask and could
// not. They are matched on text because git offers no exit code for it.
var credentialFailures = []string{
	"terminal prompts disabled",
	"could not read Username",
	"could not read Password",
	"Authentication failed",
	"Permission denied (publickey)",
	"Host key verification failed",
	"no askpass",
}

func isCredentialFailure(detail string) bool {
	for _, marker := range credentialFailures {
		if strings.Contains(detail, marker) {
			return true
		}
	}
	return false
}

// Available reports whether git can be used at all.
func Available() bool {
	_, err := exec.LookPath("git")
	return err == nil
}

// gitDir returns the repository's git directory, absolute.
func (r runner) gitDir(ctx context.Context) (string, error) {
	out, err := r.read(ctx, "rev-parse", "--absolute-git-dir")
	if err != nil {
		return "", err
	}
	if out == "" {
		return "", fmt.Errorf("git: no git directory")
	}
	return out, nil
}
