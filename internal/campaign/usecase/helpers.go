package usecase

import (
	"project-srv/internal/campaign"
	"project-srv/internal/model"
)

const (
	campaignSortCreatedAtDesc = "created_at_desc"
	campaignSortFavoriteDesc  = "favorite_desc"
)

// validateStatus checks if the given status is a valid CampaignStatus.
func validateStatus(status string) error {
	switch model.CampaignStatus(status) {
	case model.CampaignStatusPending, model.CampaignStatusActive, model.CampaignStatusPaused, model.CampaignStatusArchived:
		return nil
	default:
		return campaign.ErrInvalidStatus
	}
}

func validateSort(sort string) error {
	switch sort {
	case "", campaignSortCreatedAtDesc, campaignSortFavoriteDesc:
		return nil
	default:
		return campaign.ErrInvalidSort
	}
}

func favoriteCampaignForUser(item model.Campaign, userID string) model.Campaign {
	item.IsFavorite = false
	if userID == "" {
		return item
	}
	for _, favoriteUserID := range item.FavoriteUserIDs {
		if favoriteUserID == userID {
			item.IsFavorite = true
			break
		}
	}
	return item
}
