package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/amigoer/kite/internal/model"
	"github.com/amigoer/kite/internal/repo"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
)

const (
	OAuthModeLogin = "login"
	OAuthModeBind  = "bind"

	socialTicketKindLogin   = "login"
	socialTicketKindOnboard = "onboard"
)

var (
	ErrOAuthStateInvalid     = errors.New("oauth state is invalid or expired")
	ErrOAuthCodeMissing      = errors.New("oauth code is missing")
	ErrOAuthTicketInvalid    = errors.New("oauth ticket is invalid or expired")
	ErrOAuthBindingConflict  = errors.New("this third-party account is already linked to another user")
	ErrOAuthLoginRequired    = errors.New("login is required to bind a third-party account")
	ErrOAuthLastLoginMethod  = errors.New("cannot remove the last available login method")
	ErrOAuthOnboardingNeeded = errors.New("oauth onboarding is required")
)

type socialStateClaims struct {
	Provider     string `json:"provider"`
	Mode         string `json:"mode"`
	State        string `json:"state"`
	CodeVerifier string `json:"code_verifier,omitempty"`
	Nonce        string `json:"nonce,omitempty"`
	ReturnTo     string `json:"return_to"`
	UserID       string `json:"user_id,omitempty"`
	jwt.RegisteredClaims
}

type socialTicketClaims struct {
	Kind            string  `json:"kind"`
	Provider        string  `json:"provider"`
	UserID          string  `json:"user_id,omitempty"`
	ReturnTo        string  `json:"return_to"`
	ProviderUserID  string  `json:"provider_user_id,omitempty"`
	ProviderUnionID *string `json:"provider_union_id,omitempty"`
	Email           *string `json:"email,omitempty"`
	EmailVerified   bool    `json:"email_verified,omitempty"`
	DisplayName     *string `json:"display_name,omitempty"`
	AvatarURL       *string `json:"avatar_url,omitempty"`
	RawProfile      string  `json:"raw_profile,omitempty"`
	jwt.RegisteredClaims
}

// OAuthStartResult is the data a handler needs to start an OAuth redirect.
type OAuthStartResult struct {
	RedirectURL string
	CookieValue string
	ExpiresAt   time.Time
}

// UserIdentityStatus is the profile page view model for one provider.
type UserIdentityStatus struct {
	Key         string  `json:"key"`
	Label       string  `json:"label"`
	IconKey     string  `json:"icon_key"`
	Bound       bool    `json:"bound"`
	DisplayName *string `json:"display_name,omitempty"`
	Email       *string `json:"email,omitempty"`
}

// SocialAuthService orchestrates third-party login, onboarding, and account binding.
type SocialAuthService struct {
	authSvc                  *AuthService
	userRepo                 *repo.UserRepo
	identityRepo             *repo.UserIdentityRepo
	settingRepo              *repo.SettingRepo
	oauthConfigSvc           *OAuthConfigService
	registry                 *socialProviderRegistry
	jwtSecret                string
	allowRegistrationDefault bool
}

func NewSocialAuthService(
	authSvc *AuthService,
	userRepo *repo.UserRepo,
	identityRepo *repo.UserIdentityRepo,
	settingRepo *repo.SettingRepo,
	oauthConfigSvc *OAuthConfigService,
	jwtSecret string,
	allowRegistrationDefault bool,
) *SocialAuthService {
	return &SocialAuthService{
		authSvc:                  authSvc,
		userRepo:                 userRepo,
		identityRepo:             identityRepo,
		settingRepo:              settingRepo,
		oauthConfigSvc:           oauthConfigSvc,
		registry:                 newSocialProviderRegistry(&http.Client{Timeout: 10 * time.Second}),
		jwtSecret:                jwtSecret,
		allowRegistrationDefault: allowRegistrationDefault,
	}
}

// PrepareAuthRedirect creates the state cookie token and provider redirect URL.
func (s *SocialAuthService) PrepareAuthRedirect(
	ctx context.Context,
	provider, mode, returnTo, currentUserID string,
) (*OAuthStartResult, error) {
	if mode != OAuthModeBind {
		mode = OAuthModeLogin
	}
	cfg, err := s.oauthConfigSvc.RequireEnabledProvider(ctx, provider)
	if err != nil {
		return nil, err
	}
	if mode == OAuthModeBind && currentUserID == "" {
		return nil, ErrOAuthLoginRequired
	}

	rawState, err := randomURLSafeString(24)
	if err != nil {
		return nil, err
	}
	codeVerifier := ""
	if provider == "github" || provider == "google" {
		codeVerifier, err = randomURLSafeString(48)
		if err != nil {
			return nil, err
		}
	}
	nonce := ""
	if provider == "google" {
		nonce, err = randomURLSafeString(24)
		if err != nil {
			return nil, err
		}
	}

	now := time.Now()
	exp := now.Add(10 * time.Minute)
	claims := &socialStateClaims{
		Provider:     provider,
		Mode:         mode,
		State:        rawState,
		CodeVerifier: codeVerifier,
		Nonce:        nonce,
		ReturnTo:     sanitizeReturnTo(returnTo, defaultReturnTo(mode)),
		UserID:       currentUserID,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(exp),
			Subject:   provider,
		},
	}
	cookieValue, err := s.signClaims(claims)
	if err != nil {
		return nil, err
	}

	providerClient, err := s.registry.Get(provider)
	if err != nil {
		return nil, err
	}
	redirectURL, err := providerClient.BuildAuthURL(cfg, rawState, cfg.CallbackURL, codeVerifier, nonce)
	if err != nil {
		return nil, err
	}

	return &OAuthStartResult{
		RedirectURL: redirectURL,
		CookieValue: cookieValue,
		ExpiresAt:   exp,
	}, nil
}

// InspectStateCookie returns the mode and return target encoded in the state cookie.
func (s *SocialAuthService) InspectStateCookie(cookieValue string) (string, string, error) {
	claims := &socialStateClaims{}
	if err := s.parseClaims(cookieValue, claims); err != nil {
		return "", "", err
	}
	return claims.Mode, claims.ReturnTo, nil
}

// HandleCallback validates provider state and returns the frontend redirect target.
func (s *SocialAuthService) HandleCallback(
	ctx context.Context,
	provider, code, returnedState, cookieValue, currentUserID string,
) (string, error) {
	if strings.TrimSpace(code) == "" {
		return "", ErrOAuthCodeMissing
	}

	stateClaims := &socialStateClaims{}
	if err := s.parseClaims(cookieValue, stateClaims); err != nil {
		return "", ErrOAuthStateInvalid
	}
	if stateClaims.Provider != provider || stateClaims.State != returnedState {
		return "", ErrOAuthStateInvalid
	}

	cfg, err := s.oauthConfigSvc.RequireEnabledProvider(ctx, provider)
	if err != nil {
		return "", err
	}
	providerClient, err := s.registry.Get(provider)
	if err != nil {
		return "", err
	}
	profile, err := providerClient.Exchange(ctx, cfg, code, cfg.CallbackURL, stateClaims.CodeVerifier, stateClaims.Nonce)
	if err != nil {
		return "", err
	}

	if stateClaims.Mode == OAuthModeBind {
		if currentUserID == "" || currentUserID != stateClaims.UserID {
			return "", ErrOAuthLoginRequired
		}
		if _, err := s.linkIdentityToUser(ctx, currentUserID, profile); err != nil {
			return "", err
		}
		return appendQueryValue(stateClaims.ReturnTo, map[string]string{
			"social_status": "linked",
			"provider":      profile.Provider,
		}), nil
	}

	return s.resolveLoginRedirect(ctx, stateClaims.ReturnTo, profile)
}

// ExchangeLoginTicket converts a short-lived login ticket into the real JWT pair.
func (s *SocialAuthService) ExchangeLoginTicket(ctx context.Context, ticket string) (*TokenPair, *model.User, string, error) {
	claims := &socialTicketClaims{}
	if err := s.parseClaims(ticket, claims); err != nil || claims.Kind != socialTicketKindLogin || claims.UserID == "" {
		return nil, nil, "", ErrOAuthTicketInvalid
	}
	user, err := s.userRepo.GetByID(ctx, claims.UserID)
	if err != nil {
		return nil, nil, "", ErrOAuthTicketInvalid
	}
	if !user.IsActive {
		return nil, nil, "", ErrUserInactive
	}
	pair, err := s.authSvc.IssueTokenPair(user)
	if err != nil {
		return nil, nil, "", err
	}
	return pair, user, sanitizeReturnTo(claims.ReturnTo, "/user/dashboard"), nil
}

// CompleteOnboarding creates a new account from a pending social login ticket.
func (s *SocialAuthService) CompleteOnboarding(
	ctx context.Context,
	ticket, username, email string,
) (*TokenPair, *model.User, string, error) {
	claims := &socialTicketClaims{}
	if err := s.parseClaims(ticket, claims); err != nil || claims.Kind != socialTicketKindOnboard {
		return nil, nil, "", ErrOAuthTicketInvalid
	}
	if !s.allowRegistration(ctx) {
		return nil, nil, "", ErrRegistrationClosed
	}

	profile := &SocialProfile{
		Provider:        claims.Provider,
		ProviderUserID:  claims.ProviderUserID,
		ProviderUnionID: claims.ProviderUnionID,
		Email:           claims.Email,
		EmailVerified:   claims.EmailVerified,
		DisplayName:     claims.DisplayName,
		AvatarURL:       claims.AvatarURL,
		RawProfile:      claims.RawProfile,
	}

	user, err := s.authSvc.CreateSocialUser(ctx, strings.TrimSpace(username), strings.TrimSpace(email), profile.DisplayName, profile.AvatarURL)
	if err != nil {
		return nil, nil, "", err
	}
	if _, err := s.linkIdentityToUser(ctx, user.ID, profile); err != nil {
		return nil, nil, "", err
	}

	pair, err := s.authSvc.IssueTokenPair(user)
	if err != nil {
		return nil, nil, "", err
	}
	return pair, user, sanitizeReturnTo(claims.ReturnTo, "/user/dashboard"), nil
}

// ListUserIdentities returns provider binding status for the current user profile page.
func (s *SocialAuthService) ListUserIdentities(ctx context.Context, userID string) ([]UserIdentityStatus, error) {
	configs, err := s.oauthConfigSvc.ListProviders(ctx)
	if err != nil {
		return nil, err
	}
	identities, err := s.identityRepo.ListByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	byProvider := make(map[string]model.UserIdentity, len(identities))
	for _, identity := range identities {
		byProvider[identity.Provider] = identity
	}

	items := make([]UserIdentityStatus, 0, len(configs))
	for _, cfg := range configs {
		item := UserIdentityStatus{
			Key:     cfg.Key,
			Label:   cfg.Label,
			IconKey: cfg.IconKey,
		}
		if identity, ok := byProvider[cfg.Key]; ok {
			item.Bound = true
			item.DisplayName = identity.DisplayName
			item.Email = identity.Email
		}
		items = append(items, item)
	}
	return items, nil
}

// UnlinkIdentity removes a third-party binding when at least one login method remains.
func (s *SocialAuthService) UnlinkIdentity(ctx context.Context, userID, provider string) error {
	if _, err := s.oauthConfigSvc.providerMeta(provider); err != nil {
		return err
	}

	if _, err := s.identityRepo.GetByUserAndProvider(ctx, userID, provider); err != nil {
		return err
	}
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	count, err := s.identityRepo.CountByUserID(ctx, userID)
	if err != nil {
		return err
	}
	if !user.HasLocalPassword && count <= 1 {
		return ErrOAuthLastLoginMethod
	}
	return s.identityRepo.DeleteByUserAndProvider(ctx, userID, provider)
}

func (s *SocialAuthService) resolveLoginRedirect(ctx context.Context, returnTo string, profile *SocialProfile) (string, error) {
	if identity, err := s.identityRepo.GetByProviderUserID(ctx, profile.Provider, profile.ProviderUserID); err == nil {
		user, err := s.userRepo.GetByID(ctx, identity.UserID)
		if err != nil {
			return "", err
		}
		if !user.IsActive {
			return "", ErrUserInactive
		}
		if _, err := s.linkIdentityToUser(ctx, user.ID, profile); err != nil {
			return "", err
		}
		return s.buildLoginRedirect(user.ID, returnTo)
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return "", err
	}

	if verifiedEmail := verifiedEmail(profile); verifiedEmail != "" {
		user, err := s.userRepo.GetByEmail(ctx, verifiedEmail)
		if err == nil {
			if !user.IsActive {
				return "", ErrUserInactive
			}
			if _, err := s.linkIdentityToUser(ctx, user.ID, profile); err != nil {
				return "", err
			}
			return s.buildLoginRedirect(user.ID, returnTo)
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return "", err
		}
	}

	if !s.allowRegistration(ctx) {
		return "", ErrRegistrationClosed
	}

	if verifiedEmail := verifiedEmail(profile); verifiedEmail != "" {
		user, err := s.createUserFromProfile(ctx, profile, verifiedEmail)
		if err != nil {
			return "", err
		}
		if _, err := s.linkIdentityToUser(ctx, user.ID, profile); err != nil {
			return "", err
		}
		return s.buildLoginRedirect(user.ID, returnTo)
	}

	ticket, err := s.signClaims(&socialTicketClaims{
		Kind:            socialTicketKindOnboard,
		Provider:        profile.Provider,
		ReturnTo:        sanitizeReturnTo(returnTo, "/user/dashboard"),
		ProviderUserID:  profile.ProviderUserID,
		ProviderUnionID: profile.ProviderUnionID,
		Email:           profile.Email,
		EmailVerified:   profile.EmailVerified,
		DisplayName:     profile.DisplayName,
		AvatarURL:       profile.AvatarURL,
		RawProfile:      profile.RawProfile,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(5 * time.Minute)),
			Subject:   profile.Provider,
		},
	})
	if err != nil {
		return "", err
	}
	return "/login/complete-social?ticket=" + url.QueryEscape(ticket), nil
}

func (s *SocialAuthService) buildLoginRedirect(userID, returnTo string) (string, error) {
	ticket, err := s.signClaims(&socialTicketClaims{
		Kind:     socialTicketKindLogin,
		UserID:   userID,
		ReturnTo: sanitizeReturnTo(returnTo, "/user/dashboard"),
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(5 * time.Minute)),
			Subject:   userID,
		},
	})
	if err != nil {
		return "", err
	}
	return "/login/callback?ticket=" + url.QueryEscape(ticket), nil
}

func (s *SocialAuthService) createUserFromProfile(ctx context.Context, profile *SocialProfile, email string) (*model.User, error) {
	username, err := s.generateUniqueUsername(ctx, profile, email)
	if err != nil {
		return nil, err
	}
	return s.authSvc.CreateSocialUser(ctx, username, email, profile.DisplayName, profile.AvatarURL)
}

func (s *SocialAuthService) generateUniqueUsername(ctx context.Context, profile *SocialProfile, email string) (string, error) {
	base := usernameSlug(email)
	if base == "" && profile.DisplayName != nil {
		base = usernameSlug(*profile.DisplayName)
	}
	if base == "" {
		base = usernameSlug(profile.Provider + "-" + profile.ProviderUserID)
	}
	if len(base) < 3 {
		base = base + "user"
	}
	if len(base) > 24 {
		base = base[:24]
	}
	base = strings.Trim(base, "-_.")
	if len(base) < 3 {
		base = "kite-user"
	}

	if exists, err := s.userRepo.ExistsByUsername(ctx, base); err != nil {
		return "", err
	} else if !exists {
		return base, nil
	}

	for i := 1; i <= 9999; i++ {
		suffix := strconv.Itoa(i)
		candidate := base
		if len(candidate)+len(suffix)+1 > 32 {
			candidate = candidate[:32-len(suffix)-1]
		}
		candidate = strings.Trim(candidate, "-_.") + "-" + suffix
		exists, err := s.userRepo.ExistsByUsername(ctx, candidate)
		if err != nil {
			return "", err
		}
		if !exists {
			return candidate, nil
		}
	}
	return "", errors.New("failed to generate unique username")
}

func (s *SocialAuthService) linkIdentityToUser(ctx context.Context, userID string, profile *SocialProfile) (*model.UserIdentity, error) {
	if _, err := s.userRepo.GetByID(ctx, userID); err != nil {
		return nil, err
	}
	if existing, err := s.identityRepo.GetByProviderUserID(ctx, profile.Provider, profile.ProviderUserID); err == nil {
		if existing.UserID != userID {
			return nil, ErrOAuthBindingConflict
		}
		assignProfile(existing, profile)
		if err := s.identityRepo.Update(ctx, existing); err != nil {
			return nil, err
		}
		return existing, nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	now := time.Now()
	identity := &model.UserIdentity{
		ID:              randomUUID(),
		UserID:          userID,
		Provider:        profile.Provider,
		ProviderUserID:  profile.ProviderUserID,
		ProviderUnionID: profile.ProviderUnionID,
		Email:           profile.Email,
		EmailVerified:   profile.EmailVerified,
		DisplayName:     profile.DisplayName,
		AvatarURL:       profile.AvatarURL,
		RawProfile:      profile.RawProfile,
		LastLoginAt:     &now,
	}
	if err := s.identityRepo.Create(ctx, identity); err != nil {
		return nil, err
	}
	return identity, nil
}

func (s *SocialAuthService) allowRegistration(ctx context.Context) bool {
	if s.settingRepo == nil {
		return s.allowRegistrationDefault
	}
	enabled, err := s.settingRepo.GetBool(ctx, "allow_registration", s.allowRegistrationDefault)
	if err != nil {
		return s.allowRegistrationDefault
	}
	return enabled
}

func (s *SocialAuthService) signClaims(claims jwt.Claims) (string, error) {
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(s.jwtSecret))
}

func (s *SocialAuthService) parseClaims(token string, claims jwt.Claims) error {
	parsed, err := jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(s.jwtSecret), nil
	})
	if err != nil || !parsed.Valid {
		return ErrOAuthTicketInvalid
	}
	return nil
}

func assignProfile(identity *model.UserIdentity, profile *SocialProfile) {
	now := time.Now()
	identity.ProviderUnionID = profile.ProviderUnionID
	identity.Email = profile.Email
	identity.EmailVerified = profile.EmailVerified
	identity.DisplayName = profile.DisplayName
	identity.AvatarURL = profile.AvatarURL
	identity.RawProfile = profile.RawProfile
	identity.LastLoginAt = &now
}

func verifiedEmail(profile *SocialProfile) string {
	if profile == nil || !profile.EmailVerified || profile.Email == nil {
		return ""
	}
	return strings.TrimSpace(*profile.Email)
}

func sanitizeReturnTo(value, fallback string) string {
	candidate := strings.TrimSpace(value)
	if candidate == "" {
		return fallback
	}
	parsed, err := url.Parse(candidate)
	if err != nil || parsed.IsAbs() || parsed.Host != "" {
		return fallback
	}
	if !strings.HasPrefix(parsed.Path, "/") {
		return fallback
	}
	return parsed.String()
}

func appendQueryValue(path string, params map[string]string) string {
	parsed, err := url.Parse(sanitizeReturnTo(path, "/user/dashboard"))
	if err != nil {
		return "/user/dashboard"
	}
	query := parsed.Query()
	for key, value := range params {
		query.Set(key, value)
	}
	parsed.RawQuery = query.Encode()
	return parsed.String()
}

func defaultReturnTo(mode string) string {
	if mode == OAuthModeBind {
		return "/user/profile"
	}
	return "/user/dashboard"
}

var usernameSanitizer = regexp.MustCompile(`[^a-z0-9._-]+`)

func usernameSlug(value string) string {
	lower := strings.ToLower(strings.TrimSpace(value))
	if at := strings.Index(lower, "@"); at > 0 {
		lower = lower[:at]
	}
	lower = strings.NewReplacer(" ", "-", "/", "-", "\\", "-", ":", "-").Replace(lower)
	lower = usernameSanitizer.ReplaceAllString(lower, "-")
	lower = strings.Trim(lower, "-_.")
	if len(lower) > 32 {
		lower = lower[:32]
	}
	return lower
}

func randomURLSafeString(size int) (string, error) {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func pkceChallenge(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func randomUUID() string {
	token, err := generateRandomToken(16)
	if err != nil {
		raw := make([]byte, 16)
		_, _ = rand.Read(raw)
		return hex.EncodeToString(raw)
	}
	return token
}
