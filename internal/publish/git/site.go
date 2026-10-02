package git

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/kite-plus/kite/internal/publish"
	"github.com/kite-plus/kite/internal/stamp"
)

// siteChecker reads the stamp a build leaves in the live site, to tell
// whether a commit has reached it.
//
// It works whatever hosts the site, Netlify and a server of one's own as
// much as Pages, because it asks the site rather than the host; and it costs
// no GitHub request, so it goes on answering while GitHub's allowance is
// spent. It cannot tell a failed deployment from a slow one; only the host
// knows that, and deployChecker asks it where it can.
type siteChecker struct {
	client *http.Client
	now    func() time.Time

	mu      sync.Mutex
	stamped map[string]stampProbe
	answers map[string]*siteAnswer
}

// stampProbe is whether a site has a stamp at all.
type stampProbe struct {
	has     bool
	checked time.Time
	asking  bool
}

// siteAnswer is what the site says about one commit.
type siteAnswer struct {
	step   publish.Step
	since  time.Time
	next   time.Time
	asking bool
}

func newSiteChecker(now func() time.Time) *siteChecker {
	return &siteChecker{
		client:  &http.Client{Timeout: 10 * time.Second},
		now:     now,
		stamped: make(map[string]stampProbe),
		answers: make(map[string]*siteAnswer),
	}
}

// look reports whether head is live at the site at addr, and refreshes that
// in the background when it is due, on the same schedule as a deployment
// GitHub is asked about. includes reports whether the commit a site was built
// from contains head.
func (c *siteChecker) look(addr, head string, pushed bool, includes func(sha string) bool) publish.Step {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !pushed {
		// All there is to know is whether the site would say, once head is
		// pushed and built.
		probe, known := c.stamped[addr]
		if !probe.asking && (!known || c.now().Sub(probe.checked) > probeAgain) {
			probe.asking = true
			c.stamped[addr] = probe
			go c.probe(addr)
		}
		if probe.has {
			return publish.StepPending
		}
		return publish.StepNotApplicable
	}

	key := addr + "@" + head
	answer := c.answers[key]
	if answer == nil {
		answer = &siteAnswer{step: publish.StepPending, since: c.now()}
		c.answers[key] = answer
	}
	if !answer.asking && answer.step != publish.StepDone && !c.now().Before(answer.next) {
		answer.asking = true
		go c.refresh(key, addr, head, includes)
	}
	return answer.step
}

func (c *siteChecker) probe(addr string) {
	ctx, cancel := context.WithTimeout(context.Background(), c.client.Timeout)
	defer cancel()
	_, has, err := c.read(ctx, addr)

	c.mu.Lock()
	defer c.mu.Unlock()
	probe := c.stamped[addr]
	probe.asking = false
	if err == nil {
		probe.has, probe.checked = has, c.now()
	}
	c.stamped[addr] = probe
}

func (c *siteChecker) refresh(key, addr, head string, includes func(string) bool) {
	ctx, cancel := context.WithTimeout(context.Background(), c.client.Timeout)
	defer cancel()
	commit, has, err := c.read(ctx, addr)

	c.mu.Lock()
	defer c.mu.Unlock()
	answer := c.answers[key]
	answer.asking = false
	switch {
	case err != nil:
		// Unreachable for now; what was known stands.
	case !has:
		answer.step = publish.StepNotApplicable
	case commit == head || includes(commit):
		answer.step = publish.StepDone
	default:
		answer.step = publish.StepPending
	}
	if err == nil {
		probe := c.stamped[addr]
		probe.has, probe.checked = has, c.now()
		c.stamped[addr] = probe
	}

	if answer.step == publish.StepPending {
		every := pendingEvery
		if c.now().Sub(answer.since) >= slowAfter {
			every = slowEvery
		}
		answer.next = c.now().Add(every)
		return
	}
	answer.next = c.now().Add(probeAgain)
}

// read fetches the site's stamp. It reports false, without an error, when
// the site answers with anything that is not one: a site built without a
// stamp, by an older Kite, or one whose host serves a page for every address.
func (c *siteChecker) read(ctx context.Context, addr string) (string, bool, error) {
	// A query of its own steps past a cache in front of the site, which
	// would otherwise go on serving the stamp from before the push.
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		addr+"/"+stamp.File+"?t="+strconv.FormatInt(c.now().UnixNano(), 36), nil)
	if err != nil {
		return "", false, err
	}
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Accept", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return "", false, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return "", false, nil
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<10))
	if err != nil {
		return "", false, err
	}
	s, err := stamp.Decode(body)
	if err != nil {
		return "", false, nil
	}
	return s.Commit, true, nil
}

// publicAddress returns the site's address when it is one readers can reach,
// without a trailing slash, and empty otherwise: the localhost a new site
// starts with, a private network, or a name kept for examples.
func publicAddress(raw string) string {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" || u.User != nil {
		return ""
	}
	host := strings.ToLower(strings.TrimSuffix(u.Hostname(), "."))
	if ip := net.ParseIP(host); ip != nil {
		if ip.IsLoopback() || ip.IsPrivate() || ip.IsUnspecified() || ip.IsLinkLocalUnicast() {
			return ""
		}
	} else {
		if !strings.Contains(host, ".") {
			return ""
		}
		for _, reserved := range []string{"localhost", "local", "test", "invalid", "example", "internal", "lan", "home.arpa"} {
			if host == reserved || strings.HasSuffix(host, "."+reserved) {
				return ""
			}
		}
		for _, example := range []string{"example.com", "example.net", "example.org"} {
			if host == example || strings.HasSuffix(host, "."+example) {
				return ""
			}
		}
	}
	return strings.TrimSuffix(u.Scheme+"://"+u.Host+u.EscapedPath(), "/")
}
