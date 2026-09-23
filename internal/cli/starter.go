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
  # scheduled.yml calls this once a scheduled post has fallen due.
  workflow_call:

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
        shell: bash # with pipefail, so a failed build is not hidden by tee
        run: kite build --verify --json | tee build.json

      # Tells scheduled.yml when there is next something to publish.
      - name: Record the next scheduled post
        run: jq -r '.next_due // "none"' build.json > .kite-next-due
      - uses: actions/cache/save@v4
        with:
          path: .kite-next-due
          key: kite-next-due-${{ github.run_id }}-${{ github.run_attempt }}

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

// scheduledWorkflow publishes scheduled posts once their time has come.
//
// It is a workflow of its own because GitHub turns off a workflow with a
// schedule in a public repository that has had no commits for 60 days, and
// turns it off for every trigger. A schedule inside the deploy workflow would
// stop a quiet blog from deploying even when it next pushes a post.
const scheduledWorkflow = `# Publishes scheduled posts once their time has come.
#
# Every hour it reads when the next scheduled post falls due, as the last
# build recorded it, and runs the deploy workflow only once that time has
# passed. A run with nothing due ends after that check.
#
# GitHub can delay these runs, and turns them off in a public repository with
# no commits for 60 days; turn them back on under the Actions tab. Deploying
# on push is a separate workflow and keeps working either way.
name: Publish scheduled posts

on:
  schedule:
    - cron: "17 * * * *"

permissions: {}

jobs:
  due:
    runs-on: ubuntu-latest
    outputs:
      build: ${{ steps.check.outputs.build }}
    steps:
      - uses: actions/cache/restore@v4
        with:
          path: .kite-next-due
          key: kite-next-due
          restore-keys: kite-next-due-

      # With no record, or one that cannot be read, it builds to find out.
      - id: check
        run: |
          due=$(cat .kite-next-due 2>/dev/null || echo unknown)
          echo "next scheduled post: $due"
          case "$due" in
            none) build=false ;;
            unknown) build=true ;;
            *)
              at=$(date -u -d "$due" +%s 2>/dev/null) || at=0
              if [ "$(date -u +%s)" -ge "$at" ]; then build=true; else build=false; fi
              ;;
          esac
          echo "build=$build" >> "$GITHUB_OUTPUT"

  deploy:
    needs: due
    if: needs.due.outputs.build == 'true'
    permissions:
      contents: read
      pages: write
      id-token: write
    uses: ./.github/workflows/deploy.yml
`

// WorkflowPath is where the deploy workflow lives, and SchedulePath the one
// that publishes scheduled posts.
var (
	WorkflowPath = filepath.Join(".github", "workflows", "deploy.yml")
	SchedulePath = filepath.Join(".github", "workflows", "scheduled.yml")
)

// writeWorkflow puts the workflows in place and returns what it wrote.
//
// An existing deploy workflow is left alone, and the scheduled one with it:
// the file belongs to the repository, an author may have edited it, and the
// scheduled workflow only works with a deploy workflow it can call.
func writeWorkflow(root, branch string) ([]string, error) {
	if _, err := os.Stat(filepath.Join(root, WorkflowPath)); err == nil {
		return nil, nil
	}
	if branch == "" {
		branch = "main"
	}
	fill := strings.NewReplacer(
		"@@BRANCH@@", branch,
		"@@GO@@", buildinfo.GoVersion(),
		"@@VERSION@@", installVersion(),
	)

	var written []string
	for _, w := range []struct{ path, body string }{
		{WorkflowPath, deployWorkflow},
		{SchedulePath, scheduledWorkflow},
	} {
		target := filepath.Join(root, w.path)
		if _, err := os.Stat(target); err == nil {
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return written, err
		}
		if err := os.WriteFile(target, []byte(fill.Replace(w.body)), 0o644); err != nil {
			return written, err
		}
		written = append(written, w.path)
	}
	return written, nil
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
