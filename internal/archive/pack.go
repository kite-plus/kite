package archive

import (
	"archive/zip"
	"io"
	"maps"
	"slices"
	"time"
)

// packedAt is the time every file of a packed archive carries, so that the
// same files always pack to the same bytes and a checksum names a version.
var packedAt = time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)

// Pack writes a theme or a plugin as the zip archive a release carries: its
// files, named with forward slashes, under one folder called top, in name
// order.
func Pack(w io.Writer, top string, files map[string][]byte) error {
	zw := zip.NewWriter(w)
	for _, name := range slices.Sorted(maps.Keys(files)) {
		h := &zip.FileHeader{Name: top + "/" + name, Method: zip.Deflate, Modified: packedAt}
		h.SetMode(0o644)
		fw, err := zw.CreateHeader(h)
		if err != nil {
			return err
		}
		if _, err := fw.Write(files[name]); err != nil {
			return err
		}
	}
	return zw.Close()
}
