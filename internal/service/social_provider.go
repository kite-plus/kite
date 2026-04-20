package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"

	oidc "github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

// SocialProfile is the normalized user profile returned by any provider.
type SocialProfile struct {
	Provider        string
	ProviderUserID  string
	ProviderUnionID *string
	Email           *string
	EmailVerified   bool
	DisplayName     *string
	AvatarURL       *string
	RawProfile      string
}

type socialProvider interface {
	BuildAuthURL(cfg *OAuthProviderConfig, state, redirectURI, codeVerifier, nonce string) (string, error)
	Exchange(ctx context.Context, cfg *OAuthProviderConfig, code, redirectURI, codeVerifier, nonce string) (*SocialProfile, error)
}

type socialProviderRegistry struct {
	providers map[string]socialProvider
}

func newSocialProviderRegistry(httpClient *http.Client) *socialProviderRegistry {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &socialProviderRegistry{
		providers: map[string]socialProvider{
			"github": &githubSocialProvider{httpClient: httpClient},
			"google": &googleSocialProvider{httpClient: httpClient},
			"wechat": &wechatSocialProvider{httpClient: httpClient},
		},
	}
}

func (r *socialProviderRegistry) Get(provider string) (socialProvider, error) {
	item, ok := r.providers[provider]
	if !ok {
		return nil, ErrOAuthProviderUnsupported
	}
	return item, nil
}

type githubSocialProvider struct {
	httpClient *http.Client
}

func (p *githubSocialProvider) BuildAuthURL(cfg *OAuthProviderConfig, state, redirectURI, codeVerifier, _ string) (string, error) {
	values := url.Values{}
	values.Set("client_id", cfg.ClientID)
	values.Set("redirect_uri", redirectURI)
	values.Set("scope", strings.Join(cfg.Scopes, " "))
	values.Set("state", state)
	values.Set("code_challenge", pkceChallenge(codeVerifier))
	values.Set("code_challenge_method", "S256")
	return "https://github.com/login/oauth/authorize?" + values.Encode(), nil
}

func (p *githubSocialProvider) Exchange(ctx context.Context, cfg *OAuthProviderConfig, code, redirectURI, codeVerifier, _ string) (*SocialProfile, error) {
	values := url.Values{}
	values.Set("client_id", cfg.ClientID)
	values.Set("client_secret", cfg.ClientSecret)
	values.Set("code", code)
	values.Set("redirect_uri", redirectURI)
	values.Set("code_verifier", codeVerifier)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://github.com/login/oauth/access_token", strings.NewReader(values.Encode()))
	if err != nil {
		return nil, formatOAuthError("github", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		Error       string `json:"error"`
		Description string `json:"error_description"`
	}
	if err := doJSON(p.httpClient, req, &tokenResp); err != nil {
		return nil, formatOAuthError("github", err)
	}
	if tokenResp.AccessToken == "" {
		if tokenResp.Description != "" {
			return nil, formatOAuthError("github", errors.New(tokenResp.Description))
		}
		return nil, formatOAuthError("github", errors.New("missing access token"))
	}

	userReq, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/user", nil)
	if err != nil {
		return nil, formatOAuthError("github", err)
	}
	userReq.Header.Set("Accept", "application/vnd.github+json")
	userReq.Header.Set("Authorization", "Bearer "+tokenResp.AccessToken)

	var githubUser struct {
		ID        int64   `json:"id"`
		Login     string  `json:"login"`
		Name      *string `json:"name"`
		AvatarURL *string `json:"avatar_url"`
		Email     *string `json:"email"`
	}
	rawUser, err := doJSONRaw(p.httpClient, userReq, &githubUser)
	if err != nil {
		return nil, formatOAuthError("github", err)
	}

	emailReq, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/user/emails", nil)
	if err != nil {
		return nil, formatOAuthError("github", err)
	}
	emailReq.Header.Set("Accept", "application/vnd.github+json")
	emailReq.Header.Set("Authorization", "Bearer "+tokenResp.AccessToken)

	var emails []struct {
		Email    string `json:"email"`
		Verified bool   `json:"verified"`
		Primary  bool   `json:"primary"`
	}
	_, err = doJSONRaw(p.httpClient, emailReq, &emails)
	if err != nil {
		return nil, formatOAuthError("github", err)
	}

	profile := &SocialProfile{
		Provider:       "github",
		ProviderUserID: fmt.Sprintf("%d", githubUser.ID),
		AvatarURL:      cleanStringPtr(githubUser.AvatarURL),
		RawProfile:     string(rawUser),
	}
	if displayName := firstNonEmptyPtr(githubUser.Name, &githubUser.Login); displayName != nil {
		profile.DisplayName = cleanStringPtr(displayName)
	}

	for _, email := range emails {
		if email.Verified && email.Primary {
			profile.Email = cleanStringPtr(&email.Email)
			profile.EmailVerified = true
			return profile, nil
		}
	}
	for _, email := range emails {
		if email.Verified {
			profile.Email = cleanStringPtr(&email.Email)
			profile.EmailVerified = true
			return profile, nil
		}
	}
	if githubUser.Email != nil {
		profile.Email = cleanStringPtr(githubUser.Email)
	}
	return profile, nil
}

type googleSocialProvider struct {
	httpClient *http.Client
	mu         sync.Mutex
	issuer     *oidc.Provider
	issuerErr  error
}

func (p *googleSocialProvider) BuildAuthURL(cfg *OAuthProviderConfig, state, redirectURI, codeVerifier, nonce string) (string, error) {
	oauthCfg := oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		RedirectURL:  redirectURI,
		Scopes:       cfg.Scopes,
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://accounts.google.com/o/oauth2/v2/auth",
			TokenURL: "https://oauth2.googleapis.com/token",
		},
	}
	return oauthCfg.AuthCodeURL(
		state,
		oauth2.AccessTypeOnline,
		oauth2.S256ChallengeOption(codeVerifier),
		oauth2.SetAuthURLParam("nonce", nonce),
	), nil
}

func (p *googleSocialProvider) Exchange(ctx context.Context, cfg *OAuthProviderConfig, code, redirectURI, codeVerifier, nonce string) (*SocialProfile, error) {
	oauthCfg := oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		RedirectURL:  redirectURI,
		Scopes:       cfg.Scopes,
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://accounts.google.com/o/oauth2/v2/auth",
			TokenURL: "https://oauth2.googleapis.com/token",
		},
	}
	token, err := oauthCfg.Exchange(ctx, code, oauth2.VerifierOption(codeVerifier))
	if err != nil {
		return nil, formatOAuthError("google", err)
	}

	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok || strings.TrimSpace(rawIDToken) == "" {
		return nil, formatOAuthError("google", errors.New("missing id_token"))
	}

	issuer, err := p.getIssuer(ctx)
	if err != nil {
		return nil, formatOAuthError("google", err)
	}
	verifier := issuer.Verifier(&oidc.Config{ClientID: cfg.ClientID})
	idToken, err := verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return nil, formatOAuthError("google", err)
	}

	var claims struct {
		Sub           string `json:"sub"`
		Email         string `json:"email"`
		EmailVerified bool   `json:"email_verified"`
		Name          string `json:"name"`
		Picture       string `json:"picture"`
		Nonce         string `json:"nonce"`
	}
	if err := idToken.Claims(&claims); err != nil {
		return nil, formatOAuthError("google", err)
	}
	if nonce != "" && claims.Nonce != nonce {
		return nil, formatOAuthError("google", ErrOAuthStateInvalid)
	}

	rawClaims, err := json.Marshal(claims)
	if err != nil {
		return nil, formatOAuthError("google", err)
	}

	profile := &SocialProfile{
		Provider:       "google",
		ProviderUserID: claims.Sub,
		EmailVerified:  claims.EmailVerified,
		RawProfile:     string(rawClaims),
	}
	if claims.Email != "" {
		profile.Email = stringPtr(claims.Email)
	}
	if claims.Name != "" {
		profile.DisplayName = stringPtr(claims.Name)
	}
	if claims.Picture != "" {
		profile.AvatarURL = stringPtr(claims.Picture)
	}
	return profile, nil
}

func (p *googleSocialProvider) getIssuer(ctx context.Context) (*oidc.Provider, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.issuer != nil || p.issuerErr != nil {
		return p.issuer, p.issuerErr
	}
	p.issuer, p.issuerErr = oidc.NewProvider(ctx, "https://accounts.google.com")
	return p.issuer, p.issuerErr
}

type wechatSocialProvider struct {
	httpClient *http.Client
}

func (p *wechatSocialProvider) BuildAuthURL(cfg *OAuthProviderConfig, state, redirectURI, _, _ string) (string, error) {
	values := url.Values{}
	values.Set("appid", cfg.ClientID)
	values.Set("redirect_uri", redirectURI)
	values.Set("response_type", "code")
	values.Set("scope", strings.Join(cfg.Scopes, " "))
	values.Set("state", state)
	return "https://open.weixin.qq.com/connect/qrconnect?" + values.Encode() + "#wechat_redirect", nil
}

func (p *wechatSocialProvider) Exchange(ctx context.Context, cfg *OAuthProviderConfig, code, _, _, _ string) (*SocialProfile, error) {
	tokenURL := "https://api.weixin.qq.com/sns/oauth2/access_token?appid=" +
		url.QueryEscape(cfg.ClientID) +
		"&secret=" + url.QueryEscape(cfg.ClientSecret) +
		"&code=" + url.QueryEscape(code) +
		"&grant_type=authorization_code"

	tokenReq, err := http.NewRequestWithContext(ctx, http.MethodGet, tokenURL, nil)
	if err != nil {
		return nil, formatOAuthError("wechat", err)
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		OpenID      string `json:"openid"`
		UnionID     string `json:"unionid"`
		ErrCode     int    `json:"errcode"`
		ErrMsg      string `json:"errmsg"`
	}
	if err := doJSON(p.httpClient, tokenReq, &tokenResp); err != nil {
		return nil, formatOAuthError("wechat", err)
	}
	if tokenResp.ErrCode != 0 {
		return nil, formatOAuthError("wechat", errors.New(tokenResp.ErrMsg))
	}

	userURL := "https://api.weixin.qq.com/sns/userinfo?access_token=" +
		url.QueryEscape(tokenResp.AccessToken) +
		"&openid=" + url.QueryEscape(tokenResp.OpenID) +
		"&lang=zh_CN"
	userReq, err := http.NewRequestWithContext(ctx, http.MethodGet, userURL, nil)
	if err != nil {
		return nil, formatOAuthError("wechat", err)
	}

	var userResp struct {
		OpenID     string `json:"openid"`
		UnionID    string `json:"unionid"`
		Nickname   string `json:"nickname"`
		HeadImgURL string `json:"headimgurl"`
		ErrCode    int    `json:"errcode"`
		ErrMsg     string `json:"errmsg"`
	}
	rawUser, err := doJSONRaw(p.httpClient, userReq, &userResp)
	if err != nil {
		return nil, formatOAuthError("wechat", err)
	}
	if userResp.ErrCode != 0 {
		return nil, formatOAuthError("wechat", errors.New(userResp.ErrMsg))
	}

	profile := &SocialProfile{
		Provider:       "wechat",
		ProviderUserID: tokenResp.OpenID,
		RawProfile:     string(rawUser),
	}
	if union := firstNonEmpty(tokenResp.UnionID, userResp.UnionID); union != "" {
		profile.ProviderUnionID = stringPtr(union)
	}
	if userResp.Nickname != "" {
		profile.DisplayName = stringPtr(userResp.Nickname)
	}
	if userResp.HeadImgURL != "" {
		profile.AvatarURL = stringPtr(userResp.HeadImgURL)
	}
	return profile, nil
}

func doJSON(client *http.Client, req *http.Request, target any) error {
	_, err := doJSONRaw(client, req, target)
	return err
}

func doJSONRaw(client *http.Client, req *http.Request, target any) ([]byte, error) {
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	if err := json.Unmarshal(body, target); err != nil {
		return nil, err
	}
	return body, nil
}

func cleanStringPtr(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func stringPtr(value string) *string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func firstNonEmptyPtr(values ...*string) *string {
	for _, value := range values {
		if cleaned := cleanStringPtr(value); cleaned != nil {
			return cleaned
		}
	}
	return nil
}
