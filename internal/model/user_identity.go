package model

import "time"

// UserIdentity stores a third-party login identity bound to a Kite user.
type UserIdentity struct {
	ID              string     `gorm:"column:id;primaryKey" json:"id"`
	UserID          string     `gorm:"column:user_id;index;not null" json:"user_id"`
	Provider        string     `gorm:"column:provider;not null;uniqueIndex:idx_user_identity_provider_uid" json:"provider"`
	ProviderUserID  string     `gorm:"column:provider_user_id;not null;uniqueIndex:idx_user_identity_provider_uid" json:"provider_user_id"`
	ProviderUnionID *string    `gorm:"column:provider_union_id" json:"provider_union_id,omitempty"`
	Email           *string    `gorm:"column:email" json:"email,omitempty"`
	EmailVerified   bool       `gorm:"column:email_verified;not null;default:false" json:"email_verified"`
	DisplayName     *string    `gorm:"column:display_name" json:"display_name,omitempty"`
	AvatarURL       *string    `gorm:"column:avatar_url" json:"avatar_url,omitempty"`
	RawProfile      string     `gorm:"column:raw_profile;type:text;not null;default:''" json:"-"`
	LastLoginAt     *time.Time `gorm:"column:last_login_at" json:"last_login_at,omitempty"`
	CreatedAt       time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt       time.Time  `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (UserIdentity) TableName() string { return "user_identities" }
