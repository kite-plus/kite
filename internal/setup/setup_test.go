package setup_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/kite-plus/kite/internal/auth"
	"github.com/kite-plus/kite/internal/setup"
)

var frozen = time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)

func newFlow(t *testing.T) (string, *auth.Guard, *setup.Flow) {
	t.Helper()
	root := t.TempDir()
	guard := auth.NewWithClock(nil, func() time.Time { return frozen })

	flow, err := setup.NewWithClock(root, guard, func() time.Time { return frozen })
	if err != nil {
		t.Fatalf("setup.New: %v", err)
	}
	return root, guard, flow
}

func TestAProjectWithAnAccountHasNothingToSetUp(t *testing.T) {
	root := t.TempDir()
	if _, err := auth.SetPassword(root, "admin", "correct horse battery"); err != nil {
		t.Fatal(err)
	}
	account, err := auth.Load(root)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := setup.New(root, auth.New(account)); !errors.Is(err, auth.ErrAlreadyConfigured) {
		t.Errorf("setup.New on a configured project = %v, want ErrAlreadyConfigured", err)
	}
}

// The token is the whole of what keeps an unconfigured server on a public
// address from belonging to whoever finds the port first.
func TestOnlyTheTokenTheServerPrintedIsAccepted(t *testing.T) {
	_, _, flow := newFlow(t)

	for _, wrong := range []string{"", " ", strings.ToUpper(flow.Token()), flow.Token() + "x"} {
		if err := flow.Check(wrong); !errors.Is(err, setup.ErrBadToken) {
			t.Errorf("Check(%q) = %v, want ErrBadToken", wrong, err)
		}
	}
	if err := flow.Check(flow.Token()); err != nil {
		t.Errorf("the printed token was refused: %v", err)
	}
}

// A token nobody was given must not be reachable by guessing at machine
// speed, so failures start costing time.
func TestRepeatedWrongTokensStartBeingRefusedOutright(t *testing.T) {
	_, _, flow := newFlow(t)

	var refused *auth.TooManyAttempts
	for range 20 {
		if err := flow.Check("wrong"); errors.As(err, &refused) {
			break
		}
	}
	if refused == nil {
		t.Fatal("wrong tokens can be tried without limit")
	}
	// And the right one is refused too while the wait is on, or the backoff
	// would be nothing more than a suggestion.
	if err := flow.Check(flow.Token()); !errors.As(err, &refused) {
		t.Errorf("Check during a backoff = %v, want TooManyAttempts", err)
	}
}

func TestFinishingSetupGuardsTheServerAndSpendsTheToken(t *testing.T) {
	root, guard, flow := newFlow(t)
	token := flow.Token()

	if guard.Required() {
		t.Fatal("the guard wants a password before there is an account")
	}
	account, err := flow.Complete("editor", "correct horse battery")
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}

	if !guard.Required() || guard.User() != "editor" {
		t.Errorf("guard required=%v user=%q, want true and editor", guard.Required(), guard.User())
	}
	if err := account.Verify("editor", "correct horse battery"); err != nil {
		t.Errorf("the account will not take its own password: %v", err)
	}
	if flow.Pending() {
		t.Error("setup is still pending after it finished")
	}

	// A spent token is not left on disk to be found in a backup later.
	if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(setup.TokenFile))); !os.IsNotExist(err) {
		t.Errorf("the token file survived setup: %v", err)
	}
	// And a second attempt cannot take the server over.
	if _, err := flow.Complete("someone", "another password"); !errors.Is(err, setup.ErrDone) {
		t.Errorf("a second Complete = %v, want ErrDone", err)
	}
	if err := flow.Check(token); !errors.Is(err, setup.ErrDone) {
		t.Errorf("Check after setup = %v, want ErrDone", err)
	}
}

// A container that restarts must not invalidate the link its operator just
// copied out of the log.
func TestARestartKeepsTheSameToken(t *testing.T) {
	root, _, flow := newFlow(t)

	again, err := setup.New(root, auth.New(nil))
	if err != nil {
		t.Fatalf("setup.New: %v", err)
	}
	if again.Token() != flow.Token() {
		t.Errorf("token changed across a restart: %q then %q", flow.Token(), again.Token())
	}

	stored, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(setup.TokenFile)))
	if err != nil {
		t.Fatalf("read %s: %v", setup.TokenFile, err)
	}
	if strings.TrimSpace(string(stored)) != flow.Token() {
		t.Error("the stored token is not the one being checked against")
	}
}

// An operator who would rather not read a token out of a log can name one.
func TestTheEnvironmentCanNameTheToken(t *testing.T) {
	t.Setenv(setup.TokenEnv, "  a secret i chose  ")

	root := t.TempDir()
	flow, err := setup.New(root, auth.New(nil))
	if err != nil {
		t.Fatal(err)
	}
	if got, want := flow.Token(), "a secret i chose"; got != want {
		t.Errorf("token = %q, want %q", got, want)
	}
	// Nothing was written: a token that came from the environment is already
	// somewhere the operator keeps secrets.
	if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(setup.TokenFile))); !os.IsNotExist(err) {
		t.Errorf("a token from the environment was also written to disk: %v", err)
	}
}

func TestANilFlowIsAServerWithNothingToSetUp(t *testing.T) {
	var none *setup.Flow
	if none.Pending() {
		t.Error("a server that was never waiting reports that it is")
	}
}

// Two browsers can have the form open at once, and both can press the button.
// Exactly one of them may create the account, or the password that ends up
// stored is whichever one happened to finish second.
func TestOnlyOneOfTwoSimultaneousSetupsWins(t *testing.T) {
	root, guard, flow := newFlow(t)

	var wg sync.WaitGroup
	results := make([]error, 2)
	passwords := []string{"first password here", "second password here"}

	wg.Add(len(results))
	for i := range results {
		go func() {
			defer wg.Done()
			_, results[i] = flow.Complete("admin", passwords[i])
		}()
	}
	wg.Wait()

	var won int
	for _, err := range results {
		switch {
		case err == nil:
			won++
		case errors.Is(err, setup.ErrDone):
		default:
			t.Errorf("unexpected refusal: %v", err)
		}
	}
	if won != 1 {
		t.Fatalf("%d of 2 setups succeeded, want exactly 1", won)
	}

	// And what was stored is a password one of them actually chose.
	stored, err := auth.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	var matched bool
	for _, password := range passwords {
		if stored.Verify("admin", password) == nil {
			matched = true
		}
	}
	if !matched {
		t.Error("the stored account takes neither password that was submitted")
	}
	if !guard.Required() {
		t.Error("the studio is still open after setup finished")
	}
}
