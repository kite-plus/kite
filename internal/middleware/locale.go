package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/kite-plus/kite/internal/i18n"
)

// localeContextKey is the gin-context key handlers use when they want the
// locale without poking at request.Context. We expose it as a string so
// callers can read it via c.Get / c.MustGet without importing this package.
const localeContextKey = "locale"

// Locale resolves the per-request locale and stamps it onto both the gin
// context (for handlers using c.Get) and the request's context.Context (for
// downstream code paths — huma operations, repos, services — that only see
// the standard context). Order of precedence:
//
//  1. The kite_locale cookie, set by the language switcher in the public
//     header / admin sidebar. This is the most explicit signal we have.
//  2. The Accept-Language header from the browser.
//  3. The package default, English.
//
// Mounted near the top of the router so every downstream middleware /
// handler / template can rely on the locale being present without extra
// nil-checks.
func Locale() gin.HandlerFunc {
	return func(c *gin.Context) {
		var cookieValue string
		if cookie, err := c.Cookie(i18n.LocaleCookieName); err == nil {
			cookieValue = cookie
		}
		acceptLang := c.GetHeader("Accept-Language")

		locale := i18n.Pick(cookieValue, acceptLang)

		c.Set(localeContextKey, locale)
		c.Request = c.Request.WithContext(i18n.WithLocale(c.Request.Context(), locale))
		c.Next()
	}
}

// LocaleFromGin pulls the locale stamped by [Locale] off the gin context.
// Useful for handlers that don't want to thread context.Context through
// just to call [i18n.FromContext]. Returns the package default when the
// middleware hasn't run (e.g. test contexts that bypass the router).
func LocaleFromGin(c *gin.Context) i18n.Locale {
	if v, ok := c.Get(localeContextKey); ok {
		if l, ok := v.(i18n.Locale); ok && i18n.IsSupported(l) {
			return l
		}
	}
	return i18n.DefaultLocale
}
