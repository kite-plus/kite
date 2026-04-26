package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/kite-plus/kite/internal/version"
)

// HTTP headers exposed by [APIVersion] so clients can pin / feature-detect.
const (
	HeaderAPIVersion    = "X-Kite-API-Version"    // wire-format version, bumped on breaking changes
	HeaderServerVersion = "X-Kite-Server-Version" // build identity (release tag or git sha)
)

// APIVersion stamps every response with the wire-format version and the
// running build identity. The header pair lets clients:
//
//   - Refuse to talk to a server with an incompatible API version (without
//     parsing every response body to detect drift).
//   - Surface "your client is out of date" notices when the server is newer
//     than what the client knows about.
//   - Correlate bug reports with a specific commit even when the user
//     installed via Docker tag rather than a release.
//
// The values are read from [version.APIVersion] (constant, baked at build
// time) and [version.Get] (runtime, populated from -ldflags).
func APIVersion() gin.HandlerFunc {
	build := version.Get()
	serverID := build.Version
	if build.Commit != "" && build.Commit != "unknown" {
		serverID = serverID + "+" + build.Commit
	}
	return func(c *gin.Context) {
		c.Header(HeaderAPIVersion, version.APIVersion)
		c.Header(HeaderServerVersion, serverID)
		c.Next()
	}
}
