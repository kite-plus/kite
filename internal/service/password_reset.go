package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/kite-plus/kite/internal/model"
	"github.com/kite-plus/kite/internal/repo"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// PasswordResetService owns the "I forgot my password" flow. A user
// submits either their username or email; we look up the account,
// mail a 6-digit code to the address on file, and let them exchange
// the code for a new password.
//
// We deliberately never tell the client whether a given identifier
// exists — every well-formed request returns the same shape, whether
// the account is real or not. This prevents trivial user enumeration
// via the forgot-password endpoint.
type PasswordResetService struct {
	userRepo    *repo.UserRepo
	settingRepo *repo.SettingRepo
	verRepo     *repo.EmailVerificationRepo
	emailSvc    *EmailService
}

func NewPasswordResetService(
	userRepo *repo.UserRepo,
	settingRepo *repo.SettingRepo,
	verRepo *repo.EmailVerificationRepo,
	emailSvc *EmailService,
) *PasswordResetService {
	return &PasswordResetService{
		userRepo:    userRepo,
		settingRepo: settingRepo,
		verRepo:     verRepo,
		emailSvc:    emailSvc,
	}
}

// ErrPasswordResetIdentifierRequired fires when the client submits a
// blank identifier — it's the only "request" error we surface, since
// everything else (missing account, cooldown, inactive user) is
// swallowed to avoid leaking account existence.
var ErrPasswordResetIdentifierRequired = errors.New("identifier is required")

// RequestPasswordReset looks up a user by username or email. If the
// account exists, is active, and the resend cooldown has passed, a
// fresh code is stored and emailed. The caller always receives the
// same "expires_at" shape regardless — real UX difference only shows
// up once the code arrives in the target inbox.
//
// Returning `expires_at` even for unknown identifiers costs us
// nothing: the UI just shows "check your inbox if this account
// exists", and no email is ever sent from a spoofed request.
func (s *PasswordResetService) RequestPasswordReset(ctx context.Context, rawIdentifier string) (time.Time, error) {
	identifier := strings.TrimSpace(rawIdentifier)
	if identifier == "" {
		return time.Time{}, ErrPasswordResetIdentifierRequired
	}

	fakeExpiry := time.Now().Add(emailCodeTTL)

	user := s.lookupForReset(ctx, identifier)
	if user == nil {
		// No account, inactive account, or social-only account. Return
		// the same shape a real request would. The response is a lie,
		// but it's a safe one — nothing has been persisted or sent.
		return fakeExpiry, nil
	}

	// Resend cooldown. We silently swallow hits inside the window so
	// a real user who mashes the button sees "code is on its way"
	// without us revealing that an account exists.
	latest, err := s.verRepo.LatestForUser(ctx, user.ID, model.EmailVerifyPurposePasswordReset)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return time.Time{}, err
	}
	if latest != nil && time.Since(latest.CreatedAt) < emailResendAfter {
		return fakeExpiry, nil
	}

	settings, err := s.settingRepo.GetAll(ctx)
	if err != nil {
		return time.Time{}, fmt.Errorf("load smtp settings: %w", err)
	}
	cfg, err := ResolveSMTPConfig(settings)
	if err != nil {
		// SMTP misconfiguration is a real server problem, not an
		// enumeration vector — surface it so the admin can fix it.
		return time.Time{}, fmt.Errorf("%w: %v", ErrSMTPNotConfigured, err)
	}

	code, err := generateNumericCode(6)
	if err != nil {
		return time.Time{}, fmt.Errorf("generate code: %w", err)
	}
	expiresAt := time.Now().Add(emailCodeTTL)

	row := &model.EmailVerification{
		ID:        uuid.NewString(),
		UserID:    user.ID,
		NewEmail:  user.Email, // "target" address is simply the user's current email
		CodeHash:  hashCode(code),
		Purpose:   model.EmailVerifyPurposePasswordReset,
		ExpiresAt: expiresAt,
	}
	if err := s.verRepo.Create(ctx, row); err != nil {
		return time.Time{}, err
	}

	siteName := strings.TrimSpace(settings[SiteNameSettingKey])
	if siteName == "" {
		siteName = "Kite"
	}
	subject, body := buildPasswordResetMessage(siteName, user.Username, code, emailCodeTTL)
	if err := s.emailSvc.Send(ctx, *cfg, user.Email, subject, body); err != nil {
		// Retire the row so the next retry isn't blocked by cooldown
		// on a request whose email never made it out.
		_ = s.verRepo.MarkConsumed(ctx, row.ID, time.Now())
		return time.Time{}, fmt.Errorf("send reset email: %w", err)
	}

	return expiresAt, nil
}

// ConfirmPasswordReset finishes the flow: verify the code, set a
// fresh password hash, and bump token_version so every pre-existing
// session on the account is instantly revoked. Callers who were
// logged in elsewhere will therefore be kicked back to /login. 2FA
// remains untouched — if it was on, the next login will still see
// the challenge page.
func (s *PasswordResetService) ConfirmPasswordReset(ctx context.Context, rawIdentifier, code, newPassword string) error {
	identifier := strings.TrimSpace(rawIdentifier)
	if identifier == "" {
		return ErrPasswordResetIdentifierRequired
	}
	code = strings.TrimSpace(code)
	if code == "" {
		return ErrVerificationCodeWrong
	}

	user := s.lookupForReset(ctx, identifier)
	if user == nil {
		// Treat "no such user" as "no such code" so we don't give
		// enumeration signal here either.
		return ErrVerificationNotFound
	}

	v, err := s.verRepo.GetLatestPending(ctx, user.ID, model.EmailVerifyPurposePasswordReset, user.Email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrVerificationNotFound
		}
		return err
	}
	if time.Now().After(v.ExpiresAt) {
		return ErrVerificationExpired
	}
	if hashCode(code) != v.CodeHash {
		return ErrVerificationCodeWrong
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash new password: %w", err)
	}
	user.PasswordHash = string(hash)
	user.HasLocalPassword = true
	if err := s.userRepo.Update(ctx, user); err != nil {
		return fmt.Errorf("update password: %w", err)
	}
	if err := s.userRepo.BumpTokenVersion(ctx, user.ID); err != nil {
		return fmt.Errorf("revoke sessions: %w", err)
	}

	// Burn every still-pending reset code on this account so a stolen
	// unused code can't be redeemed for a second reset.
	now := time.Now()
	_ = s.verRepo.InvalidateUserPending(ctx, user.ID, model.EmailVerifyPurposePasswordReset, now)

	return nil
}

// lookupForReset returns the user row for a reset-eligible account,
// or nil for anything we want to treat as "no account". Social-only
// accounts are intentionally excluded — they have no local password
// to reset, so the reset flow can't help them.
func (s *PasswordResetService) lookupForReset(ctx context.Context, identifier string) *model.User {
	// Try username first, then email. Username matches are case
	// sensitive (they always are in this app); email is normalised.
	user, err := s.userRepo.GetByUsername(ctx, identifier)
	if err != nil {
		normalized, nerr := normalizeEmail(identifier)
		if nerr != nil {
			return nil
		}
		user, err = s.userRepo.GetByEmail(ctx, normalized)
		if err != nil {
			return nil
		}
	}
	if !user.IsActive {
		return nil
	}
	if !user.HasLocalPassword {
		return nil
	}
	return user
}

func buildPasswordResetMessage(siteName, username, code string, ttl time.Duration) (string, string) {
	subject := fmt.Sprintf("[%s] 密码重置验证码", siteName)
	body := strings.Join([]string{
		fmt.Sprintf("你好 %s，", username),
		"",
		fmt.Sprintf("这是来自 %s 的密码重置验证码：", siteName),
		"",
		fmt.Sprintf("    %s", code),
		"",
		fmt.Sprintf("验证码将在 %d 分钟内有效。", int(ttl.Minutes())),
		"如果这不是你的操作，可以忽略本邮件，账户密码不会发生变化。",
	}, "\r\n")
	return subject, body
}
