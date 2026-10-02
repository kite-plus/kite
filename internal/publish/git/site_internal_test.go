package git

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/kite-plus/kite/internal/publish"
	"github.com/kite-plus/kite/internal/stamp"
)

const (
	siteHead  = "0123456789abcdef0123456789abcdef01234567"
	siteLater = "89abcdef0123456789abcdef0123456789abcdef"
	siteOlder = "fedcba9876543210fedcba9876543210fedcba98"
)

// fakeSite serves a build stamp, or whatever a host serves in its place.
type fakeSite struct {
	mu       sync.Mutex
	body     string
	status   int
	queries  []string
	requests int
}

func (f *fakeSite) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.requests++
	f.queries = append(f.queries, r.URL.RawQuery)
	if r.URL.Path != "/blog/"+stamp.File {
		http.NotFound(w, r)
		return
	}
	w.WriteHeader(f.status)
	_, _ = w.Write([]byte(f.body))
}

func (f *fakeSite) serve(status int, body string) {
	f.mu.Lock()
	f.status, f.body = status, body
	f.mu.Unlock()
}

func (f *fakeSite) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.requests
}

func newSite(t *testing.T, now func() time.Time) (*fakeSite, *siteChecker, string) {
	t.Helper()
	site := &fakeSite{status: http.StatusOK, body: string(stamp.Encode(siteOlder))}
	srv := httptest.NewServer(site)
	t.Cleanup(srv.Close)
	return site, newSiteChecker(now), srv.URL + "/blog"
}

// settleSite waits for every read that look started to finish.
func settleSite(t *testing.T, c *siteChecker) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		c.mu.Lock()
		busy := false
		for _, a := range c.answers {
			busy = busy || a.asking
		}
		for _, p := range c.stamped {
			busy = busy || p.asking
		}
		c.mu.Unlock()
		if !busy {
			return
		}
		if time.Now().After(deadline) {
			t.Fatal("a read of the site did not finish in five seconds")
		}
		time.Sleep(time.Millisecond)
	}
}

func lookAtSite(t *testing.T, c *siteChecker, addr string, includes func(string) bool) publish.Step {
	t.Helper()
	c.look(addr, siteHead, true, includes)
	settleSite(t, c)
	return c.look(addr, siteHead, true, includes)
}

func TestTheSiteSaysWhetherACommitIsLive(t *testing.T) {
	laterContainsHead := func(sha string) bool { return sha == siteLater }
	for name, tc := range map[string]struct {
		status int
		body   string
		want   publish.Step
	}{
		"built from it":                 {http.StatusOK, string(stamp.Encode(siteHead)), publish.StepDone},
		"built from a later commit":     {http.StatusOK, string(stamp.Encode(siteLater)), publish.StepDone},
		"still built from an older one": {http.StatusOK, string(stamp.Encode(siteOlder)), publish.StepPending},
		"built without a stamp":         {http.StatusNotFound, "", publish.StepNotApplicable},
		"a page for every address":      {http.StatusOK, "<!doctype html><title>Home</title>", publish.StepNotApplicable},
	} {
		t.Run(name, func(t *testing.T) {
			site, c, addr := newSite(t, time.Now)
			site.serve(tc.status, tc.body)
			if got := lookAtSite(t, c, addr, laterContainsHead); got != tc.want {
				t.Errorf("step = %q, want %q", got, tc.want)
			}
		})
	}
}

// A cache in front of the site would go on serving the stamp from before
// the push, so each read asks for it afresh.
func TestEachReadOfTheStampStepsPastACache(t *testing.T) {
	c := &clock{t: time.Now()}
	site, checker, addr := newSite(t, c.now)
	lookAtSite(t, checker, addr, never)
	c.add(time.Minute)
	lookAtSite(t, checker, addr, never)

	site.mu.Lock()
	defer site.mu.Unlock()
	if len(site.queries) < 2 || site.queries[0] == "" || site.queries[0] == site.queries[1] {
		t.Errorf("queries = %q, want a different one each time", site.queries)
	}
}

// A push goes live a while after it is made: the site is asked again each
// minute, then each five, and not at all once the commit is there.
func TestTheSiteIsAskedUntilTheCommitIsLive(t *testing.T) {
	c := &clock{t: time.Now()}
	site, checker, addr := newSite(t, c.now)

	for range 10 {
		if step := lookAtSite(t, checker, addr, never); step != publish.StepPending {
			t.Fatalf("step = %q while the site is still built from an older commit", step)
		}
		c.add(30 * time.Second)
	}
	if n := site.count(); n != 5 {
		t.Errorf("%d reads in five minutes, want one a minute", n)
	}

	site.serve(http.StatusOK, string(stamp.Encode(siteHead)))
	c.add(time.Minute)
	if step := lookAtSite(t, checker, addr, never); step != publish.StepDone {
		t.Fatalf("step = %q once the site is built from the commit", step)
	}
	settled := site.count()
	for range 5 {
		c.add(10 * time.Minute)
		lookAtSite(t, checker, addr, never)
	}
	if n := site.count() - settled; n != 0 {
		t.Errorf("a live commit was asked about %d more times", n)
	}
}

// Before a push, the step says whether the site would tell once it is made.
func TestBeforeAPushTheSiteSaysWhetherItWillTell(t *testing.T) {
	for name, tc := range map[string]struct {
		status int
		want   publish.Step
	}{
		"a site with a stamp": {http.StatusOK, publish.StepPending},
		"a site without one":  {http.StatusNotFound, publish.StepNotApplicable},
	} {
		site, c, addr := newSite(t, time.Now)
		if tc.status != http.StatusOK {
			site.serve(tc.status, "")
		}
		c.look(addr, siteHead, false, never)
		settleSite(t, c)
		if step := c.look(addr, siteHead, false, never); step != tc.want {
			t.Errorf("%s: step %q, want %q", name, step, tc.want)
		}
	}
}

func TestOnlyAnAddressReadersReachIsAsked(t *testing.T) {
	for raw, want := range map[string]string{
		"https://www.kite.plus":        "https://www.kite.plus",
		"https://www.kite.plus/":       "https://www.kite.plus",
		"https://acme.github.io/site/": "https://acme.github.io/site",
		"http://blog.example.dev:8080": "http://blog.example.dev:8080",
		"http://localhost:1717":        "",
		"http://127.0.0.1:1717":        "",
		"http://[::1]:1717":            "",
		"http://192.168.1.20":          "",
		"http://kite.local":            "",
		"http://kite.localhost":        "",
		"https://example.com/blog/":    "",
		"https://www.example.org":      "",
		"http://intranet":              "",
		"ftp://files.kite.plus":        "",
		"":                             "",
		"https://user:pw@kite.plus":    "",
	} {
		if got := publicAddress(raw); got != want {
			t.Errorf("publicAddress(%q) = %q, want %q", raw, got, want)
		}
	}
}

func TestTheHostAndTheSiteAreTakenTogether(t *testing.T) {
	const addr = "https://www.kite.plus"
	paused := time.Now().Add(time.Hour)
	for name, tc := range map[string]struct {
		host deployLook
		site publish.Step
		want deployLook
	}{
		"the site has it, whatever the host says": {
			deployLook{step: publish.StepPending, host: publish.HostVercel},
			publish.StepDone,
			deployLook{step: publish.StepDone, url: addr + "/", host: publish.HostVercel},
		},
		"the host says it is live": {
			deployLook{step: publish.StepDone, url: "https://site-abc.vercel.app", host: publish.HostVercel},
			publish.StepNotApplicable,
			deployLook{step: publish.StepDone, url: "https://site-abc.vercel.app", host: publish.HostVercel},
		},
		"the host says it failed": {
			deployLook{step: publish.StepFailed, host: publish.HostCloudflarePages},
			publish.StepPending,
			deployLook{step: publish.StepFailed, host: publish.HostCloudflarePages},
		},
		"only the site will tell": {
			deployLook{step: publish.StepNotApplicable},
			publish.StepPending,
			deployLook{step: publish.StepPending},
		},
		"GitHub is paused but the site answers": {
			deployLook{step: publish.StepPending, pausedUntil: paused},
			publish.StepPending,
			deployLook{step: publish.StepPending},
		},
		"GitHub is paused and nothing else tells": {
			deployLook{step: publish.StepPending, pausedUntil: paused},
			publish.StepNotApplicable,
			deployLook{step: publish.StepPending, pausedUntil: paused},
		},
		"neither tells": {
			deployLook{step: publish.StepNotApplicable},
			publish.StepNotApplicable,
			deployLook{step: publish.StepNotApplicable},
		},
	} {
		if got := combine(tc.host, tc.site, addr); got != tc.want {
			t.Errorf("%s: got %+v, want %+v", name, got, tc.want)
		}
	}
}
