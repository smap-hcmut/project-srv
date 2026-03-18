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
}

// UpdateOutput is the output after updating a project.
type UpdateOutput struct {
	Project model.Project
}

// ActivationReadiness is the local readiness contract used by LifecycleManager.
type ActivationReadiness struct {
	ProjectID   string
	CanActivate bool
	Errors      []ActivationReadinessError
}

// ActivationReadinessError describes one readiness blocker.
type ActivationReadinessError struct {
	Code    string
	Message string
}

// ActivateOutput is the output after activating a project.
type ActivateOutput struct {
	Project model.Project
}

// PauseOutput is the output after pausing a project.
type PauseOutput struct {
	Project model.Project
}

// ResumeOutput is the output after resuming a project.
type ResumeOutput struct {
	Project model.Project
}

// ArchiveOutput is the output after archiving a project.
type ArchiveOutput struct {
	Project model.Project
}

// UnarchiveOutput is the output after unarchiving a project.
type UnarchiveOutput struct {
	Project model.Project
}
