package api

import (
	"io"
	"mime"
	"net/http"
	"path"
	"strings"

	"github.com/kite-plus/kite/internal/content"
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

// upload stores the file a request carries, beside an item or, with no
// owner, among the site's own files.
func (s *Server) upload(w http.ResponseWriter, r *http.Request, view View, owner *content.Content) {
	file, header, err := r.FormFile("file")
	if err != nil {
		failField(w, http.StatusBadRequest, CodeInvalidRequest, "file",
			"send the file as multipart form data under the name \"file\"")
		return
	}
	defer func() { _ = file.Close() }()

	name := path.Base(strings.ReplaceAll(header.Filename, `\`, "/"))
	ext := strings.ToLower(path.Ext(name))
	if !allowedMedia[ext] {
		failField(w, http.StatusUnsupportedMediaType, CodeInvalidRequest, "file",
			"this kind of file cannot be uploaded: "+ext)
		return
	}

	data, err := io.ReadAll(io.LimitReader(file, maxUpload+1))
	if err != nil {
		s.failErr(w, err)
		return
	}
	if len(data) > maxUpload {
		fail(w, http.StatusRequestEntityTooLarge, CodeInvalidRequest, "the file is too large")
		return
	}

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
		Name: path.Base(stored),
		Path: stored,
		Size: len(data),
		Type: mime.TypeByExtension(ext),
	}
	if owner != nil {
		// A bundle publishes its files beside the page, so the markdown links
		// them by name alone. That is what keeps the source readable in an
		// editor and on GitHub.
		media.Link = path.Base(stored)
		media.URL = path.Join(view.Resolver.For(owner), path.Base(stored))
	} else {
		// static/ is published at the root of the site, so a file of the
		// site's own is named by its path there.
		media.Link = "/" + strings.TrimPrefix(stored, "static/")
		media.URL = view.Resolver.Rel(media.Link)
	}
	writeJSON(w, http.StatusCreated, media)
}

// handleDeleteMedia removes a file from an item's bundle.
func (s *Server) handleDeleteMedia(w http.ResponseWriter, r *http.Request) {
	view := s.src()
	if !s.writable(w, view) {
		return
	}

	id := content.ID(r.PathValue("id"))
	name := path.Base(r.PathValue("name"))
	if name == "." || name == "/" || name == "" {
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
