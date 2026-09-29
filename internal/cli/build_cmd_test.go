package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/kite-plus/kite/internal/project"
)

// runKite runs a command line the way the binary would, in a project.
func runKite(t *testing.T, root string, args ...string) string {
	t.Helper()
	out, err := tryKite(t, root, args...)
	if err != nil {
		t.Fatalf("kite %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return out
}

// tryKite is runKite for a command line that may fail.
func tryKite(t *testing.T, root string, args ...string) (string, error) {
	t.Helper()
	t.Chdir(root)
	cmd := newRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs(args)
	err := cmd.ExecuteContext(t.Context())
	return out.String(), err
}

// setOutput writes build.output into a project kite init wrote.
func setOutput(t *testing.T, root, output string) {
	t.Helper()
	file := filepath.Join(root, project.ConfigName)
	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	edited := strings.Replace(string(data), "output: public", "output: '"+output+"'", 1)
	if edited == string(data) {
		t.Fatalf("%s names no output:\n%s", project.ConfigName, data)
	}
	if err := os.WriteFile(file, []byte(edited), 0o644); err != nil {
		t.Fatal(err)
	}
}

func newSite(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if _, err := create(t.Context(), plan{Root: root, Title: "Due", BaseURL: "https://example.com", Language: "en"}); err != nil {
		t.Fatalf("create: %v", err)
	}
	return root
}

func writePost(t *testing.T, root, id, slug, front string) {
	t.Helper()
	dir := filepath.Join(root, "content", "posts", slug)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	body := "---\nid: " + id + "\ntitle: " + slug + "\nslug: " + slug + "\n" + front + "---\n\nText.\n"
	if err := os.WriteFile(filepath.Join(dir, "index.md"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// The deploy workflow only builds on a schedule once a post has fallen due,
// and it learns when that is from what the last build reported.
func TestBuildReportsWhenTheNextScheduledPostIsDue(t *testing.T) {
	root := newSite(t)

	var report buildReport
	if err := json.Unmarshal([]byte(runKite(t, root, "build", "--json")), &report); err != nil {
		t.Fatal(err)
	}
	if report.NextDue != "" {
		t.Errorf("next_due = %q with nothing scheduled", report.NextDue)
	}

	writePost(t, root, "01J8KQ2P3R4S5T6V7W8X9YZ001", "later", "status: scheduled\npublished_at: 2099-06-01T08:00:00Z\n")
	writePost(t, root, "01J8KQ2P3R4S5T6V7W8X9YZ002", "soon", "status: scheduled\npublished_at: 2099-01-01T08:00:00+08:00\n")

	report = buildReport{}
	if err := json.Unmarshal([]byte(runKite(t, root, "build", "--json")), &report); err != nil {
		t.Fatal(err)
	}
	if want := "2099-01-01T00:00:00Z"; report.NextDue != want {
		t.Errorf("next_due = %q, want the earlier post, in UTC: %q", report.NextDue, want)
	}

	if out := runKite(t, root, "build"); !strings.Contains(out, "next scheduled post is due 2099-01-01T00:00:00Z") {
		t.Errorf("the build does not say when to build again:\n%s", out)
	}
}

// entries lists what a directory holds, leaving out the .kite a build keeps
// its records in.
func entries(t *testing.T, dir string) []string {
	t.Helper()
	list, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range list {
		if e.Name() != ".kite" {
			names = append(names, e.Name())
		}
	}
	return names
}

// An absolute output is where the site goes, whichever of kite.yaml,
// KITE_BUILD_OUTPUT and --output gives it. Joined to the project root, it
// would land in a copy of its own path inside the project.
func TestAnAbsoluteOutputTakesTheSiteOutOfTheProject(t *testing.T) {
	for _, via := range []string{"kite.yaml", "KITE_BUILD_OUTPUT", "--output"} {
		t.Run(via, func(t *testing.T) {
			t.Setenv("KITE_BUILD_OUTPUT", "")
			root := newSite(t)
			elsewhere := filepath.Join(t.TempDir(), "site")

			args := []string{"build", "--json"}
			switch via {
			case "kite.yaml":
				setOutput(t, root, elsewhere)
			case "KITE_BUILD_OUTPUT":
				t.Setenv("KITE_BUILD_OUTPUT", elsewhere)
			case "--output":
				args = append(args, "--output", elsewhere)
			}
			before := entries(t, root)

			var report buildReport
			if err := json.Unmarshal([]byte(runKite(t, root, args...)), &report); err != nil {
				t.Fatal(err)
			}
			if report.Output != elsewhere {
				t.Errorf("output = %q, want %q", report.Output, elsewhere)
			}
			if _, err := os.Stat(filepath.Join(elsewhere, "index.html")); err != nil {
				t.Errorf("the site is not where it was sent: %v", err)
			}
			if after := entries(t, root); !slices.Equal(after, before) {
				t.Errorf("the build added to the project: %v, was %v", after, before)
			}
		})
	}
}

// A build replaces its output whole, so one sent to the project itself is
// refused, whichever of --output, kite.yaml and KITE_BUILD_OUTPUT sends it,
// and the project is left as it was.
func TestABuildIntoTheProjectIsRefused(t *testing.T) {
	for _, via := range []string{"--output", "kite.yaml", "KITE_BUILD_OUTPUT"} {
		t.Run(via, func(t *testing.T) {
			t.Setenv("KITE_BUILD_OUTPUT", "")
			root := newSite(t)

			args := []string{"build"}
			switch via {
			case "--output":
				args = append(args, "--output", ".")
			case "kite.yaml":
				setOutput(t, root, ".")
			case "KITE_BUILD_OUTPUT":
				t.Setenv("KITE_BUILD_OUTPUT", root)
			}
			before := entries(t, root)

			out, err := tryKite(t, root, args...)
			if err == nil || !strings.Contains(err.Error(), "would replace the project") {
				t.Fatalf("kite build = %v, want a refusal\n%s", err, out)
			}
			if after := entries(t, root); !slices.Equal(after, before) {
				t.Errorf("the project went from %v to %v", before, after)
			}
		})
	}
}

// workflow is the part of a GitHub Actions workflow these tests look at.
type workflow struct {
	On   map[string]any `yaml:"on"`
	Jobs map[string]struct {
		Needs any    `yaml:"needs"`
		If    string `yaml:"if"`
		Uses  string `yaml:"uses"`
		Steps []struct {
			ID   string            `yaml:"id"`
			Uses string            `yaml:"uses"`
			Run  string            `yaml:"run"`
			With map[string]string `yaml:"with"`
			Env  map[string]string `yaml:"env"`
		} `yaml:"steps"`
	} `yaml:"jobs"`
}

func readWorkflows(t *testing.T) (deploy, scheduled workflow) {
	t.Helper()
	root := t.TempDir()
	written, err := writeWorkflow(root, "main")
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(written, []string{WorkflowPath, SchedulePath}) {
		t.Fatalf("wrote %v", written)
	}
	for path, into := range map[string]*workflow{WorkflowPath: &deploy, SchedulePath: &scheduled} {
		data, err := os.ReadFile(filepath.Join(root, path))
		if err != nil {
			t.Fatal(err)
		}
		if err := yaml.Unmarshal(data, into); err != nil {
			t.Fatalf("%s is not YAML: %v", path, err)
		}
	}
	return deploy, scheduled
}

// GitHub turns a workflow with a schedule off, for every trigger, in a public
// repository with no commits for 60 days. A schedule in the deploy workflow
// would leave a quiet blog that pushes a new post not deploying at all.
func TestTheScheduleNeverSitsInTheDeployWorkflow(t *testing.T) {
	deploy, scheduled := readWorkflows(t)
	if _, ok := deploy.On["schedule"]; ok {
		t.Error("the deploy workflow has a schedule")
	}
	if _, ok := deploy.On["workflow_call"]; !ok {
		t.Error("the deploy workflow cannot be called by the scheduled one")
	}
	if _, ok := scheduled.On["schedule"]; !ok || len(scheduled.On) != 1 {
		t.Errorf("the scheduled workflow should run on a schedule only: %v", scheduled.On)
	}
}

// The scheduled workflow reads the report by the name of one field. Renaming
// that field would leave every scheduled run finding nothing due, with
// nothing failing to say so.
func TestTheScheduledWorkflowReadsTheDueTimeTheBuildReports(t *testing.T) {
	field, ok := reflect.TypeFor[buildReport]().FieldByName("NextDue")
	if !ok {
		t.Fatal("buildReport has no NextDue")
	}
	key, _, _ := strings.Cut(field.Tag.Get("json"), ",")

	deploy, scheduled := readWorkflows(t)

	var reads, saves, restores bool
	for _, s := range deploy.Jobs["build"].Steps {
		reads = reads || strings.Contains(s.Run, "."+key+" ")
		saves = saves || strings.HasPrefix(s.Uses, "actions/cache/save@") && s.With["path"] == ".kite-next-due"
	}
	for _, s := range scheduled.Jobs["due"].Steps {
		restores = restores || strings.HasPrefix(s.Uses, "actions/cache/restore@") && s.With["path"] == ".kite-next-due"
	}
	if !reads {
		t.Errorf("no build step reads .%s from the report", key)
	}
	if !saves || !restores {
		t.Errorf("the due time is not carried between runs: saved %v, restored %v", saves, restores)
	}

	call := scheduled.Jobs["deploy"]
	if call.Uses != "./"+filepath.ToSlash(WorkflowPath) {
		t.Errorf("the scheduled workflow calls %q", call.Uses)
	}
	if call.Needs != "due" || !strings.Contains(call.If, "needs.due.outputs.build") {
		t.Errorf("deploying is not gated on the due check: needs %v, if %q", call.Needs, call.If)
	}
}

// A project site is published under the repository's name, and a site built
// for the address kite.yaml still holds, often the one it was previewed at,
// would link every page to the wrong place. The workflow builds for the
// address Pages reports instead.
func TestTheDeployWorkflowBuildsForWherePagesPublishes(t *testing.T) {
	deploy, _ := readWorkflows(t)
	configured, built := -1, -1
	for i, s := range deploy.Jobs["build"].Steps {
		if strings.HasPrefix(s.Uses, "actions/configure-pages@") && s.ID == "pages" {
			configured = i
		}
		if strings.Contains(s.Run, "kite build") {
			built = i
			if got := s.Env["KITE_SITE_BASEURL"]; got != "${{ steps.pages.outputs.base_url }}" {
				t.Errorf("the build runs with KITE_SITE_BASEURL = %q", got)
			}
		}
	}
	if configured < 0 || built < 0 || configured > built {
		t.Errorf("configure-pages is step %d and the build step %d; the address has to be known first", configured, built)
	}
}

// The deploy workflow uploads one directory, and a project kite init wrote
// builds into it without being told where.
func TestTheDeployWorkflowUploadsWhereANewSiteBuilds(t *testing.T) {
	deploy, _ := readWorkflows(t)
	uploaded := ""
	for _, s := range deploy.Jobs["build"].Steps {
		if strings.HasPrefix(s.Uses, "actions/upload-pages-artifact@") {
			uploaded = s.With["path"]
		}
	}
	if uploaded == "" {
		t.Fatal("the deploy workflow uploads nothing")
	}

	t.Setenv("KITE_BUILD_OUTPUT", "")
	root := newSite(t)
	runKite(t, root, "build")
	if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(uploaded), "index.html")); err != nil {
		t.Errorf("a new site does not build where the workflow uploads, %s: %v", uploaded, err)
	}
}

// The workflow installs the release that wrote it, so the site builds the
// same way in a year. A release stamps its version without the v its tag
// has, and a build that is no release installs the latest one.
func TestTheWorkflowInstallsTheReleaseThatWroteIt(t *testing.T) {
	for v, want := range map[string]string{
		"0.1.0":              "v0.1.0",
		"v0.1.0":             "v0.1.0",
		"v0.2.0-rc.1":        "v0.2.0-rc.1",
		"v0.1.0-3-gabc1234":  "latest",
		"v0.1.0-dirty":       "latest",
		"26200b0":            "latest",
		"577feeb-dirty":      "latest",
		"0.1.1-snapshot-abc": "latest",
		"dev":                "latest",
	} {
		if got := pinnedVersion(v); got != want {
			t.Errorf("pinnedVersion(%q) = %q, want %q", v, got, want)
		}
	}
}

// An author's own deploy workflow is theirs, and a scheduled workflow that
// calls into it would fail on a file that cannot be called.
func TestAnExistingDeployWorkflowIsLeftAloneWithNoScheduleBesideIt(t *testing.T) {
	root := t.TempDir()
	own := filepath.Join(root, WorkflowPath)
	if err := os.MkdirAll(filepath.Dir(own), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(own, []byte("name: Mine\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	written, err := writeWorkflow(root, "main")
	if err != nil {
		t.Fatal(err)
	}
	if len(written) != 0 {
		t.Errorf("wrote %v beside an existing deploy workflow", written)
	}
	if data, _ := os.ReadFile(own); string(data) != "name: Mine\n" {
		t.Errorf("the existing workflow was changed:\n%s", data)
	}
}
