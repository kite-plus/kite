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
)

// runKite runs a command line the way the binary would, in a project.
func runKite(t *testing.T, root string, args ...string) string {
	t.Helper()
	t.Chdir(root)
	cmd := newRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs(args)
	if err := cmd.ExecuteContext(t.Context()); err != nil {
		t.Fatalf("kite %s: %v\n%s", strings.Join(args, " "), err, out.String())
	}
	return out.String()
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

// workflow is the part of a GitHub Actions workflow these tests look at.
type workflow struct {
	On   map[string]any `yaml:"on"`
	Jobs map[string]struct {
		Needs any    `yaml:"needs"`
		If    string `yaml:"if"`
		Uses  string `yaml:"uses"`
		Steps []struct {
			Uses string            `yaml:"uses"`
			Run  string            `yaml:"run"`
			With map[string]string `yaml:"with"`
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
