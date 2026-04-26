// Package errcodes is the single source of truth for the business error codes
// returned in the {code, message, data} response envelope. Each code is a
// 5-digit decimal where the first three digits mirror the HTTP status (400,
// 401, 404 …) and the remaining two digits subdivide that status. Centralising
// them here lets the OpenAPI spec, the generated SDKs and the human-facing
// docs/error-codes.md stay in lock-step with the runtime.
//
// Conventions:
//   - Code 0 = success (used by the [handler.Success] / [handler.Created]
//     helpers).
//   - Codes are typed [Code] so the compiler catches typos when handlers
//     reference a non-existent constant.
//   - Some sub-codes (40001, 40002 …) are intentionally generic: the same
//     numeric code may be returned for unrelated validation failures from
//     different endpoints, with the human-readable message carrying the
//     specifics. Clients should classify by HTTP status range first and read
//     the message; matching on a specific 400xx sub-code is only meaningful
//     when the docs explicitly call it out (e.g. 40102 SessionRevoked).
//   - The catalog below pairs every code with its HTTP status and a short
//     summary so the OpenAPI generator and docs builder can emit a complete
//     reference.
package errcodes

import "net/http"

// Code is the type used for the `code` field in the API response envelope.
type Code int

// Success is the sentinel value indicating a non-error response. The HTTP
// status carries the intent (200 vs 201) — `code` only flags business errors.
const Success Code = 0

// 400xx — bad request / validation failures. Sub-codes are endpoint-specific
// variations on "your input is malformed". Clients should display the
// `message` field rather than branch on the precise sub-code.
const (
	BadRequest         Code = 40000 // generic — payload binding failed or fields missing
	InvalidArgumentA   Code = 40001 // first endpoint-specific validation variant (e.g. email format, TOTP code, password mismatch)
	InvalidArgumentB   Code = 40002 // second variant (e.g. email unchanged, password mismatch on disable-2fa)
	InvalidArgumentC   Code = 40003 // third variant (e.g. verify code expired, local password not set)
	InvalidUsername    Code = 40010 // username rejected by the policy (length / charset)
	InvalidPassword    Code = 40011 // password rejected by the policy (length / strength)
	InvalidOAuthState  Code = 40020 // OAuth state/code mismatch or expiry
	InvalidOAuthConfig Code = 40030 // admin saved an invalid provider configuration
	UnknownProvider    Code = 40031 // OAuth provider name not recognised
)

// 401xx — unauthenticated. The client is missing credentials, the credentials
// are invalid, or a session was revoked.
const (
	Unauthorized   Code = 40100 // generic 401 — authentication required or failed
	InvalidToken   Code = 40101 // bearer token malformed, unknown or expired
	SessionRevoked Code = 40102 // JWT version is older than the user's current version
)

// 403xx — authenticated but forbidden. The credentials are valid but the
// caller is not allowed to perform this operation.
const (
	Forbidden Code = 40300 // generic 403 — admin-only or feature-flag gated
)

// 404xx — the requested resource doesn't exist or has been deleted.
const (
	NotFound Code = 40400
)

// 409xx — the request would conflict with existing state (resource already
// exists, 2FA already enabled, etc.).
const (
	Conflict Code = 40900
)

// 410xx — the resource was once available but is gone permanently.
const (
	Gone Code = 41000 // verification code expired, share link revoked
)

// 413xx / 415xx — payload-level rejections.
const (
	PayloadTooLarge   Code = 41300 // upload exceeds the configured size cap
	UnsupportedFormat Code = 41500 // file extension or MIME type is on the deny list
)

// 424xx / 429xx — preconditions and rate limiting.
const (
	UpstreamFailed  Code = 42400 // dependent service (SMTP, OAuth provider) returned an error
	TooManyRequests Code = 42900 // per-IP / per-user rate limit hit
)

// 500xx / 507xx — server failures the client cannot recover from by retrying
// the same request unchanged.
const (
	InternalError       Code = 50000
	InsufficientStorage Code = 50700 // user quota or backend disk is full
)

// CodeInfo describes one error code for documentation and generated SDKs.
type CodeInfo struct {
	Code       Code
	HTTPStatus int
	Name       string
	Summary    string
	ClientHint string
}

// Catalog enumerates every business error code. Tests iterate this to
// guarantee every constant has a documented entry; the OpenAPI generator
// consumes it directly.
var Catalog = []CodeInfo{
	{Success, http.StatusOK, "Success", "Operation succeeded.", ""},

	{BadRequest, http.StatusBadRequest, "BadRequest", "Generic validation failure (binding or required fields).", "Surface message; do not retry without changing input."},
	{InvalidArgumentA, http.StatusBadRequest, "InvalidArgumentA", "Endpoint-specific validation variant #1 (see message for details).", "Surface message; user re-enters the offending field."},
	{InvalidArgumentB, http.StatusBadRequest, "InvalidArgumentB", "Endpoint-specific validation variant #2.", "Surface message; user re-enters the offending field."},
	{InvalidArgumentC, http.StatusBadRequest, "InvalidArgumentC", "Endpoint-specific validation variant #3.", "Surface message; user re-enters the offending field."},
	{InvalidUsername, http.StatusBadRequest, "InvalidUsername", "Username violates policy.", "Re-prompt with policy hints."},
	{InvalidPassword, http.StatusBadRequest, "InvalidPassword", "Password violates policy.", "Re-prompt with policy hints."},
	{InvalidOAuthState, http.StatusBadRequest, "InvalidOAuthState", "OAuth state/code is invalid.", "Restart the OAuth flow."},
	{InvalidOAuthConfig, http.StatusBadRequest, "InvalidOAuthConfig", "OAuth provider config is invalid.", "Admin must fix provider settings; clients should hide the provider."},
	{UnknownProvider, http.StatusBadRequest, "UnknownProvider", "OAuth provider not recognised.", "Refresh the available providers list."},

	{Unauthorized, http.StatusUnauthorized, "Unauthorized", "Authentication required or failed.", "Drop credentials and prompt sign-in."},
	{InvalidToken, http.StatusUnauthorized, "InvalidToken", "Token malformed, unknown, or expired.", "Try refresh once; on second 401 force sign-in."},
	{SessionRevoked, http.StatusUnauthorized, "SessionRevoked", "Server-side session was revoked.", "Drop credentials, force sign-in; do not retry."},

	{Forbidden, http.StatusForbidden, "Forbidden", "Caller lacks permission for this operation.", "Surface message; admin features should hide UI for non-admin role."},

	{NotFound, http.StatusNotFound, "NotFound", "Resource doesn't exist or was deleted.", "Refresh local cache; do not retry the same path."},

	{Conflict, http.StatusConflict, "Conflict", "Operation would conflict with existing state.", "Surface message; client may retry with different input."},

	{Gone, http.StatusGone, "Gone", "Resource was previously valid but is no longer.", "Force a fresh request from the source-of-truth."},

	{PayloadTooLarge, http.StatusRequestEntityTooLarge, "PayloadTooLarge", "Upload exceeds configured size cap.", "Surface limit; offer chunked upload if available."},
	{UnsupportedFormat, http.StatusUnsupportedMediaType, "UnsupportedFormat", "File type is on the deny list.", "Surface message; do not retry."},

	{UpstreamFailed, http.StatusFailedDependency, "UpstreamFailed", "Upstream dependency (SMTP, OAuth) failed.", "Retry with backoff; surface degraded-service notice."},
	{TooManyRequests, http.StatusTooManyRequests, "TooManyRequests", "Per-IP or per-user rate limit hit.", "Honour Retry-After if present; back off exponentially."},

	{InternalError, http.StatusInternalServerError, "InternalError", "Unexpected server failure.", "Retry once with backoff; report if persistent."},
	{InsufficientStorage, http.StatusInsufficientStorage, "InsufficientStorage", "User quota or backend storage is full.", "Surface quota dialog; do not retry."},
}

// HTTPStatus returns the canonical HTTP status for a known code, or 500 when
// the code isn't in the catalog.
func HTTPStatus(c Code) int {
	for i := range Catalog {
		if Catalog[i].Code == c {
			return Catalog[i].HTTPStatus
		}
	}
	return http.StatusInternalServerError
}
