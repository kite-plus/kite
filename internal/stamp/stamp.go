// Package stamp is the small file a build leaves at the root of a site
// to say which commit it was made from.
//
// The studio reads it back from the live site to tell whether a push has
// reached readers. That works on any host, since it asks the site itself
// rather than the host, and needs neither credentials nor GitHub's allowance
// for anonymous requests.
package stamp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"strings"
	"time"
)

// File is where the stamp is written, from the root of the site.
const File = "kite-build.json"

// Stamp is what the file holds.
type Stamp struct {
	Commit string `json:"commit"`
}

// commitVars are the variables hosts set to the commit they are building,
// in the order they are trusted: GitHub Actions, Vercel, Cloudflare Pages,
// Netlify and GitLab CI. A host builds the commit it names, which a checkout
// of a shallow or detached clone may not say as plainly.
var commitVars = []string{"GITHUB_SHA", "VERCEL_GIT_COMMIT_SHA", "CF_PAGES_COMMIT_SHA", "COMMIT_REF", "CI_COMMIT_SHA"}

// Commit returns the commit a site in root is being built from: the one its
// host names in the environment, or else the HEAD of the repository root is
// in. It is empty when neither says, as for a site that is not in git.
func Commit(root string, getenv func(string) string) string {
	for _, name := range commitVars {
		if sha := strings.ToLower(strings.TrimSpace(getenv(name))); valid(sha) {
			return sha
		}
	}
	return head(root)
}

func head(root string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", "-C", root, "rev-parse", "--verify", "--quiet", "HEAD")
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0", "GIT_OPTIONAL_LOCKS=0")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	if sha := strings.TrimSpace(string(out)); valid(sha) {
		return sha
	}
	return ""
}

// valid reports whether s is a full commit id, sha1 or sha256.
func valid(s string) bool {
	if len(s) != 40 && len(s) != 64 {
		return false
	}
	for _, c := range s {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return false
		}
	}
	return true
}

// Encode returns the file's bytes for a commit.
func Encode(commit string) []byte {
	data, _ := json.Marshal(Stamp{Commit: commit})
	return append(data, '\n')
}

// errNotAStamp is what Decode says of anything else, such as the page a host
// serves in place of a file it does not have.
var errNotAStamp = errors.New("stamp: not a build stamp")

// Decode reads a stamp, refusing anything that does not name a commit.
func Decode(data []byte) (Stamp, error) {
	var s Stamp
	dec := json.NewDecoder(bytes.NewReader(data))
	if err := dec.Decode(&s); err != nil || !valid(strings.ToLower(s.Commit)) {
		return Stamp{}, errNotAStamp
	}
	s.Commit = strings.ToLower(s.Commit)
	return s, nil
}
