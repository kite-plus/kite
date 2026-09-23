package git

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/kite-plus/kite/internal/publish"
)

// fakeGitHub answers the two deployment endpoints the checker uses.
type fakeGitHub struct {
	mu          sync.Mutex
	deployments map[string][]ghDeployment // by owner/name, newest first
	statuses    map[int64][]ghStatus      // newest first
	private     bool
	remaining   int
	requests    int
}

func (f *fakeGitHub) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.requests++
	w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(f.remaining))
	w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(time.Hour).Unix(), 10))
	if f.private {
		http.NotFound(w, r)
		return
	}

	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	switch {
	case len(parts) == 4 && parts[3] == "deployments":
		if r.URL.Query().Get("environment") != pagesEnvironment {
			http.Error(w, "unexpected environment", http.StatusBadRequest)
			return
		}
		var out []ghDeployment
		for _, d := range f.deployments[parts[1]+"/"+parts[2]] {
			if sha := r.URL.Query().Get("sha"); sha == "" || sha == d.SHA {
				out = append(out, d)
			}
		}
		_ = json.NewEncoder(w).Encode(append([]ghDeployment{}, out...))
	case len(parts) == 6 && parts[5] == "statuses":
		id, _ := strconv.ParseInt(parts[4], 10, 64)
		_ = json.NewEncoder(w).Encode(append([]ghStatus{}, f.statuses[id]...))
	default:
		http.NotFound(w, r)
	}
}

func newFake(t *testing.T) (*fakeGitHub, *deployChecker) {
	t.Helper()
	fake := &fakeGitHub{
		deployments: map[string][]ghDeployment{},
		statuses:    map[int64][]ghStatus{},
		remaining:   60,
	}
	srv := httptest.NewServer(fake)
	t.Cleanup(srv.Close)
	return fake, newDeployChecker(srv.URL, time.Now)
}

func never(string) bool { return false }

func TestADeploymentIsReadFromItsNewestStatus(t *testing.T) {
	for _, tc := range []struct {
		state string
		want  publish.Step
	}{
		{"success", publish.StepDone},
		{"inactive", publish.StepDone},
		{"in_progress", publish.StepPending},
		{"queued", publish.StepPending},
		{"waiting", publish.StepPending},
		{"failure", publish.StepFailed},
		{"error", publish.StepFailed},
	} {
		t.Run(tc.state, func(t *testing.T) {
			fake, d := newFake(t)
			fake.deployments["acme/site"] = []ghDeployment{{ID: 7, SHA: "abc"}}
			fake.statuses[7] = []ghStatus{
				{ID: 2, State: tc.state, EnvironmentURL: "https://acme.github.io/site/"},
				{ID: 1, State: "queued"},
			}
			step, _, err := d.ask(t.Context(), "acme/site", "abc", never)
			if err != nil {
				t.Fatal(err)
			}
			if step != tc.want {
				t.Errorf("step = %q, want %q", step, tc.want)
			}
		})
	}
}

func TestTheLiveAddressComesFromTheDeployment(t *testing.T) {
	fake, d := newFake(t)
	fake.deployments["acme/site"] = []ghDeployment{{ID: 7, SHA: "abc"}}
	fake.statuses[7] = []ghStatus{{ID: 1, State: "success", EnvironmentURL: "https://acme.github.io/site/"}}

	_, link, err := d.ask(t.Context(), "acme/site", "abc", never)
	if err != nil || link != "https://acme.github.io/site/" {
		t.Errorf("url = %q, %v", link, err)
	}
}

// A later commit's deployment replaces an earlier one that was still waiting,
// and it deploys the earlier commit's content along with its own.
func TestANewerDeploymentThatContainsTheCommitCounts(t *testing.T) {
	fake, d := newFake(t)
	fake.deployments["acme/site"] = []ghDeployment{{ID: 9, SHA: "later"}, {ID: 8, SHA: "older"}}
	fake.statuses[9] = []ghStatus{{ID: 1, State: "success"}}

	contains := func(sha string) bool { return sha == "later" }
	if step, _, _ := d.ask(t.Context(), "acme/site", "mine", contains); step != publish.StepDone {
		t.Errorf("step = %q, want done", step)
	}
	if step, _, _ := d.ask(t.Context(), "acme/site", "mine", never); step != publish.StepPending {
		t.Errorf("a deployment that does not contain the commit counted: %q", step)
	}
}

func TestARepositoryThatDoesNotReportDeploymentsIsNotApplicable(t *testing.T) {
	t.Run("never deployed to Pages", func(t *testing.T) {
		_, d := newFake(t)
		if step, _, err := d.ask(t.Context(), "acme/site", "abc", never); err != nil || step != publish.StepNotApplicable {
			t.Errorf("step = %q, %v", step, err)
		}
	})
	t.Run("private", func(t *testing.T) {
		fake, d := newFake(t)
		fake.private = true
		if step, _, err := d.ask(t.Context(), "acme/site", "abc", never); err != nil || step != publish.StepNotApplicable {
			t.Errorf("step = %q, %v", step, err)
		}
	})
}

// An anonymous caller has sixty requests an hour, shared with everything else
// on the machine, so the checker stops well before they run out.
func TestTheCheckerStopsAskingWhenTheAllowanceRunsLow(t *testing.T) {
	fake, d := newFake(t)
	fake.remaining = spare
	fake.deployments["acme/site"] = []ghDeployment{{ID: 7, SHA: "abc"}}

	// The first answer says the allowance is nearly spent, so the status it
	// would have gone on to ask for is not asked for.
	if _, _, err := d.ask(t.Context(), "acme/site", "abc", never); !errors.Is(err, errQuiet) {
		t.Errorf("err = %v, want errQuiet", err)
	}
	if _, _, err := d.ask(t.Context(), "acme/site", "abc", never); !errors.Is(err, errQuiet) {
		t.Errorf("err = %v, want errQuiet", err)
	}
	if fake.requests != 1 {
		t.Errorf("%d requests were made, want only the one that reported the allowance", fake.requests)
	}
}

// look answers from what it knows without waiting, and asks in the
// background; once a deployment is live it is not asked about again.
func TestLookNeverWaitsAndStopsAskingOnceLive(t *testing.T) {
	fake, d := newFake(t)
	fake.deployments["acme/site"] = []ghDeployment{{ID: 7, SHA: "abc"}}
	fake.statuses[7] = []ghStatus{{ID: 1, State: "success"}}

	if step, _ := d.look("acme/site", "abc", true, never); step != publish.StepPending {
		t.Errorf("first look = %q, want pending until GitHub has answered", step)
	}
	deadline := time.Now().Add(5 * time.Second)
	for {
		step, _ := d.look("acme/site", "abc", true, never)
		if step == publish.StepDone {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("still %q after five seconds", step)
		}
		time.Sleep(10 * time.Millisecond)
	}

	fake.mu.Lock()
	settled := fake.requests
	fake.mu.Unlock()
	for range 5 {
		d.look("acme/site", "abc", true, never)
	}
	time.Sleep(50 * time.Millisecond)
	fake.mu.Lock()
	defer fake.mu.Unlock()
	if fake.requests != settled {
		t.Errorf("a finished deployment was asked about %d more times", fake.requests-settled)
	}
}

func TestGitHubRemotesAreRecognizedInEveryForm(t *testing.T) {
	for remote, want := range map[string]string{
		"https://github.com/acme/site.git":         "acme/site",
		"https://github.com/acme/site":             "acme/site",
		"https://token@github.com/acme/site.git":   "acme/site",
		"git@github.com:acme/site.git":             "acme/site",
		"ssh://git@github.com/acme/site.git":       "acme/site",
		"ssh://git@ssh.github.com:443/acme/site":   "acme/site",
		"https://GitHub.com/Acme/Site.git":         "Acme/Site",
		"https://gitlab.com/acme/site.git":         "",
		"git@gitlab.com:acme/site.git":             "",
		"/srv/git/site.git":                        "",
		"https://github.com/acme":                  "",
		"https://github.com/acme/site/extra/parts": "",
	} {
		got, ok := githubRepo(remote)
		if (want == "") == ok || got != want {
			t.Errorf("githubRepo(%q) = %q, %v; want %q", remote, got, ok, want)
		}
	}
}
