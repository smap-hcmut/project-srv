package usecase

import (
	"project-srv/internal/campaign"
	"project-srv/internal/model"
)

// validateStatus checks if the given status is a valid CampaignStatus.
func validateStatus(status string) error {
	switch model.CampaignStatus(status) {
	case model.CampaignStatusActive, model.CampaignStatusInactive, model.CampaignStatusArchived:
		return nil
	default:
		return campaign.ErrInvalidStatus
	}
}
