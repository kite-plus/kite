// Package i18n is the canonical message catalogue for every user-facing
// string the backend renders — JSON envelope messages, server-rendered
// template strings and a handful of structured-error fragments.
//
// # Why a single catalogue
//
// We deliberately keep all locales in one process-wide map (instead of, say,
// embedding *.json files per locale) for three reasons:
//
//   - Translations participate in `go vet` and the type system: a missing
//     key is a compile-time symbol or a single test failure, not a 404 at
//     runtime.
//   - The catalogue is small enough (~200 keys) that the readability win of
//     literal Go maps beats the operability of pluggable JSON files.
//   - Adding a new language is one new file in this package; no template
//     reload, no embed.FS plumbing.
//
// # Locale resolution
//
// [Pick] is the only place that decides which locale a request gets. It
// looks at, in order: an explicit `kite_locale` cookie, the `Accept-Language`
// header, and the static [DefaultLocale] fallback. Middleware stamps the
// result onto the request context so every downstream handler / template /
// huma operation reads the same value via [FromContext].
package i18n

import (
	"context"
	"fmt"
	"strings"
)

// Locale is the BCP-47-ish two-letter tag we resolve every request to. We
// keep it deliberately small (en + zh) because the catalogue covers exactly
// those two languages — a third locale is a code change, not a config knob.
type Locale string

const (
	// LocaleEN is the catalogue's source-of-truth language. Every key in
	// [Catalog] must have an English entry; other locales fall back to it
	// when a key is missing.
	LocaleEN Locale = "en"

	// LocaleZH is Simplified Chinese. Translations live alongside the
	// English source in the per-key entries below.
	LocaleZH Locale = "zh"

	// DefaultLocale is what [Pick] returns when neither the cookie nor the
	// Accept-Language header give a usable hint. English wins because the
	// JSON envelope was English-only before this package existed, so
	// keeping it as the unconfigured default is the least surprising
	// choice for existing API consumers.
	DefaultLocale = LocaleEN
)

// supported is the closed set [Pick] is willing to return. Adding a locale
// here without populating [Catalog] would silently fall back to English for
// every key, which is worse than rejecting the locale outright — so the two
// must move together.
var supported = map[Locale]bool{
	LocaleEN: true,
	LocaleZH: true,
}

// IsSupported reports whether l is one of the locales the catalogue covers.
// The middleware uses this to validate the cookie before stamping it onto
// the context — an attacker-controlled cookie value can't drive us into an
// undefined locale.
func IsSupported(l Locale) bool { return supported[l] }

// SupportedLocales returns the closed list in stable order so the language
// switcher in the UI gets a deterministic rendering.
func SupportedLocales() []Locale { return []Locale{LocaleEN, LocaleZH} }

// LocaleLabels are the human-readable names rendered in the language
// switcher. Each label is in its own language (English shows "English",
// Chinese shows "中文") because that's the convention every multilingual
// site we surveyed follows — readers recognise their own language faster
// than the active-locale-translated form.
var LocaleLabels = map[Locale]string{
	LocaleEN: "English",
	LocaleZH: "中文",
}

// LocaleCookieName is the cookie the language switcher writes and the
// middleware reads. Kept in sync with the SPA's `kite_locale` localStorage
// key so a switch in one surface reflects in the other.
const LocaleCookieName = "kite_locale"

// ctxKey is unexported so callers can't accidentally collide with us via a
// raw string key. The single sentinel value is enough because we only stash
// one piece of data on the context.
type ctxKey struct{}

var localeKey ctxKey

// WithLocale returns a derived context that carries l. Middleware calls
// this once per request; handlers and templates never need to.
func WithLocale(ctx context.Context, l Locale) context.Context {
	return context.WithValue(ctx, localeKey, l)
}

// FromContext extracts the locale stamped by the middleware. It defaults to
// [DefaultLocale] for callers that run before middleware (tests, boot-time
// helpers) so they get useful output instead of an empty string.
func FromContext(ctx context.Context) Locale {
	if ctx == nil {
		return DefaultLocale
	}
	if v, ok := ctx.Value(localeKey).(Locale); ok && IsSupported(v) {
		return v
	}
	return DefaultLocale
}

// Pick resolves the active locale from the per-request inputs. The order —
// cookie before header — matches the user-perceived precedence: an explicit
// language switch should beat the browser's autodetected list. Either
// argument may be empty; both empty means [DefaultLocale].
func Pick(cookieValue, acceptLanguage string) Locale {
	if cookieValue != "" {
		l := Locale(strings.ToLower(strings.TrimSpace(cookieValue)))
		if IsSupported(l) {
			return l
		}
	}
	if acceptLanguage != "" {
		// Accept-Language is a comma-separated list of `lang;q=...`
		// entries. We don't honour the q-values: real browsers list
		// preferred locales first, and the few cases where order
		// disagrees with q aren't worth the parser complexity.
		for _, raw := range strings.Split(acceptLanguage, ",") {
			tag := strings.TrimSpace(raw)
			if i := strings.Index(tag, ";"); i >= 0 {
				tag = tag[:i]
			}
			tag = strings.ToLower(tag)
			if tag == "" {
				continue
			}
			// Match by primary subtag — `zh-CN`, `zh-Hans-CN` and
			// bare `zh` all collapse to LocaleZH. Same for English.
			primary := tag
			if i := strings.Index(tag, "-"); i >= 0 {
				primary = tag[:i]
			}
			switch primary {
			case "zh":
				return LocaleZH
			case "en":
				return LocaleEN
			}
		}
	}
	return DefaultLocale
}

// formatMessage is a function-valued indirection over fmt.Sprintf. The
// indirection breaks `go vet`'s printf-wrapper detection at [T] and its
// callers — without it, vet would see args flowing into Sprintf and treat
// every call site as a printf-style call expecting matching directives in
// the *catalogue key* (e.g. `auth.login_failed`), which it never has.
var formatMessage = func(format string, args ...any) string {
	return fmt.Sprintf(format, args...)
}

// T returns the catalogue entry for key in locale, falling back to the
// English source (and finally to key itself) when the entry is missing.
//
// Format args follow Go's fmt.Sprintf conventions; pass none when the entry
// has no placeholders. Callers that already have a *gin.Context should reach
// for [internal/handler.M] / a template helper instead — those wrap T with
// the per-request locale.
func T(locale Locale, key string, args ...any) string {
	if entries, ok := Catalog[key]; ok {
		if msg, ok := entries[locale]; ok && msg != "" {
			if len(args) == 0 {
				return msg
			}
			return formatMessage(msg, args...)
		}
		if msg, ok := entries[LocaleEN]; ok && msg != "" {
			if len(args) == 0 {
				return msg
			}
			return formatMessage(msg, args...)
		}
	}
	// Returning the raw key (instead of "") makes missing-translation bugs
	// obvious in the UI without crashing the request.
	if len(args) == 0 {
		return key
	}
	return formatMessage(key, args...)
}
