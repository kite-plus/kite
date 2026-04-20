package model

import "time"

// User is a registered account. The role column distinguishes admins from
// regular users; regular users self-register through the sign-up page.
type User struct {
	ID                 string    `gorm:"column:id;primaryKey" json:"id"`
	Username           string    `gorm:"column:username;uniqueIndex;not null" json:"username"`
	Nickname           *string   `gorm:"column:nickname" json:"nickname,omitempty"`
	Email              string    `gorm:"column:email;uniqueIndex;not null" json:"email"`
	AvatarURL          *string   `gorm:"column:avatar_url" json:"avatar_url,omitempty"`
	PasswordHash       string    `gorm:"column:password_hash;not null" json:"-"`
	HasLocalPassword   bool      `gorm:"column:has_local_password;not null;default:true" json:"has_local_password"`
	Role               string    `gorm:"column:role;default:user" json:"role"`                          // admin / user
	StorageLimit       int64     `gorm:"column:storage_limit;default:10737418240" json:"storage_limit"` // 10GB default; -1 means unlimited
	StorageUsed        int64     `gorm:"column:storage_used;default:0" json:"storage_used"`
	IsActive           bool      `gorm:"column:is_active;default:true" json:"is_active"`
	PasswordMustChange bool      `gorm:"column:password_must_change;default:false" json:"password_must_change"` // forces account reset on first login
	CreatedAt          time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt          time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (User) TableName() string { return "users" }

// HasStorageSpace reports whether the user has room to store fileSize more bytes.
func (u *User) HasStorageSpace(fileSize int64) bool {
	if u.StorageLimit == -1 {
		return true
	}
	return u.StorageUsed+fileSize <= u.StorageLimit
}
