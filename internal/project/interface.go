package project

import (
	"context"
)

// UseCase defines the business logic interface for Project operations.
type UseCase interface {
	Create(ctx context.Context, input CreateInput) (CreateOutput, error)
	Detail(ctx context.Context, id string) (DetailOutput, error)
	List(ctx context.Context, input ListInput) (ListOutput, error)
	Update(ctx context.Context, input UpdateInput) (UpdateOutput, error)
	Archive(ctx context.Context, id string) error
}
