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

	for _, name := range []string{"", "   ", "Go", "two\nlines"} {
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
