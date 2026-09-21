module github.com/kite-plus/kite

go 1.26.4

// Pinned exactly, not as a floor. A floating patch version means the same
// tag rebuilt later compiles with a different compiler and produces different
// bytes, which would make the release checksums unverifiable.
toolchain go1.26.8

require (
	github.com/alecthomas/chroma/v2 v2.27.0
	github.com/fsnotify/fsnotify v1.10.1
	github.com/oklog/ulid/v2 v2.1.2
	github.com/spf13/cobra v1.10.2
	github.com/yuin/goldmark v1.8.6
	github.com/yuin/goldmark-highlighting/v2 v2.0.0-20230729083705-37449abec8cc
	gopkg.in/yaml.v3 v3.0.1
	modernc.org/sqlite v1.59.0
)

require (
	github.com/dlclark/regexp2/v2 v2.2.1 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/mattn/go-isatty v0.0.24 // indirect
	github.com/ncruces/go-strftime v1.0.0 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	github.com/spf13/pflag v1.0.9 // indirect
	golang.org/x/crypto v0.57.0 // indirect
	golang.org/x/sys v0.48.0 // indirect
	golang.org/x/term v0.46.0 // indirect
	modernc.org/libc v1.75.7 // indirect
	modernc.org/mathutil v1.7.1 // indirect
	modernc.org/memory v1.12.1 // indirect
)
