package api

import (
	"context"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/kite-plus/kite/internal/errcodes"
	"github.com/kite-plus/kite/internal/i18n"
	"github.com/kite-plus/kite/internal/middleware"
	"github.com/kite-plus/kite/internal/model"
)

// TokenSummary is the public projection of an API token. The plaintext token
// is never included — it's only returned once, on creation.
type TokenSummary struct {
	ID        string     `json:"id" doc:"Stable token identifier."`
	Name      string     `json:"name" doc:"Caller-supplied label, e.g. 'My MacBook'."`
	LastUsed  *time.Time `json:"last_used,omitempty" doc:"Wall-clock timestamp of the last successful API call. Null when never used."`
	ExpiresAt *time.Time `json:"expires_at,omitempty" doc:"Wall-clock expiry. Null means the token never expires until deleted."`
	CreatedAt time.Time  `json:"created_at"`
}

// CreateTokenInput is the body for POST /tokens. ExpiresIn is in days; null
// or zero requests a token that never expires.
type CreateTokenInput struct {
	Body struct {
		Name      string `json:"name" required:"true" minLength:"1" maxLength:"100" doc:"Human-readable label shown in the management UI." example:"PicGo on my laptop"`
		ExpiresIn *int   `json:"expires_in,omitempty" minimum:"0" maximum:"3650" doc:"Lifetime in days. Omit or 0 for a token that never expires."`
	}
}

// CreatedToken is the one-time response: includes the plaintext token. The
// client must store this immediately — subsequent reads only return the
// summary fields.
type CreatedToken struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	Token     string     `json:"token" doc:"Plaintext token. ONLY returned at creation time — never readable again. Store securely (Keychain / DPAPI / libsecret)."`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

// CreateTokenOutput wraps the one-time payload. The HTTP status is pinned to
// 201 by the operation's DefaultStatus — we deliberately don't expose a
// `Status int` field here because huma reads such a field via reflection and
// would let its zero value override the default (resulting in a stray 200).
type CreateTokenOutput struct {
	Body Envelope[CreatedToken]
}

// ListTokensInput is empty — auth context identifies the user.
type ListTokensInput struct{}

// ListTokensOutput returns the caller's tokens (summary form).
type ListTokensOutput struct {
	Body Envelope[[]TokenSummary]
}

// DeleteTokenInput pins the token by URL path parameter.
type DeleteTokenInput struct {
	ID string `path:"id" doc:"Token identifier returned by /tokens."`
}

// DeleteTokenOutput carries the empty success envelope.
type DeleteTokenOutput struct {
	Body Envelope[struct{}]
}

// registerTokens registers PAT management endpoints. All operations require
// a signed-in user; tokens are scoped to the caller's user_id.
func registerTokens(api huma.API, deps Deps) {
	authMW := huma.Middlewares{ginMW(deps.AuthMW)}

	huma.Register(api, huma.Operation{
		OperationID: "tokens-list",
		Method:      http.MethodGet,
		Path:        "/api/v1/tokens",
		Summary:     "List the caller's API tokens",
		Description: "Returns a summary view of every PAT belonging to the authenticated caller. Plaintext tokens are NOT included; if the caller lost theirs they need to delete and recreate.",
		Tags:        []string{"Tokens"},
		Security:    []map[string][]string{{"bearer": {}}},
		Middlewares: authMW,
	}, func(ctx context.Context, _ *ListTokensInput) (*ListTokensOutput, error) {
		c := ginContextFromHuma(ctx)
		if c == nil {
			return nil, ErrKey(ctx, errcodes.InternalError, i18n.KeyErrMissingRequestCtx)
		}
		userID := c.GetString(middleware.ContextKeyUserID)
		if userID == "" {
			return nil, ErrKey(ctx, errcodes.Unauthorized, i18n.KeyErrMissingUserCtx)
		}
		records, err := deps.AuthSvc.TokenRepo().ListByUser(ctx, userID)
		if err != nil {
			return nil, ErrKey(ctx, errcodes.InternalError, i18n.KeyTokenListFailed)
		}
		out := make([]TokenSummary, 0, len(records))
		for i := range records {
			out = append(out, summaryFromRecord(&records[i]))
		}
		return &ListTokensOutput{Body: Ok(out)}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID:   "tokens-create",
		Method:        http.MethodPost,
		Path:          "/api/v1/tokens",
		Summary:       "Mint a new API token",
		Description:   "Creates a Personal Access Token. The plaintext value is returned ONCE in the response — clients must persist it immediately. Use these tokens with `Authorization: Bearer <token>` for any /api/v1/* endpoint that requires authentication.",
		Tags:          []string{"Tokens"},
		Security:      []map[string][]string{{"bearer": {}}},
		Middlewares:   authMW,
		DefaultStatus: http.StatusCreated,
	}, func(ctx context.Context, in *CreateTokenInput) (*CreateTokenOutput, error) {
		c := ginContextFromHuma(ctx)
		if c == nil {
			return nil, ErrKey(ctx, errcodes.InternalError, i18n.KeyErrMissingRequestCtx)
		}
		userID := c.GetString(middleware.ContextKeyUserID)
		if userID == "" {
			return nil, ErrKey(ctx, errcodes.Unauthorized, i18n.KeyErrMissingUserCtx)
		}

		var expiresAt *time.Time
		if in.Body.ExpiresIn != nil && *in.Body.ExpiresIn > 0 {
			t := time.Now().Add(time.Duration(*in.Body.ExpiresIn) * 24 * time.Hour)
			expiresAt = &t
		}
		plain, token, err := deps.AuthSvc.CreateAPIToken(ctx, userID, in.Body.Name, expiresAt)
		if err != nil {
			return nil, ErrKey(ctx, errcodes.InternalError, i18n.KeyTokenCreateFailed)
		}
		return &CreateTokenOutput{Body: Ok(CreatedToken{
			ID:        token.ID,
			Name:      token.Name,
			Token:     plain,
			ExpiresAt: token.ExpiresAt,
			CreatedAt: token.CreatedAt,
		})}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "tokens-delete",
		Method:      http.MethodDelete,
		Path:        "/api/v1/tokens/{id}",
		Summary:     "Revoke an API token",
		Description: "Deletes a PAT by id. The deletion is immediate — any in-flight requests using the revoked token will return 401 InvalidToken on their next call.",
		Tags:        []string{"Tokens"},
		Security:    []map[string][]string{{"bearer": {}}},
		Middlewares: authMW,
	}, func(ctx context.Context, in *DeleteTokenInput) (*DeleteTokenOutput, error) {
		c := ginContextFromHuma(ctx)
		if c == nil {
			return nil, ErrKey(ctx, errcodes.InternalError, i18n.KeyErrMissingRequestCtx)
		}
		userID := c.GetString(middleware.ContextKeyUserID)
		if userID == "" {
			return nil, ErrKey(ctx, errcodes.Unauthorized, i18n.KeyErrMissingUserCtx)
		}
		if err := deps.AuthSvc.TokenRepo().Delete(ctx, in.ID, userID); err != nil {
			return nil, ErrKey(ctx, errcodes.NotFound, i18n.KeyTokenNotFound)
		}
		return &DeleteTokenOutput{Body: Ok(struct{}{})}, nil
	})
}

// summaryFromRecord shapes a model.APIToken into the public summary form.
func summaryFromRecord(t *model.APIToken) TokenSummary {
	return TokenSummary{
		ID:        t.ID,
		Name:      t.Name,
		LastUsed:  t.LastUsed,
		ExpiresAt: t.ExpiresAt,
		CreatedAt: t.CreatedAt,
	}
}
