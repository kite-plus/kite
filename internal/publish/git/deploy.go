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
// the one the workflow kite init writes deploys to. A commit deployed there
// and somewhere else as well is reported from here.
const pagesEnvironment = "github-pages"

const (
	// perRequest is how often the checker may ask GitHub, and burst how many
	// requests it may save up, so that a check needing two is not held back
	// a minute between them. An anonymous caller gets sixty an hour, shared
	// with everything else on the network, and running out costs the rest of
	// the hour; a minute's delay in saying a site is live costs nothing.
	perRequest = time.Minute
	burst      = 3

	// checkCost is the most requests one check makes: the list of
	// deployments, then the status of the one that matters.
	checkCost = 2

	// pendingEvery is how often a deployment still under way is asked about,
	// until slowAfter has passed since the push was first seen; then it is
	// slowEvery, so a deployment that never comes costs a dozen requests an
	// hour.
	pendingEvery = time.Minute
	slowAfter    = 10 * time.Minute
	slowEvery    = 5 * time.Minute

	// probeAgain is how long an answer that could still change -- a failed
	// deployment can be run again, and a host can be connected -- is taken at
	// its word before it is asked for again.
	probeAgain = 10 * time.Minute

	// listed is how many of a repository's newest deployments are read to
	// find a commit's.
	listed = 30

	// spare is how many of the hour's anonymous requests are left alone, for
	// whatever else on this machine talks to GitHub.
	spare = 10
)

// errQuiet is returned while the checker is holding back from GitHub.
var errQuiet = errors.New("git: not asking GitHub for now")

// deployChecker asks GitHub whether a commit has been deployed.
//
// Whether a site is live is something only its host knows. GitHub records
// every Pages deployment against the commit it deployed, and other hosts,
// such as Vercel, record theirs in the same place, so for a repository
// there the question has an answer. For a host that records nothing, the
// answer is that nobody reports it, which is better than a step that stays
// pending forever.
//
// Nothing here waits on the network while it is asked. The studio asks every
// few seconds, and an anonymous caller gets sixty requests an hour, with or
// without an ETag; so an answer is returned from what is known, and a stale
// one is refreshed in the background, at most once a minute across every
// commit and repository, and never again once it has finished.
type deployChecker struct {
	api    string
	client *http.Client
	now    func() time.Time

	mu      sync.Mutex
	reports map[string]reportProbe
	answers map[string]*deployAnswer

	// tokens is how many requests may be made now, and filled when the last
	// one was added. It goes below zero when two checks start together.
	tokens int
	filled time.Time

	// quiet is when GitHub may be asked again after a failure or a spent
	// allowance, and limited the same for a spent allowance alone.
	quiet   time.Time
	limited time.Time
}

// reportProbe is whether a repository records deployments at all.
type reportProbe struct {
	uses    bool
	checked time.Time
	asking  bool
}

// deployAnswer is what is known about one commit's deployment.
type deployAnswer struct {
	step publish.Step
	url  string

	// id is the deployment found for the commit while it is under way, so
	// later checks ask for its status alone.
	id int64

	since  time.Time
	next   time.Time
	asking bool
}

// deployLook is what look knows about a deployment. pausedUntil is set while
// GitHub's allowance is spent and the step is still pending.
type deployLook struct {
	step        publish.Step
	url         string
	pausedUntil time.Time
}

func newDeployChecker(api string, now func() time.Time) *deployChecker {
	if api == "" {
		api = GitHubAPI
	}
	return &deployChecker{
		api:     strings.TrimSuffix(api, "/"),
		client:  &http.Client{Timeout: 10 * time.Second},
		now:     now,
		reports: make(map[string]reportProbe),
		answers: make(map[string]*deployAnswer),
		tokens:  burst,
		filled:  now(),
	}
}

// look reports what is known about head's deployment in repo, and refreshes
// that in the background when it is due. includes reports whether a deployed
// commit contains head, which a newer deployment that superseded head's own
// does.
func (d *deployChecker) look(repo, head string, pushed bool, includes func(sha string) bool) deployLook {
	d.mu.Lock()
	defer d.mu.Unlock()

	if !pushed {
		// Nothing of head is deployed until it is pushed; all there is to
		// know is whether this repository would report it when it is.
		probe, known := d.reports[repo]
		if !probe.asking && (!known || d.now().Sub(probe.checked) > probeAgain) && d.afford(1) {
			probe.asking = true
			d.reports[repo] = probe
			go d.probe(repo)
		}
		if known && !probe.uses && !probe.checked.IsZero() {
			return deployLook{step: publish.StepNotApplicable}
		}
		return deployLook{step: publish.StepPending}
	}

	key := repo + "@" + head
	answer := d.answers[key]
	if answer == nil {
		answer = &deployAnswer{step: publish.StepPending, since: d.now()}
		d.answers[key] = answer
	}
	// Once live, a deployment stays that way; anything else may still move.
	if !answer.asking && answer.step != publish.StepDone && !d.now().Before(answer.next) && d.afford(checkCost) {
		answer.asking = true
		go d.refresh(key, repo, head, answer.id, includes)
	}
	out := deployLook{step: answer.step, url: answer.url}
	if answer.step == publish.StepPending && d.now().Before(d.limited) {
		out.pausedUntil = d.limited
	}
	return out
}

// afford reports whether n requests may be made now. It spends nothing; get
// spends one for each request it makes. The caller holds d.mu.
func (d *deployChecker) afford(n int) bool {
	now := d.now()
	if gained := int(now.Sub(d.filled) / perRequest); gained > 0 {
		d.tokens += gained
		d.filled = d.filled.Add(time.Duration(gained) * perRequest)
	}
	if d.tokens >= burst {
		// A full allowance saves up no more.
		d.tokens, d.filled = burst, now
	}
	return d.tokens >= n
}

func (d *deployChecker) probe(repo string) {
	ctx, cancel := context.WithTimeout(context.Background(), d.client.Timeout)
	defer cancel()
	uses, err := d.recordsDeployments(ctx, repo)

	d.mu.Lock()
	defer d.mu.Unlock()
	probe := d.reports[repo]
	probe.asking = false
	if err == nil {
		probe.uses, probe.checked = uses, d.now()
	}
	d.reports[repo] = probe
}

func (d *deployChecker) refresh(key, repo, head string, known int64, includes func(string) bool) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*d.client.Timeout)
	defer cancel()
	step, link, id, err := d.ask(ctx, repo, head, known, includes)

	d.mu.Lock()
	defer d.mu.Unlock()
	answer := d.answers[key]
	answer.asking = false
	if err == nil {
		answer.step, answer.url = step, link
		// A failed deployment can be run again as a new one, so only one
		// still under way is kept to be asked about by its status.
		answer.id = 0
		if step == publish.StepPending {
			answer.id = id
		}
	}
	if answer.step == publish.StepPending {
		every := pendingEvery
		if d.now().Sub(answer.since) >= slowAfter {
			every = slowEvery
		}
		answer.next = d.now().Add(every)
		return
	}
	answer.next = d.now().Add(probeAgain)
}

// ask finds head's deployment and how it went, and the id of the deployment
// it read, if any. known, when not zero, is a deployment of head found
// before, and only its status is asked for.
func (d *deployChecker) ask(ctx context.Context, repo, head string, known int64, includes func(string) bool) (publish.Step, string, int64, error) {
	if known != 0 {
		step, link, err := d.status(ctx, repo, known)
		return step, link, known, err
	}

	// One list answers every question there is before a status: whether the
	// repository records deployments at all, whether head has one of its
	// own, and whether a later commit's replaced it, which deploys head's
	// content all the same.
	var latest []ghDeployment
	found, err := d.get(ctx, "/repos/"+repo+"/deployments?per_page="+strconv.Itoa(listed), &latest)
	if err != nil {
		return "", "", 0, err
	}
	if !found || len(latest) == 0 {
		return publish.StepNotApplicable, "", 0, nil
	}

	var own []ghDeployment
	for _, dep := range latest {
		if dep.SHA == head {
			own = append(own, dep)
		}
	}
	var dep ghDeployment
	switch {
	case len(own) > 0:
		dep = preferred(own)
	case includes(preferred(latest).SHA):
		dep = preferred(latest)
	default:
		return publish.StepPending, "", 0, nil
	}
	step, link, err := d.status(ctx, repo, dep.ID)
	return step, link, dep.ID, err
}

// recordsDeployments reports whether a repository has ever recorded a
// deployment, to any environment.
func (d *deployChecker) recordsDeployments(ctx context.Context, repo string) (bool, error) {
	var latest []ghDeployment
	found, err := d.get(ctx, "/repos/"+repo+"/deployments?per_page=1", &latest)
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
	ID          int64  `json:"id"`
	SHA         string `json:"sha"`
	Environment string `json:"environment"`
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

// preferred picks the deployment to report: the newest to Pages, since a
// repository that deploys there may also send previews elsewhere, and
// otherwise the newest to any environment. Hosts do not reliably mark which
// of their environments is production, so nothing else tells them apart.
func preferred(ds []ghDeployment) ghDeployment {
	var pages []ghDeployment
	for _, dep := range ds {
		if dep.Environment == pagesEnvironment {
			pages = append(pages, dep)
		}
	}
	if len(pages) > 0 {
		return newest(pages)
	}
	return newest(ds)
}

// get reads one GitHub API resource into v. It reports false when there is
// no such resource, which is also how a private repository looks to an
// anonymous caller.
func (d *deployChecker) get(ctx context.Context, path string, v any) (bool, error) {
	d.mu.Lock()
	quiet := d.now().Before(d.quiet)
	if !quiet {
		d.tokens--
	}
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
	until := d.now().Add(probeAgain)
	if reset, err := strconv.ParseInt(h.Get("X-RateLimit-Reset"), 10, 64); err == nil {
		until = time.Unix(reset, 0)
	}
	d.mu.Lock()
	d.quiet, d.limited = until, until
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
