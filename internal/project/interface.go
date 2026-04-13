package project

import (
	"context"

	"project-srv/internal/domain"
)

// UseCase defines the business logic interface for Project operations.
type UseCase interface {
	Create(ctx context.Context, input CreateInput) (CreateOutput, error)
	Detail(ctx context.Context, id string) (DetailOutput, error)
	List(ctx context.Context, input ListInput) (ListOutput, error)
	Update(ctx context.Context, input UpdateInput) (UpdateOutput, error)
	Favorite(ctx context.Context, id string) error
	Unfavorite(ctx context.Context, id string) error
	Delete(ctx context.Context, id string) error
	ListDomains(ctx context.Context) ([]domain.Domain, error)

	LifecycleManager
}

// LifecycleManager defines the integration boundary for project lifecycle orchestration.
type LifecycleManager interface {
	Activate(ctx context.Context, id string) (ActivateOutput, error)
	Pause(ctx context.Context, id string) (PauseOutput, error)
	Resume(ctx context.Context, id string) (ResumeOutput, error)
	Archive(ctx context.Context, id string) (ArchiveOutput, error)
	Unarchive(ctx context.Context, id string) (UnarchiveOutput, error)
	GetActivationReadiness(ctx context.Context, input ActivationReadinessInput) (ActivationReadiness, error)
}
