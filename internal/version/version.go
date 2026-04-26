// Package version exposes the build-time identity of the running binary.
//
// The vars below are overridable through `-ldflags "-X ..."` at link time so
// the Makefile, Dockerfile and CI pipeline can stamp the current git SHA and
// release tag into the binary without touching the source tree. When they
// aren't set (e.g. a raw `go run`), we fall back to the sentinel "dev" so the
// /health endpoint is always answerable.
package version

import (
	"runtime/debug"
	"strings"
	"sync"
)

// Sentinel values used when no ldflags injection happened. Consumers should
// treat the string "dev" as "this is a developer build, not a tagged release".
const (
	unknownVersion = "dev"
	unknownCommit  = "unknown"
	unknownDate    = "unknown"
)

// APIVersion is the wire-format version of the public HTTP API. Bumped when
// existing endpoints change shape in a breaking way; new endpoints alone do
// NOT cause a bump (clients can feature-detect from the OpenAPI spec). The
// number is stamped on every response via the X-Kite-API-Version header so
// clients can refuse to talk to an incompatible server early.
const APIVersion = "1"

// The following vars are overwritten at link time:
//
//	go build -ldflags "-X github.com/kite-plus/kite/internal/version.Version=v1.0.0 \
//	                   -X github.com/kite-plus/kite/internal/version.Commit=$(git rev-parse HEAD) \
//	                   -X github.com/kite-plus/kite/internal/version.Date=$(date -u +%FT%TZ)"
var (
	Version = unknownVersion // Semver or git tag (e.g. "v1.0.0" or "v1.0.0-5-g<sha>").
	Commit  = unknownCommit  // Short or full git SHA.
	Date    = unknownDate    // RFC3339 UTC build timestamp.
)

var (
	infoOnce sync.Once
	info     Info
)

// Info is the immutable build descriptor returned by Get. Kept small on
// purpose so it's cheap to embed in health responses and log lines.
type Info struct {
	Version string `json:"version"`
	Commit  string `json:"commit"`
	Date    string `json:"date"`
	Go      string `json:"go"`
}

// Get returns the merged build info. On the first call it consults
// runtime/debug.ReadBuildInfo to backfill any ldflags that were omitted —
// this keeps `go install` / `go run` builds from reporting "unknown" when
// the VCS metadata is already available on disk.
func Get() Info {
	infoOnce.Do(func() {
		info = Info{
			Version: strings.TrimSpace(Version),
			Commit:  strings.TrimSpace(Commit),
			Date:    strings.TrimSpace(Date),
		}
		if info.Version == "" {
			info.Version = unknownVersion
		}
		if info.Commit == "" {
			info.Commit = unknownCommit
		}
		if info.Date == "" {
			info.Date = unknownDate
		}

		if bi, ok := debug.ReadBuildInfo(); ok {
			info.Go = bi.GoVersion
			// Only fill these from debug.BuildInfo when the ldflag path
			// did not already supply a value — operator intent wins.
			if info.Commit == unknownCommit {
				for _, setting := range bi.Settings {
					if setting.Key == "vcs.revision" && setting.Value != "" {
						info.Commit = setting.Value
					}
					if setting.Key == "vcs.time" && info.Date == unknownDate && setting.Value != "" {
						info.Date = setting.Value
					}
				}
			}
		}
	})
	return info
}
