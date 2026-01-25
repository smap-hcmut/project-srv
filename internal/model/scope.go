package model

const (
	ScopeTypeAccess = "access"
	SMAPAPI         = "smap-api"
)

type Scope struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"` // USER or ADMIN
}

// IsAdmin checks if the scope has admin role
func (s Scope) IsAdmin() bool {
	return s.Role == RoleAdmin
}

// IsUser checks if the scope has user role
func (s Scope) IsUser() bool {
	return s.Role == RoleUser
}
