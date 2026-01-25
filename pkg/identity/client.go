package identity

import (
	"context"
)

// Client defines the interface for the Identity Service client.
type Client interface {
	// GetUser retrieves a user by ID.
	GetUser(ctx context.Context, id string) (User, error)

	// GetUserSubscription retrieves the active subscription for a user.
	GetUserSubscription(ctx context.Context, userID string) (Subscription, error)
}
