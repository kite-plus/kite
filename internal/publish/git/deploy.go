package git

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/kite-plus/kite/internal/publish"
)

// GitHubAPI is where deployments are asked about.
const GitHubAPI = "https://api.github.com"

// pagesEnvironment is the environment actions/deploy-pages deploys to, and so
// the one the workflow kite init writes deploys to.
const pagesEnvironment = "github-pages"

const (
	// firstWait and longestWait bound how often a deployment still under way
	// is asked about again: soon at first, then less and less often.
	firstWait   = 15 * time.Second
	longestWait = 10 * time.Minute

	// probeAgain is how long an answer that could still change -- a failed
	// deployment can be run again, and Pages can be turned on -- is taken at
	// its word before it is asked for again.
	probeAgain = 10 * time.Minute

	// spare is how many of the hour's anonymous requests are left alone, for
	// whatever else on this machine talks to GitHub.
	spare = 10
)

// errQuiet is returned while the checker is holding back from GitHub.
var errQuiet = errors.New("git: not asking GitHub for now")

// deployChecker asks GitHub whether a commit has been deployed to Pages.
//
// Whether a site is live is something only its host knows. GitHub records
// every Pages deployment against the commit it deployed, so for a repository
// there the question has an answer. For any other host, or a repository that
// does not deploy to Pages, the answer is that nobody reports it, which is
// better than a step that stays pending forever.
//
// Nothing here waits on the network while it is asked. The studio asks every
// few seconds, and an anonymous caller gets sixty requests an hour, with or
// without an ETag; so an answer is returned from what is known, and a stale
// one is refreshed in the background, less often the longer a deployment
// takes, and never again once it has finished.
type deployChecker struct {
	api    string
	client *http.Client
	now    func() time.Time

	mu      sync.Mutex
	pages   map[string]pagesProbe
	answers map[string]*deployAnswer
	quiet   time.Time
}

// pagesProbe is whether a repository deploys to Pages at all.
type pagesProbe struct {
	uses    bool
	checked time.Time
	asking  bool
}

// deployAnswer is what is known about one commit's deployment.
type deployAnswer struct {
	step publish.Step
	url  string

	next   time.Time
	wait   time.Duration
	asking bool
}

func newDeployChecker(api string, now func() time.Time) *deployChecker {
	if api == "" {
		api = GitHubAPI
	}
	return &deployChecker{
		api:     strings.TrimSuffix(api, "/"),
		client:  &http.Client{Timeout: 10 * time.Second},
		now:     now,
		pages:   make(map[string]pagesProbe),
		answers: make(map[string]*deployAnswer),
	}
}

// look reports what is known about head's deployment in repo, and refreshes
// that in the background when it is due. includes reports whether a deployed
// commit contains head, which a newer deployment that superseded head's own
// does.
func (d *deployChecker) look(repo, head string, pushed bool, includes func(sha string) bool) (publish.Step, string) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if !pushed {
		// Nothing of head is deployed until it is pushed; all there is to
		// know is whether this repository would report it when it is.
		probe, known := d.pages[repo]
		if !probe.asking && (!known || d.now().Sub(probe.checked) > probeAgain) {
			probe.asking = true
			d.pages[repo] = probe
			go d.probe(repo)
		}
		if known && !probe.uses && !probe.checked.IsZero() {
			return publish.StepNotApplicable, ""
		}
		return publish.StepPending, ""
	}

	key := repo + "@" + head
	answer := d.answers[key]
	if answer == nil {
		answer = &deployAnswer{step: publish.StepPending}
		d.answers[key] = answer
	}
	// Once live, a deployment stays that way; anything else may still move.
	if !answer.asking && answer.step != publish.StepDone && !d.now().Before(answer.next) {
		answer.asking = true
		go d.refresh(key, repo, head, includes)
	}
	return answer.step, answer.url
}

func (d *deployChecker) probe(repo string) {
	ctx, cancel := context.WithTimeout(context.Background(), d.client.Timeout)
	defer cancel()
	uses, err := d.usesPages(ctx, repo)

	d.mu.Lock()
	defer d.mu.Unlock()
	probe := d.pages[repo]
	probe.asking = false
	if err == nil {
		probe.uses, probe.checked = uses, d.now()
	}
	d.pages[repo] = probe
}

func (d *deployChecker) refresh(key, repo, head string, includes func(string) bool) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*d.client.Timeout)
	defer cancel()
	step, link, err := d.ask(ctx, repo, head, includes)

	d.mu.Lock()
	defer d.mu.Unlock()
	answer := d.answers[key]
	answer.asking = false
	if err == nil {
		answer.step, answer.url = step, link
	}
	if answer.step == publish.StepPending {
		// Under way, or not known: ask again later, and later still the
		// time after, so a deployment that never comes costs a handful of
		// requests rather than all of them.
		answer.wait = min(max(2*answer.wait, firstWait), longestWait)
		answer.next = d.now().Add(answer.wait)
		return
	}
	answer.next = d.now().Add(probeAgain)
}

// ask finds head's deployment and how it went.
func (d *deployChecker) ask(ctx context.Context, repo, head string, includes func(string) bool) (publish.Step, string, error) {
	var own []ghDeployment
	found, err := d.get(ctx, "/repos/"+repo+"/deployments?environment="+pagesEnvironment+
		"&sha="+url.QueryEscape(head)+"&per_page=5", &own)
	if err != nil {
		return "", "", err
	}
	if found && len(own) > 0 {
		return d.status(ctx, repo, newest(own).ID)
	}

	// No deployment of head itself. The repository may not deploy to Pages
	// at all, head's may not have started yet, or a later commit's may have
	// replaced it, which deploys head's content all the same.
	var latest []ghDeployment
	found, err = d.get(ctx, "/repos/"+repo+"/deployments?environment="+pagesEnvironment+"&per_page=5", &latest)
	if err != nil {
		return "", "", err
	}
	if !found || len(latest) == 0 {
		return publish.StepNotApplicable, "", nil
	}
	if last := newest(latest); last.SHA != head && includes(last.SHA) {
		return d.status(ctx, repo, last.ID)
	}
	return publish.StepPending, "", nil
}

// usesPages reports whether a repository has ever deployed to Pages.
func (d *deployChecker) usesPages(ctx context.Context, repo string) (bool, error) {
	var latest []ghDeployment
	found, err := d.get(ctx, "/repos/"+repo+"/deployments?environment="+pagesEnvironment+"&per_page=1", &latest)
	return found && len(latest) > 0, err
}

// status reads how a deployment went.
func (d *deployChecker) status(ctx context.Context, repo string, id int64) (publish.Step, string, error) {
	var statuses []ghStatus
	if _, err := d.get(ctx, "/repos/"+repo+"/deployments/"+strconv.FormatInt(id, 10)+"/statuses?per_page=5", &statuses); err != nil {
		return "", "", err
	}
	if len(statuses) == 0 {
		return publish.StepPending, "", nil
	}
	last := slices.MaxFunc(statuses, func(a, b ghStatus) int { return cmp.Compare(a.ID, b.ID) })
	switch last.State {
	case "success":
		return publish.StepDone, last.EnvironmentURL, nil
	case "inactive":
		// Replaced by a newer deployment, which means this one went live.
		return publish.StepDone, "", nil
	case "failure", "error":
		return publish.StepFailed, "", nil
	default:
		// queued, pending, in_progress, and whatever GitHub adds next, such
		// as waiting on an environment's protection rules.
		return publish.StepPending, "", nil
	}
}

type ghDeployment struct {
	ID  int64  `json:"id"`
	SHA string `json:"sha"`
}

type ghStatus struct {
	ID             int64  `json:"id"`
	State          string `json:"state"`
	EnvironmentURL string `json:"environment_url"`
}

// newest picks the most recent deployment. GitHub lists them newest first,
// but does not promise to, and ids only ever grow.
func newest(ds []ghDeployment) ghDeployment {
	return slices.MaxFunc(ds, func(a, b ghDeployment) int { return cmp.Compare(a.ID, b.ID) })
}

// get reads one GitHub API resource into v. It reports false when there is
// no such resource, which is also how a private repository looks to an
// anonymous caller.
func (d *deployChecker) get(ctx context.Context, path string, v any) (bool, error) {
	d.mu.Lock()
	quiet := d.now().Before(d.quiet)
	d.mu.Unlock()
	if quiet {
		return false, errQuiet
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, d.api+path, nil)
	if err != nil {
		return false, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := d.client.Do(req)
	if err != nil {
		d.holdBack(time.Minute)
		return false, err
	}
	defer func() { _ = resp.Body.Close() }()
	d.budget(resp.Header)

	switch resp.StatusCode {
	case http.StatusOK:
		body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		if err != nil {
			return false, err
		}
		return true, json.Unmarshal(body, v)
	case http.StatusNotFound:
		return false, nil
	default:
		return false, fmt.Errorf("git: GitHub answered %s for %s", resp.Status, path)
	}
}

// budget stops asking while too little of the hour's allowance is left.
func (d *deployChecker) budget(h http.Header) {
	remaining, err := strconv.Atoi(h.Get("X-RateLimit-Remaining"))
	if err != nil || remaining > spare {
		return
	}
	reset, err := strconv.ParseInt(h.Get("X-RateLimit-Reset"), 10, 64)
	if err != nil {
		d.holdBack(longestWait)
		return
	}
	d.mu.Lock()
	d.quiet = time.Unix(reset, 0)
	d.mu.Unlock()
}

func (d *deployChecker) holdBack(wait time.Duration) {
	d.mu.Lock()
	d.quiet = d.now().Add(wait)
	d.mu.Unlock()
}

// githubRepo reads "owner/name" out of a remote URL on github.com, in any of
// the forms git accepts for one.
func githubRepo(remote string) (string, bool) {
	var path string
	switch {
	case strings.HasPrefix(remote, "git@github.com:"):
		path = strings.TrimPrefix(remote, "git@github.com:")
	default:
		u, err := url.Parse(remote)
		if err != nil {
			return "", false
		}
		// ssh.github.com is GitHub's ssh on port 443, for networks that
		// block 22.
		if host := strings.ToLower(u.Hostname()); host != "github.com" && host != "ssh.github.com" {
			return "", false
		}
		path = u.Path
	}
	path = strings.TrimSuffix(strings.Trim(path, "/"), ".git")
	owner, name, ok := strings.Cut(path, "/")
	if !ok || owner == "" || name == "" || strings.Contains(name, "/") {
		return "", false
	}
	return owner + "/" + name, true
}
