package identity

import "errors"

var (
	ErrIdentityUnavailable     = errors.New("identity service unavailable")
	ErrIdentityInvalidResponse = errors.New("invalid response from identity service")
	ErrIdentityTimeout         = errors.New("identity service timeout")
	ErrUserNotFound            = errors.New("user not found")
	ErrSubscriptionNotFound    = errors.New("subscription not found")
	ErrUnauthorized            = errors.New("unauthorized request to identity service")
)
