package microservice

import "context"

// IngestUseCase defines internal ingest lifecycle operations consumed by project-srv.
type IngestUseCase interface {
	GetActivationReadiness(ctx context.Context, input ActivationReadinessInput) (ActivationReadiness, error)
	Activate(ctx context.Context, projectID string) error
	Pause(ctx context.Context, projectID string) error
	Resume(ctx context.Context, projectID string) error
}
