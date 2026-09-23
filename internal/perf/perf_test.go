package perf_test

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand/v2"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/kite-plus/kite/internal/serve"
	"github.com/kite-plus/kite/internal/site"
)

// The targets, from docs/design/architecture.md §29.
const (
	buildTarget  = 2 * time.Second        // M0: a full build
	listTarget   = 300 * time.Millisecond // M2: the list's first screen
	reloadTarget = 500 * time.Millisecond // M1: an edit reaching the browser
	studioTarget = 3 * time.Second        // M3: an edit reaching the studio
	branchTarget = 3 * time.Second        // M2: a checkout reaching the list
)

func needPerf(t *testing.T) {
	t.Helper()
	if os.Getenv("KITE_PERF") == "" {
		t.Skip("set KITE_PERF=1, or run make perf")
	}
}

// size is the number of posts, which the targets are stated for.
func size() int {
	if n, err := strconv.Atoi(os.Getenv("KITE_PERF_POSTS")); err == nil && n > 0 {
		return n
	}
	return 2000
}

// measured reports a timing against its target, failing when it is missed.
func measured(t *testing.T, what string, got, target time.Duration) {
	t.Helper()
	t.Logf("%-44s %8s   target %s", what, got.Round(time.Millisecond), target)
	if got > target {
		t.Errorf("%s took %s, over the %s target", what, got.Round(time.Millisecond), target)
	}
}

var vocabulary = strings.Fields(`
	a about after again against all also always another answer around away back
	because before being below between both build came can change close could
	day different does down each early end enough even every example far few
	file find first follow found from full give good great group hand hard head
	help here high home house idea important into keep kind know large last
	late learn leave left less light line little long look made make many might
	more most move much must name near need never next night nothing number
	often old once only open order other over own page part people place plan
	point possible public put question quite read real right same seem set
	several should show side since small some something sometimes soon start
	still story such system take tell than that their them then there these
	thing think those though through time today together turn under until use
	very want water well went what when where which while whole why will with
	without word work world would write year young`)

var (
	tags       = vocabulary[:40]
	categories = []string{"Notes", "Essays", "Tools", "Travel", "Reading", "Code", "Photos", "Life"}
)

// article is a post of about eight hundred words with the markdown a real
// one has: headings, emphasis, a link, a list, a quote and a code block.
func article(r *rand.Rand, i int) string {
	sentence := func(n int) string {
		w := make([]string, n)
		for j := range w {
			w[j] = vocabulary[r.IntN(len(vocabulary))]
		}
		w[0] = strings.ToUpper(w[0][:1]) + w[0][1:]
		return strings.Join(w, " ") + "."
	}
	para := func(sentences int) string {
		s := make([]string, sentences)
		for j := range s {
			s[j] = sentence(8 + r.IntN(12))
		}
		switch r.IntN(3) {
		case 0:
			s[0] = "**" + strings.TrimSuffix(s[0], ".") + "**."
		case 1:
			s[len(s)-1] = "It links to [another page](/posts/post-" + fmt.Sprintf("%04d", r.IntN(1000)) + "/)."
		default:
			s[len(s)/2] = "Run `kite build` and see."
		}
		return strings.Join(s, " ")
	}

	var b strings.Builder
	fmt.Fprintf(&b, "%s\n\n## %s\n\n%s\n\n", para(5), strings.TrimSuffix(sentence(4), "."), para(7))
	for range 4 {
		fmt.Fprintf(&b, "- %s\n", sentence(6))
	}
	fmt.Fprintf(&b, "\n```go\nfunc example%d() int {\n\treturn %d\n}\n```\n\n", i, i)
	fmt.Fprintf(&b, "## %s\n\n%s\n\n> %s\n\n%s\n", strings.TrimSuffix(sentence(3), "."), para(8), sentence(12), para(8))
	return b.String()
}

func postID(i int) string { return fmt.Sprintf("01J8KQ2P3R4S5T%012d", i) }

func postFile(root string, i int) string {
	return filepath.Join(root, "content", "posts", fmt.Sprintf("post-%04d", i), "index.md")
}

func post(i int, title string) string {
	r := rand.New(rand.NewPCG(uint64(i), 7))
	published := time.Date(2021, 1, 1, 9, 0, 0, 0, time.UTC).Add(time.Duration(i) * 21 * time.Hour)
	picked := make([]string, 0, 3)
	for len(picked) < 3 {
		if tag := tags[r.IntN(len(tags))]; !slices.Contains(picked, tag) {
			picked = append(picked, tag)
		}
	}
	return fmt.Sprintf("---\nid: %s\ntitle: %s\nslug: post-%04d\nstatus: published\npublished_at: %s\ntags: [%s]\ncategories: [%s]\n---\n\n%s",
		postID(i), title, i, published.Format(time.RFC3339), strings.Join(picked, ", "),
		categories[r.IntN(len(categories))], article(r, i))
}

func write(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// newSite writes a site of size() posts. Its files are dated an hour ago, as
// the files of a real repository are: a file written a moment before it is
// indexed is read again on the next pass whatever its stat says, which would
// measure a situation no author is in.
func newSite(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	write(t, filepath.Join(root, "kite.yaml"),
		"site:\n  title: Perf\n  baseURL: https://example.com\n  language: en\nbuild:\n  output: public\n")
	for i := range size() {
		write(t, postFile(root, i), post(i, fmt.Sprintf("Post %04d", i)))
	}
	write(t, filepath.Join(root, "content", "pages", "about.md"),
		"---\nid: 01J8KQ2P3R4S5T6V7W8X9YZZZZ\ntitle: About\nslug: about\nstatus: published\n---\n\nAbout this site.\n")
	backdate(t, root)
	return root
}

func backdate(t *testing.T, root string) {
	t.Helper()
	then := time.Now().Add(-time.Hour)
	err := filepath.WalkDir(filepath.Join(root, "content"), func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		return os.Chtimes(p, then, then)
	})
	if err != nil {
		t.Fatal(err)
	}
}

func open(t *testing.T, root string) *site.Site {
	t.Helper()
	s, err := site.Open(t.Context(), root)
	if err != nil {
		t.Fatalf("site.Open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

// M0: `kite build` on a checkout with nothing cached, so indexing every file
// is part of it, and again with the index in place.
func TestAFullBuild(t *testing.T) {
	needPerf(t)
	root := newSite(t)

	start := time.Now()
	s := open(t, root)
	indexed := time.Since(start)
	stats, _, err := s.Build(t.Context(), site.BuildOptions{})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%d targets rendered; indexing took %s", stats.Rendered, indexed.Round(time.Millisecond))
	measured(t, fmt.Sprintf("full build of %d posts, nothing cached", size()), time.Since(start), buildTarget)

	start = time.Now()
	if _, _, err := s.Build(t.Context(), site.BuildOptions{OutDir: filepath.Join(root, "again")}); err != nil {
		t.Fatal(err)
	}
	measured(t, "full build with the index in place", time.Since(start), buildTarget)
}

// M2: what the post list asks for when it opens, all at once as a browser
// asks: the first page with its total, and a count for every status tab.
func TestTheListsFirstScreen(t *testing.T) {
	needPerf(t)
	s := open(t, newSite(t))
	srv, err := serve.New(t.Context(), s, serve.Options{Admin: true})
	if err != nil {
		t.Fatal(err)
	}
	web := httptest.NewServer(srv.Handler())
	defer web.Close()

	urls := []string{web.URL + "/api/v1/contents?kind=post&limit=20&count=true"}
	for _, status := range []string{"", "published", "draft", "scheduled", "archived"} {
		u := web.URL + "/api/v1/contents?kind=post&limit=1&count=true"
		if status != "" {
			u += "&status=" + status
		}
		urls = append(urls, u)
	}
	screen := func() time.Duration {
		start := time.Now()
		var wg sync.WaitGroup
		for _, u := range urls {
			wg.Go(func() {
				resp, err := http.Get(u)
				if err != nil {
					t.Error(err)
					return
				}
				defer func() { _ = resp.Body.Close() }()
				if _, err := io.Copy(io.Discard, resp.Body); err != nil || resp.StatusCode != http.StatusOK {
					t.Errorf("GET %s: %d %v", u, resp.StatusCode, err)
				}
			})
		}
		wg.Wait()
		return time.Since(start)
	}

	measured(t, "list first screen, first time", screen(), listTarget)
	var times []time.Duration
	for range 9 {
		times = append(times, screen())
	}
	slices.Sort(times)
	measured(t, "list first screen, median of nine", times[len(times)/2], listTarget)
}

// running is a server with its watcher, as `kite run` starts one.
type running struct {
	base   string
	events <-chan string
}

func start(t *testing.T, root string) *running {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := listener.Addr().String()
	_ = listener.Close()

	srv, err := serve.New(t.Context(), open(t, root), serve.Options{
		Addr: addr, Watch: true, LiveReload: true, Admin: true, Drafts: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	go func() { _ = srv.ListenAndServe(ctx) }()

	base := "http://" + addr
	deadline := time.Now().Add(10 * time.Second)
	for {
		if resp, err := http.Get(base + "/"); err == nil {
			_ = resp.Body.Close()
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("the server did not come up")
		}
		time.Sleep(20 * time.Millisecond)
	}

	// The reload stream a served page listens on.
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, base+serve.ReloadPath, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	events := make(chan string, 64)
	go func() {
		defer func() { _ = resp.Body.Close() }()
		defer close(events)
		lines := bufio.NewScanner(resp.Body)
		for lines.Scan() {
			if name, ok := strings.CutPrefix(lines.Text(), "event: "); ok {
				events <- name
			}
		}
	}()
	r := &running{base: base, events: events}
	r.await(t, "hello", 5*time.Second)
	return r
}

func (r *running) await(t *testing.T, want string, within time.Duration) {
	t.Helper()
	timeout := time.After(within)
	for {
		select {
		case got, ok := <-r.events:
			if !ok {
				t.Fatal("the reload stream closed")
			}
			if got == want {
				return
			}
		case <-timeout:
			t.Fatalf("no %q event within %s", want, within)
		}
	}
}

func (r *running) get(t *testing.T, path string) string {
	t.Helper()
	resp, err := http.Get(r.base + path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	return string(body)
}

// until polls a check until it holds, returning how long that took.
func until(t *testing.T, within time.Duration, holds func() bool) time.Duration {
	t.Helper()
	start := time.Now()
	for !holds() {
		if time.Since(start) > within {
			t.Fatalf("still not so after %s", within)
		}
		time.Sleep(5 * time.Millisecond)
	}
	return time.Since(start)
}

// M1 and M3: a file edited in an editor, seen by a page that is open on it,
// and by the studio.
func TestAnEditReachesTheBrowserAndTheStudio(t *testing.T) {
	needPerf(t)
	root := newSite(t)
	r := start(t, root)

	var worst time.Duration
	for n, i := range []int{11, size() / 2, size() - 3} {
		title := fmt.Sprintf("Edited %d", n)
		began := time.Now()
		write(t, postFile(root, i), post(i, title))
		r.await(t, "reload", 5*time.Second)
		if page := r.get(t, fmt.Sprintf("/posts/post-%04d/", i)); !strings.Contains(page, title) {
			t.Fatalf("the reloaded page does not show the edit")
		}
		worst = max(worst, time.Since(began))
	}
	measured(t, "edit to a reloaded page, worst of three", worst, reloadTarget)

	i := size() / 3
	write(t, postFile(root, i), post(i, "Edited for the studio"))
	took := until(t, 2*studioTarget, func() bool {
		var item struct{ Title string }
		return json.Unmarshal([]byte(r.get(t, "/api/v1/contents/"+postID(i))), &item) == nil &&
			item.Title == "Edited for the studio"
	})
	measured(t, "edit to the studio", took, studioTarget)
}

func git(t *testing.T, root string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = root
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=Perf", "GIT_AUTHOR_EMAIL=perf@example.com",
		"GIT_COMMITTER_NAME=Perf", "GIT_COMMITTER_EMAIL=perf@example.com",
		"GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null",
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
}

// M2: a checkout while the studio is open, which has to reach the list
// quickly and re-read only what the checkout changed.
func TestABranchSwitchReachesTheList(t *testing.T) {
	needPerf(t)
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	root := newSite(t)
	git(t, root, "init", "-q", "-b", "main")
	git(t, root, "add", "-A")
	git(t, root, "commit", "-q", "-m", "posts")

	// The other branch changes ten posts, adds one and removes one.
	git(t, root, "checkout", "-q", "-b", "other")
	for i := range 10 {
		n := i * (size() / 10)
		write(t, postFile(root, n), post(n, fmt.Sprintf("Branch %d", i)))
	}
	write(t, postFile(root, size()), post(size(), "Added on the branch"))
	if err := os.RemoveAll(filepath.Dir(postFile(root, 1))); err != nil {
		t.Fatal(err)
	}
	git(t, root, "add", "-A")
	git(t, root, "commit", "-q", "-m", "branch")
	git(t, root, "checkout", "-q", "main")
	backdate(t, root)

	// Re-reading only what changed, measured on the index directly so the
	// count is exact.
	s, err := site.Open(t.Context(), root)
	if err != nil {
		t.Fatal(err)
	}
	git(t, root, "checkout", "-q", "other")
	stats, err := s.Index.Reconcile(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("checkout reconciled in %s: %d re-read, %d removed, %d untouched",
		stats.Duration.Round(time.Millisecond), stats.Indexed, stats.Removed, stats.Skipped)
	if stats.Indexed != 11 || stats.Removed != 1 {
		t.Errorf("a checkout of 11 changed files and 1 removal re-read %d and removed %d", stats.Indexed, stats.Removed)
	}
	git(t, root, "checkout", "-q", "main")
	if _, err := s.Index.Reconcile(t.Context()); err != nil {
		t.Fatal(err)
	}
	_ = s.Close()
	backdate(t, root)

	// And end to end, with the watcher noticing the checkout.
	r := start(t, root)
	began := time.Now()
	git(t, root, "checkout", "-q", "other")
	until(t, 2*branchTarget, func() bool {
		var list struct{ Items []struct{ Title string } }
		return json.Unmarshal([]byte(r.get(t, "/api/v1/contents?kind=post&q=Added+on+the+branch")), &list) == nil &&
			len(list.Items) == 1
	})
	measured(t, "checkout to the list", time.Since(began), branchTarget)
}
