package api

import (
	"context"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humagin"
	"github.com/gin-gonic/gin"
)

// ginContextKey is the value type the [GinContextInjector] middleware stuffs
// into the request context so typed huma handlers can retrieve the underlying
// *gin.Context (needed for cookie writes, X-Forwarded-Proto checks, etc.).
type ginContextKey struct{}

// GinContextInjector is a gin middleware that copies the *gin.Context onto
// the request's context.Context. Typed handlers in this package retrieve it
// via [ginContextFromHuma]. Without this middleware the typed handlers would
// have no way to set HttpOnly cookies on the response, since huma's typed
// handler signature surfaces only context.Context, not *gin.Context.
func GinContextInjector() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := context.WithValue(c.Request.Context(), ginContextKey{}, c)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

// ginContextFromHuma returns the underlying *gin.Context for a typed handler
// invocation. Returns nil when the request didn't pass through
// [GinContextInjector] (e.g. unit tests that hit the huma API directly).
func ginContextFromHuma(ctx context.Context) *gin.Context {
	if c, ok := ctx.Value(ginContextKey{}).(*gin.Context); ok {
		return c
	}
	return nil
}

// ginMW lifts a regular gin middleware into a huma per-operation middleware
// so we can pin auth / admin requirements without forking the originals from
// internal/middleware. The huma adapter's gin context is unwrapped so the
// middleware sees the same *gin.Context it would in a vanilla gin route.
func ginMW(mw gin.HandlerFunc) func(huma.Context, func(huma.Context)) {
	if mw == nil {
		return func(ctx huma.Context, next func(huma.Context)) { next(ctx) }
	}
	return func(ctx huma.Context, next func(huma.Context)) {
		c := humagin.Unwrap(ctx)
		mw(c)
		if c.IsAborted() {
			// gin already wrote the response; signal huma to stop the chain.
			return
		}
		next(ctx)
	}
}

// Cookie helpers — duplicated from internal/handler/auth_social.go because
// importing that package would create a cycle (handler→api→handler). The
// cookie names, paths and SameSite policies must stay in lockstep with the
// original helpers; tests on the legacy gin handlers cover the contract.
const (
	accessTokenCookieName  = "access_token"
	refreshTokenCookieName = "refresh_token"
	refreshTokenCookiePath = "/api/v1/auth"
)

func writeAccessTokenCookieGin(c *gin.Context, token string, expiresAt time.Time) {
	maxAge := maxAgeFromExpiry(expiresAt)
	if token == "" {
		maxAge = -1
	}
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     accessTokenCookieName,
		Value:    token,
		Path:     "/",
		Expires:  expiresAt,
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   isSecureRequestGin(c),
		SameSite: http.SameSiteLaxMode,
	})
}

func writeRefreshTokenCookieGin(c *gin.Context, token string, expiresAt time.Time) {
	maxAge := maxAgeFromExpiry(expiresAt)
	if token == "" {
		maxAge = -1
	}
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     refreshTokenCookieName,
		Value:    token,
		Path:     refreshTokenCookiePath,
		Expires:  expiresAt,
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   isSecureRequestGin(c),
		SameSite: http.SameSiteStrictMode,
	})
}

// maxAgeFromExpiry mirrors the legacy helper — returns -1 (deleted cookie)
// when the expiry is in the past, otherwise the seconds-until-expiry.
func maxAgeFromExpiry(expiresAt time.Time) int {
	if expiresAt.IsZero() {
		return 0
	}
	d := time.Until(expiresAt)
	if d <= 0 {
		return -1
	}
	return int(d.Seconds())
}

// isSecureRequestGin returns true if the inbound request came in over TLS.
// It honours the X-Forwarded-Proto header so reverse-proxied deployments
// still flag cookies as Secure.
func isSecureRequestGin(c *gin.Context) bool {
	if c.Request.TLS != nil {
		return true
	}
	if proto := c.GetHeader("X-Forwarded-Proto"); proto == "https" {
		return true
	}
	return false
}
