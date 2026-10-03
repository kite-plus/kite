package cli

import (
	"encoding/json"
	"encoding/xml"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/kite-plus/kite/internal/project"
)

// pingCall is a ping as an update service reads it.
type pingCall struct {
	Method string   `xml:"methodName"`
	Params []string `xml:"params>param>value>string"`
}

// pingService is an update service that answers every ping with status, and
// keeps what it was sent.
func pingService(t *testing.T, status int) (string, *[]pingCall) {
	t.Helper()
	var calls []pingCall
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, _ := io.ReadAll(r.Body)
		var c pingCall
		if err := xml.Unmarshal(data, &c); err != nil {
			t.Errorf("the ping is not XML-RPC: %v\n%s", err, data)
		}
		calls = append(calls, c)
		w.WriteHeader(status)
		_, _ = io.WriteString(w, `<?xml version="1.0"?><methodResponse><params><param><value><struct>`+
			`<member><name>flerror</name><value><boolean>0</boolean></value></member>`+
			`<member><name>message</name><value>Thanks for the ping.</value></member>`+
			`</struct></value></param></params></methodResponse>`)
	}))
	t.Cleanup(srv.Close)
	return srv.URL, &calls
}

// editConfig rewrites a project's kite.yaml.
func editConfig(t *testing.T, root string, edit func(string) string) {
	t.Helper()
	file := filepath.Join(root, project.ConfigName)
	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, []byte(edit(string(data))), 0o644); err != nil {
		t.Fatal(err)
	}
}

// withPings lists update services under publish.ping.
func withPings(t *testing.T, root string, endpoints ...string) {
	t.Helper()
	editConfig(t, root, func(config string) string {
		config += "\npublish:\n  ping:\n"
		for _, e := range endpoints {
			config += "    - " + e + "\n"
		}
		return config
	})
}

// After a deploy the workflow tells each service where the site is, at the
// address the deploy gave it, which KITE_SITE_BASEURL carries as it does for
// a build.
func TestPingTellsEachServiceWhereTheSiteIs(t *testing.T) {
	t.Setenv("KITE_SITE_BASEURL", "https://owner.github.io/notes")
	root := newSite(t)
	endpoint, calls := pingService(t, http.StatusOK)
	withPings(t, root, endpoint)

	out := runKite(t, root, "ping")
	if !strings.Contains(out, "pinged "+endpoint+": Thanks for the ping.") {
		t.Errorf("kite ping said:\n%s", out)
	}
	const site = "https://owner.github.io/notes"
	want := pingCall{Method: "weblogUpdates.extendedPing", Params: []string{"Due", site, site, site + "/rss.xml"}}
	if len(*calls) != 1 || (*calls)[0].Method != want.Method || !slices.Equal((*calls)[0].Params, want.Params) {
		t.Errorf("the service was sent %+v, want %+v", *calls, want)
	}
}

// A service that cannot be told is said so, the rest are told all the same,
// and the command fails, which the deploy workflow does not let fail a
// deploy.
func TestAFailedPingFailsOnceEveryServiceIsTried(t *testing.T) {
	root := newSite(t)
	down, _ := pingService(t, http.StatusInternalServerError)
	up, calls := pingService(t, http.StatusOK)
	withPings(t, root, down, up)

	out, err := tryKite(t, root, "ping")
	if err == nil || err.Error() != "1 of 2 ping(s) failed" {
		t.Errorf("kite ping = %v, want 1 of 2 failed", err)
	}
	if !strings.Contains(out, "could not ping "+down+": answered 500 Internal Server Error") ||
		!strings.Contains(out, "pinged "+up) {
		t.Errorf("kite ping said:\n%s", out)
	}
	if len(*calls) != 1 {
		t.Errorf("the service after the failed one was pinged %d times", len(*calls))
	}

	out, err = tryKite(t, root, "ping", "--json")
	if err == nil {
		t.Error("kite ping --json succeeded with a ping failed")
	}
	var report pingReport
	if err := json.Unmarshal([]byte(out), &report); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	if len(report.Pings) != 2 || report.Pings[0].OK || !report.Pings[1].OK ||
		report.Method != "weblogUpdates.extendedPing" || report.Feed != "https://example.com/rss.xml" {
		t.Errorf("report = %+v", report)
	}
}

// A site that lists no update service, as every site made before pings, has
// nothing to ping, which is no failure, and needs no address for it.
func TestPingWithNothingToPingSucceeds(t *testing.T) {
	t.Setenv("KITE_SITE_BASEURL", "")
	root := t.TempDir()
	if _, err := create(t.Context(), plan{Root: root, Title: "Nowhere", Language: "en"}); err != nil {
		t.Fatal(err)
	}
	if out := runKite(t, root, "ping"); !strings.Contains(out, "nothing to ping") {
		t.Errorf("kite ping said:\n%s", out)
	}
	var report pingReport
	if err := json.Unmarshal([]byte(runKite(t, root, "ping", "--json")), &report); err != nil {
		t.Fatal(err)
	}
	if report.Pings == nil || len(report.Pings) != 0 {
		t.Errorf("report = %+v, want an empty list of pings", report)
	}
}

// A ping names the site by its address, so one with none asks no service.
// A site that writes no feed has none to name, and sends the plain ping.
func TestPingNeedsAnAddressAndNamesAFeedOnlyIfThereIsOne(t *testing.T) {
	t.Setenv("KITE_SITE_BASEURL", "")
	root := t.TempDir()
	if _, err := create(t.Context(), plan{Root: root, Title: "Nowhere", Language: "en"}); err != nil {
		t.Fatal(err)
	}
	endpoint, calls := pingService(t, http.StatusOK)
	withPings(t, root, endpoint)

	if _, err := tryKite(t, root, "ping"); err == nil || !strings.Contains(err.Error(), "site.baseURL is empty") {
		t.Errorf("kite ping = %v, want a refusal naming site.baseURL", err)
	}
	if len(*calls) != 0 {
		t.Errorf("a site with no address was pinged for: %+v", *calls)
	}

	t.Setenv("KITE_SITE_BASEURL", "https://example.com")
	editConfig(t, root, func(config string) string {
		return strings.Replace(config, "  output: public\n", "  output: public\n  feed: false\n", 1)
	})
	runKite(t, root, "ping")
	want := pingCall{Method: "weblogUpdates.ping", Params: []string{"Nowhere", "https://example.com"}}
	if len(*calls) != 1 || (*calls)[0].Method != want.Method || !slices.Equal((*calls)[0].Params, want.Params) {
		t.Errorf("the service was sent %+v, want %+v", *calls, want)
	}
}
