package api

import (
	"html/template"
	"io/fs"
	"strings"
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
	// safeHTML 将字符串标记为安全 HTML，不做转义
	"safeHTML": func(s string) template.HTML {
		return template.HTML(s)
	},
	// add 加法
	"add": func(a, b int) int { return a + b },
	// subtract 减法
	"subtract": func(a, b int) int { return a - b },
	// currentYear 返回当前年份
	"currentYear": func() int { return time.Now().Year() },
	// iconifyURL 将 Iconify icon ID（如 "lucide:home"）转换为 SVG API URL
	"iconifyURL": func(icon string) string {
		if icon == "" {
			return ""
		}
		// 将 "lucide:home" 转为 "lucide/home"
		path := strings.Replace(icon, ":", "/", 1)
		return "https://api.iconify.design/" + path + ".svg"
	},
}

func loadTemplateSet(templateFS fs.FS) (*template.Template, error) {
	return template.New("").Funcs(templateFuncs).ParseFS(
		templateFS,
		"templates/*.html",
		"templates/partials/*.html",
		"templates/pages/*.html",
	)
}
