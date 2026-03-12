package project

import (
	"project-srv/internal/model"

	"github.com/smap-hcmut/shared-libs/go/paginator"
)

// CreateInput is the input for creating a project.
type CreateInput struct {
	CampaignID  string
	Name        string
	Description string
	Brand       string
	EntityType  string
	EntityName  string
}

// CreateOutput is the output after creating a project.
type CreateOutput struct {
	Project model.Project
}

// DetailOutput is the output for getting project detail.
type DetailOutput struct {
	Project model.Project
}

// ListInput is the input for listing projects.
type ListInput struct {
	CampaignID string
	Status     string
	Name       string
	Brand      string
	EntityType string
	Paginator  paginator.PaginateQuery
}

// ListOutput is the output for listing projects.
type ListOutput struct {
	Projects  []model.Project
	Paginator paginator.Paginator
}

// UpdateInput is the input for updating a project.
type UpdateInput struct {
	ID          string
	Name        string
	Description string
	Brand       string
	EntityType  string
	EntityName  string
	Status      string
}

// UpdateOutput is the output after updating a project.
type UpdateOutput struct {
	Project model.Project
}
