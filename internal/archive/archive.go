// Package archive reads a theme or a plugin out of the zip archive it is
// uploaded as, refusing one that would write outside itself or fill a disk.
package archive

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"slices"
	"strings"
	"testing/fstest"
)

// Unpack reads a theme or a plugin, kind, out of a zip archive: from
// its top, or from the one folder everything in it sits in, as an archive of
// a repository has it. manifest is the file that has to be there. It reports
// what is wrong with the archive when it cannot.
func Unpack(archive []byte, kind, manifest string, maxSize int64, maxFiles int) (map[string][]byte, string) {
	zr, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
	if err != nil {
		return nil, "the file is not a zip archive"
	}

	var names []string
	for _, f := range zr.File {
		if name := archived(f.Name); name != "" && !strings.HasSuffix(name, "/") {
			names = append(names, name)
		}
	}
	root := ""
	missing := "the archive has no " + manifest + " at its top or in a single folder"
	if !slices.Contains(names, manifest) {
		if len(names) == 0 {
			return nil, "the archive is empty"
		}
		top, _, _ := strings.Cut(names[0], "/")
		for _, name := range names {
			if !strings.HasPrefix(name, top+"/") {
				return nil, missing
			}
		}
		if !slices.Contains(names, top+"/"+manifest) {
			return nil, missing
		}
		root = top + "/"
	}

	files := make(map[string][]byte)
	var total int64
	for _, f := range zr.File {
		name := archived(f.Name)
		if name == "" || strings.HasSuffix(name, "/") {
			continue
		}
		rel := strings.TrimPrefix(name, root)
		if !fs.ValidPath(rel) {
			return nil, "the archive holds a path that leads out of the " + kind + ": " + f.Name
		}
		if !f.Mode().IsRegular() {
			return nil, "the archive holds something other than a plain file: " + f.Name
		}
		if len(files) == maxFiles {
			return nil, fmt.Sprintf("a %s may hold at most %d files", kind, maxFiles)
		}
		rc, err := f.Open()
		if err != nil {
			return nil, "the archive cannot be read: " + err.Error()
		}
		// The sizes an archive declares are not trusted: the bytes are
		// counted as they come out.
		data, err := io.ReadAll(io.LimitReader(rc, maxSize-total+1))
		_ = rc.Close()
		if err != nil {
			return nil, "the archive cannot be read: " + err.Error()
		}
		total += int64(len(data))
		if total > maxSize {
			return nil, fmt.Sprintf("a %s may take at most %d MB unpacked", kind, maxSize>>20)
		}
		files[rel] = data
	}
	return files, ""
}

// archived is the name of a file in an archive with forward slashes, or ""
// for what a computer adds to an archive on its own: a folder of macOS
// metadata, a Finder or Explorer file, a repository's .git.
func archived(name string) string {
	name = strings.ReplaceAll(name, `\`, "/")
	for part := range strings.SplitSeq(name, "/") {
		switch part {
		case "__MACOSX", ".git", ".DS_Store", "Thumbs.db", "desktop.ini":
			return ""
		}
	}
	return name
}

// FS holds unpacked files as a filesystem a theme or plugin can be loaded
// from.
func FS(files map[string][]byte) fs.FS {
	fsys := make(fstest.MapFS, len(files))
	for name, data := range files {
		fsys[name] = &fstest.MapFile{Data: data, Mode: 0o644}
	}
	return fsys
}
