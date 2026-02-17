package campaign

import (
	"context"
)

// UseCase defines the business logic interface for Campaign operations.
type UseCase interface {
	// Campaign CRUD
	Create(ctx context.Context, input CreateInput) (CreateOutput, error)
	Detail(ctx context.Context, id string) (DetailOutput, error)
	List(ctx context.Context, input ListInput) (ListOutput, error)
	Update(ctx context.Context, input UpdateInput) (UpdateOutput, error)
	Archive(ctx context.Context, id string) error
}
