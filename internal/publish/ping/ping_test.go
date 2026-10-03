package ping_test

import (
	"context"
	"encoding/xml"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/kite-plus/kite/internal/publish/ping"
)

// call is a ping as a service reads it.
type call struct {
	Method string   `xml:"methodName"`
	Params []string `xml:"params>param>value>string"`
}

// answer is a methodResponse whose struct holds flerror and message.
func answer(flerror bool, message string) string {
	flag := "0"
	if flerror {
		flag = "1"
	}
	return `<?xml version="1.0"?>
<methodResponse><params><param><value><struct>
  <member><name>flerror</name><value><boolean>` + flag + `</boolean></value></member>
  <member><name>message</name><value>` + message + `</value></member>
</struct></value></param></params></methodResponse>`
}

// service answers every ping with status and body, and keeps what it was sent.
func service(t *testing.T, status int, body string) (*httptest.Server, *[]call) {
	t.Helper()
	var calls []call
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.Header.Get("Content-Type") != "text/xml" {
			t.Errorf("a ping came as %s with Content-Type %q", r.Method, r.Header.Get("Content-Type"))
		}
		data, _ := io.ReadAll(r.Body)
		var c call
		if err := xml.Unmarshal(data, &c); err != nil {
			t.Errorf("the ping is not XML-RPC: %v\n%s", err, data)
		}
		calls = append(calls, c)
		w.Header().Set("Content-Type", "text/xml")
		w.WriteHeader(status)
		_, _ = io.WriteString(w, body)
	}))
	t.Cleanup(srv.Close)
	return srv, &calls
}

var site = ping.Site{
	Title: "Notes & <Sketches>",
	URL:   "https://blog.example.com",
	Feed:  "https://blog.example.com/rss.xml",
}

// A site with a feed names it with extendedPing: the site's name, its address
// twice, once as the page that changed, and the feed. A service that takes
// the ping says so with flerror false.
func TestAPingNamesTheSiteAndItsFeed(t *testing.T) {
	srv, calls := service(t, http.StatusOK, answer(false, "Thanks for the ping."))
	message, err := ping.Send(t.Context(), srv.Client(), srv.URL, site)
	if err != nil || message != "Thanks for the ping." {
		t.Fatalf("Send = %q, %v", message, err)
	}
	want := call{
		Method: "weblogUpdates.extendedPing",
		Params: []string{site.Title, site.URL, site.URL, site.Feed},
	}
	if len(*calls) != 1 || (*calls)[0].Method != want.Method || !slices.Equal((*calls)[0].Params, want.Params) {
		t.Errorf("the service was sent %+v, want %+v", *calls, want)
	}
}

// A site with no feed has none to name, and sends the plain ping.
func TestASiteWithNoFeedSendsThePlainPing(t *testing.T) {
	srv, calls := service(t, http.StatusOK, answer(false, "ok"))
	plain := ping.Site{Title: "Notes", URL: "https://blog.example.com"}
	if _, err := ping.Send(t.Context(), srv.Client(), srv.URL, plain); err != nil {
		t.Fatal(err)
	}
	if c := (*calls)[0]; c.Method != "weblogUpdates.ping" || !slices.Equal(c.Params, []string{"Notes", "https://blog.example.com"}) {
		t.Errorf("the service was sent %+v", c)
	}
}

// A ping fails when the service cannot take it, and says why.
func TestAPingThatIsNotTakenFails(t *testing.T) {
	fault := `<?xml version="1.0"?>
<methodResponse><fault><value><struct>
  <member><name>faultCode</name><value><int>-32602</int></value></member>
  <member><name>faultString</name><value><string>wrong parameters</string></value></member>
</struct></value></fault></methodResponse>`

	for _, tc := range []struct {
		name   string
		status int
		body   string
		want   string
	}{
		{"flerror", http.StatusOK, answer(true, "This blog is not listed."), "This blog is not listed."},
		{"fault", http.StatusOK, fault, "fault -32602: wrong parameters"},
		{"an HTTP error", http.StatusServiceUnavailable, "busy", "answered 503 Service Unavailable"},
		{"no XML-RPC", http.StatusOK, "<html><body>Hello</body></html>", "not an XML-RPC response"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			srv, _ := service(t, tc.status, tc.body)
			message, err := ping.Send(t.Context(), srv.Client(), srv.URL, site)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Errorf("Send = %q, %v; want an error saying %q", message, err, tc.want)
			}
		})
	}
}

// A service that does not answer in time fails the ping rather than holding
// up the rest.
func TestAPingThatIsNotAnsweredInTimeFails(t *testing.T) {
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		select {
		case <-r.Context().Done():
		case <-release:
		}
	}))
	t.Cleanup(srv.Close)
	t.Cleanup(func() { close(release) })

	ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
	defer cancel()
	start := time.Now()
	_, err := ping.Send(ctx, srv.Client(), srv.URL, site)
	if err == nil || !strings.Contains(err.Error(), "no answer within") {
		t.Errorf("Send = %v, want it to say there was no answer", err)
	}
	if took := time.Since(start); took > 5*time.Second {
		t.Errorf("Send waited %s", took)
	}
}
