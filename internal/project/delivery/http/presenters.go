package http

import (
	"project-srv/internal/model"
	"project-srv/internal/project"

	"github.com/smap-hcmut/shared-libs/go/paginator"
)

// --- Request DTOs ---

// createReq represents project creation request
type createReq struct {
	CampaignID  string `json:"-"`                                                                                                  // Parent campaign ID (from path param)
	Name        string `json:"name" binding:"required" example:"VinFast VF8 Monitoring"`                                           // Project name (required)
	Description string `json:"description" example:"Monitor discussions about VF8 electric SUV"`                                   // Project description
	Brand       string `json:"brand" example:"VinFast"`                                                                            // Brand name for UI grouping
	EntityType  string `json:"entity_type" binding:"required" example:"product" enums:"product,campaign,service,competitor,topic"` // Entity type (required)
	EntityName  string `json:"entity_name" binding:"required" example:"VF8"`                                                       // Specific entity name (required)
}

func (r createReq) validate() error {
	if r.Name == "" {
		return errNameRequired
	}
	if r.CampaignID == "" {
		return errCampaignRequired
	}
	if r.EntityType == "" {
		return errEntityTypeRequired
	}
	switch model.EntityType(r.EntityType) {
	case model.EntityTypeProduct, model.EntityTypeCampaign, model.EntityTypeService, model.EntityTypeCompetitor, model.EntityTypeTopic:
		// valid
	default:
		return errInvalidEntity
	}
	if r.EntityName == "" {
		return errEntityNameRequired
	}
	return nil
}

func (r createReq) toInput() project.CreateInput {
	return project.CreateInput{
		CampaignID:  r.CampaignID,
		Name:        r.Name,
		Description: r.Description,
		Brand:       r.Brand,
		EntityType:  r.EntityType,
		EntityName:  r.EntityName,
	}
}

type detailReq struct {
	ID string
}

func (r detailReq) toInput() string {
	return r.ID
}

// updateReq represents project update request
type updateReq struct {
	ID          string `json:"-"`                                                                               // Project ID (from path param)
	Name        string `json:"name" example:"VinFast VF8 Monitoring - Updated"`                                 // Project name
	Description string `json:"description" example:"Monitor discussions about VF8 electric SUV - Updated"`      // Project description
	Brand       string `json:"brand" example:"VinFast"`                                                         // Brand name
	EntityType  string `json:"entity_type" example:"product" enums:"product,campaign,service,competitor,topic"` // Entity type
	EntityName  string `json:"entity_name" example:"VF8"`                                                       // Specific entity name
}

func (r updateReq) validate() error {
	if r.EntityType != "" {
		switch model.EntityType(r.EntityType) {
		case model.EntityTypeProduct, model.EntityTypeCampaign, model.EntityTypeService, model.EntityTypeCompetitor, model.EntityTypeTopic:
			// valid
		default:
			return errInvalidEntity
		}
	}
	return nil
}

func (r updateReq) toInput() project.UpdateInput {
	return project.UpdateInput{
		ID:          r.ID,
		Name:        r.Name,
		Description: r.Description,
		Brand:       r.Brand,
		EntityType:  r.EntityType,
		EntityName:  r.EntityName,
	}
}

type archiveReq struct {
	ID string
}

func (r archiveReq) toInput() string {
	return r.ID
}

type listReq struct {
	paginator.PaginateQuery
	CampaignID string `form:"-"`
	Status     string `form:"status"`
	Name       string `form:"name"`
	Brand      string `form:"brand"`
	EntityType string `form:"entity_type"`
}

func (r listReq) toInput() project.ListInput {
	r.PaginateQuery.Adjust()
	return project.ListInput{
		CampaignID: r.CampaignID,
		Status:     r.Status,
		Name:       r.Name,
		Brand:      r.Brand,
		EntityType: r.EntityType,
		Paginator:  r.PaginateQuery,
	}
}

// --- Response DTOs ---

// projectResp represents project data in API responses
type projectResp struct {
	ID           string `json:"id" example:"550e8400-e29b-41d4-a716-446655440002"`                                                                                                     // Project UUID
	CampaignID   string `json:"campaign_id" example:"550e8400-e29b-41d4-a716-446655440000"`                                                                                            // Parent campaign UUID
	Name         string `json:"name" example:"VinFast VF8 Monitoring"`                                                                                                                 // Project name
	Description  string `json:"description,omitempty" example:"Monitor discussions about VF8 electric SUV"`                                                                            // Project description
	Brand        string `json:"brand,omitempty" example:"VinFast"`                                                                                                                     // Brand name for UI grouping
	EntityType   string `json:"entity_type" example:"product" enums:"product,campaign,service,competitor,topic"`                                                                       // Entity type
	EntityName   string `json:"entity_name" example:"VF8"`                                                                                                                             // Specific entity name
	Status       string `json:"status" example:"DRAFT" enums:"DRAFT,ACTIVE,PAUSED,ARCHIVED"`                                                                                           // Project status
	ConfigStatus string `json:"config_status,omitempty" example:"DRAFT" enums:"DRAFT,CONFIGURING,ONBOARDING,ONBOARDING_DONE,DRYRUN_RUNNING,DRYRUN_SUCCESS,DRYRUN_FAILED,ACTIVE,ERROR"` // Project configuration status
	CreatedBy    string `json:"created_by" example:"550e8400-e29b-41d4-a716-446655440001"`                                                                                             // Creator user UUID
	CreatedAt    string `json:"created_at" example:"2026-02-18T00:00:00Z"`                                                                                                             // Creation timestamp
	UpdatedAt    string `json:"updated_at" example:"2026-02-18T00:00:00Z"`                                                                                                             // Last update timestamp
}

// createResp wraps project creation response
type createResp struct {
	Project projectResp `json:"project"` // Created project data
}

// detailResp wraps project detail response
type detailResp struct {
	Project projectResp `json:"project"` // Project details
}

// listResp wraps paginated project list response
type listResp struct {
	Projects  []projectResp               `json:"projects"`  // List of projects
	Paginator paginator.PaginatorResponse `json:"paginator"` // Pagination metadata
}

// updateResp wraps project update response
type updateResp struct {
	Project projectResp `json:"project"` // Updated project data
}

// lifecycleResp wraps project lifecycle response.
type lifecycleResp struct {
	Project projectResp `json:"project"`
}

// --- Response Mappers ---

func (h *handler) newCreateResp(o project.CreateOutput) createResp {
	return createResp{Project: toProjectResp(o.Project)}
}

func (h *handler) newDetailResp(o project.DetailOutput) detailResp {
	return detailResp{Project: toProjectResp(o.Project)}
}

func (h *handler) newListResp(o project.ListOutput) listResp {
	projects := make([]projectResp, len(o.Projects))
	for i, p := range o.Projects {
		projects[i] = toProjectResp(p)
	}
	return listResp{
		Projects:  projects,
		Paginator: o.Paginator.ToResponse(),
	}
}

func (h *handler) newUpdateResp(o project.UpdateOutput) updateResp {
	return updateResp{Project: toProjectResp(o.Project)}
}

func (h *handler) newLifecycleResp(p model.Project) lifecycleResp {
	return lifecycleResp{Project: toProjectResp(p)}
}

// --- Internal Mapper ---

func toProjectResp(p model.Project) projectResp {
	resp := projectResp{
		ID:         p.ID,
		CampaignID: p.CampaignID,
		Name:       p.Name,
		EntityType: string(p.EntityType),
		EntityName: p.EntityName,
		Status:     string(p.Status),
		CreatedBy:  p.CreatedBy,
		CreatedAt:  p.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:  p.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	if p.Description != "" {
		resp.Description = p.Description
	}
	if p.Brand != "" {
		resp.Brand = p.Brand
	}
	if p.ConfigStatus != "" {
		resp.ConfigStatus = string(p.ConfigStatus)
	}

	return resp
}
