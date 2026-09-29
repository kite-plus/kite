package api_test

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/kite-plus/kite/internal/api"
)

// sources reads every post of a newProject fixture, by index.
func sources(t *testing.T, root string, n int) []string {
	t.Helper()
	out := make([]string, n)
	for i := range n {
		data, err := os.ReadFile(postPath(root, i))
		if err != nil {
			t.Fatal(err)
		}
		out[i] = string(data)
	}
	return out
}

func postPath(root string, i int) string {
	return filepath.Join(root, "content", "posts", fmt.Sprintf("post-%02d", i), "index.md")
}

func postID(i int) string { return fmt.Sprintf("01J8KQ2P3R4S5T6V7W8X9YZ%03d", i) }

func termPath(taxonomy, term string) string {
	return api.Prefix + "/taxonomies/" + taxonomy + "/terms/" + url.PathEscape(term)
}

// readTerm reads a term and the entity tag a change to it sends back.
func readTerm(t *testing.T, h http.Handler, taxonomy, term string) (api.TermDetail, string) {
	t.Helper()
	rec := send(t, h, http.MethodGet, termPath(taxonomy, term), nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET %s/%s: %d\n%s", taxonomy, term, rec.Code, rec.Body.String())
	}
	tag := rec.Header().Get("ETag")
	if tag == "" {
		t.Fatal("a term was read without an ETag, so a change to it has nothing to send back")
	}
	return decode[api.TermDetail](t, rec), tag
}

func TestATermListsEveryItemThatCarriesIt(t *testing.T) {
	h, _ := newWritableServer(t, newProject(t, 10))

	term, _ := readTerm(t, h, "tags", "Notes")
	if term.Term != "Notes" || term.Count != 5 || len(term.Items) != 5 || term.URL == "" {
		t.Fatalf("Notes = %+v, want 5 items and a URL", term)
	}
	for _, item := range term.Items {
		if !slices.Contains(item.Taxonomies["tags"], "Notes") {
			t.Errorf("%s is listed but does not carry Notes", item.ID)
		}
		if item.Locator == "" || item.Revision == "" {
			t.Errorf("%s does not say which file a change would touch", item.ID)
		}
	}

	get[api.ErrorBody](t, h, termPath("tags", "Nowhere"), http.StatusNotFound)
	get[api.ErrorBody](t, h, termPath("nonsense", "Notes"), http.StatusNotFound)
}

func TestRenamingATermRewritesOnlyItsLineOnEveryItemCarryingIt(t *testing.T) {
	root := newProject(t, 6)
	h, _ := newWritableServer(t, root)
	before := sources(t, root, 6)

	_, tag := readTerm(t, h, "tags", "Notes")
	rec := send(t, h, http.MethodPut, termPath("tags", "Notes"), api.TermRename{Name: " Journal "},
		map[string]string{"If-Match": tag})
	if rec.Code != http.StatusOK {
		t.Fatalf("rename: %d\n%s", rec.Code, rec.Body.String())
	}
	renamed := decode[api.TermDetail](t, rec)
	if renamed.Term != "Journal" || renamed.Count != 3 {
		t.Errorf("renamed term = %+v, want Journal on 3 items", renamed)
	}
	if rec.Header().Get("ETag") == "" {
		t.Error("the renamed term came back without an ETag")
	}

	after := sources(t, root, 6)
	for i := range after {
		changed := changedLines(before[i], after[i])
		if i%2 == 1 {
			if len(changed) != 0 {
				t.Errorf("post %d does not carry Notes but changed:\n%s", i, strings.Join(changed, "\n"))
			}
			continue
		}
		if len(changed) != 1 || !strings.Contains(changed[0], "+tags: [Go, Journal]") {
			t.Errorf("post %d changed %d lines, want only its tags:\n%s", i, len(changed), strings.Join(changed, "\n"))
		}
	}

	get[api.ErrorBody](t, h, termPath("tags", "Notes"), http.StatusNotFound)
	terms := get[api.List[api.TermCount]](t, h, api.Prefix+"/taxonomies/tags/terms", http.StatusOK)
	for _, tc := range terms.Items {
		if tc.Term == "Notes" {
			t.Error("the term list still counts Notes after the rename")
		}
	}
}

// Renaming a term to one that exists is how two terms become one, and an item
// that carried both must not end up with the survivor twice.
func TestRenamingIntoATermInUseMergesTheTwo(t *testing.T) {
	root := newProject(t, 4)
	h, _ := newWritableServer(t, root)

	_, tag := readTerm(t, h, "tags", "Notes")
	rec := send(t, h, http.MethodPut, termPath("tags", "Notes"), api.TermRename{Name: "Go"},
		map[string]string{"If-Match": tag})
	if rec.Code != http.StatusOK {
		t.Fatalf("merge: %d\n%s", rec.Code, rec.Body.String())
	}
	if merged := decode[api.TermDetail](t, rec); merged.Count != 4 {
		t.Errorf("Go is on %d items after the merge, want 4", merged.Count)
	}
	for i, source := range sources(t, root, 4) {
		if !strings.Contains(source, "tags: [Go]\n") {
			t.Errorf("post %d does not carry Go exactly once:\n%s", i, source)
		}
	}
}

func TestRemovingATermKeepsTheItemsAndDropsAnEmptyList(t *testing.T) {
	root := newProject(t, 4)
	h, _ := newWritableServer(t, root)

	_, tag := readTerm(t, h, "categories", "Tech")
	rec := send(t, h, http.MethodDelete, termPath("categories", "Tech"), nil, map[string]string{"If-Match": tag})
	if rec.Code != http.StatusNoContent {
		t.Fatalf("remove: %d\n%s", rec.Code, rec.Body.String())
	}
	for i, source := range sources(t, root, 4) {
		if strings.Contains(source, "categories") {
			t.Errorf("post %d kept an empty categories key:\n%s", i, source)
		}
		if !strings.Contains(source, "title: Post") {
			t.Errorf("post %d lost more than its term:\n%s", i, source)
		}
	}
	get[api.ErrorBody](t, h, termPath("categories", "Tech"), http.StatusNotFound)

	_, tag = readTerm(t, h, "tags", "Notes")
	if rec := send(t, h, http.MethodDelete, termPath("tags", "Notes"), nil,
		map[string]string{"If-Match": tag}); rec.Code != http.StatusNoContent {
		t.Fatalf("remove Notes: %d\n%s", rec.Code, rec.Body.String())
	}
	if source := sources(t, root, 1)[0]; !strings.Contains(source, "tags: [Go]\n") {
		t.Errorf("post 0 should keep Go alone:\n%s", source)
	}
}

// A term change is confirmed against a list of items, so it must not go on to
// change a list that is no longer the one that was shown.
func TestATermChangeAgainstAStaleReadChangesNothing(t *testing.T) {
	root := newProject(t, 4)
	h, _ := newWritableServer(t, root)

	_, stale := readTerm(t, h, "tags", "Notes")

	item, itemTag := load(t, h, postID(2))
	draft := draftOf(item)
	draft.Title = "Edited meanwhile"
	if rec := send(t, h, http.MethodPut, api.Prefix+"/contents/"+item.ID, draft,
		map[string]string{"If-Match": itemTag}); rec.Code != http.StatusOK {
		t.Fatalf("edit: %d\n%s", rec.Code, rec.Body.String())
	}
	before := sources(t, root, 4)

	rec := send(t, h, http.MethodPut, termPath("tags", "Notes"), api.TermRename{Name: "Journal"},
		map[string]string{"If-Match": stale})
	if rec.Code != http.StatusConflict {
		t.Fatalf("a rename against a stale read returned %d, want 409\n%s", rec.Code, rec.Body.String())
	}
	if code := decode[api.ErrorBody](t, rec).Error.Code; code != api.CodeConflict {
		t.Errorf("code = %q, want %q", code, api.CodeConflict)
	}
	if rec := send(t, h, http.MethodDelete, termPath("tags", "Notes"), nil,
		map[string]string{"If-Match": stale}); rec.Code != http.StatusConflict {
		t.Errorf("a removal against a stale read returned %d, want 409", rec.Code)
	}
	if after := sources(t, root, 4); !slices.Equal(before, after) {
		t.Error("a refused term change still changed files")
	}

	if rec := send(t, h, http.MethodPut, termPath("tags", "Notes"), api.TermRename{Name: "Journal"},
		nil); rec.Code != http.StatusPreconditionRequired {
		t.Errorf("a rename without If-Match returned %d, want 428", rec.Code)
	}
}

// A trashed item keeps its terms, so a term change that skipped it would come
// back the moment the item is restored.
func TestATermChangeReachesTheTrash(t *testing.T) {
	root := newProject(t, 4)
	h, _ := newWritableServer(t, root)

	_, itemTag := load(t, h, postID(2))
	if rec := send(t, h, http.MethodDelete, api.Prefix+"/contents/"+postID(2), nil,
		map[string]string{"If-Match": itemTag}); rec.Code != http.StatusNoContent {
		t.Fatalf("delete: %d", rec.Code)
	}

	term, tag := readTerm(t, h, "tags", "Notes")
	if term.Count != 1 || len(term.Items) != 2 {
		t.Fatalf("Notes = %d live of %d items, want 1 of 2", term.Count, len(term.Items))
	}
	for _, item := range term.Items {
		if item.Trashed != (item.ID == postID(2)) {
			t.Errorf("%s trashed = %v", item.ID, item.Trashed)
		}
	}

	if rec := send(t, h, http.MethodPut, termPath("tags", "Notes"), api.TermRename{Name: "Journal"},
		map[string]string{"If-Match": tag}); rec.Code != http.StatusOK {
		t.Fatalf("rename: %d\n%s", rec.Code, rec.Body.String())
	}
	source, err := os.ReadFile(postPath(root, 2))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(source), "tags: [Go, Journal]") || !strings.Contains(string(source), "deleted_at") {
		t.Errorf("the trashed post should carry the new name and stay trashed:\n%s", source)
	}
}

func TestATermNameMustBeUsable(t *testing.T) {
	h, _ := newWritableServer(t, newProject(t, 2))
	_, tag := readTerm(t, h, "tags", "Go")

	for _, name := range []string{"", "   ", "Go", "two\nlines", "-", " / -"} {
		rec := send(t, h, http.MethodPut, termPath("tags", "Go"), api.TermRename{Name: name},
			map[string]string{"If-Match": tag})
		if rec.Code != http.StatusBadRequest {
			t.Errorf("renaming to %q returned %d, want 400", name, rec.Code)
		}
	}
}

// A term is free text, so it can hold a slash; the path has to keep it in one
// segment.
func TestATermWithASlashIsOneTerm(t *testing.T) {
	h, _ := newWritableServer(t, newProject(t, 2))
	_, tag := readTerm(t, h, "tags", "Go")

	if rec := send(t, h, http.MethodPut, termPath("tags", "Go"), api.TermRename{Name: "C/C++"},
		map[string]string{"If-Match": tag}); rec.Code != http.StatusOK {
		t.Fatalf("rename: %d\n%s", rec.Code, rec.Body.String())
	}
	if term, _ := readTerm(t, h, "tags", "C/C++"); term.Count != 2 {
		t.Errorf("C/C++ is on %d items, want 2", term.Count)
	}
}

// spelled is a newProject of four posts whose tags write Go three ways: post
// 0 and 2 carry Go and Notes, post 1 carries go, and post 3 GO and Go.
func spelled(t *testing.T) string {
	t.Helper()
	root := newProject(t, 4)
	for i, tags := range map[int]string{1: "tags: [go]\n", 3: "tags: [GO, Go]\n"} {
		write(t, postPath(root, i), post(i, tags))
	}
	return root
}

// Go, go and GO share one page on the site, so the studio shows them as the
// one term the site does.
func TestATermIsEveryWayItIsWritten(t *testing.T) {
	h, _ := newWritableServer(t, spelled(t))

	for _, asked := range []string{"Go", "go", "GO"} {
		term, _ := readTerm(t, h, "tags", asked)
		if term.Term != "Go" || term.Count != 4 || len(term.Items) != 4 || !strings.HasSuffix(term.URL, "/tags/go/") {
			t.Errorf("%s = %s on %d of %d items at %s, want Go on 4 at /tags/go/",
				asked, term.Term, term.Count, len(term.Items), term.URL)
		}
		if !slices.Equal(term.Variants, []string{"GO", "go"}) {
			t.Errorf("%s is also written %v, want GO and go", asked, term.Variants)
		}
	}

	terms := get[api.List[api.TermCount]](t, h, api.Prefix+"/taxonomies/tags/terms", http.StatusOK)
	var names []string
	for _, tc := range terms.Items {
		names = append(names, fmt.Sprintf("%s:%d%v", tc.Term, tc.Count, tc.Variants))
	}
	if want := []string{"Go:4[GO go]", "Notes:2[]"}; !slices.Equal(names, want) {
		t.Errorf("terms = %v, want %v", names, want)
	}

	listed := get[api.List[api.Summary]](t, h, api.Prefix+"/contents?kind=post&term=tags:go", http.StatusOK)
	if len(listed.Items) != 4 {
		t.Errorf("filtering by tags:go lists %d posts, want 4", len(listed.Items))
	}
}

func TestRenamingATermWritesItsNewNameEveryWhereItIsWritten(t *testing.T) {
	root := spelled(t)
	h, _ := newWritableServer(t, root)

	_, tag := readTerm(t, h, "tags", "go")
	rec := send(t, h, http.MethodPut, termPath("tags", "go"), api.TermRename{Name: "Golang"},
		map[string]string{"If-Match": tag})
	if rec.Code != http.StatusOK {
		t.Fatalf("rename: %d\n%s", rec.Code, rec.Body.String())
	}
	if renamed := decode[api.TermDetail](t, rec); renamed.Term != "Golang" || renamed.Count != 4 {
		t.Errorf("renamed term = %s on %d items, want Golang on 4", renamed.Term, renamed.Count)
	}
	for i, want := range []string{"tags: [Golang, Notes]", "tags: [Golang]", "tags: [Golang, Notes]", "tags: [Golang]"} {
		if source := sources(t, root, 4)[i]; !strings.Contains(source, want+"\n") {
			t.Errorf("post %d does not carry %s:\n%s", i, want, source)
		}
	}
	get[api.ErrorBody](t, h, termPath("tags", "GO"), http.StatusNotFound)
}

// Renaming a term to its own name writes it one way, and leaves the items
// that already write it so as they are.
func TestRenamingATermToItsNameWritesItOneWay(t *testing.T) {
	root := spelled(t)
	h, _ := newWritableServer(t, root)
	before := sources(t, root, 4)

	_, tag := readTerm(t, h, "tags", "Go")
	rec := send(t, h, http.MethodPut, termPath("tags", "Go"), api.TermRename{Name: "Go"},
		map[string]string{"If-Match": tag})
	if rec.Code != http.StatusOK {
		t.Fatalf("rename: %d\n%s", rec.Code, rec.Body.String())
	}
	after := sources(t, root, 4)
	for i := range after {
		changed := changedLines(before[i], after[i])
		switch i {
		case 0, 2:
			if len(changed) != 0 {
				t.Errorf("post %d already wrote Go but changed:\n%s", i, strings.Join(changed, "\n"))
			}
		default:
			if len(changed) != 1 || !strings.Contains(changed[0], "+tags: [Go]") {
				t.Errorf("post %d changed %d lines, want its tags written as Go:\n%s",
					i, len(changed), strings.Join(changed, "\n"))
			}
		}
	}

	_, tag = readTerm(t, h, "tags", "go")
	rec = send(t, h, http.MethodPut, termPath("tags", "go"), api.TermRename{Name: "Go"},
		map[string]string{"If-Match": tag})
	if rec.Code != http.StatusBadRequest {
		t.Errorf("renaming a term written one way to that way returned %d, want 400", rec.Code)
	}
}

// Merging writes the term merged into as it was asked for on every item the
// merge changes, however each wrote it.
func TestMergingIntoATermWrittenAnotherWay(t *testing.T) {
	root := spelled(t)
	h, _ := newWritableServer(t, root)

	_, tag := readTerm(t, h, "tags", "Notes")
	rec := send(t, h, http.MethodPut, termPath("tags", "Notes"), api.TermRename{Name: "go"},
		map[string]string{"If-Match": tag})
	if rec.Code != http.StatusOK {
		t.Fatalf("merge: %d\n%s", rec.Code, rec.Body.String())
	}
	if merged := decode[api.TermDetail](t, rec); merged.Term != "go" || merged.Count != 4 {
		t.Errorf("merged term = %s on %d items, want go, now written so most, on 4", merged.Term, merged.Count)
	}
	for i, want := range []string{"tags: [go]", "tags: [go]", "tags: [go]", "tags: [GO, Go]"} {
		if source := sources(t, root, 4)[i]; !strings.Contains(source, want+"\n") {
			t.Errorf("post %d does not carry %s:\n%s", i, want, source)
		}
	}
}

func TestRemovingATermRemovesEveryWayItIsWritten(t *testing.T) {
	root := spelled(t)
	h, _ := newWritableServer(t, root)

	_, tag := readTerm(t, h, "tags", "GO")
	if rec := send(t, h, http.MethodDelete, termPath("tags", "GO"), nil,
		map[string]string{"If-Match": tag}); rec.Code != http.StatusNoContent {
		t.Fatalf("remove: %d\n%s", rec.Code, rec.Body.String())
	}
	for i, source := range sources(t, root, 4) {
		want := i%2 == 0 // posts 0 and 2 keep Notes
		if strings.Contains(source, "tags: [Notes]\n") != want || strings.Contains(strings.ToLower(source), "go]") {
			t.Errorf("post %d kept a way of writing Go, or lost Notes:\n%s", i, source)
		}
	}
}
