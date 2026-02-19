package usecase

import (
	"project-srv/internal/model"
	"project-srv/internal/project"
)

// validateStatus checks if the given status is valid for a project.
func validateStatus(status string) error {
	switch model.ProjectStatus(status) {
	case model.ProjectStatusActive, model.ProjectStatusPaused, model.ProjectStatusArchived:
		return nil
	default:
		return project.ErrInvalidStatus
	}
}

// validateEntityType checks if the given entity type is valid.
func validateEntityType(entityType string) error {
	switch model.EntityType(entityType) {
	case model.EntityTypeProduct, model.EntityTypeCampaign, model.EntityTypeService,
		model.EntityTypeCompetitor, model.EntityTypeTopic:
		return nil
	default:
		return project.ErrInvalidEntity
	}
}
