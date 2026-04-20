package repo

import (
	"context"
	"fmt"
	"time"

	"github.com/amigoer/kite/internal/model"
	"gorm.io/gorm"
)

// UserIdentityRepo is the data access layer for third-party account bindings.
type UserIdentityRepo struct {
	db *gorm.DB
}

func NewUserIdentityRepo(db *gorm.DB) *UserIdentityRepo {
	return &UserIdentityRepo{db: db}
}

// Create inserts a new identity row.
func (r *UserIdentityRepo) Create(ctx context.Context, identity *model.UserIdentity) error {
	if err := r.db.WithContext(ctx).Create(identity).Error; err != nil {
		return fmt.Errorf("create user identity: %w", err)
	}
	return nil
}

// GetByProviderUserID fetches an identity by provider and provider user id.
func (r *UserIdentityRepo) GetByProviderUserID(ctx context.Context, provider, providerUserID string) (*model.UserIdentity, error) {
	var identity model.UserIdentity
	if err := r.db.WithContext(ctx).
		Where("provider = ? AND provider_user_id = ?", provider, providerUserID).
		First(&identity).Error; err != nil {
		return nil, fmt.Errorf("get identity by provider: %w", err)
	}
	return &identity, nil
}

// GetByUserAndProvider fetches an identity by user and provider.
func (r *UserIdentityRepo) GetByUserAndProvider(ctx context.Context, userID, provider string) (*model.UserIdentity, error) {
	var identity model.UserIdentity
	if err := r.db.WithContext(ctx).
		Where("user_id = ? AND provider = ?", userID, provider).
		First(&identity).Error; err != nil {
		return nil, fmt.Errorf("get identity by user and provider: %w", err)
	}
	return &identity, nil
}

// ListByUserID returns all third-party identities bound to a user.
func (r *UserIdentityRepo) ListByUserID(ctx context.Context, userID string) ([]model.UserIdentity, error) {
	var identities []model.UserIdentity
	if err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at ASC").
		Find(&identities).Error; err != nil {
		return nil, fmt.Errorf("list identities by user: %w", err)
	}
	return identities, nil
}

// CountByUserID returns the total number of third-party identities bound to a user.
func (r *UserIdentityRepo) CountByUserID(ctx context.Context, userID string) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&model.UserIdentity{}).
		Where("user_id = ?", userID).
		Count(&count).Error; err != nil {
		return 0, fmt.Errorf("count identities by user: %w", err)
	}
	return count, nil
}

// Update persists identity field changes.
func (r *UserIdentityRepo) Update(ctx context.Context, identity *model.UserIdentity) error {
	if err := r.db.WithContext(ctx).Save(identity).Error; err != nil {
		return fmt.Errorf("update user identity: %w", err)
	}
	return nil
}

// DeleteByUserAndProvider removes the binding between a user and provider.
func (r *UserIdentityRepo) DeleteByUserAndProvider(ctx context.Context, userID, provider string) error {
	if err := r.db.WithContext(ctx).
		Where("user_id = ? AND provider = ?", userID, provider).
		Delete(&model.UserIdentity{}).Error; err != nil {
		return fmt.Errorf("delete user identity: %w", err)
	}
	return nil
}

// TouchLastLogin updates the last login timestamp.
func (r *UserIdentityRepo) TouchLastLogin(ctx context.Context, id string, at time.Time) error {
	if err := r.db.WithContext(ctx).
		Model(&model.UserIdentity{}).
		Where("id = ?", id).
		Update("last_login_at", at).Error; err != nil {
		return fmt.Errorf("touch identity last login: %w", err)
	}
	return nil
}
