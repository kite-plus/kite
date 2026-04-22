package repo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/amigoer/kite/internal/model"
	"gorm.io/gorm"
)

// EmailVerificationRepo persists short-lived email verification codes.
type EmailVerificationRepo struct {
	db *gorm.DB
}

func NewEmailVerificationRepo(db *gorm.DB) *EmailVerificationRepo {
	return &EmailVerificationRepo{db: db}
}

// Create inserts a pending verification row.
func (r *EmailVerificationRepo) Create(ctx context.Context, v *model.EmailVerification) error {
	if err := r.db.WithContext(ctx).Create(v).Error; err != nil {
		return fmt.Errorf("create email verification: %w", err)
	}
	return nil
}

// GetLatestPending returns the most recent non-consumed, non-expired row
// matching the user + purpose + new email. Returns gorm.ErrRecordNotFound
// when no active verification exists.
func (r *EmailVerificationRepo) GetLatestPending(ctx context.Context, userID, purpose, newEmail string) (*model.EmailVerification, error) {
	var v model.EmailVerification
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND purpose = ? AND new_email = ? AND consumed_at IS NULL AND expires_at > ?",
			userID, purpose, newEmail, time.Now()).
		Order("created_at DESC").
		First(&v).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		return nil, fmt.Errorf("get latest pending verification: %w", err)
	}
	return &v, nil
}

// LatestForUser returns the most recent row (regardless of status) for the
// given user + purpose, so the caller can enforce a per-user resend cooldown.
func (r *EmailVerificationRepo) LatestForUser(ctx context.Context, userID, purpose string) (*model.EmailVerification, error) {
	var v model.EmailVerification
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND purpose = ?", userID, purpose).
		Order("created_at DESC").
		First(&v).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		return nil, fmt.Errorf("get latest verification: %w", err)
	}
	return &v, nil
}

// MarkConsumed sets consumed_at on a specific row so it cannot be replayed.
func (r *EmailVerificationRepo) MarkConsumed(ctx context.Context, id string, at time.Time) error {
	if err := r.db.WithContext(ctx).Model(&model.EmailVerification{}).
		Where("id = ?", id).
		Update("consumed_at", at).Error; err != nil {
		return fmt.Errorf("mark verification consumed: %w", err)
	}
	return nil
}

// InvalidateUserPending marks every still-pending row for this user/purpose
// as consumed — called after a successful change so stale codes cannot be
// redeemed later against a different email.
func (r *EmailVerificationRepo) InvalidateUserPending(ctx context.Context, userID, purpose string, at time.Time) error {
	if err := r.db.WithContext(ctx).Model(&model.EmailVerification{}).
		Where("user_id = ? AND purpose = ? AND consumed_at IS NULL", userID, purpose).
		Update("consumed_at", at).Error; err != nil {
		return fmt.Errorf("invalidate pending verifications: %w", err)
	}
	return nil
}
