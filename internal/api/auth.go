package api

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/kite-plus/kite/internal/errcodes"
	"github.com/kite-plus/kite/internal/middleware"
	"github.com/kite-plus/kite/internal/service"
)

// LoginInput is the body the client posts to /auth/login. The username field
// accepts either the literal username or the user's email address — the
// service layer falls through both lookups.
type LoginInput struct {
	Body struct {
		Username string `json:"username" required:"true" minLength:"1" maxLength:"64" doc:"Account username or email." example:"alice"`
		Password string `json:"password" required:"true" minLength:"1" maxLength:"128" doc:"Account password."`
	}
}

// LoginData is the success payload of /auth/login. When the user has TOTP
// enabled the access/refresh tokens are empty and `pending_2fa` is true; the
// client then exchanges the challenge token at /auth/2fa/verify.
type LoginData struct {
	AccessToken      string    `json:"access_token,omitempty" doc:"Short-lived JWT access token. Empty when pending_2fa=true."`
	RefreshToken     string    `json:"refresh_token,omitempty" doc:"Refresh token; long-lived. Empty when pending_2fa=true."`
	ExpiresAt        time.Time `json:"expires_at,omitzero" doc:"Wall-clock expiry of the access token."`
	RefreshExpiresAt time.Time `json:"refresh_expires_at,omitzero" doc:"Wall-clock expiry of the refresh token."`
	Pending2FA       bool      `json:"pending_2fa,omitempty" doc:"True when the account requires a TOTP code; client must call /auth/2fa/verify next."`
	ChallengeToken   string    `json:"challenge_token,omitempty" doc:"Short-lived 2FA challenge; only present when pending_2fa=true."`
	ChallengeExpiry  time.Time `json:"challenge_expiry,omitzero" doc:"Wall-clock expiry of the challenge token."`
}

// LoginOutput wraps the typed login payload in the standard envelope.
type LoginOutput struct {
	Body Envelope[LoginData]
}

// RefreshInput accepts the refresh token in the body for non-browser clients.
// Browser clients can leave the body empty; the cookie set by /auth/login is
// used instead.
type RefreshInput struct {
	Body struct {
		RefreshToken string `json:"refresh_token,omitempty" doc:"Refresh token from a prior /auth/login response. Optional when the request carries the refresh_token cookie."`
	}
}

// RefreshOutput is shaped identically to LoginOutput so SDKs can reuse the
// same data type.
type RefreshOutput struct {
	Body Envelope[LoginData]
}

// LogoutInput is empty — auth is via Authorization header / cookie.
type LogoutInput struct{}

// LogoutOutput carries the empty success envelope.
type LogoutOutput struct {
	Body Envelope[struct{}]
}

// ProfileData is the shape of GET /profile. Mirrors the legacy gin handler so
// existing clients keep working.
type ProfileData struct {
	UserID             string    `json:"user_id" doc:"Stable user identifier (UUID)."`
	Username           string    `json:"username"`
	Nickname           *string   `json:"nickname,omitempty"`
	Email              string    `json:"email"`
	AvatarURL          *string   `json:"avatar_url,omitempty"`
	HasLocalPassword   bool      `json:"has_local_password" doc:"False when the account is OAuth-only and has never set a local password."`
	Role               string    `json:"role" doc:"Either \"admin\" or \"user\"." enum:"admin,user"`
	PasswordMustChange bool      `json:"password_must_change" doc:"True for the bootstrap admin account; clients must redirect to first-login flow."`
	StorageLimit       int64     `json:"storage_limit" doc:"Per-user upload quota in bytes; 0 means no quota."`
	StorageUsed        int64     `json:"storage_used" doc:"Bytes consumed by this user's files."`
	TOTPEnabled        bool      `json:"totp_enabled"`
	CreatedAt          time.Time `json:"created_at"`
}

// ProfileInput carries no fields — the user identity comes from the Auth
// middleware's context.
type ProfileInput struct{}

// ProfileOutput wraps the profile payload.
type ProfileOutput struct {
	Body Envelope[ProfileData]
}

// registerAuth wires the typed authentication operations onto the huma API.
// The corresponding gin routes in router/auth.go must be removed for the
// paths registered here, or gin will report duplicate-handler conflicts.
func registerAuth(api huma.API, deps Deps) {
	// Public auth ops inherit the per-IP rate limit so credential stuffing
	// doesn't get cheaper just because the route is now huma-typed.
	publicMW := huma.Middlewares{}
	if deps.AuthRateLimitMW != nil {
		publicMW = append(publicMW, ginMW(deps.AuthRateLimitMW))
	}

	huma.Register(api, huma.Operation{
		OperationID: "auth-login",
		Method:      http.MethodPost,
		Path:        "/api/v1/auth/login",
		Summary:     "Sign in with username and password",
		Description: "Authenticates the caller and returns a token pair. When TOTP is enabled the response carries `pending_2fa` instead — the client must follow up with `/auth/2fa/verify` using the returned `challenge_token`.\n\nA successful response also sets the `access_token` and `refresh_token` HttpOnly cookies for browser clients; non-browser clients should keep the JSON body's tokens.",
		Tags:        []string{"Auth"},
		Middlewares: publicMW,
	}, func(ctx context.Context, in *LoginInput) (*LoginOutput, error) {
		result, err := deps.AuthSvc.LoginOrChallenge(ctx, in.Body.Username, in.Body.Password)
		if err != nil {
			if errors.Is(err, service.ErrInvalidCredentials) || errors.Is(err, service.ErrUserInactive) {
				return nil, Errf(errcodes.Unauthorized, "%s", err.Error())
			}
			return nil, Errf(errcodes.InternalError, "login failed")
		}

		// 2FA branch: hand back the challenge instead of a token pair.
		if result.Challenge != nil {
			return &LoginOutput{Body: Ok(LoginData{
				Pending2FA:      true,
				ChallengeToken:  result.Challenge.Token,
				ChallengeExpiry: result.Challenge.ExpiresAt,
			})}, nil
		}

		// Non-2FA branch: also set HttpOnly cookies so the SPA stays
		// logged in without exposing the tokens to JS.
		writeAuthCookiesFromCtx(ctx, result.Tokens)

		return &LoginOutput{Body: Ok(LoginData{
			AccessToken:      result.Tokens.AccessToken,
			RefreshToken:     result.Tokens.RefreshToken,
			ExpiresAt:        result.Tokens.ExpiresAt,
			RefreshExpiresAt: result.Tokens.RefreshExpiresAt,
		})}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "auth-refresh",
		Method:      http.MethodPost,
		Path:        "/api/v1/auth/refresh",
		Summary:     "Exchange a refresh token for a fresh access/refresh pair",
		Description: "Reads the refresh token from (in order) the `refresh_token` HttpOnly cookie or the JSON body. Both the access and refresh tokens are rotated — keep the new pair and discard the old one.",
		Tags:        []string{"Auth"},
		Middlewares: publicMW,
	}, func(ctx context.Context, in *RefreshInput) (*RefreshOutput, error) {
		token := strings.TrimSpace(in.Body.RefreshToken)
		if token == "" {
			if c := ginContextFromHuma(ctx); c != nil {
				if cookie, err := c.Cookie("refresh_token"); err == nil {
					token = strings.TrimSpace(cookie)
				}
			}
		}
		if token == "" {
			return nil, Errf(errcodes.Unauthorized, "refresh_token is required")
		}
		pair, err := deps.AuthSvc.RefreshToken(ctx, token)
		if err != nil {
			return nil, Errf(errcodes.InvalidToken, "invalid refresh token")
		}
		writeAuthCookiesFromCtx(ctx, pair)
		return &RefreshOutput{Body: Ok(LoginData{
			AccessToken:      pair.AccessToken,
			RefreshToken:     pair.RefreshToken,
			ExpiresAt:        pair.ExpiresAt,
			RefreshExpiresAt: pair.RefreshExpiresAt,
		})}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "auth-logout",
		Method:      http.MethodPost,
		Path:        "/api/v1/auth/logout",
		Summary:     "Clear the session cookies",
		Description: "Browser clients should call this to drop the HttpOnly cookies. The bearer token in the Authorization header (if any) remains valid until its natural expiry — there is no server-side revocation store; if you need to invalidate JWTs use change-password / first-login-reset.",
		Tags:        []string{"Auth"},
		Security:    []map[string][]string{{"bearer": {}}},
		Middlewares: huma.Middlewares{ginMW(deps.AuthMW)},
	}, func(ctx context.Context, _ *LogoutInput) (*LogoutOutput, error) {
		clearAuthCookiesFromCtx(ctx)
		return &LogoutOutput{Body: Ok(struct{}{})}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "profile-get",
		Method:      http.MethodGet,
		Path:        "/api/v1/profile",
		Summary:     "Return the current user's profile",
		Description: "Returns the canonical profile for the authenticated caller. Used by clients to seed local state after a successful login or refresh.",
		Tags:        []string{"Auth"},
		Security:    []map[string][]string{{"bearer": {}}},
		Middlewares: huma.Middlewares{ginMW(deps.AuthMW)},
	}, func(ctx context.Context, _ *ProfileInput) (*ProfileOutput, error) {
		c := ginContextFromHuma(ctx)
		if c == nil {
			return nil, Errf(errcodes.InternalError, "missing request context")
		}
		userID := c.GetString(middleware.ContextKeyUserID)
		if userID == "" {
			return nil, Errf(errcodes.Unauthorized, "missing user context")
		}
		user, err := deps.AuthSvc.UserRepo().GetByID(ctx, userID)
		if err != nil {
			return nil, Errf(errcodes.Unauthorized, "user not found")
		}
		return &ProfileOutput{Body: Ok(ProfileData{
			UserID:             user.ID,
			Username:           user.Username,
			Nickname:           user.Nickname,
			Email:              user.Email,
			AvatarURL:          user.AvatarURL,
			HasLocalPassword:   user.HasLocalPassword,
			Role:               user.Role,
			PasswordMustChange: user.PasswordMustChange,
			StorageLimit:       user.StorageLimit,
			StorageUsed:        user.StorageUsed,
			TOTPEnabled:        user.TOTPEnabled,
			CreatedAt:          user.CreatedAt,
		})}, nil
	})
}

// writeAuthCookiesFromCtx mirrors the gin handler's cookie helper but works
// from a huma context so the typed handlers can rotate cookies the same way
// the legacy ones do.
func writeAuthCookiesFromCtx(ctx context.Context, tokens *service.TokenPair) {
	c := ginContextFromHuma(ctx)
	if c == nil || tokens == nil {
		return
	}
	writeAccessTokenCookieGin(c, tokens.AccessToken, tokens.ExpiresAt)
	refreshExpiry := tokens.RefreshExpiresAt
	if refreshExpiry.IsZero() {
		refreshExpiry = tokens.ExpiresAt
	}
	writeRefreshTokenCookieGin(c, tokens.RefreshToken, refreshExpiry)
}

func clearAuthCookiesFromCtx(ctx context.Context) {
	c := ginContextFromHuma(ctx)
	if c == nil {
		return
	}
	writeAccessTokenCookieGin(c, "", time.Unix(0, 0))
	writeRefreshTokenCookieGin(c, "", time.Unix(0, 0))
}
