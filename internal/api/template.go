package api

import (
	"html/template"
	"io/fs"
	"time"
)

// templateFuncs 注册自定义模板函数
var templateFuncs = template.FuncMap{
	// formatDate 格式化时间为 "2006-01-02" 日期字符串
	"formatDate": func(t interface{}) string {
		switch v := t.(type) {
		case time.Time:
			return v.Format("2006-01-02")
		case *time.Time:
			if v == nil {
				return ""
			}
			return v.Format("2006-01-02")
		default:
			return ""
		}
	},
	// add 加法
	"add": func(a, b int) int { return a + b },
	// subtract 减法
	"subtract": func(a, b int) int { return a - b },
}

func loadTemplateSet(templateFS fs.FS) (*template.Template, error) {
	return template.New("").Funcs(templateFuncs).ParseFS(
		templateFS,
		"templates/*.html",
		"templates/partials/*.html",
		"templates/pages/*.html",
	)
}
