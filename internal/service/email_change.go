package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"net/mail"
	"strings"
	"time"

	"github.com/amigoer/kite/internal/model"
	"github.com/amigoer/kite/internal/repo"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// EmailChangeService owns the "change my email with verification" flow.
// The ownership proof is a 6-digit code delivered to the requested new
// address; the caller must present the code to commit the change.
type EmailChangeService struct {
	userRepo    *repo.UserRepo
	settingRepo *repo.SettingRepo
	verRepo     *repo.EmailVerificationRepo
	emailSvc    *EmailService
}

func NewEmailChangeService(
	userRepo *repo.UserRepo,
	settingRepo *repo.SettingRepo,
	verRepo *repo.EmailVerificationRepo,
	emailSvc *EmailService,
) *EmailChangeService {
	return &EmailChangeService{
		userRepo:    userRepo,
		settingRepo: settingRepo,
		verRepo:     verRepo,
		emailSvc:    emailSvc,
	}
}

// Errors surfaced to callers. The handler maps these to HTTP status codes.
var (
	ErrInvalidEmailFormat    = errors.New("invalid email address format")
	ErrSameAsCurrentEmail    = errors.New("new email matches current email")
	ErrEmailTaken            = errors.New("email is already registered to another account")
	ErrVerificationCooldown  = errors.New("please wait before requesting another code")
	ErrVerificationNotFound  = errors.New("no pending verification for this email")
	ErrVerificationCodeWrong = errors.New("verification code is incorrect")
	ErrVerificationExpired   = errors.New("verification code has expired")
	ErrSMTPNotConfigured     = errors.New("SMTP is not configured; contact admin")
)

const (
	emailCodeTTL     = 10 * time.Minute
	emailResendAfter = 60 * time.Second
)

// RequestEmailChange validates the desired address, rate-limits requests and
// emails a fresh verification code. The code is stored as a SHA-256 hash;
// only the plaintext goes out over SMTP.
func (s *EmailChangeService) RequestEmailChange(ctx context.Context, userID, rawNewEmail string) (time.Time, error) {
	newEmail, err := normalizeEmail(rawNewEmail)
	if err != nil {
		return time.Time{}, err
	}

	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return time.Time{}, fmt.Errorf("get user: %w", err)
	}

	if strings.EqualFold(newEmail, user.Email) {
		return time.Time{}, ErrSameAsCurrentEmail
	}

	// Rely on the existing uniqueness index: check both username (unchanged)
	// and the new email against every other account.
	conflict, err := s.userRepo.ExistsByUsernameOrEmailExcept(ctx, user.Username, newEmail, userID)
	if err != nil {
		return time.Time{}, fmt.Errorf("check email conflict: %w", err)
	}
	if conflict {
		return time.Time{}, ErrEmailTaken
	}

	// Cooldown: if the most recent row for this user is under emailResendAfter,
	// reject with a hint so the UI can render a countdown.
	latest, err := s.verRepo.LatestForUser(ctx, userID, model.EmailVerifyPurposeEmailChange)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return time.Time{}, err
	}
	if latest != nil {
		since := time.Since(latest.CreatedAt)
		if since < emailResendAfter {
			return time.Time{}, ErrVerificationCooldown
		}
	}

	// Resolve SMTP config up front so we never create a pending row we can't
	// actually deliver. ResolveSMTPConfig returns a specific message when the
	// admin hasn't finished configuring mail.
	settings, err := s.settingRepo.GetAll(ctx)
	if err != nil {
		return time.Time{}, fmt.Errorf("load smtp settings: %w", err)
	}
	cfg, err := ResolveSMTPConfig(settings)
	if err != nil {
		return time.Time{}, fmt.Errorf("%w: %v", ErrSMTPNotConfigured, err)
	}

	code, err := generateNumericCode(6)
	if err != nil {
		return time.Time{}, fmt.Errorf("generate code: %w", err)
	}
	expiresAt := time.Now().Add(emailCodeTTL)

	row := &model.EmailVerification{
		ID:        uuid.NewString(),
		UserID:    userID,
		NewEmail:  newEmail,
		CodeHash:  hashCode(code),
		Purpose:   model.EmailVerifyPurposeEmailChange,
		ExpiresAt: expiresAt,
	}
	if err := s.verRepo.Create(ctx, row); err != nil {
		return time.Time{}, err
	}

	siteName := strings.TrimSpace(settings[SiteNameSettingKey])
	if siteName == "" {
		siteName = "Kite"
	}
	subject, body := buildEmailChangeMessage(siteName, code, emailCodeTTL)
	if err := s.emailSvc.Send(ctx, *cfg, newEmail, subject, body); err != nil {
		// Best-effort cleanup: expire the code immediately so the user isn't
		// blocked by a stale row when they retry.
		_ = s.verRepo.MarkConsumed(ctx, row.ID, time.Now())
		return time.Time{}, fmt.Errorf("send verification email: %w", err)
	}

	return expiresAt, nil
}

// ConfirmEmailChange checks the provided code against the latest pending row
// for the given (user, email) pair; on success it rotates the user's email.
func (s *EmailChangeService) ConfirmEmailChange(ctx context.Context, userID, rawNewEmail, rawCode string) (*model.User, error) {
	newEmail, err := normalizeEmail(rawNewEmail)
	if err != nil {
		return nil, err
	}
	code := strings.TrimSpace(rawCode)
	if code == "" {
		return nil, ErrVerificationCodeWrong
	}

	v, err := s.verRepo.GetLatestPending(ctx, userID, model.EmailVerifyPurposeEmailChange, newEmail)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrVerificationNotFound
		}
		return nil, err
	}
	if time.Now().After(v.ExpiresAt) {
		return nil, ErrVerificationExpired
	}
	if hashCode(code) != v.CodeHash {
		return nil, ErrVerificationCodeWrong
	}

	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}

	// Re-check uniqueness just before commit: another account could have
	// claimed the address while this code was pending.
	conflict, err := s.userRepo.ExistsByUsernameOrEmailExcept(ctx, user.Username, newEmail, userID)
	if err != nil {
		return nil, fmt.Errorf("check email conflict: %w", err)
	}
	if conflict {
		return nil, ErrEmailTaken
	}

	user.Email = newEmail
	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, fmt.Errorf("update user email: %w", err)
	}

	now := time.Now()
	if err := s.verRepo.InvalidateUserPending(ctx, userID, model.EmailVerifyPurposeEmailChange, now); err != nil {
		// Non-fatal: the email has already been rotated. Log and move on by
		// ignoring — the pending rows will expire on their own.
		_ = err
	}

	return user, nil
}

// normalizeEmail trims, lowercases and validates the RFC-5321 form. Returns
// ErrInvalidEmailFormat for anything net/mail refuses.
func normalizeEmail(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", ErrInvalidEmailFormat
	}
	addr, err := mail.ParseAddress(trimmed)
	if err != nil {
		return "", ErrInvalidEmailFormat
	}
	return strings.ToLower(addr.Address), nil
}

// generateNumericCode returns an n-digit decimal code drawn from crypto/rand
// so codes are not guessable from clock state.
func generateNumericCode(n int) (string, error) {
	var b strings.Builder
	b.Grow(n)
	ten := big.NewInt(10)
	for i := 0; i < n; i++ {
		digit, err := rand.Int(rand.Reader, ten)
		if err != nil {
			return "", err
		}
		b.WriteString(digit.String())
	}
	return b.String(), nil
}

// hashCode returns a SHA-256 hex digest of the code so plaintext codes are
// never persisted. SHA-256 is sufficient here: the code is short-lived (10
// min) and has six decimal digits, so brute force against the hash requires
// access to DB rows that also carry the target email and user.
func hashCode(code string) string {
	sum := sha256.Sum256([]byte(code))
	return hex.EncodeToString(sum[:])
}

// buildEmailChangeMessage renders the subject/body the user receives. Keep
// it plaintext for now — matches the existing test-email template.
func buildEmailChangeMessage(siteName, code string, ttl time.Duration) (string, string) {
	subject := fmt.Sprintf("[%s] 邮箱验证码", siteName)
	body := strings.Join([]string{
		fmt.Sprintf("你好，这是来自 %s 的邮箱验证码：", siteName),
		"",
		fmt.Sprintf("    %s", code),
		"",
		fmt.Sprintf("验证码将在 %d 分钟内有效，用于绑定新的账户邮箱。", int(ttl.Minutes())),
		"如果这不是你的操作，可以忽略本邮件，账户邮箱不会发生变化。",
	}, "\r\n")
	return subject, body
}
