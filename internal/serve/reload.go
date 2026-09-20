package serve

import (
	"bytes"
	"fmt"
	"html"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ReloadPath is where a page subscribes to rebuild notifications.
const ReloadPath = "/_kite/reload"

// reloadHub broadcasts "the site changed" to every open page.
//
// Server-sent events rather than a socket: the traffic is one way, the browser
// reconnects on its own, and there is no handshake to get wrong.
type reloadHub struct {
	mu      sync.Mutex
	clients map[chan struct{}]struct{}
	// generation lets a page that reconnects after a restart notice that it
	// missed a change, instead of waiting for the next one.
	generation int
}

func newReloadHub() *reloadHub {
	return &reloadHub{clients: make(map[chan struct{}]struct{})}
}

func (h *reloadHub) subscribe() (<-chan struct{}, func(), int) {
	ch := make(chan struct{}, 1)

	h.mu.Lock()
	h.clients[ch] = struct{}{}
	gen := h.generation
	h.mu.Unlock()

	return ch, func() {
		h.mu.Lock()
		delete(h.clients, ch)
		h.mu.Unlock()
	}, gen
}

// broadcast wakes every subscriber. A client that has not drained its previous
// notification does not need another one.
func (h *reloadHub) broadcast() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.generation++
	for ch := range h.clients {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
}

// handleReload streams change notifications to one page.
func (h *reloadHub) handleReload(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	events, unsubscribe, gen := h.subscribe()
	defer unsubscribe()

	// A client that has gone away fails every write; the stream ends on the
	// next loop when its context is done.
	_, _ = fmt.Fprintf(w, "event: hello\ndata: %d\n\n", gen)
	flusher.Flush()

	// A periodic comment keeps proxies from closing an idle stream.
	ping := time.NewTicker(25 * time.Second)
	defer ping.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-events:
			_, _ = fmt.Fprint(w, "event: reload\ndata: 1\n\n")
			flusher.Flush()
		case <-ping.C:
			_, _ = fmt.Fprint(w, ": ping\n\n")
			flusher.Flush()
		}
	}
}

// reloadScript is injected into served HTML. It is deliberately tiny and
// dependency free, because it has to run on a page whose own assets may be
// broken by the very edit that is about to be reloaded.
const reloadScript = `<script>
(function () {
  var es, retry = 0;
  function connect() {
    es = new EventSource(%q);
    es.addEventListener("reload", function () { location.reload(); });
    es.addEventListener("hello", function (e) {
      // A generation ahead of ours means a change was missed while the
      // connection was down, so catch up instead of waiting for the next one.
      var seen = sessionStorage.getItem("kite:gen");
      if (seen !== null && seen !== e.data) { location.reload(); return; }
      sessionStorage.setItem("kite:gen", e.data);
      retry = 0;
    });
    es.onerror = function () {
      es.close();
      retry = Math.min(retry + 1, 10);
      setTimeout(connect, retry * 200);
    };
  }
  connect();
})();
</script>
`

// problemBanner shows what the index refused to load.
//
// The indexer reports these rather than repairing them, which is right: only
// the author knows what a broken file was meant to say. But a preview is
// watched in a browser, not in a terminal, and a warning nobody reads is the
// same as no warning. A file that silently fails to appear is the worst
// outcome, because nothing tells the author where to look.
const problemBanner = `<div style="position:fixed;left:0;right:0;bottom:0;z-index:2147483647;` +
	`font:13px/1.5 ui-monospace,SFMono-Regular,Menlo,monospace;` +
	`background:#7f1d1d;color:#fee2e2;padding:10px 14px;` +
	`box-shadow:0 -1px 0 rgba(0,0,0,.25);max-height:40vh;overflow:auto">` +
	`<strong style="color:#fff">%d file(s) not indexed</strong>%s</div>`

func banner(problems []string) string {
	if len(problems) == 0 {
		return ""
	}
	var items strings.Builder
	for _, p := range problems {
		fmt.Fprintf(&items, `<div style="margin-top:4px;opacity:.9">%s</div>`, html.EscapeString(p))
	}
	return fmt.Sprintf(problemBanner, len(problems), items.String())
}

// inject places the reload script, and any problem banner, just before
// </body>, or appends them when the document has no body tag to aim at.
func inject(page []byte, problems []string) []byte {
	extra := []byte(banner(problems) + fmt.Sprintf(reloadScript, ReloadPath))

	for _, marker := range [][]byte{[]byte("</body>"), []byte("</BODY>")} {
		if i := bytes.LastIndex(page, marker); i >= 0 {
			out := make([]byte, 0, len(page)+len(extra))
			out = append(out, page[:i]...)
			out = append(out, extra...)
			out = append(out, page[i:]...)
			return out
		}
	}
	return append(page, extra...)
}

func contentLength(n int) string { return strconv.Itoa(n) }
