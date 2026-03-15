package repo

import (
	"errors"
	"fmt"
	"time"

	"github.com/amigoer/kite-blog/internal/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

var ErrAdminSessionNotFound = errors.New("admin session not found")

type AdminSessionRepository struct {
	db *gorm.DB
}

func NewAdminSessionRepository(db *gorm.DB) *AdminSessionRepository {
	return &AdminSessionRepository{db: db}
}

func (r *AdminSessionRepository) Create(session *model.AdminSession) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("admin session repository is unavailable")
	}
	if err := r.db.Create(session).Error; err != nil {
		return fmt.Errorf("create admin session: %w", err)
	}
	return nil
}

func (r *AdminSessionRepository) GetByTokenHash(tokenHash string) (*model.AdminSession, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("admin session repository is unavailable")
	}

	var session model.AdminSession
	if err := r.db.First(&session, "token_hash = ?", tokenHash).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrAdminSessionNotFound
		}
		return nil, fmt.Errorf("get admin session by token hash: %w", err)
	}

	return &session, nil
}

func (r *AdminSessionRepository) UpdateLastUsedAt(id uuid.UUID, lastUsedAt time.Time) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("admin session repository is unavailable")
	}

	result := r.db.Model(&model.AdminSession{}).Where("id = ?", id).Update("last_used_at", lastUsedAt)
	if result.Error != nil {
		return fmt.Errorf("update admin session last_used_at: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrAdminSessionNotFound
	}
	return nil
}

func (r *AdminSessionRepository) DeleteByTokenHash(tokenHash string) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("admin session repository is unavailable")
	}

	result := r.db.Delete(&model.AdminSession{}, "token_hash = ?", tokenHash)
	if result.Error != nil {
		return fmt.Errorf("delete admin session by token hash: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrAdminSessionNotFound
	}
	return nil
}
