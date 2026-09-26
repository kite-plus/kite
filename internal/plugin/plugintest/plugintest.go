// Package plugintest builds the module the plugin tests run.
package plugintest

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// Guest is the import path of the module's source.
const Guest = "github.com/kite-plus/kite/internal/plugin/testguest"

var built = sync.OnceValues(func() ([]byte, error) {
	dir, err := os.MkdirTemp("", "kite-guest-")
	if err != nil {
		return nil, err
	}
	defer func() { _ = os.RemoveAll(dir) }()
	out := filepath.Join(dir, "plugin.wasm")
	cmd := exec.Command("go", "build", "-buildmode=c-shared", "-o", out, Guest)
	cmd.Env = append(os.Environ(), "GOOS=wasip1", "GOARCH=wasm")
	if msg, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("build %s: %w\n%s", Guest, err, msg)
	}
	return os.ReadFile(out)
})

// Wasm returns testguest built for WebAssembly, once a test binary. It
// exports every hook a plugin can, and takes two settings: sign, which it
// writes into the pages it sees, and mode, which makes it misbehave.
func Wasm(t testing.TB) []byte {
	t.Helper()
	wasm, err := built()
	if err != nil {
		t.Fatal(err)
	}
	return wasm
}

// Manifest is a plugin.yaml for testguest as id, declaring hooks.
func Manifest(id string, hooks ...string) string {
	return fmt.Sprintf(`id: %s
name: Guest
version: 1.0.0
apiVersion: kite/plugin/v1
hooks: [%s]
settings:
  - key: sign
    type: string
    default: kite
  - key: mode
    type: string
`, id, strings.Join(hooks, ", "))
}
