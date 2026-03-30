package model

import (
	"time"

	"project-srv/internal/sqlboiler"
)

// ProjectStatus represents the operational status of a project
type ProjectStatus string

const (
	ProjectStatusPending  ProjectStatus = "PENDING"
	ProjectStatusActive   ProjectStatus = "ACTIVE"
	ProjectStatusPaused   ProjectStatus = "PAUSED"
	ProjectStatusArchived ProjectStatus = "ARCHIVED"
)

// ProjectConfigStatus represents the setup wizard status of a project
type ProjectConfigStatus string

const (
	ConfigStatusDraft          ProjectConfigStatus = "DRAFT"
	ConfigStatusConfiguring    ProjectConfigStatus = "CONFIGURING"
	ConfigStatusOnboarding     ProjectConfigStatus = "ONBOARDING"
	ConfigStatusOnboardingDone ProjectConfigStatus = "ONBOARDING_DONE"
	ConfigStatusDryRunRunning  ProjectConfigStatus = "DRYRUN_RUNNING"
	ConfigStatusDryRunSuccess  ProjectConfigStatus = "DRYRUN_SUCCESS"
	ConfigStatusDryRunFailed   ProjectConfigStatus = "DRYRUN_FAILED"
	ConfigStatusActive         ProjectConfigStatus = "ACTIVE"
	ConfigStatusError          ProjectConfigStatus = "ERROR"
)

// EntityType represents the type of entity being monitored
type EntityType string

const (
	EntityTypeProduct    EntityType = "product"
	EntityTypeCampaign   EntityType = "campaign"
	EntityTypeService    EntityType = "service"
	EntityTypeCompetitor EntityType = "competitor"
	EntityTypeTopic      EntityType = "topic"
)

// Project represents a specific entity monitoring unit
type Project struct {
	ID           string              `json:"id"`
	CampaignID   string              `json:"campaign_id"`
	Name         string              `json:"name"`
	Description  string              `json:"description,omitempty"`
	Brand        string              `json:"brand,omitempty"`
	EntityType   EntityType          `json:"entity_type"`
	EntityName   string              `json:"entity_name"`
	Status       ProjectStatus       `json:"status"`
	ConfigStatus ProjectConfigStatus `json:"config_status"`
	CreatedBy    string              `json:"created_by"`
	CreatedAt    time.Time           `json:"created_at"`
	UpdatedAt    time.Time           `json:"updated_at"`

	// Relations
	Campaign *Campaign `json:"campaign,omitempty"`
}

func IsValidProjectStatus(status string) bool {
	switch ProjectStatus(status) {
	case ProjectStatusPending, ProjectStatusActive, ProjectStatusPaused, ProjectStatusArchived:
		return true
	default:
		return false
	}
}

func IsValidEntityType(entityType string) bool {
	switch EntityType(entityType) {
	case EntityTypeProduct, EntityTypeCampaign, EntityTypeService, EntityTypeCompetitor, EntityTypeTopic:
		return true
	default:
		return false
	}
}

func CanActivateProjectStatus(status ProjectStatus) bool {
	return status == ProjectStatusPending
}

func CanPauseProjectStatus(status ProjectStatus) bool {
	return status == ProjectStatusActive
}

func CanResumeProjectStatus(status ProjectStatus) bool {
	return status == ProjectStatusPaused
}

func CanArchiveProjectStatus(status ProjectStatus) bool {
	return status == ProjectStatusPending || status == ProjectStatusActive || status == ProjectStatusPaused
}

func CanUnarchiveProjectStatus(status ProjectStatus) bool {
	return status == ProjectStatusArchived
}

// NewProjectFromDB converts a sqlboiler Project to a domain Project.
// Returns nil if the input is nil.
func NewProjectFromDB(db *sqlboiler.Project) *Project {
	if db == nil {
		return nil
	}

	p := &Project{
		ID:         db.ID,
		CampaignID: db.CampaignID,
		Name:       db.Name,
		EntityType: EntityType(db.EntityType),
		EntityName: db.EntityName,
		Status:     ProjectStatus(db.Status),
		CreatedBy:  db.CreatedBy,
	}

	if db.Description.Valid {
		p.Description = db.Description.String
	}
	if db.Brand.Valid {
		p.Brand = db.Brand.String
	}
	if db.ConfigStatus.Valid {
		p.ConfigStatus = ProjectConfigStatus(db.ConfigStatus.Val)
	}
	if db.CreatedAt.Valid {
		p.CreatedAt = db.CreatedAt.Time
	}
	if db.UpdatedAt.Valid {
		p.UpdatedAt = db.UpdatedAt.Time
	}

	// Load relation if eagerly loaded
	if db.R != nil && db.R.GetCampaign() != nil {
		p.Campaign = NewCampaignFromDB(db.R.GetCampaign())
	}

	return p
}
