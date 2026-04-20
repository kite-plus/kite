package handler

import (
	"errors"
	"net/http"

	"github.com/amigoer/kite/internal/service"
	"github.com/gin-gonic/gin"
)

// OAuthProviderAdminHandler manages third-party login provider settings.
type OAuthProviderAdminHandler struct {
	oauthConfigSvc *service.OAuthConfigService
}

func NewOAuthProviderAdminHandler(oauthConfigSvc *service.OAuthConfigService) *OAuthProviderAdminHandler {
	return &OAuthProviderAdminHandler{oauthConfigSvc: oauthConfigSvc}
}

type updateOAuthProviderRequest struct {
	Enabled      bool   `json:"enabled"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
}

// List returns all providers with their admin configuration metadata.
func (h *OAuthProviderAdminHandler) List(c *gin.Context) {
	items, err := h.oauthConfigSvc.ListProviders(c.Request.Context())
	if err != nil {
		ServerError(c, "failed to load oauth providers")
		return
	}
	Success(c, items)
}

// Update saves one provider's configuration.
func (h *OAuthProviderAdminHandler) Update(c *gin.Context) {
	var req updateOAuthProviderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "invalid oauth provider data: "+err.Error())
		return
	}

	item, err := h.oauthConfigSvc.UpdateProvider(c.Request.Context(), c.Param("provider"), service.OAuthProviderUpdate{
		Enabled:      req.Enabled,
		ClientID:     req.ClientID,
		ClientSecret: req.ClientSecret,
	})
	if err != nil {
		switch {
		case errors.Is(err, service.ErrOAuthProviderUnsupported):
			BadRequest(c, err.Error())
		case errors.Is(err, service.ErrOAuthProviderIncomplete):
			Fail(c, http.StatusBadRequest, 40030, err.Error())
		case errors.Is(err, service.ErrOAuthSiteURLInvalid):
			Fail(c, http.StatusBadRequest, 40031, err.Error())
		default:
			ServerError(c, "failed to update oauth provider")
		}
		return
	}

	Success(c, item)
}
