package http

import (
	"time"

	"project-srv/internal/campaign"
	"project-srv/internal/model"
	"project-srv/pkg/paginator"
)

// --- Request DTOs ---

// createReq represents campaign creation request
type createReq struct {
	Name        string `json:"name" example:"Q1 2026 VinFast Campaign"`               // Campaign name (required)
	Description string `json:"description" example:"Monitor VinFast brand sentiment"` // Campaign description
	StartDate   string `json:"start_date" example:"2026-01-01T00:00:00Z"`             // Campaign start date (RFC3339 format)
	EndDate     string `json:"end_date" example:"2026-03-31T23:59:59Z"`               // Campaign end date (RFC3339 format)
}

func (r createReq) validate() error {
	if r.Name == "" {
		return errNameRequired
	}
	if r.StartDate != "" && r.EndDate != "" {
		start, err1 := time.Parse(time.RFC3339, r.StartDate)
		end, err2 := time.Parse(time.RFC3339, r.EndDate)
		if err1 != nil || err2 != nil {
			return errInvalidDateFormat
		}
		if !start.Before(end) {
			return errInvalidDateRange
		}
	} else if r.StartDate != "" {
		if _, err := time.Parse(time.RFC3339, r.StartDate); err != nil {
			return errInvalidDateFormat
		}
	} else if r.EndDate != "" {
		if _, err := time.Parse(time.RFC3339, r.EndDate); err != nil {
			return errInvalidDateFormat
		}
	}
	return nil
}

func (r createReq) toInput() campaign.CreateInput {
	return campaign.CreateInput{
		Name:        r.Name,
		Description: r.Description,
		StartDate:   r.StartDate,
		EndDate:     r.EndDate,
	}
}

// updateReq represents campaign update request
type updateReq struct {
	Name        string `json:"name" example:"Q1 2026 VinFast Campaign - Updated"`               // Campaign name
	Description string `json:"description" example:"Monitor VinFast brand sentiment - Updated"` // Campaign description
	Status      string `json:"status" example:"ACTIVE" enums:"ACTIVE,INACTIVE,ARCHIVED"`        // Campaign status
	StartDate   string `json:"start_date" example:"2026-01-01T00:00:00Z"`                       // Campaign start date (RFC3339 format)
	EndDate     string `json:"end_date" example:"2026-03-31T23:59:59Z"`                         // Campaign end date (RFC3339 format)
}

func (r updateReq) validate() error {
	if r.Status != "" {
		switch model.CampaignStatus(r.Status) {
		case model.CampaignStatusActive, model.CampaignStatusInactive, model.CampaignStatusArchived:
			// valid
		default:
			return errInvalidStatus
		}
	}
	if r.StartDate != "" && r.EndDate != "" {
		start, err1 := time.Parse(time.RFC3339, r.StartDate)
		end, err2 := time.Parse(time.RFC3339, r.EndDate)
		if err1 != nil || err2 != nil {
			return errInvalidDateFormat
		}
		if !start.Before(end) {
			return errInvalidDateRange
		}
	} else if r.StartDate != "" {
		if _, err := time.Parse(time.RFC3339, r.StartDate); err != nil {
			return errInvalidDateFormat
		}
	} else if r.EndDate != "" {
		if _, err := time.Parse(time.RFC3339, r.EndDate); err != nil {
			return errInvalidDateFormat
		}
	}
	return nil
}

func (r updateReq) toInput(id string) campaign.UpdateInput {
	return campaign.UpdateInput{
		ID:          id,
		Name:        r.Name,
		Description: r.Description,
		Status:      r.Status,
		StartDate:   r.StartDate,
		EndDate:     r.EndDate,
	}
}

type listReq struct {
	paginator.PaginateQuery
	Status string `form:"status"`
	Name   string `form:"name"`
}

func (r listReq) toInput() campaign.ListInput {
	r.PaginateQuery.Adjust()
	return campaign.ListInput{
		Status:    r.Status,
		Name:      r.Name,
		Paginator: r.PaginateQuery,
	}
}

// --- Response DTOs ---

// campaignResp represents campaign data in API responses
type campaignResp struct {
	ID          string  `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`               // Campaign UUID
	Name        string  `json:"name" example:"Q1 2026 VinFast Campaign"`                         // Campaign name
	Description string  `json:"description,omitempty" example:"Monitor VinFast brand sentiment"` // Campaign description
	Status      string  `json:"status" example:"ACTIVE" enums:"ACTIVE,INACTIVE,ARCHIVED"`        // Campaign status
	StartDate   *string `json:"start_date,omitempty" example:"2026-01-01T00:00:00Z"`             // Campaign start date
	EndDate     *string `json:"end_date,omitempty" example:"2026-03-31T23:59:59Z"`               // Campaign end date
	CreatedBy   string  `json:"created_by" example:"550e8400-e29b-41d4-a716-446655440001"`       // Creator user UUID
	CreatedAt   string  `json:"created_at" example:"2026-02-18T00:00:00Z"`                       // Creation timestamp
	UpdatedAt   string  `json:"updated_at" example:"2026-02-18T00:00:00Z"`                       // Last update timestamp
}

// createResp wraps campaign creation response
type createResp struct {
	Campaign campaignResp `json:"campaign"` // Created campaign data
}

// detailResp wraps campaign detail response
type detailResp struct {
	Campaign campaignResp `json:"campaign"` // Campaign details
}

// listResp wraps paginated campaign list response
type listResp struct {
	Campaigns []campaignResp              `json:"campaigns"` // List of campaigns
	Paginator paginator.PaginatorResponse `json:"paginator"` // Pagination metadata
}

// updateResp wraps campaign update response
type updateResp struct {
	Campaign campaignResp `json:"campaign"` // Updated campaign data
}

// --- Response Mappers (receiver on handler) ---

func (h *handler) newCreateResp(o campaign.CreateOutput) createResp {
	return createResp{
		Campaign: toCampaignResp(o.Campaign),
	}
}

func (h *handler) newDetailResp(o campaign.DetailOutput) detailResp {
	return detailResp{
		Campaign: toCampaignResp(o.Campaign),
	}
}

func (h *handler) newListResp(o campaign.ListOutput) listResp {
	campaigns := make([]campaignResp, len(o.Campaigns))
	for i, cam := range o.Campaigns {
		campaigns[i] = toCampaignResp(cam)
	}
	return listResp{
		Campaigns: campaigns,
		Paginator: o.Paginator.ToResponse(),
	}
}

func (h *handler) newUpdateResp(o campaign.UpdateOutput) updateResp {
	return updateResp{
		Campaign: toCampaignResp(o.Campaign),
	}
}

// --- Internal Mapper ---

func toCampaignResp(c model.Campaign) campaignResp {
	resp := campaignResp{
		ID:        c.ID,
		Name:      c.Name,
		Status:    string(c.Status),
		CreatedBy: c.CreatedBy,
		CreatedAt: c.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: c.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	if c.Description != "" {
		resp.Description = c.Description
	}
	if c.StartDate != nil {
		s := c.StartDate.Format("2006-01-02T15:04:05Z07:00")
		resp.StartDate = &s
	}
	if c.EndDate != nil {
		s := c.EndDate.Format("2006-01-02T15:04:05Z07:00")
		resp.EndDate = &s
	}

	return resp
}
