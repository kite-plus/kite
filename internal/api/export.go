package api

import (
	"archive/zip"
	"io"
	"io/fs"
	"mime"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

// storedAs lists the kinds of file that are compressed already, which the
// archive would only spend time failing to shrink.
var storedAs = map[string]bool{
	".png": true, ".jpg": true, ".jpeg": true, ".gif": true, ".webp": true, ".avif": true,
	".woff": true, ".woff2": true, ".zip": true, ".gz": true,
	".mp4": true, ".webm": true, ".mp3": true, ".m4a": true, ".ogg": true, ".pdf": true,
}

// handleExport builds the site as it is published and answers with it as a
// zip archive: what kite build writes, ready to upload anywhere that serves
// static files.
//
// It is a POST although it changes nothing, because it costs a whole build:
// a page elsewhere must not be able to make a studio do one by linking here.
func (s *Server) handleExport(w http.ResponseWriter, r *http.Request) {
	view := s.src()
	if view.Export == nil {
		fail(w, http.StatusNotImplemented, CodeInternal, "this server cannot export the site")
		return
	}

	scratch, err := os.MkdirTemp("", "kite-export-*")
	if err != nil {
		s.failErr(w, err)
		return
	}
	defer func() { _ = os.RemoveAll(scratch) }()

	out := filepath.Join(scratch, "site")
	if err := view.Export(r.Context(), out); err != nil {
		s.log.Warn("export failed", "err", err)
		// What stopped the build is the author's to fix, so it is said; where
		// on this machine it was being built is not.
		fail(w, http.StatusConflict, CodeBuildFailed, strings.ReplaceAll(err.Error(), scratch, "export"))
		return
	}

	files, err := filesUnder(out)
	if err != nil {
		s.failErr(w, err)
		return
	}

	now := time.Now()
	h := w.Header()
	h.Set("Content-Type", "application/zip")
	h.Set("Content-Disposition", mime.FormatMediaType("attachment",
		map[string]string{"filename": exportName(view.Site.BaseURL, now)}))
	h.Set("Cache-Control", "no-store")
	if err := writeZip(w, out, files, now); err != nil {
		// The status has gone out already; the client is left with an
		// archive cut short, which it cannot mistake for a whole one.
		s.log.Error("export: write the archive", "err", err)
	}
}

// filesUnder lists the files below root as sorted, slash separated paths, so
// that two exports of one site list their files in the same order.
func filesUnder(root string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}
		files = append(files, filepath.ToSlash(rel))
		return nil
	})
	slices.Sort(files)
	return files, err
}

// writeZip writes files from root into an archive at its top level: unpacked,
// they are the site, and uploading them is the whole of deploying it.
func writeZip(w io.Writer, root string, files []string, at time.Time) error {
	zw := zip.NewWriter(w)
	for _, name := range files {
		header := &zip.FileHeader{Name: name, Method: zip.Deflate, Modified: at}
		if storedAs[strings.ToLower(path.Ext(name))] {
			header.Method = zip.Store
		}
		header.SetMode(0o644)

		dst, err := zw.CreateHeader(header)
		if err != nil {
			return err
		}
		src, err := os.Open(filepath.Join(root, filepath.FromSlash(name)))
		if err != nil {
			return err
		}
		_, err = io.Copy(dst, src)
		_ = src.Close()
		if err != nil {
			return err
		}
	}
	return zw.Close()
}

// exportName names the archive after where the site lives and when it was
// taken, so that two exports sit side by side in a downloads folder.
func exportName(baseURL string, at time.Time) string {
	name := "site"
	if u, err := url.Parse(baseURL); err == nil {
		// localhost says nothing about which site this is.
		if host := u.Hostname(); host != "" && host != "localhost" && net.ParseIP(host) == nil {
			name = host
		}
	}
	return name + "-" + at.Format("20060102-1504") + ".zip"
}
