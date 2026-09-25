package theme_test

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/kite-plus/kite/internal/render/theme"
)

const packedManifest = `name: x
version: 1.0.0
apiVersion: kite/v1
title: Paper
description: A quiet theme.
layouts:
  - {name: links, label: Links, description: Cards of links.}
settings:
  - key: look
    type: section
    label: Look
    fields:
      - {key: accent, type: color, label: Accent, help: Links and marks.}
      - key: scheme
        type: select
        label: Scheme
        default: auto
        options: [{value: auto, label: Automatic}, {value: dark, label: Dark}]
  - key: social
    type: group
    label: Social
    fields:
      - {key: github, type: url, label: GitHub}
  - {key: copyright, type: string, label: Copyright}
`

const zhPack = `theme:
  title: 纸
  settings:
    look: {label: 外观}
    accent: {label: 强调色}
    scheme:
      label: 配色
      options: {auto: 跟随系统}
    social:
      label: 社交
      fields:
        github: {label: GitHub 主页}
  layouts:
    links: {label: 友链}
read_more: 阅读全文
`

func packed(t *testing.T) *theme.Theme {
	t.Helper()
	th, err := theme.Load(fstest.MapFS{
		"theme.yaml":               file(packedManifest),
		"layouts/single.html":      file("x"),
		"layouts/links.html":       file("x"),
		"i18n/zh-CN.yaml":          file(zhPack),
		"i18n/README.md":           file("not a pack"),
		"layouts/_partials/x.html": file("x"),
	})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	return th
}

// A theme speaks the admin's language through its own pack, and whatever the
// pack leaves out is still said, in the words of the manifest.
func TestAThemeIsDescribedInThePacksLanguage(t *testing.T) {
	m := packed(t).Localized("zh-CN")

	if m.Title != "纸" || m.Description != "A quiet theme." {
		t.Errorf("title, description = %q, %q", m.Title, m.Description)
	}
	look := m.Settings[0]
	if look.Label != "外观" || look.Fields[0].Label != "强调色" || look.Fields[0].Help != "Links and marks." {
		t.Errorf("section = %+v", look)
	}
	scheme := look.Fields[1]
	if scheme.Label != "配色" || scheme.Options[0].Label != "跟随系统" || scheme.Options[1].Label != "Dark" {
		t.Errorf("select = %+v", scheme)
	}
	if social := m.Settings[1]; social.Label != "社交" || social.Fields[0].Label != "GitHub 主页" {
		t.Errorf("group = %+v", social)
	}
	if m.Settings[2].Label != "Copyright" {
		t.Errorf("an untranslated label = %q", m.Settings[2].Label)
	}
	if m.Layouts[0].Label != "友链" || m.Layouts[0].Description != "Cards of links." {
		t.Errorf("layout = %+v", m.Layouts[0])
	}
}

// Translating is a copy: the manifest the theme renders with keeps its own
// words, whatever language the last visitor to the admin spoke.
func TestLocalizingLeavesTheManifestAlone(t *testing.T) {
	th := packed(t)
	_ = th.Localized("zh-CN")
	if th.Manifest.Title != "Paper" || th.Manifest.Settings[0].Fields[0].Label != "Accent" ||
		th.Manifest.Layouts[0].Label != "Links" || th.Manifest.Settings[0].Fields[1].Options[0].Label != "Automatic" {
		t.Errorf("the manifest changed: %+v", th.Manifest)
	}
}

func TestTheClosestPackIsChosen(t *testing.T) {
	th := packed(t)
	for lang, want := range map[string]string{
		"zh-CN": "纸",
		"zh_cn": "纸",
		"zh":    "纸",
		"zh-TW": "纸",
		"en":    "Paper",
		"":      "Paper",
	} {
		if got := th.Localized(lang).Title; got != want {
			t.Errorf("Localized(%q).Title = %q, want %q", lang, got, want)
		}
	}
	if words := th.Pack("zh-CN"); words["read_more"] != "阅读全文" {
		t.Errorf("the rest of the pack = %v", words)
	}
}

func TestAPackIsNamedAfterALanguage(t *testing.T) {
	_, err := theme.Load(fstest.MapFS{
		"theme.yaml":          file("name: x\nversion: 1.0.0\napiVersion: kite/v1\n"),
		"layouts/single.html": file("x"),
		"i18n/chinese.yaml":   file("theme: {title: x}"),
	})
	if err == nil || !strings.Contains(err.Error(), "chinese.yaml") {
		t.Errorf("Load = %v, want the pack's name refused", err)
	}
}

func TestAScreenshotIsFoundByNameOrByTheManifest(t *testing.T) {
	base := "name: x\nversion: 1.0.0\napiVersion: kite/v1\n"
	for _, tc := range []struct {
		files fstest.MapFS
		want  string
	}{
		{fstest.MapFS{"screenshot.webp": file("x")}, "screenshot.webp"},
		{fstest.MapFS{"docs/shot.png": file("x"), "theme.yaml": file(base + "screenshot: docs/shot.png\n")}, "docs/shot.png"},
		{fstest.MapFS{"theme.yaml": file(base + "screenshot: ../outside.png\n")}, ""},
		{fstest.MapFS{}, ""},
	} {
		files := fstest.MapFS{"theme.yaml": file(base), "layouts/single.html": file("x")}
		for name, f := range tc.files {
			files[name] = f
		}
		th, err := theme.Load(files)
		if err != nil {
			t.Fatalf("Load: %v", err)
		}
		if got, _ := th.ScreenshotPath(); got != tc.want {
			t.Errorf("ScreenshotPath = %q, want %q", got, tc.want)
		}
	}
}
