package repository

import (
	"context"

	"project-srv/internal/model"
	"project-srv/pkg/paginator"
)

// Repository defines the data access interface for Campaign.
type Repository interface {
	Create(ctx context.Context, opt CreateOptions) (model.Campaign, error)
	Detail(ctx context.Context, id string) (model.Campaign, error)
	Get(ctx context.Context, opt GetOptions) ([]model.Campaign, paginator.Paginator, error)
	Update(ctx context.Context, opt UpdateOptions) (model.Campaign, error)
	Archive(ctx context.Context, id string) error
}
