package api_test

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/kite-plus/kite/internal/api"
)

// bundleWithFiles is a project whose first post keeps files beside it: one
// at the top of its bundle, one in a folder, and what is not the post's to
// list, its source, a hidden file and another post's bundle in a folder.
func bundleWithFiles(t *testing.T) (root, dir string) {
	t.Helper()
	root = newProject(t, 1)
	dir = filepath.Join(root, "content", "posts", "post-00")
	write(t, filepath.Join(dir, "river.jpg"), "a picture")
	write(t, filepath.Join(dir, "images", "map.png"), "a map")
	write(t, filepath.Join(dir, ".DS_Store"), "junk")
	write(t, filepath.Join(dir, "draft.md"), "a source")
	write(t, filepath.Join(dir, "sub", "index.md"),
		"---\nid: 01J8KQ2P3R4S5T6V7W8X9YZ901\ntitle: Inside\nslug: inside\nstatus: published\n---\n\nInside.\n")
	write(t, filepath.Join(dir, "sub", "other.jpg"), "another post's")
	return root, dir
}

func mediaPath(id, name string) string {
	return api.Prefix + "/contents/" + id + "/media/" + url.PathEscape(name)
}

func sendFile(t *testing.T, h http.Handler, method, target, filename string, data []byte) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	form := multipart.NewWriter(&buf)
	part, err := form.CreateFormFile("file", filename)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(data); err != nil {
		t.Fatal(err)
	}
	if err := form.Close(); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(method, target, &buf)
	req.Header.Set("Content-Type", form.FormDataContentType())
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

// An item's files are listed the way its page publishes them: by their path
// within its bundle, with the link the markdown writes and where the site
// serves each. A post kept as a single file keeps none of its own.
func TestAnItemsFilesAreListedAsItsPageHasThem(t *testing.T) {
	root, _ := bundleWithFiles(t)
	h, _ := newServer(t, root)
	list := get[api.List[api.Summary]](t, h, api.Prefix+"/contents?kind=post&limit=5", http.StatusOK)
	var item api.Summary
	for _, one := range list.Items {
		if one.Slug == "post-00" {
			item = one
		}
	}

	files := get[api.MediaList](t, h, api.Prefix+"/contents/"+item.ID+"/media", http.StatusOK)
	if !files.Bundle {
		t.Error("a post kept as a bundle says it keeps no files")
	}
	var names []string
	for _, f := range files.Items {
		names = append(names, f.Name)
		if f.Link != f.Name || f.URL != item.URL+f.Name {
			t.Errorf("%s: link %q and url %q, want the name and %q", f.Name, f.Link, f.URL, item.URL+f.Name)
		}
	}
	if !slices.Equal(names, []string{"images/map.png", "river.jpg"}) {
		t.Errorf("files = %q, want the picture and the map only", names)
	}
	if files.Items[1].Size != len("a picture") || files.Items[1].Type != "image/jpeg" {
		t.Errorf("river.jpg: size %d, type %q", files.Items[1].Size, files.Items[1].Type)
	}

	page := get[api.MediaList](t, h, api.Prefix+"/contents/01J8KQ2P3R4S5T6V7W8X9YZ900/media", http.StatusOK)
	if page.Bundle || len(page.Items) != 0 {
		t.Errorf("a page kept as a single file lists %+v", page)
	}
}

// A file is replaced where it is, under its name, in a folder of the bundle
// too, so every link to it still reaches it; the new version loses where a
// photo was taken, as any upload does.
func TestAFileIsReplacedWhereItIs(t *testing.T) {
	root, dir := bundleWithFiles(t)
	h, _ := newWritableServer(t, root)
	list := get[api.List[api.Summary]](t, h, api.Prefix+"/contents?kind=post&limit=5", http.StatusOK)
	id := list.Items[0].ID
	for _, one := range list.Items {
		if one.Slug == "post-00" {
			id = one.ID
		}
	}

	rec := sendFile(t, h, http.MethodPut, mediaPath(id, "images/map.png"), "new-map.png", []byte("a newer map"))
	if rec.Code != http.StatusOK {
		t.Fatalf("replace: status %d\n%s", rec.Code, rec.Body.String())
	}
	media := decode[api.Media](t, rec)
	if media.Name != "images/map.png" || media.Link != "images/map.png" {
		t.Errorf("replaced as %q linked %q, want the name it had", media.Name, media.Link)
	}
	if data, _ := os.ReadFile(filepath.Join(dir, "images", "map.png")); string(data) != "a newer map" {
		t.Errorf("images/map.png holds %q", data)
	}
	if _, err := os.Stat(filepath.Join(dir, "new-map.png")); !os.IsNotExist(err) {
		t.Error("the new version was stored under its own name as well")
	}

	photo := photoWithPlace(t)
	rec = sendFile(t, h, http.MethodPut, mediaPath(id, "river.jpg"), "IMG_0001.JPEG", photo)
	if rec.Code != http.StatusOK || !decode[api.Media](t, rec).LocationRemoved {
		t.Errorf("replacing with a photo: status %d, %s", rec.Code, rec.Body.String())
	}
	if data, _ := os.ReadFile(filepath.Join(dir, "river.jpg")); bytes.Contains(data, []byte{0, 0, 0, 31, 0, 0, 0, 1, 0, 0, 0, 13}) {
		t.Error("the photo's place was stored")
	}

	for _, tc := range []struct {
		name, file string
		want       int
	}{
		{"missing.jpg", "a.jpg", http.StatusNotFound},
		{"river.jpg", "river.png", http.StatusUnsupportedMediaType},
		{"sub/other.jpg", "other.jpg", http.StatusNotFound},
	} {
		if rec := sendFile(t, h, http.MethodPut, mediaPath(id, tc.name), tc.file, []byte("x")); rec.Code != tc.want {
			t.Errorf("replace %s with %s: status %d, want %d", tc.name, tc.file, rec.Code, tc.want)
		}
	}
}

// A file is removed by its path within the bundle, in a folder too; a path
// that leaves the bundle, names its source or reaches into another item's
// bundle removes nothing.
func TestAFileIsRemovedByItsPathInTheBundle(t *testing.T) {
	root, dir := bundleWithFiles(t)
	h, _ := newWritableServer(t, root)
	list := get[api.List[api.Summary]](t, h, api.Prefix+"/contents?kind=post&limit=5", http.StatusOK)
	var id string
	for _, one := range list.Items {
		if one.Slug == "post-00" {
			id = one.ID
		}
	}

	for _, name := range []string{"../post-00/index.md", "index.md", "sub/other.jpg", ".DS_Store", "/etc/passwd"} {
		req := httptest.NewRequest(http.MethodDelete, mediaPath(id, name), nil)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("delete %s: status %d, want 400\n%s", name, rec.Code, rec.Body.String())
		}
	}
	for _, kept := range []string{"index.md", "sub/other.jpg", ".DS_Store", "river.jpg"} {
		if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(kept))); err != nil {
			t.Errorf("%s is gone: %v", kept, err)
		}
	}

	req := httptest.NewRequest(http.MethodDelete, mediaPath(id, "images/map.png"), nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("delete images/map.png: status %d\n%s", rec.Code, rec.Body.String())
	}
	if _, err := os.Stat(filepath.Join(dir, "images", "map.png")); !os.IsNotExist(err) {
		t.Error("images/map.png is still there")
	}
}
