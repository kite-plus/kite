package repo

import (
	"context"
	"fmt"

	"github.com/amigoer/kite/internal/model"
	"gorm.io/gorm"
)

// UserRepo is the data access layer for users.
type UserRepo struct {
	db *gorm.DB
}

func NewUserRepo(db *gorm.DB) *UserRepo {
	return &UserRepo{db: db}
}

// Create inserts a new user.
func (r *UserRepo) Create(ctx context.Context, user *model.User) error {
	if err := r.db.WithContext(ctx).Create(user).Error; err != nil {
		return fmt.Errorf("create user: %w", err)
	}
	return nil
}

// GetByID fetches a user by ID.
func (r *UserRepo) GetByID(ctx context.Context, id string) (*model.User, error) {
	var user model.User
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&user).Error; err != nil {
		return nil, fmt.Errorf("get user by id: %w", err)
	}
	return &user, nil
}

// GetByUsername fetches a user by username.
func (r *UserRepo) GetByUsername(ctx context.Context, username string) (*model.User, error) {
	var user model.User
	if err := r.db.WithContext(ctx).Where("username = ?", username).First(&user).Error; err != nil {
		return nil, fmt.Errorf("get user by username: %w", err)
	}
	return &user, nil
}

// GetByEmail fetches a user by email.
func (r *UserRepo) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	var user model.User
	if err := r.db.WithContext(ctx).Where("email = ?", email).First(&user).Error; err != nil {
		return nil, fmt.Errorf("get user by email: %w", err)
	}
	return &user, nil
}

// ExistsByUsername reports whether any row has the given username.
func (r *UserRepo) ExistsByUsername(ctx context.Context, username string) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&model.User{}).
		Where("username = ?", username).
		Count(&count).Error; err != nil {
		return false, fmt.Errorf("check username exists: %w", err)
	}
	return count > 0, nil
}

// Update persists changes to a user.
func (r *UserRepo) Update(ctx context.Context, user *model.User) error {
	if err := r.db.WithContext(ctx).Save(user).Error; err != nil {
		return fmt.Errorf("update user: %w", err)
	}
	return nil
}

// UpdateStorageUsed adjusts the user's used storage counter; delta may be negative.
//
// This method is NOT suitable for enforcing quota on positive deltas because it
// does not check the limit — concurrent uploads can race past the quota. Use
// TryConsumeStorage for uploads, and keep UpdateStorageUsed only for releases
// (negative delta) and admin/internal adjustments that bypass quota checks.
func (r *UserRepo) UpdateStorageUsed(ctx context.Context, userID string, delta int64) error {
	if err := r.db.WithContext(ctx).
		Model(&model.User{}).
		Where("id = ?", userID).
		Update("storage_used", gorm.Expr("storage_used + ?", delta)).Error; err != nil {
		return fmt.Errorf("update storage used: %w", err)
	}
	return nil
}

// TryConsumeStorage atomically increments a user's storage_used counter iff the
// resulting value fits within storage_limit (or the user is unlimited, i.e.
// storage_limit == -1). It is the authoritative quota gate for uploads.
//
// Returns (true, nil) when the quota was consumed, (false, nil) when the user
// has no room left, and (_, err) only on database failure. A non-existent user
// also returns (false, nil) — the caller is expected to have established the
// user exists via the auth middleware before calling this.
//
// The WHERE clause evaluates storage_used + delta <= storage_limit inside the
// same statement as the UPDATE, so two concurrent uploads cannot both pass a
// check-then-update race. delta must be positive; pass a negative delta to
// UpdateStorageUsed (or call ReleaseStorage) for rollbacks.
func (r *UserRepo) TryConsumeStorage(ctx context.Context, userID string, delta int64) (bool, error) {
	if delta <= 0 {
		return false, fmt.Errorf("consume storage: delta must be positive, got %d", delta)
	}
	res := r.db.WithContext(ctx).
		Model(&model.User{}).
		Where("id = ? AND (storage_limit = -1 OR storage_used + ? <= storage_limit)", userID, delta).
		Update("storage_used", gorm.Expr("storage_used + ?", delta))
	if res.Error != nil {
		return false, fmt.Errorf("consume storage: %w", res.Error)
	}
	return res.RowsAffected == 1, nil
}

// ReleaseStorage is a convenience wrapper over UpdateStorageUsed that subtracts
// delta from the user's storage_used counter. It is intended for compensating
// a prior TryConsumeStorage when a later step in the upload pipeline fails.
func (r *UserRepo) ReleaseStorage(ctx context.Context, userID string, delta int64) error {
	if delta <= 0 {
		return fmt.Errorf("release storage: delta must be positive, got %d", delta)
	}
	return r.UpdateStorageUsed(ctx, userID, -delta)
}

// Delete soft-deletes a user by flipping is_active to false.
func (r *UserRepo) Delete(ctx context.Context, id string) error {
	if err := r.db.WithContext(ctx).
		Model(&model.User{}).
		Where("id = ?", id).
		Update("is_active", false).Error; err != nil {
		return fmt.Errorf("delete user: %w", err)
	}
	return nil
}

// List returns a page of users ordered by creation time descending.
func (r *UserRepo) List(ctx context.Context, page, pageSize int) ([]model.User, int64, error) {
	var users []model.User
	var total int64

	db := r.db.WithContext(ctx).Model(&model.User{})

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count users: %w", err)
	}

	offset := (page - 1) * pageSize
	if err := db.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&users).Error; err != nil {
		return nil, 0, fmt.Errorf("list users: %w", err)
	}

	return users, total, nil
}

// ExistsByUsernameOrEmail reports whether any row has the given username or email.
func (r *UserRepo) ExistsByUsernameOrEmail(ctx context.Context, username, email string) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&model.User{}).
		Where("username = ? OR email = ?", username, email).
		Count(&count).Error; err != nil {
		return false, fmt.Errorf("check user exists: %w", err)
	}
	return count > 0, nil
}

// ExistsByUsernameOrEmailExcept reports whether any row (other than excludeID) collides on
// username or email. Used when a user edits their own profile.
func (r *UserRepo) ExistsByUsernameOrEmailExcept(ctx context.Context, username, email, excludeID string) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&model.User{}).
		Where("(username = ? OR email = ?) AND id <> ?", username, email, excludeID).
		Count(&count).Error; err != nil {
		return false, fmt.Errorf("check user exists except: %w", err)
	}
	return count > 0, nil
}

// GetTokenVersion returns the user's current token_version, the revocation
// counter JWTs are validated against. It is called on every authenticated
// request, so the query is intentionally narrow — primary-key SELECT of a
// single integer column. Callers treat "record not found" as an invalid
// token so a deleted user cannot keep using outstanding JWTs.
func (r *UserRepo) GetTokenVersion(ctx context.Context, userID string) (int, error) {
	var version int
	err := r.db.WithContext(ctx).
		Model(&model.User{}).
		Where("id = ?", userID).
		Select("token_version").
		Scan(&version).Error
	if err != nil {
		return 0, fmt.Errorf("get token version: %w", err)
	}
	return version, nil
}

// BumpTokenVersion atomically increments the user's token_version so every
// outstanding JWT signed against the previous value fails the middleware
// check. Called after password changes / resets to revoke stolen sessions.
func (r *UserRepo) BumpTokenVersion(ctx context.Context, userID string) error {
	res := r.db.WithContext(ctx).
		Model(&model.User{}).
		Where("id = ?", userID).
		Update("token_version", gorm.Expr("token_version + 1"))
	if res.Error != nil {
		return fmt.Errorf("bump token version: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return fmt.Errorf("bump token version: user %q not found", userID)
	}
	return nil
}

// Count returns the total user count; used by the setup flow to detect fresh installs.
func (r *UserRepo) Count(ctx context.Context) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&model.User{}).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("count users: %w", err)
	}
	return count, nil
}
