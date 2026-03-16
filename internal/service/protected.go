package service

import (
	"fmt"
	"regexp"
	"strings"
)

// protectedBlockRegex 匹配 :::protected HTML 渲染后的 div 块
// 后端 Markdown→HTML 已将 :::protected 转为 <div data-protected="true" ...>...</div>
var protectedBlockHTMLRegex = regexp.MustCompile(
	`<div[^>]*data-protected="true"[^>]*>[\s\S]*?</div>`,
)

// protectedBlockMDRegex 匹配 Markdown 中的 :::protected...:::
var protectedBlockMDRegex = regexp.MustCompile(
	`(?m)^:::protected(?:\s+([^\n]*))?\n([\s\S]*?)\n:::`,
)

// 占位符 HTML 模板
const protectedPlaceholderHTML = `<div class="kite-protected-block" data-hint="%s"><div class="kite-lock-icon">🔒</div><p class="kite-lock-hint">%s</p></div>`

// defaultProtectedHint 默认提示文字
const defaultProtectedHint = "输入密码查看隐藏内容"

// FilterProtectedHTML 将 HTML 中的 protected 块替换为占位符
func FilterProtectedHTML(html string) string {
	return protectedBlockHTMLRegex.ReplaceAllStringFunc(html, func(match string) string {
		hint := extractProtectedHint(match)
		return fmt.Sprintf(protectedPlaceholderHTML, hint, hint)
	})
}

// FilterProtectedMarkdown 将 Markdown 中的 :::protected 块替换为占位符
func FilterProtectedMarkdown(md string) string {
	return protectedBlockMDRegex.ReplaceAllStringFunc(md, func(match string) string {
		submatch := protectedBlockMDRegex.FindStringSubmatch(match)
		hint := defaultProtectedHint
		if len(submatch) > 1 && strings.TrimSpace(submatch[1]) != "" {
			hint = strings.TrimSpace(submatch[1])
		}
		return fmt.Sprintf(":::protected %s\n🔒 %s\n:::", hint, hint)
	})
}

// HasProtectedBlocks 检查内容是否包含 protected 块
func HasProtectedBlocks(contentHTML string) bool {
	return protectedBlockHTMLRegex.MatchString(contentHTML)
}

// extractProtectedHint 从 HTML 中提取 data-hint 属性
func extractProtectedHint(html string) string {
	hintRegex := regexp.MustCompile(`data-hint="([^"]*)"`)
	match := hintRegex.FindStringSubmatch(html)
	if len(match) > 1 {
		return match[1]
	}
	return defaultProtectedHint
}

// ConvertProtectedMDToHTML 将 :::protected Markdown 块转换为带 data-protected 的 HTML
func ConvertProtectedMDToHTML(md string) string {
	return protectedBlockMDRegex.ReplaceAllStringFunc(md, func(match string) string {
		submatch := protectedBlockMDRegex.FindStringSubmatch(match)
		hint := defaultProtectedHint
		body := ""
		if len(submatch) > 1 && strings.TrimSpace(submatch[1]) != "" {
			hint = strings.TrimSpace(submatch[1])
		}
		if len(submatch) > 2 {
			body = submatch[2]
		}
		return fmt.Sprintf(`<div data-protected="true" data-hint="%s">%s</div>`, hint, body)
	})
}
