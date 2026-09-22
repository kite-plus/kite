package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/kite-plus/kite/internal/config"
	"github.com/kite-plus/kite/internal/project"
)

// scripted runs the wizard against a fixed set of answers.
func scripted(t *testing.T, answers string) (plan, string) {
	t.Helper()
	// A locale would otherwise decide what the language question defaults to,
	// and the answer this test checks is the default.
	for _, key := range []string{"LC_ALL", "LC_MESSAGES", "LANG"} {
		t.Setenv(key, "C")
	}

	cmd := newInitCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetIn(strings.NewReader(answers))

	// Answered by a flag, so the question is not asked: whether it would be
	// depends on whether the directory this test runs in is inside a
	// repository, which is not something the wizard should be tested on.
	if err := cmd.Flags().Set("git", "false"); err != nil {
		t.Fatal(err)
	}

	p := plan{Root: t.TempDir()}
	if err := interview(cmd, &p); err != nil {
		t.Fatalf("interview: %v", err)
	}
	p.fill()
	return p, out.String()
}

func TestTheWizardTakesTheAnswersItIsGiven(t *testing.T) {
	p, _ := scripted(t, strings.Join([]string{
		"Notes: a journal",          // what is this site called
		"https://notes.example.com", // where will it live
		"zh-CN",                     // language
		"n",                         // write a deploy workflow
		"n",                         // set a password now
	}, "\n")+"\n")

	if p.Title != "Notes: a journal" {
		t.Errorf("title = %q", p.Title)
	}
	if p.BaseURL != "https://notes.example.com" {
		t.Errorf("baseURL = %q", p.BaseURL)
	}
	if p.Language != "zh-CN" {
		t.Errorf("language = %q", p.Language)
	}
	if p.Workflow {
		t.Error("a workflow was written after being declined")
	}
	if p.Password != "" {
		t.Error("a password was set after being declined")
	}
}

// An answer nobody gave is the default, not an empty project.
func TestEmptyAnswersLeaveTheDefaults(t *testing.T) {
	p, _ := scripted(t, "\n\n\n\n\n")

	if p.Title != defaultTitle || p.BaseURL != defaultBaseURL || p.Language != defaultLanguage {
		t.Errorf("defaults not taken: %+v", p)
	}
	// The workflow question defaults to yes, and the password question to no.
	if !p.Workflow {
		t.Error("the deploy workflow was not written by default")
	}
	if p.Password != "" {
		t.Error("a password was set without being asked for")
	}
}

// A language tag that is not one produces a site claiming to be written in a
// language that does not exist, so the question is asked again.
func TestAnImpossibleLanguageIsAskedForAgain(t *testing.T) {
	p, out := scripted(t, strings.Join([]string{
		"My Site", "https://example.com",
		"not a language", // refused
		"ja",             // accepted
		"n", "n",
	}, "\n")+"\n")

	if p.Language != "ja" {
		t.Errorf("language = %q, want ja", p.Language)
	}
	if !strings.Contains(out, "is not a language tag") {
		t.Errorf("the wizard accepted a bad tag without saying so:\n%s", out)
	}
}

// The configuration file is written, not templated over: a title with a colon
// in it is an ordinary thing to want and an unparseable file if it is not
// quoted.
func TestATitleThatNeedsQuotingGetsIt(t *testing.T) {
	for _, title := range []string{
		"Notes: a journal", "#1 of many", "true", "2026", "what's here", `say "hello"`,
	} {
		body := starterConfig(plan{Title: title, BaseURL: "https://example.com", Language: "en"})

		var parsed struct {
			Site struct{ Title string } `yaml:"site"`
		}
		if err := yaml.Unmarshal([]byte(body), &parsed); err != nil {
			t.Errorf("title %q produced a file that will not parse: %v\n%s", title, err, body)
			continue
		}
		if parsed.Site.Title != title {
			t.Errorf("title %q came back as %q", title, parsed.Site.Title)
		}
	}
}

func TestCreateWritesAProjectTheRestOfKiteCanOpen(t *testing.T) {
	root := t.TempDir()
	created, err := create(t.Context(), plan{
		Root:     root,
		Title:    "Field Notes",
		BaseURL:  "https://notes.example.com",
		Language: "zh-CN",
		Workflow: true,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	for _, want := range []string{project.ConfigName, ".gitignore", WorkflowPath} {
		if !slices.Contains(created, want) {
			t.Errorf("%s was not reported as created: %v", want, created)
		}
		if _, err := os.Stat(filepath.Join(root, want)); err != nil {
			t.Errorf("%s was reported but is not there: %v", want, err)
		}
	}

	cfg, err := config.Load(root)
	if err != nil {
		t.Fatalf("the new project will not open: %v", err)
	}
	if cfg.Site.Title != "Field Notes" || cfg.Site.Language != "zh-CN" {
		t.Errorf("config = %+v", cfg.Site)
	}
	if cfg.Site.BaseURL != "https://notes.example.com" {
		t.Errorf("baseURL = %q", cfg.Site.BaseURL)
	}

	// Derived state must never be committed, whoever created the project.
	ignore, err := os.ReadFile(filepath.Join(root, ".gitignore"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(ignore), "/.kite/") {
		t.Errorf(".kite is not ignored:\n%s", ignore)
	}
}

// A container listens on every interface by necessity, and the line it prints
// is the line somebody is about to click.
func TestAWildcardAddressIsPrintedAsSomethingClickable(t *testing.T) {
	for addr, want := range map[string]string{
		"0.0.0.0:1717":   "http://localhost:1717",
		"[::]:1717":      "http://localhost:1717",
		":1717":          "http://localhost:1717",
		"127.0.0.1:1717": "http://127.0.0.1:1717",
		"example.com:80": "http://example.com:80",
	} {
		if got := displayURL(addr); got != want {
			t.Errorf("displayURL(%q) = %q, want %q", addr, got, want)
		}
	}
}
