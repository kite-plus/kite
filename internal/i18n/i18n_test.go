package i18n

import (
	"context"
	"testing"
)

func TestPick(t *testing.T) {
	tests := []struct {
		name       string
		cookie     string
		acceptLang string
		want       Locale
	}{
		{"empty defaults to en", "", "", LocaleEN},
		{"cookie wins over header", "zh", "en-US", LocaleZH},
		{"unknown cookie falls through", "fr", "zh-CN", LocaleZH},
		{"header zh-CN matches zh", "", "zh-CN", LocaleZH},
		{"header zh-Hans-CN matches zh", "", "zh-Hans-CN", LocaleZH},
		{"header en-US matches en", "", "en-US", LocaleEN},
		{"header q-values ignored, first wins", "", "en;q=0.5,zh;q=1.0", LocaleEN},
		{"unknown locale falls back to default", "", "fr-FR,de-DE", LocaleEN},
		{"cookie case-insensitive", "ZH", "", LocaleZH},
		{"whitespace trimmed", "  zh  ", "", LocaleZH},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Pick(tt.cookie, tt.acceptLang); got != tt.want {
				t.Fatalf("Pick(%q, %q) = %q, want %q", tt.cookie, tt.acceptLang, got, tt.want)
			}
		})
	}
}

func TestT(t *testing.T) {
	if got := T(LocaleEN, KeySuccess); got != "success" {
		t.Fatalf("T(en, success) = %q, want success", got)
	}
	if got := T(LocaleZH, KeySuccess); got != "成功" {
		t.Fatalf("T(zh, success) = %q, want 成功", got)
	}
	// Missing key returns the key itself so a stray reference is loud
	// rather than silent.
	if got := T(LocaleEN, "no.such.key"); got != "no.such.key" {
		t.Fatalf("T(en, no.such.key) = %q, want no.such.key", got)
	}
	// Format args. We pass the catalogue key through a variable so vet
	// doesn't try to apply printf-checking to the key constant — T's first
	// argument is a key, not a format string.
	key := KeyAuthInvalidProfile
	if got := T(LocaleEN, key, "field"); got != "invalid profile data: field" {
		t.Fatalf("T(en, invalid_profile, field) = %q", got)
	}
	if got := T(LocaleZH, key, "field"); got != "资料数据无效：field" {
		t.Fatalf("T(zh, invalid_profile, field) = %q", got)
	}
}

func TestContext(t *testing.T) {
	ctx := context.Background()
	if got := FromContext(ctx); got != DefaultLocale {
		t.Fatalf("empty ctx = %q, want default", got)
	}
	ctx = WithLocale(ctx, LocaleZH)
	if got := FromContext(ctx); got != LocaleZH {
		t.Fatalf("ctx with zh = %q, want zh", got)
	}
}

func TestCatalogCoverage(t *testing.T) {
	// Every key must have at least an English translation. A missing
	// English entry would silently fall back to the raw key string in
	// production, which is much worse to debug than a test failure here.
	for key, entries := range Catalog {
		if entries[LocaleEN] == "" {
			t.Errorf("catalog key %q is missing English translation", key)
		}
	}
}
