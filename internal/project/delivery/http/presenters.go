package http

import (
	"project-srv/internal/model"
	"project-srv/internal/project"
	"strings"

	"github.com/google/uuid"
	"github.com/smap-hcmut/shared-libs/go/paginator"
)

// --- Request DTOs ---

// createReq represents project creation request
type createReq struct {
	CampaignID     string `json:"-"`                                                                                                  // Parent campaign ID (from path param)
	Name           string `json:"name" binding:"required" example:"VinFast VF8 Monitoring"`                                           // Project name (required)
	Description    string `json:"description" example:"Monitor discussions about VF8 electric SUV"`                                   // Project description
	Brand          string `json:"brand" example:"VinFast"`                                                                            // Brand name for UI grouping
	EntityType     string `json:"entity_type" binding:"required" example:"product" enums:"product,campaign,service,competitor,topic"` // Entity type (required)
	EntityName     string `json:"entity_name" binding:"required" example:"VF8"`                                                       // Specific entity name (required)
	DomainTypeCode string `json:"domain_type_code" binding:"required" example:"ev"`                                                   // Analysis/business domain code
}

func (r createReq) validate() error {
	if r.Name == "" {
		return errNameRequired
	}
	if r.CampaignID == "" {
		return errCampaignRequired
	}
	if !isValidUUID(r.CampaignID) {
		return errWrongBody
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
	if strings.TrimSpace(r.DomainTypeCode) == "" {
		return errDomainTypeRequired
	}
	return nil
}

func (r createReq) toInput() project.CreateInput {
	return project.CreateInput{
		CampaignID:     r.CampaignID,
		Name:           r.Name,
		Description:    r.Description,
		Brand:          r.Brand,
		EntityType:     r.EntityType,
		EntityName:     r.EntityName,
		DomainTypeCode: strings.TrimSpace(r.DomainTypeCode),
	}
}

type detailReq struct {
	ID string
}

func (r detailReq) validate() error {
	if !isValidUUID(r.ID) {
		return errWrongBody
	}
	return nil
}

func (r detailReq) toInput() string {
	return r.ID
}

// updateReq represents project update request
type updateReq struct {
	ID             string `json:"-"`                                                                               // Project ID (from path param)
	Name           string `json:"name" example:"VinFast VF8 Monitoring - Updated"`                                 // Project name
	Description    string `json:"description" example:"Monitor discussions about VF8 electric SUV - Updated"`      // Project description
	Brand          string `json:"brand" example:"VinFast"`                                                         // Brand name
	EntityType     string `json:"entity_type" example:"product" enums:"product,campaign,service,competitor,topic"` // Entity type
	EntityName     string `json:"entity_name" example:"VF8"`                                                       // Specific entity name
	DomainTypeCode string `json:"domain_type_code,omitempty" example:"ev"`                                         // Analysis/business domain code
}

func (r updateReq) validate() error {
	if !isValidUUID(r.ID) {
		return errWrongBody
	}
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
		ID:             r.ID,
		Name:           r.Name,
		Description:    r.Description,
		Brand:          r.Brand,
		EntityType:     r.EntityType,
		EntityName:     r.EntityName,
		DomainTypeCode: strings.TrimSpace(r.DomainTypeCode),
	}
}

type archiveReq struct {
	ID string
}

func (r archiveReq) validate() error {
	if !isValidUUID(r.ID) {
		return errWrongBody
	}
	return nil
}

func (r archiveReq) toInput() string {
	return r.ID
}

type activationReadinessReq struct {
	ID      string `form:"-"`
	Command string `form:"command"`
}

func (r activationReadinessReq) validate() error {
	if !isValidUUID(r.ID) {
		return errWrongBody
	}
	switch r.Command {
	case "", string(project.ActivationReadinessCommandActivate), string(project.ActivationReadinessCommandResume):
		return nil
	default:
		return errWrongBody
	}
}

func (r activationReadinessReq) toInput() project.ActivationReadinessInput {
	return project.ActivationReadinessInput{
		ProjectID: r.ID,
		Command:   project.ActivationReadinessCommand(r.Command),
	}
}

type listReq struct {
	paginator.PaginateQuery
	CampaignID   string `form:"-"`
	Status       string `form:"status"`
	Name         string `form:"name"`
	Brand        string `form:"brand"`
	EntityType   string `form:"entity_type"`
	FavoriteOnly bool   `form:"favorite_only"`
	Sort         string `form:"sort"`
}

func (r listReq) validate() error {
	if !isValidUUID(r.CampaignID) {
		return errWrongBody
	}
	switch r.Sort {
	case "", "created_at_desc", "favorite_desc":
	default:
		return errWrongQuery
	}
	return nil
}

func (r listReq) toInput() project.ListInput {
	r.PaginateQuery.Adjust()
	return project.ListInput{
		CampaignID:   r.CampaignID,
		Status:       r.Status,
		Name:         r.Name,
		Brand:        r.Brand,
		EntityType:   r.EntityType,
		FavoriteOnly: r.FavoriteOnly,
		Sort:         r.Sort,
		Paginator:    r.PaginateQuery,
	}
}

type favoriteListReq struct {
	paginator.PaginateQuery
	CampaignID   string `form:"campaign_id"`
	Status       string `form:"status"`
	Name         string `form:"name"`
	Brand        string `form:"brand"`
	EntityType   string `form:"entity_type"`
	FavoriteOnly bool   `form:"favorite_only"`
	Sort         string `form:"sort"`
}

func (r favoriteListReq) validate() error {
	if r.CampaignID != "" && !isValidUUID(r.CampaignID) {
		return errWrongQuery
	}
	switch r.Sort {
	case "", "created_at_desc", "favorite_desc":
	default:
		return errWrongQuery
	}
	return nil
}

func (r favoriteListReq) toInput() project.ListInput {
	r.PaginateQuery.Adjust()
	return project.ListInput{
		CampaignID:   r.CampaignID,
		Status:       r.Status,
		Name:         r.Name,
		Brand:        r.Brand,
		EntityType:   r.EntityType,
		FavoriteOnly: true,
		Sort:         r.Sort,
		Paginator:    r.PaginateQuery,
	}
}

// --- Response DTOs ---

// projectResp represents project data in API responses
type projectResp struct {
	ID             string `json:"id" example:"550e8400-e29b-41d4-a716-446655440002"`                               // Project UUID
	CampaignID     string `json:"campaign_id" example:"550e8400-e29b-41d4-a716-446655440000"`                      // Parent campaign UUID
	Name           string `json:"name" example:"VinFast VF8 Monitoring"`                                           // Project name
	Description    string `json:"description,omitempty" example:"Monitor discussions about VF8 electric SUV"`      // Project description
	Brand          string `json:"brand,omitempty" example:"VinFast"`                                               // Brand name for UI grouping
	EntityType     string `json:"entity_type" example:"product" enums:"product,campaign,service,competitor,topic"` // Entity type
	EntityName     string `json:"entity_name" example:"VF8"`                                                       // Specific entity name
	DomainTypeCode string `json:"domain_type_code" example:"ev"`                                                   // Analysis/business domain code
	Status         string `json:"status" example:"PENDING" enums:"PENDING,ACTIVE,PAUSED,ARCHIVED"`                 // Project status
	IsFavorite     bool   `json:"is_favorite" example:"false"`                                                     // Whether current user favorited this project
	CreatedBy      string `json:"created_by" example:"550e8400-e29b-41d4-a716-446655440001"`                       // Creator user UUID
	CreatedAt      string `json:"created_at" example:"2026-02-18T00:00:00Z"`                                       // Creation timestamp
	UpdatedAt      string `json:"updated_at" example:"2026-02-18T00:00:00Z"`                                       // Last update timestamp
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

// activationReadinessErrorResp describes one readiness blocker.
type activationReadinessErrorResp struct {
	Code         string `json:"code" example:"TARGET_DRYRUN_MISSING"`
	Message      string `json:"message" example:"crawl target has never been dry-run"`
	DataSourceID string `json:"data_source_id,omitempty" example:"550e8400-e29b-41d4-a716-446655440011"`
	TargetID     string `json:"target_id,omitempty" example:"550e8400-e29b-41d4-a716-446655440012"`
}

// activationReadinessResp wraps readiness output from project lifecycle manager + local status.
type activationReadinessResp struct {
	ProjectID                string                         `json:"project_id" example:"550e8400-e29b-41d4-a716-446655440002"`
	ProjectStatus            string                         `json:"project_status" example:"PENDING" enums:"PENDING,ACTIVE,PAUSED,ARCHIVED"`
	DataSourceCount          int                            `json:"data_source_count" example:"2"`
	HasDatasource            bool                           `json:"has_datasource" example:"true"`
	PassiveUnconfirmedCount  int                            `json:"passive_unconfirmed_count" example:"0"`
	MissingTargetDryrunCount int                            `json:"missing_target_dryrun_count" example:"1"`
	FailedTargetDryrunCount  int                            `json:"failed_target_dryrun_count" example:"0"`
	CanProceed               bool                           `json:"can_proceed" example:"false"`
	Errors                   []activationReadinessErrorResp `json:"errors"`
}

// --- Response Mappers ---

func (h *handler) newCreateResp(o project.CreateOutput) createResp {
	return createResp{Project: h.toProjectResp(o.Project)}
}

func (h *handler) newDetailResp(o project.DetailOutput) detailResp {
	return detailResp{Project: h.toProjectResp(o.Project)}
}

func (h *handler) newListResp(o project.ListOutput) listResp {
	projects := make([]projectResp, len(o.Projects))
	for i, p := range o.Projects {
		projects[i] = h.toProjectResp(p)
	}
	return listResp{
		Projects:  projects,
		Paginator: o.Paginator.ToResponse(),
	}
}

func (h *handler) newUpdateResp(o project.UpdateOutput) updateResp {
	return updateResp{Project: h.toProjectResp(o.Project)}
}

func (h *handler) newLifecycleResp(p model.Project) lifecycleResp {
	return lifecycleResp{Project: h.toProjectResp(p)}
}

func (h *handler) newActivationReadinessResp(o project.ActivationReadiness) activationReadinessResp {
	errors := make([]activationReadinessErrorResp, 0, len(o.Errors))
	for _, e := range o.Errors {
		errors = append(errors, activationReadinessErrorResp{
			Code:         string(e.Code),
			Message:      e.Message,
			DataSourceID: e.DataSourceID,
			TargetID:     e.TargetID,
		})
	}

	return activationReadinessResp{
		ProjectID:                o.ProjectID,
		ProjectStatus:            string(o.ProjectStatus),
		DataSourceCount:          o.DataSourceCount,
		HasDatasource:            o.HasDatasource,
		PassiveUnconfirmedCount:  o.PassiveUnconfirmedCount,
		MissingTargetDryrunCount: o.MissingTargetDryrunCount,
		FailedTargetDryrunCount:  o.FailedTargetDryrunCount,
		CanProceed:               o.CanProceed,
		Errors:                   errors,
	}
}

// --- Internal Mapper ---

func (h *handler) toProjectResp(p model.Project) projectResp {
	resp := projectResp{
		ID:             p.ID,
		CampaignID:     p.CampaignID,
		Name:           p.Name,
		EntityType:     string(p.EntityType),
		EntityName:     p.EntityName,
		DomainTypeCode: p.DomainTypeCode,
		Status:         string(p.Status),
		IsFavorite:     p.IsFavorite,
		CreatedBy:      p.CreatedBy,
		CreatedAt:      p.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:      p.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	if p.Description != "" {
		resp.Description = p.Description
	}
	if p.Brand != "" {
		resp.Brand = p.Brand
	}

	return resp
}

func isValidUUID(value string) bool {
	_, err := uuid.Parse(strings.TrimSpace(value))
	return err == nil
}
