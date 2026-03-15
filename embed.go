package kiteblog

import "embed"

//go:embed templates/*.tmpl ui/admin/dist/*
var resources embed.FS

func TemplateFS() embed.FS {
	return resources
}
