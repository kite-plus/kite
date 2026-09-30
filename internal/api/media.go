package api

import (
	"io"
	"mime"
	"net/http"
	"path"
	"strings"

	"github.com/kite-plus/kite/internal/content"
	"github.com/kite-plus/kite/internal/render/img"
)

// maxUpload bounds one uploaded file.
const maxUpload = 32 << 20

// allowedMedia is what an author may drop into a bundle.
//
// A page bundle sits inside the content directory and is published beside the
// page, so an upload is a file that will be served from the site's own origin.
// An open list would make it a place to host anything.
var allowedMedia = map[string]bool{
	".png": true, ".jpg": true, ".jpeg": true, ".gif": true, ".webp": true,
	".avif": true, ".svg": true, ".ico": true,
	".mp4": true, ".webm": true, ".mp3": true, ".m4a": true, ".ogg": true,
	".pdf": true, ".txt": true, ".csv": true, ".json": true,
	".woff": true, ".woff2": true,
}

// handleUpload puts a file into an item's bundle and reports the link that
// reaches it.
func (s *Server) handleUpload(w http.ResponseWriter, r *http.Request) {
	view := s.src()
	if !s.writable(w, view) {
		return
	}
	owner, err := view.Reader.Get(r.Context(), content.ID(r.PathValue("id")))
	if err != nil {
		s.failErr(w, err)
		return
	}
	s.upload(w, r, view, owner)
}

// handleUploadSiteMedia puts a file among the site's own, for something that
// belongs to no one item, such as a logo a theme setting names.
func (s *Server) handleUploadSiteMedia(w http.ResponseWriter, r *http.Request) {
	view := s.src()
	if !s.writable(w, view) {
		return
	}
	s.upload(w, r, view, nil)
}

// received is a file a request carries, as it is to be stored.
type received struct {
	name string
	data []byte
	// located says the photo named where it was taken, which was taken out.
	located bool
}

// receive reads the file a request carries, refusing a kind the site does not
// host and one too large.
func (s *Server) receive(w http.ResponseWriter, r *http.Request) (received, bool) {
	file, header, err := r.FormFile("file")
	if err != nil {
		failField(w, http.StatusBadRequest, CodeInvalidRequest, "file",
			"send the file as multipart form data under the name \"file\"")
		return received{}, false
	}
	defer func() { _ = file.Close() }()

	name := path.Base(strings.ReplaceAll(header.Filename, `\`, "/"))
	ext := strings.ToLower(path.Ext(name))
	if !allowedMedia[ext] {
		failField(w, http.StatusUnsupportedMediaType, CodeInvalidRequest, "file",
			"this kind of file cannot be uploaded: "+ext)
		return received{}, false
	}

	data, err := io.ReadAll(io.LimitReader(file, maxUpload+1))
	if err != nil {
		s.failErr(w, err)
		return received{}, false
	}
	if len(data) > maxUpload {
		fail(w, http.StatusRequestEntityTooLarge, CodeInvalidRequest, "the file is too large")
		return received{}, false
	}
	// What is uploaded is published, and a phone writes where it was taken
	// into every photo, so that goes before the photo is stored.
	data, located := img.WithoutLocation(data)
	return received{name: name, data: data, located: located}, true
}

// upload stores the file a request carries, beside an item or, with no
// owner, among the site's own files.
func (s *Server) upload(w http.ResponseWriter, r *http.Request, view View, owner *content.Content) {
	got, ok := s.receive(w, r)
	if !ok {
		return
	}
	name, data, located := got.name, got.data, got.located
	ext := strings.ToLower(path.Ext(name))

	var id content.ID
	if owner != nil {
		id = owner.ID
	}
	res, err := s.apply(r, view, content.ChangeSet{
		Ops:     []content.Op{content.PutMedia{Owner: id, Name: name, Data: data}},
		Message: "media: " + name,
	})
	if err != nil {
		s.failWrite(w, view, r, err)
		return
	}
	if len(res.Written) == 0 {
		s.failErr(w, errNothingWritten)
		return
	}

	// The store may have chosen another name rather than replace a file, so
	// the link is built from what it actually wrote.
	stored := res.Written[len(res.Written)-1]
	media := Media{
		Name:            path.Base(stored),
		Path:            stored,
		Size:            len(data),
		Type:            mime.TypeByExtension(ext),
		LocationRemoved: located,
	}
	if site, ok := strings.CutPrefix(stored, "static/"); ok {
		// static/ is published at the root of the site, so a file of the
		// site's own, or of an item kept as a single file, is named by its
		// path there.
		media.Link = "/" + site
		media.URL = view.Resolver.Rel(media.Link)
	} else {
		// A bundle publishes its files beside the page, so the markdown links
		// them by name alone. That is what keeps the source readable in an
		// editor and on GitHub.
		media.Link = path.Base(stored)
		media.URL = path.Join(view.Resolver.For(owner), path.Base(stored))
	}
	writeJSON(w, http.StatusCreated, media)
}

// handleListMedia lists the files an item keeps beside it, which are
// published with its page.
func (s *Server) handleListMedia(w http.ResponseWriter, r *http.Request) {
	view := s.src()
	owner, err := view.Reader.Get(r.Context(), content.ID(r.PathValue("id")))
	if err != nil {
		s.failErr(w, err)
		return
	}
	list := MediaList{Items: []Media{}}
	if view.Bundle != nil {
		files, kept, err := view.Bundle(owner)
		if err != nil {
			s.failErr(w, err)
			return
		}
		list.Bundle = kept
		for _, f := range files {
			list.Items = append(list.Items, bundled(view, owner, f))
		}
	}
	writeJSON(w, http.StatusOK, list)
}

// bundled describes a file of an item's bundle. A bundle publishes its files
// beside the page, so the markdown links one by its path within it.
func bundled(view View, owner *content.Content, f BundleFile) Media {
	return Media{
		Name: f.Name,
		Path: f.Path,
		URL:  path.Join(view.Resolver.For(owner), f.Name),
		Link: f.Name,
		Size: int(f.Size),
		Type: mime.TypeByExtension(strings.ToLower(path.Ext(f.Name))),
	}
}

// handleReplaceMedia puts a new version of a file of an item's bundle in its
// place, under its name, so that every link to it still reaches it.
func (s *Server) handleReplaceMedia(w http.ResponseWriter, r *http.Request) {
	view := s.src()
	if !s.writable(w, view) {
		return
	}
	owner, err := view.Reader.Get(r.Context(), content.ID(r.PathValue("id")))
	if err != nil {
		s.failErr(w, err)
		return
	}
	name := r.PathValue("name")
	var old *BundleFile
	if view.Bundle != nil {
		files, _, err := view.Bundle(owner)
		if err != nil {
			s.failErr(w, err)
			return
		}
		for i := range files {
			if files[i].Name == name {
				old = &files[i]
			}
		}
	}
	if old == nil {
		fail(w, http.StatusNotFound, CodeNotFound, "this item keeps no file called "+name)
		return
	}

	got, ok := s.receive(w, r)
	if !ok {
		return
	}
	// The name stays, and with it what a browser takes the file for.
	if !sameKind(got.name, name) {
		failField(w, http.StatusUnsupportedMediaType, CodeInvalidRequest, "file",
			"a file is replaced by one of its own kind: "+path.Ext(name))
		return
	}
	if _, err := s.apply(r, view, content.ChangeSet{
		Ops:     []content.Op{content.PutMedia{Owner: owner.ID, Name: name, Data: got.data, Replace: true}},
		Message: "media: replace " + name,
	}); err != nil {
		s.failWrite(w, view, r, err)
		return
	}
	media := bundled(view, owner, BundleFile{Name: name, Path: old.Path, Size: int64(len(got.data))})
	media.LocationRemoved = got.located
	writeJSON(w, http.StatusOK, media)
}

// sameKind reports whether two file names are of one kind of file, as
// photo.jpg and photo.jpeg are.
func sameKind(a, b string) bool {
	x, y := strings.ToLower(path.Ext(a)), strings.ToLower(path.Ext(b))
	return x == y || mime.TypeByExtension(x) != "" && mime.TypeByExtension(x) == mime.TypeByExtension(y)
}

// handleDeleteMedia removes a file from an item's bundle, named by its path
// within it.
func (s *Server) handleDeleteMedia(w http.ResponseWriter, r *http.Request) {
	view := s.src()
	if !s.writable(w, view) {
		return
	}

	id := content.ID(r.PathValue("id"))
	name := r.PathValue("name")
	if name == "" {
		failField(w, http.StatusBadRequest, CodeInvalidRequest, "name", "name is required")
		return
	}

	if _, err := s.apply(r, view, content.ChangeSet{
		Ops:     []content.Op{content.DeleteMedia{Owner: id, Name: name}},
		Message: "media: remove " + name,
	}); err != nil {
		s.failWrite(w, view, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
