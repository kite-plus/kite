package model

import "time"

// EmailVerificationPurpose enumerates what a verification code unlocks.
// Kept as typed constants so callers cannot pass a free-form string.
const (
	EmailVerifyPurposeEmailChange   = "email_change"
	EmailVerifyPurposePasswordReset = "password_reset"
)

// EmailVerification stores a short-lived hashed verification code tied to a
// user and a target email address. Codes are single-use; ConsumedAt is set
// when a successful verification retires the row.
type EmailVerification struct {
	ID         string     `gorm:"column:id;primaryKey" json:"id"`
	UserID     string     `gorm:"column:user_id;index;not null" json:"user_id"`
	NewEmail   string     `gorm:"column:new_email;not null" json:"new_email"`
	CodeHash   string     `gorm:"column:code_hash;not null" json:"-"`
	Purpose    string     `gorm:"column:purpose;index;not null" json:"purpose"`
	ExpiresAt  time.Time  `gorm:"column:expires_at;not null" json:"expires_at"`
	ConsumedAt *time.Time `gorm:"column:consumed_at" json:"consumed_at,omitempty"`
	CreatedAt  time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

func (EmailVerification) TableName() string { return "email_verifications" }
