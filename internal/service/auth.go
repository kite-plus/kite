package service

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/amigoer/kite-blog/internal/config"
	"github.com/amigoer/kite-blog/internal/model"
	"github.com/amigoer/kite-blog/internal/repo"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrAdminAuthDisabled       = errors.New("admin auth is disabled")
	ErrInvalidAdminCredentials = errors.New("invalid admin credentials")
	ErrAdminUnauthorized       = errors.New("admin unauthorized")
)

type AdminLoginInput struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type AdminSessionMeta struct {
	IP        string
	UserAgent string
}

type AdminProfile struct {
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	Email       string `json:"email"`
	Bio         string `json:"bio"`
	Avatar      string `json:"avatar"`
	Website     string `json:"website"`
	Location    string `json:"location"`
}

type AdminCurrentUser struct {
	AuthEnabled    bool         `json:"auth_enabled"`
	Authenticated  bool         `json:"authenticated"`
	User           AdminProfile `json:"user"`
	SessionExpires *time.Time   `json:"session_expires,omitempty"`
}

type AdminAuthService struct {
	cfg         *config.Config
	sessionRepo *repo.AdminSessionRepository
}

func NewAdminAuthService(cfg *config.Config, sessionRepo *repo.AdminSessionRepository) *AdminAuthService {
	return &AdminAuthService{cfg: cfg, sessionRepo: sessionRepo}
}

func (s *AdminAuthService) IsEnabled() bool {
	return s != nil && s.cfg != nil && s.cfg.Admin.Enabled
}

func (s *AdminAuthService) SessionTTL() time.Duration {
	if s == nil || s.cfg == nil || s.cfg.Admin.SessionTTLHours <= 0 {
		return 168 * time.Hour
	}
	return time.Duration(s.cfg.Admin.SessionTTLHours) * time.Hour
}

func (s *AdminAuthService) Login(input AdminLoginInput, meta AdminSessionMeta) (string, *AdminCurrentUser, error) {
	if !s.IsEnabled() {
		return "", nil, ErrAdminAuthDisabled
	}
	if s.sessionRepo == nil {
		return "", nil, fmt.Errorf("admin auth service is unavailable")
	}

	username := strings.TrimSpace(input.Username)
	if username == "" || input.Password == "" {
		return "", nil, ErrInvalidAdminCredentials
	}
	if username != strings.TrimSpace(s.cfg.Admin.Username) {
		return "", nil, ErrInvalidAdminCredentials
	}
	if err := bcrypt.CompareHashAndPassword([]byte(s.cfg.Admin.PasswordHash), []byte(input.Password)); err != nil {
		return "", nil, ErrInvalidAdminCredentials
	}

	rawToken, err := generateAdminSessionToken()
	if err != nil {
		return "", nil, fmt.Errorf("generate admin session token: %w", err)
	}

	now := time.Now().UTC()
	expiresAt := now.Add(s.SessionTTL())
	session := &model.AdminSession{
		TokenHash:  hashAdminSessionToken(rawToken, s.cfg.Admin.SessionSecret),
		ExpiresAt:  expiresAt,
		LastUsedAt: now,
		IP:         strings.TrimSpace(meta.IP),
		UserAgent:  strings.TrimSpace(meta.UserAgent),
	}

	if err := s.sessionRepo.Create(session); err != nil {
		return "", nil, err
	}

	return rawToken, s.buildCurrentUser(true, &expiresAt), nil
}

func (s *AdminAuthService) GetCurrent(rawToken string) (*AdminCurrentUser, error) {
	if !s.IsEnabled() {
		return s.buildCurrentUser(false, nil), nil
	}
	if s.sessionRepo == nil {
		return nil, fmt.Errorf("admin auth service is unavailable")
	}

	rawToken = strings.TrimSpace(rawToken)
	if rawToken == "" {
		return nil, ErrAdminUnauthorized
	}

	tokenHash := hashAdminSessionToken(rawToken, s.cfg.Admin.SessionSecret)
	session, err := s.sessionRepo.GetByTokenHash(tokenHash)
	if err != nil {
		if errors.Is(err, repo.ErrAdminSessionNotFound) {
			return nil, ErrAdminUnauthorized
		}
		return nil, err
	}

	now := time.Now().UTC()
	if session.ExpiresAt.Before(now) {
		_ = s.sessionRepo.DeleteByTokenHash(tokenHash)
		return nil, ErrAdminUnauthorized
	}

	if err := s.sessionRepo.UpdateLastUsedAt(session.ID, now); err != nil && !errors.Is(err, repo.ErrAdminSessionNotFound) {
		return nil, err
	}

	return s.buildCurrentUser(true, &session.ExpiresAt), nil
}

func (s *AdminAuthService) Logout(rawToken string) error {
	if !s.IsEnabled() {
		return nil
	}
	if s.sessionRepo == nil {
		return fmt.Errorf("admin auth service is unavailable")
	}

	rawToken = strings.TrimSpace(rawToken)
	if rawToken == "" {
		return nil
	}

	tokenHash := hashAdminSessionToken(rawToken, s.cfg.Admin.SessionSecret)
	if err := s.sessionRepo.DeleteByTokenHash(tokenHash); err != nil && !errors.Is(err, repo.ErrAdminSessionNotFound) {
		return err
	}
	return nil
}

func (s *AdminAuthService) buildCurrentUser(authenticated bool, expiresAt *time.Time) *AdminCurrentUser {
	profile := AdminProfile{}
	if s != nil && s.cfg != nil {
		profile = AdminProfile{
			Username:    strings.TrimSpace(s.cfg.Admin.Username),
			DisplayName: strings.TrimSpace(s.cfg.Admin.Profile.DisplayName),
			Email:       strings.TrimSpace(s.cfg.Admin.Profile.Email),
			Bio:         strings.TrimSpace(s.cfg.Admin.Profile.Bio),
			Avatar:      strings.TrimSpace(s.cfg.Admin.Profile.Avatar),
			Website:     strings.TrimSpace(s.cfg.Admin.Profile.Website),
			Location:    strings.TrimSpace(s.cfg.Admin.Profile.Location),
		}
		if profile.DisplayName == "" {
			profile.DisplayName = profile.Username
		}
	}

	return &AdminCurrentUser{
		AuthEnabled:    s.IsEnabled(),
		Authenticated:  authenticated,
		User:           profile,
		SessionExpires: expiresAt,
	}
}

func generateAdminSessionToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func hashAdminSessionToken(rawToken string, sessionSecret string) string {
	sum := sha256.Sum256([]byte(sessionSecret + ":" + rawToken))
	return hex.EncodeToString(sum[:])
}
