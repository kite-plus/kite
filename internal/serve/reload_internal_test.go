package serve

import (
	"bufio"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// A page stores the generation a reload names, and reloads again whenever it
// is greeted with another one. Unless the two agree, a single change sends
// every open page into reloading forever.
func TestAReloadNamesTheGenerationThePageIsGreetedWith(t *testing.T) {
	hub := newReloadHub()
	srv := httptest.NewServer(http.HandlerFunc(hub.handleReload))
	// Registered first so it runs last, once the streams it waits for are closed.
	t.Cleanup(srv.Close)

	// A change made before the page connected, so the count is under way.
	hub.broadcast()

	page := listen(t, srv.URL)
	next(t, page, "hello")
	hub.broadcast()
	sent := next(t, page, "reload")

	if greeted := next(t, listen(t, srv.URL), "hello"); greeted != sent {
		t.Errorf("a reload to generation %s is greeted with %s", sent, greeted)
	}
}

func listen(t *testing.T, url string) *bufio.Reader {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, url, nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })
	return bufio.NewReader(resp.Body)
}

// next reads up to the next event of a kind and returns its data.
func next(t *testing.T, stream *bufio.Reader, event string) string {
	t.Helper()
	found := false
	for {
		line, err := stream.ReadString('\n')
		if err != nil {
			t.Fatalf("waiting for %s: %v", event, err)
		}
		line = strings.TrimSuffix(line, "\n")
		switch {
		case line == "event: "+event:
			found = true
		case found && strings.HasPrefix(line, "data: "):
			return strings.TrimPrefix(line, "data: ")
		}
	}
}
