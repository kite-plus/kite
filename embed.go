//go:build !dev

package kite

import "embed"

//go:embed all:web/admin/dist
var AdminFS embed.FS
