// Package themes embeds the themes shipped inside the binary.
package themes

import (
	"embed"
	"io/fs"
)

//go:embed all:default
var builtin embed.FS

// Default returns the filesystem of the theme Kite ships with, rooted at the
// theme directory. Embedding it keeps "download one binary and run" true.
func Default() fs.FS {
	sub, err := fs.Sub(builtin, "default")
	if err != nil {
		panic("themes: default theme is missing from the binary: " + err.Error())
	}
	return sub
}
