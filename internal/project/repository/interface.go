package repository

import (
	"context"

	"project-srv/internal/model"
	"project-srv/pkg/paginator"
)

// Repository defines the data access interface for Project.
type Repository interface {
	Create(ctx context.Context, opt CreateOptions) (model.Project, error)
	Detail(ctx context.Context, id string) (model.Project, error)
	Get(ctx context.Context, opt GetOptions) ([]model.Project, paginator.Paginator, error)
	Update(ctx context.Context, opt UpdateOptions) (model.Project, error)
	Archive(ctx context.Context, id string) error
}
