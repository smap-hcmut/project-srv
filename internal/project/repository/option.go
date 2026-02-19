package repository

import (
	"project-srv/pkg/paginator"
)

// CreateOptions contains the data needed to create a new project.
type CreateOptions struct {
	CampaignID  string
	Name        string
	Description string
	Brand       string
	EntityType  string
	EntityName  string
	CreatedBy   string
}

// GetOptions contains filters for listing projects.
type GetOptions struct {
	CampaignID string
	Status     string
	Name       string
	Brand      string
	EntityType string
	Paginator  paginator.PaginateQuery
}

// UpdateOptions contains the data for updating a project.
type UpdateOptions struct {
	ID          string
	Name        string
	Description string
	Brand       string
	EntityType  string
	EntityName  string
	Status      string
}
