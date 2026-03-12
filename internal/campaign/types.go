package campaign

import (
	"project-srv/internal/model"

	"github.com/smap-hcmut/shared-libs/go/paginator"
)

// CreateInput is the input for creating a campaign.
type CreateInput struct {
	Name        string
	Description string
	StartDate   string // RFC3339
	EndDate     string // RFC3339
}

// CreateOutput is the output after creating a campaign.
type CreateOutput struct {
	Campaign model.Campaign
}

// DetailOutput is the output for getting campaign detail.
type DetailOutput struct {
	Campaign model.Campaign
}

// ListInput is the input for listing campaigns.
type ListInput struct {
	Status    string
	Name      string
	Paginator paginator.PaginateQuery
}

// ListOutput is the output for listing campaigns.
type ListOutput struct {
	Campaigns []model.Campaign
	Paginator paginator.Paginator
}

// UpdateInput is the input for updating a campaign.
type UpdateInput struct {
	ID          string
	Name        string
	Description string
	Status      string
	StartDate   string // RFC3339
	EndDate     string // RFC3339
}

// UpdateOutput is the output after updating a campaign.
type UpdateOutput struct {
	Campaign model.Campaign
}
