package api

import (
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humagin"
	"github.com/gin-gonic/gin"
	"github.com/kite-plus/kite/internal/errcodes"
	"github.com/kite-plus/kite/internal/service"
	"github.com/kite-plus/kite/internal/version"
)

// Deps bundles the service-layer dependencies typed handlers need. New
// endpoints can pull what they want without forcing every operation to take
// every dependency in its constructor.
type Deps struct {
	AuthSvc         *service.AuthService
	AuthMW          gin.HandlerFunc // applied to operations that require sign-in
	AdminMW         gin.HandlerFunc // applied to admin-only operations
	AuthRateLimitMW gin.HandlerFunc // per-IP rate limit for unauthenticated auth endpoints
}

// Register attaches the typed OpenAPI-described operations to the gin engine
// and returns the huma.API instance so callers can publish the spec or mount
// extra routes around it.
//
// Operation handlers live in sibling files (auth.go, tokens.go, …). Each one
// calls huma.Register against the api returned here.
func Register(r *gin.Engine, deps Deps) huma.API {
	cfg := huma.DefaultConfig("Kite", version.APIVersion)
	cfg.Info.Description = "Public HTTP API for the Kite media-hosting server. " +
		"Authentication is via Bearer token (JWT or PAT). The wire format wraps every " +
		"response in {code, message, data}; non-zero `code` values are documented in " +
		"the errcodes catalog."
	cfg.Info.License = &huma.License{Name: "MIT"}

	// Move the spec, docs UI and JSON-Schema mounts under /api/v1 so they sit
	// alongside the operations they describe instead of polluting the root.
	cfg.OpenAPIPath = "/api/v1/openapi"
	cfg.DocsPath = "/api/v1/docs"
	cfg.SchemasPath = "/api/v1/schemas"

	// Server URL hint for SDK generators. Empty string lets the spec be
	// served relative to whatever origin loads it.
	cfg.Servers = []*huma.Server{
		{URL: "/", Description: "Same-origin (default)"},
	}

	// Document the two authentication schemes operations can require.
	if cfg.Components == nil {
		cfg.Components = &huma.Components{}
	}
	cfg.Components.SecuritySchemes = map[string]*huma.SecurityScheme{
		"bearer": {
			Type:         "http",
			Scheme:       "bearer",
			BearerFormat: "JWT or PAT (personal access token)",
			Description:  "Send `Authorization: Bearer <token>`. JWT issued by /auth/login and refreshed via /auth/refresh; PATs minted at /tokens never expire unless deleted.",
		},
	}

	// Override huma's default error builder so non-success responses come
	// out as our {code, message, data: null} envelope rather than RFC 7807
	// problem details. Kept tiny: huma supplies the HTTP status, we map it
	// back to a generic errcodes constant if no specific one was thrown.
	huma.NewError = func(status int, message string, errs ...error) huma.StatusError {
		return &APIError{
			Status:  status,
			Code:    defaultCodeFor(status),
			Message: message,
		}
	}

	api := humagin.New(r, cfg)

	registerAuth(api, deps)
	registerTokens(api, deps)

	return api
}

// defaultCodeFor maps an HTTP status to the generic errcodes constant for
// that status range. Used when huma synthesises errors (e.g. on body
// validation) and a more specific business code wasn't supplied.
func defaultCodeFor(status int) errcodes.Code {
	switch status {
	case http.StatusBadRequest, http.StatusUnprocessableEntity:
		return errcodes.BadRequest
	case http.StatusUnauthorized:
		return errcodes.Unauthorized
	case http.StatusForbidden:
		return errcodes.Forbidden
	case http.StatusNotFound:
		return errcodes.NotFound
	case http.StatusConflict:
		return errcodes.Conflict
	case http.StatusGone:
		return errcodes.Gone
	case http.StatusRequestEntityTooLarge:
		return errcodes.PayloadTooLarge
	case http.StatusUnsupportedMediaType:
		return errcodes.UnsupportedFormat
	case http.StatusFailedDependency:
		return errcodes.UpstreamFailed
	case http.StatusTooManyRequests:
		return errcodes.TooManyRequests
	case http.StatusInsufficientStorage:
		return errcodes.InsufficientStorage
	}
	return errcodes.InternalError
}
