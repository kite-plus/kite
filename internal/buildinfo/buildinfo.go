// Package buildinfo carries values stamped into the binary at link time.
package buildinfo

// Version is the release version, set via -ldflags at build time.
var Version = "dev"

// Commit is the source revision the binary was built from.
var Commit = "none"

// ThemeAPIVersion is the theme contract version this binary implements. A theme
// declaring a different apiVersion is refused rather than rendered best effort.
const ThemeAPIVersion = "kite/v1"

// PluginABIVersion is the WebAssembly host ABI version. A plugin reporting a
// different value is refused at load time.
const PluginABIVersion = 1

// BuildABIVersion changes whenever build cache keying changes, invalidating
// every cached artifact.
const BuildABIVersion = 1
