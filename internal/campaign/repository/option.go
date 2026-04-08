package repository

import (
	"time"

	"github.com/smap-hcmut/shared-libs/go/paginator"
)

// CreateOptions contains the data needed to create a new campaign.
type CreateOptions struct {
	Name        string
	Description string
	StartDate   *time.Time
	EndDate     *time.Time
	CreatedBy   string
}

// GetOptions contains filters for listing campaigns.
type GetOptions struct {
	Status        string
	Name          string
	CreatedBy     string
	FavoriteOnly  bool
	Sort          string
	CurrentUserID string
	Paginator     paginator.PaginateQuery
}

// UpdateOptions contains the data for updating a campaign.
type UpdateOptions struct {
	ID          string
	Name        string
	Description string
	Status      string
	StartDate   *time.Time
	EndDate     *time.Time
}
