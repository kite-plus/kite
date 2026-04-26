package handler

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kite-plus/kite/internal/i18n"
	"github.com/kite-plus/kite/internal/middleware"
	"github.com/kite-plus/kite/internal/repo"
	"github.com/kite-plus/kite/internal/service"
)

// TokenHandler handles API token management HTTP requests.
type TokenHandler struct {
	authSvc   *service.AuthService
	tokenRepo *repo.APITokenRepo
}

func NewTokenHandler(authSvc *service.AuthService, tokenRepo *repo.APITokenRepo) *TokenHandler {
	return &TokenHandler{authSvc: authSvc, tokenRepo: tokenRepo}
}

type createTokenRequest struct {
	Name      string `json:"name" binding:"required,max=100"`
	ExpiresIn *int   `json:"expires_in"` // lifetime in days; nil means never expires
}

// Create issues a new API token.
func (h *TokenHandler) Create(c *gin.Context) {
	var req createTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, M(c, i18n.KeyTokenInvalidData, err.Error()))
		return
	}

	userID := c.GetString(middleware.ContextKeyUserID)

	var expiresAt *time.Time
	if req.ExpiresIn != nil && *req.ExpiresIn > 0 {
		t := time.Now().Add(time.Duration(*req.ExpiresIn) * 24 * time.Hour)
		expiresAt = &t
	}

	plainToken, token, err := h.authSvc.CreateAPIToken(c.Request.Context(), userID, req.Name, expiresAt)
	if err != nil {
		ServerError(c, M(c, i18n.KeyTokenCreateFailed))
		return
	}

	// The plaintext token is only returned here at creation time.
	Created(c, gin.H{
		"id":         token.ID,
		"name":       token.Name,
		"token":      plainToken,
		"expires_at": token.ExpiresAt,
		"created_at": token.CreatedAt,
	})
}

// List returns the current user's API tokens.
func (h *TokenHandler) List(c *gin.Context) {
	userID := c.GetString(middleware.ContextKeyUserID)

	tokens, err := h.tokenRepo.ListByUser(c.Request.Context(), userID)
	if err != nil {
		ServerError(c, M(c, i18n.KeyTokenListFailed))
		return
	}

	Success(c, tokens)
}

// Delete removes an API token.
func (h *TokenHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	userID := c.GetString(middleware.ContextKeyUserID)

	if err := h.tokenRepo.Delete(c.Request.Context(), id, userID); err != nil {
		NotFound(c, M(c, i18n.KeyTokenNotFound))
		return
	}

	Success(c, nil)
}
