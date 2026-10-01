package apps

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// indexServer serves an index with an ETag, counting what it is asked.
type indexServer struct {
	srv           *httptest.Server
	body          atomic.Value
	full, revalid atomic.Int32
	down          atomic.Bool
}

func newIndexServer(t *testing.T, ix *Index) *indexServer {
	s := &indexServer{}
	s.set(t, ix)
	s.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.down.Load() {
			http.Error(w, "down", http.StatusServiceUnavailable)
			return
		}
		body := s.body.Load().([]byte)
		tag := `"` + sum(body)[:16] + `"`
		if r.Header.Get("If-None-Match") == tag {
			s.revalid.Add(1)
			w.WriteHeader(http.StatusNotModified)
			return
		}
		s.full.Add(1)
		w.Header().Set("ETag", tag)
		_, _ = w.Write(body)
	}))
	t.Cleanup(s.srv.Close)
	return s
}

func (s *indexServer) set(t *testing.T, ix *Index) {
	data, err := json.Marshal(ix)
	if err != nil {
		t.Fatal(err)
	}
	s.body.Store(data)
}

func oneApp(versions ...string) *Index {
	a := &App{Kind: "theme", ID: "paper", Repo: "someone/paper", Title: map[string]string{"en": "Paper"}}
	for _, v := range versions {
		a.Versions = append(a.Versions, &Release{Version: v, API: "kite/v1"})
	}
	return &Index{Format: Format, Apps: []*App{a}}
}

func TestTheIndexIsKeptForAnHourAndAskedForAgainByItsTag(t *testing.T) {
	s := newIndexServer(t, oneApp("1.0.0"))
	now := time.Date(2026, 10, 1, 8, 0, 0, 0, time.UTC)
	c := NewClient([]string{s.srv.URL + "/index.json"}, t.TempDir())
	c.Now = func() time.Time { return now }

	ix, got, err := c.Index(t.Context(), false)
	if err != nil || ix.Find("theme", "paper") == nil || got.Offline || !got.At.Equal(now) {
		t.Fatalf("first read: %+v %+v %v", ix, got, err)
	}
	now = now.Add(59 * time.Minute)
	if _, _, err := c.Index(t.Context(), false); err != nil || s.full.Load() != 1 || s.revalid.Load() != 0 {
		t.Errorf("within the hour it asked again: %d full, %d revalidated, %v", s.full.Load(), s.revalid.Load(), err)
	}
	now = now.Add(2 * time.Minute)
	if _, got, err := c.Index(t.Context(), false); err != nil || s.full.Load() != 1 || s.revalid.Load() != 1 || !got.At.Equal(now) {
		t.Errorf("after the hour: %d full, %d revalidated, %+v, %v", s.full.Load(), s.revalid.Load(), got, err)
	}
	s.set(t, oneApp("1.1.0", "1.0.0"))
	ix, _, err = c.Index(t.Context(), true)
	if err != nil || s.full.Load() != 2 || ix.Find("theme", "paper").Versions[0].Version != "1.1.0" {
		t.Errorf("a refresh did not fetch the new index: %d full, %v", s.full.Load(), err)
	}
}

func TestTheNextAddressAnswersWhenOneDoesNot(t *testing.T) {
	down := newIndexServer(t, oneApp("1.0.0"))
	down.down.Store(true)
	up := newIndexServer(t, oneApp("1.0.0"))
	c := NewClient([]string{down.srv.URL + "/index.json", up.srv.URL + "/index.json"}, "")
	if _, got, err := c.Index(t.Context(), false); err != nil || got.URL != up.srv.URL+"/index.json" {
		t.Errorf("Index = %+v, %v", got, err)
	}
}

func TestOfflineTheKeptIndexIsUsedAndSaysHowOld(t *testing.T) {
	s := newIndexServer(t, oneApp("1.0.0"))
	cache := t.TempDir()
	c := NewClient([]string{s.srv.URL + "/index.json"}, cache)
	if _, _, err := c.Index(t.Context(), false); err != nil {
		t.Fatal(err)
	}
	s.down.Store(true)
	ix, got, err := c.Index(t.Context(), true)
	if err != nil || ix.Find("theme", "paper") == nil || !got.Offline || got.Err == nil {
		t.Errorf("offline: %+v, %v", got, err)
	}

	// A kept copy of another index is not this one.
	other := NewClient([]string{"https://example.invalid/index.json"}, cache)
	other.HTTP = s.srv.Client()
	if _, _, err := other.Index(t.Context(), false); err == nil {
		t.Error("another index was read from this one's copy")
	}
	if _, _, err := NewClient([]string{s.srv.URL}, t.TempDir()).Index(t.Context(), false); err == nil ||
		!strings.Contains(err.Error(), "could not be read") {
		t.Errorf("no answer and no copy: %v", err)
	}
}

func TestAnArchiveIsTheBytesItsChecksumNames(t *testing.T) {
	good := []byte("PK the archive")
	var fetches atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fetches.Add(1)
		switch r.URL.Path {
		case "/tampered.zip":
			_, _ = w.Write([]byte("PK something else"))
		case "/good.zip":
			_, _ = w.Write(good)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	cache := t.TempDir()
	c := NewClient(nil, cache)
	rel := &Release{Version: "1.0.0", Archive: Archive{
		URLs:   []string{srv.URL + "/missing.zip", srv.URL + "/tampered.zip", srv.URL + "/good.zip"},
		SHA256: sum(good),
		Size:   int64(len(good)),
	}}
	data, err := c.Archive(t.Context(), rel)
	if err != nil || string(data) != string(good) {
		t.Fatalf("Archive = %q, %v", data, err)
	}
	if kept, err := os.ReadFile(filepath.Join(cache, "archives", sum(good)+".zip")); err != nil || string(kept) != string(good) {
		t.Errorf("kept %q, %v", kept, err)
	}
	srv.Close()
	if data, err := c.Archive(t.Context(), rel); err != nil || string(data) != string(good) || fetches.Load() != 3 {
		t.Errorf("the kept archive was not used: %v, %d fetches", err, fetches.Load())
	}

	rel.Archive.URLs = rel.Archive.URLs[1:2]
	rel.Archive.SHA256 = sum([]byte("PK yet another"))
	if _, err := c.Archive(t.Context(), rel); err == nil || !strings.Contains(err.Error(), "could not be fetched") {
		t.Errorf("no address served the archive: %v", err)
	}
	rel.Archive.SHA256 = "not a checksum"
	if _, err := c.Archive(t.Context(), rel); err == nil || !strings.Contains(err.Error(), "no sha256") {
		t.Errorf("a version without a checksum: %v", err)
	}
}

func TestTheReleaseInstalledIsTheNewestThatFits(t *testing.T) {
	a := &App{Kind: "theme", ID: "paper", Versions: []*Release{
		{Version: "3.0.0", API: "kite/v2"},
		{Version: "2.0.0", API: "kite/v1", Requires: ">=0.2.0"},
		{Version: "1.2.0", API: "kite/v1", Yanked: true},
		{Version: "1.1.0", API: "kite/v1", Requires: ">=0.1.0 <2.0.0"},
		{Version: "1.0.0", API: "kite/v1"},
	}}
	pick := func(want, kite string) string {
		r, err := a.Pick(want, kite)
		if err != nil {
			return "error: " + err.Error()
		}
		return r.Version
	}
	for _, c := range []struct{ want, kite, got string }{
		{"", "0.1.4", "1.1.0"},
		{"", "0.2.0", "2.0.0"},
		{"", "dev", "2.0.0"},
		{"1.0.0", "0.1.4", "1.0.0"},
		{"v1.0.0", "0.1.4", "1.0.0"},
		{"2.0.0", "0.1.4", "error: theme paper requires Kite >=0.2.0"},
		{"1.2.0", "0.1.4", "error: theme paper 1.2.0 is yanked"},
		{"3.0.0", "0.1.4", "error: theme paper 3.0.0 is written to kite/v2"},
		{"9.9.9", "0.1.4", "error: theme paper has no version 9.9.9; the index lists 3.0.0, 2.0.0, 1.2.0, 1.1.0, 1.0.0"},
	} {
		if got := pick(c.want, c.kite); !strings.HasPrefix(got, c.got) {
			t.Errorf("Pick(%q) on Kite %s = %q, want %q", c.want, c.kite, got, c.got)
		}
	}

	none := &App{Kind: "plugin", ID: "late", Versions: []*Release{{Version: "1.0.0", API: "kite/plugin/v1", Requires: ">=9.0.0"}}}
	if _, err := none.Pick("", "0.1.4"); err == nil || !strings.Contains(err.Error(), "no version of plugin late works with this Kite") {
		t.Errorf("nothing fits: %v", err)
	}
	a.Delisted = "It was abandoned."
	if _, err := a.Pick("", "0.1.4"); err == nil || !strings.Contains(err.Error(), "no longer listed: It was abandoned.") {
		t.Errorf("delisted: %v", err)
	}
}

func TestAPackageIsKnownByItsRepositoryAndItsWords(t *testing.T) {
	a := &App{Kind: "theme", ID: "vane", Repo: "kite-plus/theme-vane",
		Title:       map[string]string{"en": "Vane", "zh-CN": "风标"},
		Description: map[string]string{"en": "A documentation theme."},
		Tags:        []string{"docs"}}
	for home, want := range map[string]bool{
		"https://github.com/kite-plus/theme-vane":     true,
		"https://github.com/Kite-Plus/theme-vane/":    true,
		"https://github.com/kite-plus/theme-vane.git": true,
		"https://github.com/someone/theme-vane":       false,
		"https://example.com/kite-plus/theme-vane":    false,
		"": false,
		"https://github.com/kite-plus/theme-vane-fork": false,
	} {
		if a.ProjectOf(home) != want {
			t.Errorf("ProjectOf(%q) = %v", home, !want)
		}
	}
	for words, want := range map[string]bool{"": true, "风标": true, "DOCUMENTATION theme": true, "docs vane": true, "blog": false} {
		if a.Matches(strings.Fields(words)) != want {
			t.Errorf("Matches(%q) = %v", words, !want)
		}
	}
	if Text(a.Title, "zh-CN") != "风标" || Text(a.Description, "zh-CN") != "A documentation theme." {
		t.Error("Text does not fall back to English")
	}
}
