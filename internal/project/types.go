package project

import (
	"project-srv/internal/model"
	"time"

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
	ProjectID                string
	ProjectStatus            model.ProjectStatus
	DataSourceCount          int
	HasDatasource            bool
	PassiveUnconfirmedCount  int
	MissingTargetDryrunCount int
	FailedTargetDryrunCount  int
	CanProceed               bool
	Errors                   []ActivationReadinessError
}

// ActivationReadinessError describes one readiness blocker.
type ActivationReadinessError struct {
	Code         string
	Message      string
	DataSourceID string
	TargetID     string
}

// LifecycleEventName is a typed event name for project lifecycle.
type LifecycleEventName string

const (
	ProjectLifecycleEventActivated  LifecycleEventName = "project.lifecycle.activated"
	ProjectLifecycleEventPaused     LifecycleEventName = "project.lifecycle.paused"
	ProjectLifecycleEventResumed    LifecycleEventName = "project.lifecycle.resumed"
	ProjectLifecycleEventArchived   LifecycleEventName = "project.lifecycle.archived"
	ProjectLifecycleEventUnarchived LifecycleEventName = "project.lifecycle.unarchived"
)

// LifecycleEvent is emitted to Kafka after local lifecycle status transitions.
type LifecycleEvent struct {
	EventName   LifecycleEventName `json:"event_name"`
	ProjectID   string             `json:"project_id"`
	CampaignID  string             `json:"campaign_id"`
	Status      string             `json:"status"`
	TriggeredBy string             `json:"triggered_by"`
	OccurredAt  time.Time          `json:"occurred_at"`
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
