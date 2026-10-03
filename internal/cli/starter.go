package cli

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/kite-plus/kite/internal/kitew"
)

// deployWorkflow is the GitHub Actions workflow `kite init` writes.
//
// It is a file in the repository rather than something Kite talks to over an
// API. Deployment belongs to the hosting platform, and a workflow the author
// can read, edit and delete is the honest shape for that: changing host means
// editing this file, not migrating a CMS.
//
// It builds with kitew, which runs the release kite.lock pins: the one that
// wrote the project, until the author moves the pin. A workflow that floated
// to the newest release would rebuild the same commit differently later,
// which is the thing reproducible builds exist to prevent.
//
// Its last job pings the update services the site lists once the deploy is
// live. It is its own job, allowed to fail, so a service that is down never
// fails a deployment.
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
    outputs:
      base_url: ${{ steps.pages.outputs.base_url }}
    steps:
      - uses: actions/checkout@v7

      # Where Pages publishes the site: under the repository's name for a
      # project site, or at its own domain.
      - id: pages
        uses: actions/configure-pages@v6

      # Built for that address rather than the one kite.yaml names, which
      # may still be the one the site was previewed at, and with the Kite
      # release kite.lock pins, which kitew downloads and checks first, so
      # this commit builds the same way in a year as it does today. --verify
      # builds twice and compares every byte, so a site that would deploy
      # differently on a second run fails here instead.
      - name: Build
        shell: bash # with pipefail, so a failed build is not hidden by tee
        env:
          KITE_SITE_BASEURL: ${{ steps.pages.outputs.base_url }}
        run: sh ./kitew build --verify --json | tee build.json

      # Tells scheduled.yml when there is next something to publish.
      - name: Record the next scheduled post
        run: jq -r '.next_due // "none"' build.json > .kite-next-due
      - uses: actions/cache/save@v6
        with:
          path: .kite-next-due
          key: kite-next-due-${{ github.run_id }}-${{ github.run_attempt }}

      - uses: actions/upload-pages-artifact@v5
        with:
          path: public
          # A site's static/.well-known is published like any other file.
          include-hidden-files: true

  deploy:
    needs: build
    runs-on: ubuntu-latest
    environment:
      name: github-pages
      url: ${{ steps.deployment.outputs.page_url }}
    steps:
      - id: deployment
        uses: actions/deploy-pages@v5

  # Tells the update services publish.ping in kite.yaml lists, such as
  # Explore, that the site has changed, for the address Pages published it
  # at. A site that lists none pings nothing, and a ping that fails is
  # reported here without failing the deployment.
  ping:
    needs: [build, deploy]
    runs-on: ubuntu-latest
    continue-on-error: true
    permissions:
      contents: read
    steps:
      - uses: actions/checkout@v7
      - name: Ping
        env:
          KITE_SITE_BASEURL: ${{ needs.build.outputs.base_url }}
        run: sh ./kitew ping
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
      - uses: actions/cache/restore@v6
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
	WorkflowPath = kitew.Workflow
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
	fill := strings.NewReplacer("@@BRANCH@@", branch)

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
