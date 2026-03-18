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
	Activate(ctx context.Context, id string) (ActivateOutput, error)
	Pause(ctx context.Context, id string) (PauseOutput, error)
	Resume(ctx context.Context, id string) (ResumeOutput, error)
	Archive(ctx context.Context, id string) (ArchiveOutput, error)
	Unarchive(ctx context.Context, id string) (UnarchiveOutput, error)
	Delete(ctx context.Context, id string) error

	LifecycleManager
}

// LifecycleManager defines the integration boundary for project lifecycle orchestration.
type LifecycleManager interface {
	GetActivationReadiness(ctx context.Context, projectID string) (ActivationReadiness, error)
	ActivateProject(ctx context.Context, projectID string) error
	PauseProject(ctx context.Context, projectID string) error
	ResumeProject(ctx context.Context, projectID string) error
}
