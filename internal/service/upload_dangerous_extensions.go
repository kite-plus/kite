package service

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	UploadDangerousExtensionRulesSettingKey = "upload.dangerous_extension_rules"
	UploadDangerousRenameSuffixSettingKey   = "upload.dangerous_rename_suffix"

	DangerousExtensionActionBlock  = "block"
	DangerousExtensionActionRename = "rename"

	DefaultDangerousRenameSuffixValue = "blocked"
	DangerousRenameMimeType           = "application/octet-stream"
)

var (
	dangerousExtensionPattern   = regexp.MustCompile(`^\.[a-z0-9]+$`)
	dangerousRenameSuffixRegexp = regexp.MustCompile(`^[a-z0-9_-]+$`)
	defaultDangerousExtensions  = []string{".exe", ".bat", ".cmd", ".sh", ".ps1"}
)

type DangerousExtensionRule struct {
	Ext    string `json:"ext"`
	Action string `json:"action"`
}

type DangerousExtensionDecision struct {
	Action     string
	MatchedExt string
	SafeName   string
	SafeExt    string
}

func DefaultDangerousExtensionRules(forbiddenExts []string) string {
	source := forbiddenExts
	if len(source) == 0 {
		source = defaultDangerousExtensions
	}

	rules := make([]DangerousExtensionRule, 0, len(source))
	seen := make(map[string]struct{}, len(source))
	for _, rawExt := range source {
		ext, err := normalizeDangerousExtension(rawExt)
		if err != nil {
			continue
		}
		if _, ok := seen[ext]; ok {
			continue
		}
		seen[ext] = struct{}{}
		rules = append(rules, DangerousExtensionRule{
			Ext:    ext,
			Action: DangerousExtensionActionBlock,
		})
	}

	if len(rules) == 0 {
		rules = []DangerousExtensionRule{{
			Ext:    ".exe",
			Action: DangerousExtensionActionBlock,
		}}
	}

	payload, err := json.Marshal(rules)
	if err != nil {
		return `[{"ext":".exe","action":"block"}]`
	}
	return string(payload)
}

func NormalizeDangerousExtensionRules(raw string) (string, error) {
	rules, err := ParseDangerousExtensionRules(raw)
	if err != nil {
		return "", err
	}

	payload, err := json.Marshal(rules)
	if err != nil {
		return "", fmt.Errorf("规则序列化失败")
	}
	return string(payload), nil
}

func ParseDangerousExtensionRules(raw string) ([]DangerousExtensionRule, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, fmt.Errorf("规则列表不能为空")
	}
	if !strings.HasPrefix(trimmed, "[") {
		return nil, fmt.Errorf("规则列表必须是 JSON 数组")
	}

	var rules []DangerousExtensionRule
	if err := json.Unmarshal([]byte(trimmed), &rules); err != nil {
		return nil, fmt.Errorf("规则列表不是合法的 JSON 数组")
	}

	normalized := make([]DangerousExtensionRule, 0, len(rules))
	seen := make(map[string]struct{}, len(rules))
	for _, rule := range rules {
		ext, err := normalizeDangerousExtension(rule.Ext)
		if err != nil {
			return nil, err
		}
		action, err := normalizeDangerousExtensionAction(rule.Action)
		if err != nil {
			return nil, err
		}
		if _, ok := seen[ext]; ok {
			return nil, fmt.Errorf("扩展名 %s 不能重复", ext)
		}
		seen[ext] = struct{}{}
		normalized = append(normalized, DangerousExtensionRule{
			Ext:    ext,
			Action: action,
		})
	}

	return normalized, nil
}

func NormalizeDangerousRenameSuffix(raw string) (string, error) {
	normalized := strings.TrimSpace(raw)
	normalized = strings.TrimLeft(normalized, ".")
	normalized = strings.ToLower(normalized)
	if normalized == "" {
		return "", fmt.Errorf("安全后缀不能为空")
	}
	if !dangerousRenameSuffixRegexp.MatchString(normalized) {
		return "", fmt.Errorf("安全后缀只能包含字母、数字、- 或 _")
	}
	return normalized, nil
}

func ResolveDangerousExtensionRules(raw string, fallback []string) []DangerousExtensionRule {
	rules, err := ParseDangerousExtensionRules(raw)
	if err == nil {
		return rules
	}

	rules, err = ParseDangerousExtensionRules(DefaultDangerousExtensionRules(fallback))
	if err == nil {
		return rules
	}

	return []DangerousExtensionRule{{
		Ext:    ".exe",
		Action: DangerousExtensionActionBlock,
	}}
}

func ResolveDangerousRenameSuffix(raw string) string {
	suffix, err := NormalizeDangerousRenameSuffix(raw)
	if err == nil {
		return suffix
	}
	return DefaultDangerousRenameSuffixValue
}

func DecideDangerousExtension(filename string, rules []DangerousExtensionRule, renameSuffix string) (DangerousExtensionDecision, bool, error) {
	ext := strings.ToLower(filepath.Ext(strings.TrimSpace(filename)))
	if ext == "" {
		return DangerousExtensionDecision{}, false, nil
	}

	for _, rule := range rules {
		if ext != rule.Ext {
			continue
		}
		if rule.Action == DangerousExtensionActionBlock {
			return DangerousExtensionDecision{
				Action:     DangerousExtensionActionBlock,
				MatchedExt: ext,
			}, true, nil
		}

		suffix, err := NormalizeDangerousRenameSuffix(renameSuffix)
		if err != nil {
			return DangerousExtensionDecision{}, false, err
		}
		return DangerousExtensionDecision{
			Action:     DangerousExtensionActionRename,
			MatchedExt: ext,
			SafeName:   BuildDangerousRenamedFilename(filename, suffix),
			SafeExt:    suffix,
		}, true, nil
	}

	return DangerousExtensionDecision{}, false, nil
}

func BuildDangerousRenamedFilename(filename, renameSuffix string) string {
	return filename + "." + renameSuffix
}

func normalizeDangerousExtension(raw string) (string, error) {
	normalized := strings.TrimSpace(raw)
	normalized = strings.ToLower(normalized)
	if normalized == "" {
		return "", fmt.Errorf("扩展名不能为空")
	}
	if !strings.HasPrefix(normalized, ".") {
		return "", fmt.Errorf("扩展名 %q 必须以 . 开头", normalized)
	}
	if !dangerousExtensionPattern.MatchString(normalized) {
		return "", fmt.Errorf("扩展名 %q 格式无效，仅支持单段后缀", normalized)
	}
	return normalized, nil
}

func normalizeDangerousExtensionAction(raw string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(raw))
	switch normalized {
	case DangerousExtensionActionBlock, DangerousExtensionActionRename:
		return normalized, nil
	default:
		return "", fmt.Errorf("扩展名动作 %q 无效，仅支持 block 或 rename", raw)
	}
}
