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
		var out []ghDeployment
		for _, d := range f.deployments[parts[1]+"/"+parts[2]] {
			sha, env := r.URL.Query().Get("sha"), r.URL.Query().Get("environment")
			if (sha == "" || sha == d.SHA) && (env == "" || env == d.Environment) {
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
	return newFakeAt(t, time.Now)
}

func newFakeAt(t *testing.T, now func() time.Time) (*fakeGitHub, *deployChecker) {
	t.Helper()
	fake := &fakeGitHub{
		deployments: map[string][]ghDeployment{},
		statuses:    map[int64][]ghStatus{},
		remaining:   60,
	}
	srv := httptest.NewServer(fake)
	t.Cleanup(srv.Close)
	return fake, newDeployChecker(srv.URL, now)
}

func (f *fakeGitHub) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.requests
}

// clock is a time a test moves by hand.
type clock struct {
	mu sync.Mutex
	t  time.Time
}

func (c *clock) now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.t
}

func (c *clock) add(dt time.Duration) {
	c.mu.Lock()
	c.t = c.t.Add(dt)
	c.mu.Unlock()
}

// settle waits for every check that look started to finish.
func settle(t *testing.T, d *deployChecker) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		d.mu.Lock()
		busy := false
		for _, a := range d.answers {
			busy = busy || a.asking
		}
		for _, p := range d.reports {
			busy = busy || p.asking
		}
		d.mu.Unlock()
		if !busy {
			return
		}
		if time.Now().After(deadline) {
			t.Fatal("a check did not finish in five seconds")
		}
		time.Sleep(time.Millisecond)
	}
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
			step, _, _, err := d.ask(t.Context(), "acme/site", "abc", 0, never)
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

	_, link, _, err := d.ask(t.Context(), "acme/site", "abc", 0, never)
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
	if step, _, _, _ := d.ask(t.Context(), "acme/site", "mine", 0, contains); step != publish.StepDone {
		t.Errorf("step = %q, want done", step)
	}
	if step, _, _, _ := d.ask(t.Context(), "acme/site", "mine", 0, never); step != publish.StepPending {
		t.Errorf("a deployment that does not contain the commit counted: %q", step)
	}
}

// Hosts other than Pages, such as Vercel, record their deployments on GitHub
// too, in environments of their own naming.
func TestADeploymentToAnotherHostIsReported(t *testing.T) {
	fake, d := newFake(t)
	fake.deployments["acme/site"] = []ghDeployment{{ID: 7, SHA: "abc", Environment: "Production"}}
	fake.statuses[7] = []ghStatus{{ID: 1, State: "success", EnvironmentURL: "https://site-abc.vercel.app"}}

	step, link, _, err := d.ask(t.Context(), "acme/site", "abc", 0, never)
	if err != nil || step != publish.StepDone || link != "https://site-abc.vercel.app" {
		t.Errorf("step = %q, url = %q, %v", step, link, err)
	}
}

// A repository that once deployed to Pages and then moved to another host
// keeps its old Pages deployments; a commit is reported from where it went.
func TestARepositoryThatMovedOffPagesIsReportedFromItsNewHost(t *testing.T) {
	fake, d := newFake(t)
	fake.deployments["acme/site"] = []ghDeployment{
		{ID: 9, SHA: "abc", Environment: "Production"},
		{ID: 3, SHA: "old", Environment: pagesEnvironment},
	}
	fake.statuses[9] = []ghStatus{{ID: 1, State: "success"}}

	if step, _, _, err := d.ask(t.Context(), "acme/site", "abc", 0, never); err != nil || step != publish.StepDone {
		t.Errorf("step = %q, %v; want done", step, err)
	}
}

// A commit deployed to Pages and previewed elsewhere is reported from Pages.
func TestPagesIsPreferredOverAnotherEnvironment(t *testing.T) {
	fake, d := newFake(t)
	fake.deployments["acme/site"] = []ghDeployment{
		{ID: 9, SHA: "abc", Environment: "Preview"},
		{ID: 8, SHA: "abc", Environment: pagesEnvironment},
	}
	fake.statuses[9] = []ghStatus{{ID: 1, State: "success"}}
	fake.statuses[8] = []ghStatus{{ID: 1, State: "in_progress"}}

	if step, _, _, _ := d.ask(t.Context(), "acme/site", "abc", 0, never); step != publish.StepPending {
		t.Errorf("step = %q, want the Pages deployment's, still under way", step)
	}
}

func TestARepositoryThatDoesNotReportDeploymentsIsNotApplicable(t *testing.T) {
	t.Run("never deployed", func(t *testing.T) {
		_, d := newFake(t)
		if step, _, _, err := d.ask(t.Context(), "acme/site", "abc", 0, never); err != nil || step != publish.StepNotApplicable {
			t.Errorf("step = %q, %v", step, err)
		}
	})
	t.Run("private", func(t *testing.T) {
		fake, d := newFake(t)
		fake.private = true
		if step, _, _, err := d.ask(t.Context(), "acme/site", "abc", 0, never); err != nil || step != publish.StepNotApplicable {
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
	if _, _, _, err := d.ask(t.Context(), "acme/site", "abc", 0, never); !errors.Is(err, errQuiet) {
		t.Errorf("err = %v, want errQuiet", err)
	}
	if _, _, _, err := d.ask(t.Context(), "acme/site", "abc", 0, never); !errors.Is(err, errQuiet) {
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

	if step := d.look("acme/site", "abc", true, never).step; step != publish.StepPending {
		t.Errorf("first look = %q, want pending until GitHub has answered", step)
	}
	deadline := time.Now().Add(5 * time.Second)
	for {
		step := d.look("acme/site", "abc", true, never).step
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

// Once a commit's deployment is found, a check asks for its status alone.
func TestAFoundDeploymentIsThenAskedAboutByItsStatusAlone(t *testing.T) {
	fake, d := newFake(t)
	fake.deployments["acme/site"] = []ghDeployment{{ID: 7, SHA: "abc", Environment: "Production"}}
	fake.statuses[7] = []ghStatus{{ID: 1, State: "in_progress"}}

	step, _, id, err := d.ask(t.Context(), "acme/site", "abc", 0, never)
	if err != nil || step != publish.StepPending || id != 7 {
		t.Fatalf("step = %q, id = %d, %v", step, id, err)
	}
	if n := fake.count(); n != 2 {
		t.Errorf("the first check made %d requests, want the list and the status", n)
	}
	if _, _, _, err := d.ask(t.Context(), "acme/site", "abc", id, never); err != nil {
		t.Fatal(err)
	}
	if n := fake.count(); n != 3 {
		t.Errorf("the second check made %d requests, want only the status", n-2)
	}
}

// However many commits are pushed, the checker asks GitHub at most once a
// minute, after the few requests it may save up.
func TestTheCheckerAsksAtMostOnceAMinute(t *testing.T) {
	c := &clock{t: time.Now()}
	fake, d := newFakeAt(t, c.now)
	var heads []string
	for i := range 10 {
		head := "commit" + strconv.Itoa(i)
		heads = append(heads, head)
		fake.deployments["acme/site"] = append(fake.deployments["acme/site"], ghDeployment{ID: int64(i + 1), SHA: head})
		fake.statuses[int64(i+1)] = []ghStatus{{ID: 1, State: "in_progress"}}
	}

	const minutes = 10
	for elapsed := time.Duration(0); elapsed < minutes*time.Minute; elapsed += 5 * time.Second {
		for _, head := range heads {
			d.look("acme/site", head, true, never)
			settle(t, d)
		}
		c.add(5 * time.Second)
	}
	if n := fake.count(); n > burst+minutes || n < minutes {
		t.Errorf("%d requests in %d minutes, want about one a minute and at most %d", n, minutes, burst+minutes)
	}
}

// A deployment still under way is asked about every minute for ten minutes,
// then every five, so one that never finishes costs a few requests an hour.
func TestAPendingDeploymentIsAskedLessOftenAfterTenMinutes(t *testing.T) {
	c := &clock{t: time.Now()}
	fake, d := newFakeAt(t, c.now)
	fake.deployments["acme/site"] = []ghDeployment{{ID: 7, SHA: "abc", Environment: "Production"}}
	fake.statuses[7] = []ghStatus{{ID: 1, State: "in_progress"}}

	var firstTen int
	for elapsed := time.Duration(0); elapsed < time.Hour; elapsed += 5 * time.Second {
		if elapsed == slowAfter {
			firstTen = fake.count()
		}
		d.look("acme/site", "abc", true, never)
		settle(t, d)
		c.add(5 * time.Second)
	}
	// The list and the status at once, then the status at each minute to
	// the tenth; then at the tenth and every five minutes after it.
	if firstTen != 2+9 {
		t.Errorf("%d requests in the first ten minutes, want 11", firstTen)
	}
	if n := fake.count(); n != 2+10+9 {
		t.Errorf("%d requests in the hour, want 21", n)
	}
}

// While GitHub's allowance is spent, a pending deployment says when it will
// be asked about again rather than seeming to deploy forever.
func TestASpentAllowanceSaysWhenTheCheckerAsksAgain(t *testing.T) {
	fake, d := newFake(t)
	fake.remaining = spare
	fake.deployments["acme/site"] = []ghDeployment{{ID: 7, SHA: "abc"}}

	d.look("acme/site", "abc", true, never)
	settle(t, d)
	seen := d.look("acme/site", "abc", true, never)
	if seen.step != publish.StepPending || seen.pausedUntil.Before(time.Now().Add(50*time.Minute)) {
		t.Errorf("step %q, paused until %v; want pending until GitHub's reset an hour away", seen.step, seen.pausedUntil)
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
