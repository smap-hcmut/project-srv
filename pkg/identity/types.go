package identity

import (
	"time"
)

// User represents the user information returned by the Identity Service.
type User struct {
	ID        string    `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	FullName  string    `json:"full_name"`
	AvatarURL string    `json:"avatar_url"`
	IsActive  bool      `json:"is_active"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Plan represents the plan information embedded in the subscription.
type Plan struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Code        string  `json:"code"`
	MaxUsage    int     `json:"max_usage"`
	Description *string `json:"description,omitempty"`
}

// Subscription represents the subscription information returned by the Identity Service.
type Subscription struct {
	ID          string     `json:"id"`
	UserID      string     `json:"user_id"`
	PlanID      string     `json:"plan_id"`
	Status      string     `json:"status"`
	TrialEndsAt *time.Time `json:"trial_ends_at,omitempty"`
	StartsAt    time.Time  `json:"starts_at"`
	EndsAt      *time.Time `json:"ends_at,omitempty"`
	CancelledAt *time.Time `json:"cancelled_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	Plan        Plan       `json:"plan"`
}

// Response wrappers
type userResponse struct {
	Data User `json:"data"`
}

type subscriptionResponse struct {
	Data Subscription `json:"data"`
}
