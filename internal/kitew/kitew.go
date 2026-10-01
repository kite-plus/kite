// Package kitew pins the Kite release a project builds with and writes the
// scripts that run it: kitew for Linux and macOS, kitew.ps1 for Windows.
//
// The scripts read the pin from kite.lock, download the release the first
// time, check it against the release's checksums.txt, whose own sha256 the
// pin records, and run it, so that the author's machine and the deploy build
// the same site with the same Kite (docs/design/architecture.md 17). The
// archive names and checksums.txt they download are the contract
// .goreleaser.yaml keeps.
package kitew

import (
	"bytes"
	"context"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/kite-plus/kite/internal/buildinfo"
	"github.com/kite-plus/kite/internal/lock"
)

//go:embed kitew kitew.ps1
var scripts embed.FS

// Scripts are the files Write puts at the root of a project.
var Scripts = []string{"kitew", "kitew.ps1"}

// Releases is where releases are downloaded from, unless KITE_DOWNLOAD_URL
// names a mirror, as it does for the scripts.
const Releases = "https://github.com/kite-plus/kite/releases/download"

// Base is where releases are downloaded from.
func Base() string {
	if base := os.Getenv("KITE_DOWNLOAD_URL"); base != "" {
		return strings.TrimRight(base, "/")
	}
	return Releases
}

// release matches a version a tag names: the version a release stamps, as
// 0.1.0, or what git describe says at a tag, as v0.1.0 or v0.2.0-rc.1.
// Commits past a tag, as v0.1.0-3-gabc1234, and a dirty tree name no tag.
var release = regexp.MustCompile(`^v?\d+\.\d+\.\d+(-[0-9A-Za-z.]+)?$`)

// Release is the release a version names, without its v, and false for a
// build from source, which no release stands for.
func Release(v string) (string, bool) {
	if !release.MatchString(v) || strings.HasSuffix(v, "-dirty") {
		return "", false
	}
	return strings.TrimPrefix(v, "v"), true
}

// Write puts the scripts at the root of a project, kitew executable, and
// returns those it changed.
func Write(root string) ([]string, error) {
	var changed []string
	for _, name := range Scripts {
		data, err := scripts.ReadFile(name)
		if err != nil {
			return changed, err
		}
		mode := fs.FileMode(0o644)
		if name == "kitew" {
			mode = 0o755
		}
		target := filepath.Join(root, name)
		if old, err := os.ReadFile(target); err == nil && bytes.Equal(old, data) {
			info, err := os.Stat(target)
			if err != nil {
				return changed, err
			}
			if runtime.GOOS == "windows" || info.Mode().Perm() == mode {
				continue
			}
		} else if err := os.WriteFile(target, data, mode); err != nil {
			return changed, err
		}
		// WriteFile keeps the mode of a file that was there.
		if err := os.Chmod(target, mode); err != nil {
			return changed, err
		}
		changed = append(changed, name)
	}
	return changed, nil
}

// Installed reports whether a project has the scripts.
func Installed(root string) bool {
	for _, name := range Scripts {
		if _, err := os.Stat(filepath.Join(root, name)); err != nil {
			return false
		}
	}
	return true
}

// Workflow is the deploy workflow kite init writes.
var Workflow = filepath.Join(".github", "workflows", "deploy.yml")

// Deploy says how a project's deploy workflow gets Kite: "kitew", "self"
// when it installs a Kite of its own, as those written before kitew did, or
// "" when there is no deploy workflow.
func Deploy(root string) string {
	data, err := os.ReadFile(filepath.Join(root, Workflow))
	switch {
	case err != nil:
		return ""
	case bytes.Contains(data, []byte("kitew")):
		return "kitew"
	}
	return "self"
}

// ErrNoRelease is reported, wrapped, for a version that was never released.
var ErrNoRelease = errors.New("no such release")

// Pin makes the pin of a release: its version, and the sha256 of the
// checksums.txt it is downloaded with. It fails with ErrNoRelease when there
// is no such release; when the list cannot be fetched for any other reason,
// the pin it returns has the version alone, and the error says why.
func Pin(ctx context.Context, client *http.Client, version string) (lock.Pin, error) {
	pin := lock.Pin{Version: version}
	addr := Base() + "/v" + version + "/checksums.txt"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, addr, nil)
	if err != nil {
		return pin, err
	}
	req.Header.Set("User-Agent", "kite/"+buildinfo.Version)
	resp, err := client.Do(req)
	if err != nil {
		return pin, err
	}
	defer func() { _ = resp.Body.Close() }()
	switch {
	case resp.StatusCode == http.StatusNotFound:
		return pin, fmt.Errorf("%w: Kite %s was never released", ErrNoRelease, version)
	case resp.StatusCode != http.StatusOK:
		return pin, fmt.Errorf("%s answered %s", addr, resp.Status)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return pin, err
	}
	if !bytes.Contains(data, []byte("  kite_"+version+"_")) {
		return pin, fmt.Errorf("%s lists no archive of Kite %s", addr, version)
	}
	sum := sha256.Sum256(data)
	pin.Checksums = "sha256:" + hex.EncodeToString(sum[:])
	return pin, nil
}
