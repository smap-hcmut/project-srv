package scope

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	// TokenExpirationDuration is the default expiration time for JWT tokens (1 week)
	TokenExpirationDuration = time.Hour * 24 * 7
)

// Manager defines the interface for JWT token management.
type Manager interface {
	// Verify verifies a JWT token and returns the payload if valid.
	Verify(token string) (Payload, error)
}

// Payload represents the JWT token claims.
type Payload struct {
	jwt.RegisteredClaims
	UserID   string `json:"sub"`      // Subject (user ID)
	Username string `json:"username"` // Username
	Role     string `json:"role"`     // User role (USER, ADMIN)
	Type     string `json:"type"`     // Token type (e.g., "access", "refresh")
	Refresh  bool   `json:"refresh"`  // Whether this is a refresh token
}

type implManager struct {
	secretKey string
}

// New creates a new JWT manager with the provided secret key.
func New(secretKey string) Manager {
	if secretKey == "" {
		panic("JWT secret key cannot be empty")
	}
	return &implManager{
		secretKey: secretKey,
	}
}

// Verify verifies the JWT token and returns the payload if valid.
// It checks the token signature, expiration, and claims structure.
func (m implManager) Verify(token string) (Payload, error) {
	if token == "" {
		return Payload{}, fmt.Errorf("%w: token is empty", ErrInvalidToken)
	}

	keyFunc := func(token *jwt.Token) (interface{}, error) {
		// Verify signing method is HMAC
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("%w: unexpected signing method: %v", ErrInvalidToken, token.Header["alg"])
		}
		return []byte(m.secretKey), nil
	}

	var payload Payload
	jwtToken, err := jwt.ParseWithClaims(token, &payload, keyFunc)
	if err != nil {
		return Payload{}, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}

	// Verify token is valid
	if !jwtToken.Valid {
		return Payload{}, fmt.Errorf("%w: token is not valid", ErrInvalidToken)
	}

	return payload, nil
}
