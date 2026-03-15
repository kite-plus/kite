package api

import (
	"html/template"
	"io/fs"
)

func loadTemplateSet(templateFS fs.FS) (*template.Template, error) {
	return template.ParseFS(templateFS, "*.tmpl")
}
