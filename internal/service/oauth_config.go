package service

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"

	"github.com/amigoer/kite/internal/repo"
)

var (
	ErrOAuthProviderUnsupported = errors.New("oauth provider is not supported")
	ErrOAuthProviderDisabled    = errors.New("oauth provider is disabled")
	ErrOAuthProviderIncomplete  = errors.New("oauth provider is not fully configured")
	ErrOAuthSiteURLInvalid      = errors.New("site URL must use https in production or localhost during development")
)

type oauthProviderMeta struct {
	Key      string
	Label    string
	IconKey  string
	Protocol string
	Scopes   []string
}

var oauthProviders = []oauthProviderMeta{
	{
		Key:      "wechat",
		Label:    "微信",
		IconKey:  "wechat",
		Protocol: "微信网站扫码登录",
		Scopes:   []string{"snsapi_login"},
	},
	{
		Key:      "github",
		Label:    "GitHub",
		IconKey:  "github",
		Protocol: "OAuth 2.0",
		Scopes:   []string{"read:user", "user:email"},
	},
	{
		Key:      "google",
		Label:    "Google",
		IconKey:  "google",
		Protocol: "OpenID Connect",
		Scopes:   []string{"openid", "profile", "email"},
	},
}

// OAuthProviderConfig is the normalized configuration and metadata for a provider.
type OAuthProviderConfig struct {
	Key          string   `json:"key"`
	Label        string   `json:"label"`
	IconKey      string   `json:"icon_key"`
	Protocol     string   `json:"protocol"`
	Enabled      bool     `json:"enabled"`
	ClientID     string   `json:"client_id"`
	ClientSecret string   `json:"-"`
	HasSecret    bool     `json:"has_secret"`
	CallbackURL  string   `json:"callback_url"`
	IsConfigured bool     `json:"is_configured"`
	Scopes       []string `json:"scopes"`
	SiteURL      string   `json:"site_url"`
	SiteURLValid bool     `json:"site_url_valid"`
}

// PublicOAuthProvider is the minimal provider payload exposed to anonymous pages.
type PublicOAuthProvider struct {
	Key     string `json:"key"`
	Label   string `json:"label"`
	IconKey string `json:"icon_key"`
}

// OAuthProviderUpdate is the admin payload for mutating one provider.
type OAuthProviderUpdate struct {
	Enabled      bool
	ClientID     string
	ClientSecret string
}

// OAuthConfigService hides the raw settings keys behind a provider-centric API.
type OAuthConfigService struct {
	settingRepo      *repo.SettingRepo
	defaultSiteURL   string
	providersByKey   map[string]oauthProviderMeta
	providersInOrder []oauthProviderMeta
}

func NewOAuthConfigService(settingRepo *repo.SettingRepo, defaultSiteURL string) *OAuthConfigService {
	byKey := make(map[string]oauthProviderMeta, len(oauthProviders))
	for _, provider := range oauthProviders {
		byKey[provider.Key] = provider
	}
	return &OAuthConfigService{
		settingRepo:      settingRepo,
		defaultSiteURL:   defaultSiteURL,
		providersByKey:   byKey,
		providersInOrder: oauthProviders,
	}
}

func (s *OAuthConfigService) providerMeta(key string) (oauthProviderMeta, error) {
	meta, ok := s.providersByKey[key]
	if !ok {
		return oauthProviderMeta{}, ErrOAuthProviderUnsupported
	}
	return meta, nil
}

func (s *OAuthConfigService) enabledKey(provider string) string {
	return "oauth_" + provider + "_enabled"
}

func (s *OAuthConfigService) clientIDKey(provider string) string {
	return "oauth_" + provider + "_client_id"
}

func (s *OAuthConfigService) clientSecretKey(provider string) string {
	return "oauth_" + provider + "_client_secret"
}

// GetSiteURL returns the effective site URL, falling back to the compiled-in default.
func (s *OAuthConfigService) GetSiteURL(ctx context.Context) (string, error) {
	if s.settingRepo == nil {
		return strings.TrimSpace(s.defaultSiteURL), nil
	}
	return s.settingRepo.GetOrDefault(ctx, "site_url", strings.TrimSpace(s.defaultSiteURL))
}

// GetProvider returns the normalized configuration for a provider.
func (s *OAuthConfigService) GetProvider(ctx context.Context, key string) (*OAuthProviderConfig, error) {
	meta, err := s.providerMeta(key)
	if err != nil {
		return nil, err
	}

	enabled := false
	if s.settingRepo != nil {
		enabled, err = s.settingRepo.GetBool(ctx, s.enabledKey(key), false)
		if err != nil {
			return nil, err
		}
	}

	clientID := ""
	clientSecret := ""
	if s.settingRepo != nil {
		clientID, err = s.settingRepo.GetOrDefault(ctx, s.clientIDKey(key), "")
		if err != nil {
			return nil, err
		}
		clientSecret, err = s.settingRepo.GetOrDefault(ctx, s.clientSecretKey(key), "")
		if err != nil {
			return nil, err
		}
	}

	siteURL, err := s.GetSiteURL(ctx)
	if err != nil {
		return nil, err
	}

	cfg := &OAuthProviderConfig{
		Key:          meta.Key,
		Label:        meta.Label,
		IconKey:      meta.IconKey,
		Protocol:     meta.Protocol,
		Enabled:      enabled,
		ClientID:     strings.TrimSpace(clientID),
		ClientSecret: strings.TrimSpace(clientSecret),
		HasSecret:    strings.TrimSpace(clientSecret) != "",
		Scopes:       append([]string(nil), meta.Scopes...),
		SiteURL:      strings.TrimSpace(siteURL),
	}
	cfg.SiteURLValid = isAllowedOAuthSiteURL(cfg.SiteURL)
	cfg.CallbackURL = buildOAuthCallbackURL(cfg.SiteURL, cfg.Key)
	cfg.IsConfigured = cfg.ClientID != "" && cfg.ClientSecret != "" && cfg.CallbackURL != ""

	return cfg, nil
}

// ListProviders returns all providers in the configured display order.
func (s *OAuthConfigService) ListProviders(ctx context.Context) ([]OAuthProviderConfig, error) {
	items := make([]OAuthProviderConfig, 0, len(s.providersInOrder))
	for _, provider := range s.providersInOrder {
		cfg, err := s.GetProvider(ctx, provider.Key)
		if err != nil {
			return nil, err
		}
		items = append(items, *cfg)
	}
	return items, nil
}

// ListPublicProviders returns the subset of providers that are enabled and fully configured.
func (s *OAuthConfigService) ListPublicProviders(ctx context.Context) ([]PublicOAuthProvider, error) {
	configs, err := s.ListProviders(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]PublicOAuthProvider, 0, len(configs))
	for _, cfg := range configs {
		if cfg.Enabled && cfg.IsConfigured && cfg.SiteURLValid {
			result = append(result, PublicOAuthProvider{
				Key:     cfg.Key,
				Label:   cfg.Label,
				IconKey: cfg.IconKey,
			})
		}
	}
	return result, nil
}

// UpdateProvider persists the admin configuration for a provider.
func (s *OAuthConfigService) UpdateProvider(ctx context.Context, key string, update OAuthProviderUpdate) (*OAuthProviderConfig, error) {
	if _, err := s.providerMeta(key); err != nil {
		return nil, err
	}

	current, err := s.GetProvider(ctx, key)
	if err != nil {
		return nil, err
	}

	clientID := strings.TrimSpace(update.ClientID)
	if clientID == "" {
		clientID = current.ClientID
	}

	clientSecret := strings.TrimSpace(update.ClientSecret)
	if clientSecret == "" {
		clientSecret = current.ClientSecret
	}

	siteURL := current.SiteURL
	if update.Enabled {
		if !isAllowedOAuthSiteURL(siteURL) {
			return nil, ErrOAuthSiteURLInvalid
		}
		if clientID == "" || clientSecret == "" {
			return nil, ErrOAuthProviderIncomplete
		}
	}

	settings := map[string]string{
		s.enabledKey(key):  boolToString(update.Enabled),
		s.clientIDKey(key): clientID,
	}
	if strings.TrimSpace(update.ClientSecret) != "" {
		settings[s.clientSecretKey(key)] = clientSecret
	}

	if s.settingRepo != nil {
		if err := s.settingRepo.SetBatch(ctx, settings); err != nil {
			return nil, err
		}
	}

	return s.GetProvider(ctx, key)
}

// RequireEnabledProvider returns a provider config only when it can be used for login.
func (s *OAuthConfigService) RequireEnabledProvider(ctx context.Context, key string) (*OAuthProviderConfig, error) {
	cfg, err := s.GetProvider(ctx, key)
	if err != nil {
		return nil, err
	}
	if !cfg.Enabled {
		return nil, ErrOAuthProviderDisabled
	}
	if !cfg.IsConfigured || !cfg.SiteURLValid {
		if !cfg.SiteURLValid {
			return nil, ErrOAuthSiteURLInvalid
		}
		return nil, ErrOAuthProviderIncomplete
	}
	return cfg, nil
}

func buildOAuthCallbackURL(siteURL, provider string) string {
	trimmed := strings.TrimRight(strings.TrimSpace(siteURL), "/")
	if trimmed == "" {
		return ""
	}
	return trimmed + "/api/v1/auth/oauth/" + provider + "/callback"
}

func isAllowedOAuthSiteURL(siteURL string) bool {
	parsed, err := url.Parse(strings.TrimSpace(siteURL))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return false
	}
	if strings.EqualFold(parsed.Scheme, "https") {
		return true
	}
	host := parsed.Hostname()
	if strings.EqualFold(parsed.Scheme, "http") {
		switch host {
		case "localhost", "127.0.0.1", "::1":
			return true
		}
		if ip := net.ParseIP(host); ip != nil && ip.IsLoopback() {
			return true
		}
	}
	return false
}

func boolToString(v bool) string {
	if v {
		return "true"
	}
	return "false"
}

func providerKeys() []string {
	keys := make([]string, 0, len(oauthProviders))
	for _, provider := range oauthProviders {
		keys = append(keys, provider.Key)
	}
	return keys
}

func formatOAuthError(provider string, err error) error {
	return fmt.Errorf("%s oauth: %w", provider, err)
}
