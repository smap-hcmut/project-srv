package model

import (
	"time"

	"project-srv/internal/sqlboiler"
)

// CampaignStatus represents the status of a campaign
type CampaignStatus string

const (
	CampaignStatusPending  CampaignStatus = "PENDING"
	CampaignStatusActive   CampaignStatus = "ACTIVE"
	CampaignStatusPaused   CampaignStatus = "PAUSED"
	CampaignStatusArchived CampaignStatus = "ARCHIVED"
)

// Campaign represents a high-level marketing campaign
type Campaign struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Status      CampaignStatus `json:"status"`
	StartDate   *time.Time     `json:"start_date,omitempty"`
	EndDate     *time.Time     `json:"end_date,omitempty"`
	CreatedBy   string         `json:"created_by"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

// NewCampaignFromDB converts a sqlboiler Campaign to a domain Campaign.
// Returns nil if the input is nil.
func NewCampaignFromDB(db *sqlboiler.Campaign) *Campaign {
	if db == nil {
		return nil
	}

	c := &Campaign{
		ID:        db.ID,
		Name:      db.Name,
		Status:    CampaignStatus(db.Status),
		CreatedBy: db.CreatedBy,
	}

	if db.Description.Valid {
		c.Description = db.Description.String
	}
	if db.StartDate.Valid {
		t := db.StartDate.Time
		c.StartDate = &t
	}
	if db.EndDate.Valid {
		t := db.EndDate.Time
		c.EndDate = &t
	}
	if db.CreatedAt.Valid {
		c.CreatedAt = db.CreatedAt.Time
	}
	if db.UpdatedAt.Valid {
		c.UpdatedAt = db.UpdatedAt.Time
	}

	return c
}
