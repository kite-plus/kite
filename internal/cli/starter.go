package cli

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/kite-plus/kite/internal/buildinfo"
)

// deployWorkflow is the GitHub Actions workflow `kite init` writes.
//
// It is a file in the repository rather than something Kite talks to over an
// API. Deployment belongs to the hosting platform, and a workflow the author
// can read, edit and delete is the honest shape for that: changing host means
// editing this file, not migrating a CMS.
//
// The version is pinned to the one that wrote the file. A workflow that
// floats to the newest release would rebuild the same commit differently
// later, which is the thing reproducible builds exist to prevent.
const deployWorkflow = `# Builds the site and publishes it to GitHub Pages.
#
# Turn Pages on first: Settings -> Pages -> Source -> GitHub Actions.
# Until then this workflow builds and then fails to deploy.
name: Deploy

on:
  push:
    branches: [@@BRANCH@@]
  workflow_dispatch:

permissions:
  contents: read
  pages: write
  id-token: write

# One deployment at a time. A newer push waits rather than canceling the one
# in flight, because a half-finished deployment is worse than a slow one.
concurrency:
  group: pages
  cancel-in-progress: false

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version: "@@GO@@"

      # Pinned to the version that wrote this file, so this commit builds the
      # same way in a year as it does today. Raise it deliberately.
      - name: Install Kite
        run: go install github.com/kite-plus/kite/cmd/kite@@@VERSION@@

      # --verify builds twice and compares every byte, so a site that would
      # deploy differently on a second run fails here instead.
      - name: Build
        run: kite build --verify

      - uses: actions/configure-pages@v5
      - uses: actions/upload-pages-artifact@v3
        with:
          path: public

  deploy:
    needs: build
    runs-on: ubuntu-latest
    environment:
      name: github-pages
      url: ${{ steps.deployment.outputs.page_url }}
    steps:
      - id: deployment
        uses: actions/deploy-pages@v4
`

// WorkflowPath is where the deploy workflow lives.
var WorkflowPath = filepath.Join(".github", "workflows", "deploy.yml")

// writeWorkflow puts the deploy workflow in place, leaving an existing one
// alone: it belongs to the repository, and an author may have edited it.
func writeWorkflow(root, branch string) (string, error) {
	target := filepath.Join(root, WorkflowPath)
	if _, err := os.Stat(target); err == nil {
		return "", nil
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return "", err
	}

	if branch == "" {
		branch = "main"
	}
	body := strings.NewReplacer(
		"@@BRANCH@@", branch,
		"@@GO@@", buildinfo.GoVersion(),
		"@@VERSION@@", installVersion(),
	).Replace(deployWorkflow)

	if err := os.WriteFile(target, []byte(body), 0o644); err != nil {
		return "", err
	}
	return WorkflowPath, nil
}

// installVersion is what `go install` should be pinned to.
//
// A released build names its own tag. A build from source has no tag anyone
// else can fetch, so it names the default branch and the workflow says what
// that means, rather than pinning to a version that does not exist.
func installVersion() string {
	v := buildinfo.Version
	if v == "" || v == "dev" || strings.Contains(v, "dirty") || !strings.HasPrefix(v, "v") {
		return "latest"
	}
	return v
}
