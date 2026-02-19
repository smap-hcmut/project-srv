package crisis

import (
	"context"
)

// UseCase defines the business logic interface for Crisis Config operations.
type UseCase interface {
	// Crisis Config CRUD
	Upsert(ctx context.Context, input UpsertInput) (UpsertOutput, error)
	Detail(ctx context.Context, projectID string) (DetailOutput, error)
	Delete(ctx context.Context, projectID string) error
}
