package site

import (
	"io/fs"
	"os"
	"path"

	"github.com/kite-plus/kite/internal/build"
	"github.com/kite-plus/kite/internal/content"
	"github.com/kite-plus/kite/internal/store/file"
)

// BundleFile is a file an item keeps in its bundle.
type BundleFile struct {
	// Name is its path within the bundle, which is how the markdown links it.
	Name string
	// Path is where it lives in the repository.
	Path string
	Size int64
}

// Bundle lists the files an item keeps in its bundle, as a build publishes
// them beside its page, and says whether it keeps a bundle at all: one kept
// as a single file has no folder of its own.
func (s *Site) Bundle(owner *content.Content) ([]BundleFile, bool, error) {
	t := s.Project.Types.Get(owner.Kind)
	if t == nil {
		return nil, false, nil
	}
	dir := file.MediaDir(t, owner.Locator)
	if dir == "" {
		return nil, false, nil
	}
	root := os.DirFS(s.Project.Root)
	names, err := build.BundleFiles(root, dir)
	if err != nil {
		return nil, true, err
	}
	files := make([]BundleFile, 0, len(names))
	for _, name := range names {
		stored := path.Join(dir, name)
		info, err := fs.Stat(root, stored)
		if err != nil {
			continue // gone since it was listed
		}
		files = append(files, BundleFile{Name: name, Path: stored, Size: info.Size()})
	}
	return files, true, nil
}
