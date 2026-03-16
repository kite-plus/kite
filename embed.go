package kiteblog

import "embed"

//go:embed all:templates ui/admin/dist/*
var resources embed.FS

// TemplateFS 返回模板文件系统（SSR 主题用）
func TemplateFS() embed.FS {
	return resources
}

// AdminFS 返回 Admin SPA 文件系统
func AdminFS() embed.FS {
	return resources
}
