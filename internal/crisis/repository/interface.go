package repository

import (
	"context"

	"project-srv/internal/model"
)

// Repository defines the data access interface for Crisis Config.
type Repository interface {
	Upsert(ctx context.Context, opt UpsertOptions) (model.CrisisConfig, error)
	Detail(ctx context.Context, projectID string) (model.CrisisConfig, error)
	Delete(ctx context.Context, projectID string) error
}
