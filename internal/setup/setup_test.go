package setup_test

import (
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/kite-plus/kite/internal/auth"
	"github.com/kite-plus/kite/internal/setup"
)

func newFlow(t *testing.T) (string, *auth.Guard, *setup.Flow) {
	t.Helper()
	root := t.TempDir()
	guard := auth.New(nil)

	flow, err := setup.New(root, guard)
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

func TestFinishingSetupGuardsTheServerAndClosesTheFlow(t *testing.T) {
	root, guard, flow := newFlow(t)

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

	// The account is on disk, so a restart comes back configured rather than
	// back at setup with the password already taken.
	stored, err := auth.Load(root)
	if err != nil {
		t.Fatalf("the account did not survive: %v", err)
	}
	if stored.User() != "editor" {
		t.Errorf("stored user = %q, want editor", stored.User())
	}
	if _, err := setup.New(root, auth.New(stored)); !errors.Is(err, auth.ErrAlreadyConfigured) {
		t.Errorf("a restarted server would offer setup again: %v", err)
	}
}

// Setup runs once. Whoever finished it owns the studio, and a second attempt
// is not a way to take it from them.
func TestSetupCannotBeRunTwice(t *testing.T) {
	_, _, flow := newFlow(t)

	if _, err := flow.Complete("editor", "correct horse battery"); err != nil {
		t.Fatal(err)
	}
	if _, err := flow.Complete("someone", "another password"); !errors.Is(err, setup.ErrDone) {
		t.Errorf("a second Complete = %v, want ErrDone", err)
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

// Nothing is left in the project but the account: the flow keeps no state of
// its own on disk to be found later.
func TestSetupLeavesNothingBehindButTheAccount(t *testing.T) {
	root, _, flow := newFlow(t)
	if _, err := flow.Complete("admin", "correct horse battery"); err != nil {
		t.Fatal(err)
	}

	entries, err := os.ReadDir(filepath.Join(root, ".kite", "secrets"))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != "account.json" {
		var names []string
		for _, e := range entries {
			names = append(names, e.Name())
		}
		t.Errorf("secrets hold %v, want only account.json", names)
	}
}

func TestANilFlowIsAServerWithNothingToSetUp(t *testing.T) {
	var none *setup.Flow
	if none.Pending() {
		t.Error("a server that was never waiting reports that it is")
	}
}
