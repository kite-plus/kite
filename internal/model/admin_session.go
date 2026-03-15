package model

import "time"

type AdminSession struct {
	BaseModel
	TokenHash  string    `gorm:"size:64;not null;uniqueIndex" json:"-"`
	ExpiresAt  time.Time `gorm:"index;not null" json:"expires_at"`
	LastUsedAt time.Time `gorm:"not null" json:"last_used_at"`
	IP         string    `gorm:"size:64" json:"ip"`
	UserAgent  string    `gorm:"size:512" json:"user_agent"`
}
