package model

import (
	"time"
)

// User role constants
const (
	RoleUser  = "USER"
	RoleAdmin = "ADMIN"
)

// User represents a user entity in the domain layer.
// This is a safe type model that doesn't depend on database-specific types.
type User struct {
	ID           string     `json:"id"`
	Username     string     `json:"username"`
	FullName     *string    `json:"full_name,omitempty"`
	PasswordHash *string    `json:"password_hash,omitempty"`
	RoleHash     *string    `json:"role_hash,omitempty"` // Encrypted role value
	AvatarURL    *string    `json:"avatar_url,omitempty"`
	IsActive     *bool      `json:"is_active,omitempty"`
	OTP          *string    `json:"otp,omitempty"`
	OTPExpiredAt *time.Time `json:"otp_expired_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	DeletedAt    *time.Time `json:"deleted_at,omitempty"`
}
