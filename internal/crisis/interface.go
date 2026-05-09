package crisis

import (
	"context"
)

// UseCase defines the business logic interface for Crisis Config operations.
type UseCase interface {
	// Crisis Config CRUD
	Upsert(ctx context.Context, input UpsertInput) (UpsertOutput, error)
	Detail(ctx context.Context, projectID string) (DetailOutput, error)
	RuntimeConfig(ctx context.Context, projectID string) (RuntimeConfigOutput, error)
	Delete(ctx context.Context, projectID string) error
	ApplyRuntime(ctx context.Context, input ApplyRuntimeInput) (ApplyRuntimeOutput, error)
}
