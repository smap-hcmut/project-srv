package http

import (
	"errors"

	"smap-project/internal/model"
	"smap-project/internal/project"
	"smap-project/pkg/paginator"
	postgres "smap-project/pkg/postgre"
	"smap-project/pkg/response"
	"smap-project/pkg/util"
)

// CreateReq
type CompetitorsReq struct {
	Name     string   `json:"name" binding:"required"`
	Keywords []string `json:"keywords" binding:"required"`
}

type CreateReq struct {
	Name          string           `json:"name" binding:"required"`
	Description   *string          `json:"description"`
	FromDate      string           `json:"from_date" binding:"required"`
	ToDate        string           `json:"to_date" binding:"required"`
	BrandName     string           `json:"brand_name" binding:"required"`
	BrandKeywords []string         `json:"brand_keywords" binding:"required"`
	Competitors   []CompetitorsReq `json:"competitors" binding:"required"`
}

func (r CreateReq) toInput() project.CreateInput {
	comKwInputs := make([]project.CompetitorKeywordOpts, len(r.Competitors))
	for i, competitor := range r.Competitors {
		comKwInputs[i] = project.CompetitorKeywordOpts{
			Name:     competitor.Name,
			Keywords: competitor.Keywords,
		}
	}

	fromDate, _ := util.StrToDateTime(r.FromDate)
	toDate, _ := util.StrToDateTime(r.ToDate)
	return project.CreateInput{
		Name:               r.Name,
		Description:        r.Description,
		FromDate:           fromDate,
		ToDate:             toDate,
		BrandName:          r.BrandName,
		BrandKeywords:      r.BrandKeywords,
		CompetitorKeywords: comKwInputs,
	}
}

func (r CreateReq) validate() error {
	_, err := util.StrToDateTime(r.FromDate)
	if err != nil {
		return errors.New("invalid from date")
	}

	_, err = util.StrToDateTime(r.ToDate)
	if err != nil {
		return errors.New("invalid to date")
	}

	return nil
}

// PatchReq represents the HTTP request for patching a project
type PatchReq struct {
	ID                 string           `uri:"id" binding:"required"`
	Description        *string          `json:"description"`
	Status             *string          `json:"status"`
	BrandKeywords      []string         `json:"brand_keywords"`
	CompetitorKeywords []CompetitorsReq `json:"competitor_keywords"`
}

func (r PatchReq) validate() error {
	if err := postgres.IsUUID(r.ID); err != nil {
		return errors.New("invalid id")
	}

	// Validate status if provided
	if r.Status != nil {
		if !model.IsValidProjectStatus(*r.Status) {
			return errors.New("invalid status")
		}
	}

	return nil
}

func (r PatchReq) toInput() project.PatchInput {
	comKwInputs := make([]project.CompetitorKeywordOpts, len(r.CompetitorKeywords))
	for i, competitor := range r.CompetitorKeywords {
		comKwInputs[i] = project.CompetitorKeywordOpts{
			Name:     competitor.Name,
			Keywords: competitor.Keywords,
		}
	}

	return project.PatchInput{
		ID:                 r.ID,
		Description:        r.Description,
		Status:             r.Status,
		BrandKeywords:      r.BrandKeywords,
		CompetitorKeywords: comKwInputs,
	}
}

// GetReq represents the HTTP request for listing projects with filters
type GetReq struct {
	IDs        []string `form:"ids"`
	Statuses   []string `form:"statuses"`
	SearchName *string  `form:"search_name"`
	Paginate   paginator.PaginateQuery
}

func (r GetReq) validate() error {
	// Validate all status values if provided
	for _, status := range r.Statuses {
		if !model.IsValidProjectStatus(status) {
			return errors.New("invalid status")
		}
	}
	return nil
}

func (r GetReq) toInput() project.GetInput {
	return project.GetInput{
		Filter: project.Filter{
			IDs:        r.IDs,
			Statuses:   r.Statuses,
			SearchName: r.SearchName,
		},
		PaginateQuery: r.Paginate,
	}
}

type DeleteReq struct {
	IDs []string `json:"ids" binding:"required"`
}

func (r DeleteReq) validate() error {
	if err := postgres.ValidateUUIDs(r.IDs); err != nil {
		return errors.New("invalid ids")
	}

	return nil
}

func (r DeleteReq) toInput() project.DeleteInput {
	return project.DeleteInput{
		IDs: r.IDs,
	}
}

// CompetitorKeywordResp represents competitor keyword in HTTP response
type CompetitorKeywordResp struct {
	CompetitorName string   `json:"competitor_name"`
	Keywords       []string `json:"keywords"`
}

type RespObj struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Status      *string           `json:"status,omitempty"`
	Description *string           `json:"description,omitempty"`
	FromDate    response.DateTime `json:"from_date,omitempty"`
	ToDate      response.DateTime `json:"to_date,omitempty"`
}

// ProjectResp represents the HTTP response for a single project
type ProjectResp struct {
	ID                 string                  `json:"id"`
	Name               string                  `json:"name"`
	Description        *string                 `json:"description,omitempty"`
	Status             string                  `json:"status"`
	FromDate           response.DateTime       `json:"from_date"`
	ToDate             response.DateTime       `json:"to_date"`
	BrandName          string                  `json:"brand_name"`
	CompetitorNames    []string                `json:"competitor_names,omitempty"`
	BrandKeywords      []string                `json:"brand_keywords"`
	CompetitorKeywords []CompetitorKeywordResp `json:"competitor_keywords,omitempty"`
	CreatedBy          RespObj                 `json:"created_by"`
	CreatedAt          response.DateTime       `json:"created_at"`
	UpdatedAt          response.DateTime       `json:"updated_at"`
}

// GetResp represents the HTTP response for multiple projects with pagination
type GetResp struct {
	Projects  []RespObj           `json:"projects"`
	Paginator paginator.Paginator `json:"paginator"`
}

func (h handler) newProjectResp(o project.ProjectOutput) ProjectResp {
	comKwResp := make([]CompetitorKeywordResp, len(o.Project.CompetitorKeywords))
	for i, ck := range o.Project.CompetitorKeywords {
		comKwResp[i] = CompetitorKeywordResp{
			CompetitorName: ck.CompetitorName,
			Keywords:       ck.Keywords,
		}
	}

	return ProjectResp{
		ID:                 o.Project.ID,
		Name:               o.Project.Name,
		Description:        o.Project.Description,
		Status:             o.Project.Status,
		FromDate:           response.DateTime(o.Project.FromDate),
		ToDate:             response.DateTime(o.Project.ToDate),
		BrandName:          o.Project.BrandName,
		CompetitorNames:    o.Project.CompetitorNames,
		BrandKeywords:      o.Project.BrandKeywords,
		CompetitorKeywords: comKwResp,
		CreatedBy: RespObj{
			ID:   o.CreatedBy.ID,
			Name: o.CreatedBy.Username,
		},
		CreatedAt: response.DateTime(o.Project.CreatedAt),
		UpdatedAt: response.DateTime(o.Project.UpdatedAt),
	}
}

func (h handler) newGetResp(o project.GetProjectOutput) GetResp {
	resp := make([]RespObj, len(o.Projects))
	for i, p := range o.Projects {
		resp[i] = RespObj{
			ID:       p.ID,
			Name:     p.Name,
			Status:   &p.Status,
			FromDate: response.DateTime(p.FromDate),
			ToDate:   response.DateTime(p.ToDate),
		}

		if p.Description != nil {
			resp[i].Description = p.Description
		}
	}

	return GetResp{
		Projects:  resp,
		Paginator: o.Paginator,
	}
}

type SuggestKeywordsResp struct {
	NicheKeywords    []string `json:"niche_keywords"`
	NegativeKeywords []string `json:"negative_keywords"`
}

func (h handler) newSuggestKeywordsResp(niche []string, negative []string) SuggestKeywordsResp {
	return SuggestKeywordsResp{
		NicheKeywords:    niche,
		NegativeKeywords: negative,
	}
}

// DryRunJobResp represents the HTTP response for a dry-run job creation
type DryRunJobResp struct {
	JobID             string   `json:"job_id"`
	Status            string   `json:"status"`
	SampledKeywords   []string `json:"sampled_keywords"`
	TotalKeywords     int      `json:"total_keywords"`
	SamplingRatio     float64  `json:"sampling_ratio"`
	EstimatedDuration string   `json:"estimated_duration"` // Duration as string for JSON
}

type DryRunKeywordsReq struct {
	Keywords []string `json:"keywords" binding:"required"`
}

func (r DryRunKeywordsReq) validate() error {
	if len(r.Keywords) == 0 {
		return errors.New("keywords cannot be empty")
	}
	return nil
}

func (r DryRunKeywordsReq) toInput() project.DryRunKeywordsInput {
	return project.DryRunKeywordsInput{
		Keywords: r.Keywords,
	}
}

// toDryRunJobResp converts project.DryRunKeywordsOutput to DryRunJobResp
func toDryRunJobResp(output project.DryRunKeywordsOutput) DryRunJobResp {
	return DryRunJobResp{
		JobID:             output.JobID,
		Status:            output.Status,
		SampledKeywords:   output.SampledKeywords,
		TotalKeywords:     output.TotalKeywords,
		SamplingRatio:     output.SamplingRatio,
		EstimatedDuration: output.EstimatedDuration.String(),
	}
}

// ProgressResp represents the HTTP response for project progress
type ProgressResp struct {
	ID              string            `json:"id"`
	Name            string            `json:"name"`
	Description     *string           `json:"description,omitempty"`
	Status          string            `json:"status"`
	FromDate        response.DateTime `json:"from_date"`
	ToDate          response.DateTime `json:"to_date"`
	TotalItems      int64             `json:"total_items"`
	ProcessedItems  int64             `json:"processed_items"`
	FailedItems     int64             `json:"failed_items"`
	ProgressPercent float64           `json:"progress_percent"`
}

func (h handler) newProgressResp(o project.ProgressOutput) ProgressResp {
	return ProgressResp{
		ID:              o.Project.ID,
		Name:            o.Project.Name,
		Description:     o.Project.Description,
		Status:          o.Status,
		FromDate:        response.DateTime(o.Project.FromDate),
		ToDate:          response.DateTime(o.Project.ToDate),
		TotalItems:      o.TotalItems,
		ProcessedItems:  o.ProcessedItems,
		FailedItems:     o.FailedItems,
		ProgressPercent: o.ProgressPercent,
	}
}

// PhaseProgressResp represents progress data for a single processing phase.
// Used in API responses to show granular progress per phase (crawl or analyze).
type PhaseProgressResp struct {
	Total           int64   `json:"total"`            // Total items to process in this phase
	Done            int64   `json:"done"`             // Completed items in this phase
	Errors          int64   `json:"errors"`           // Failed items in this phase
	ProgressPercent float64 `json:"progress_percent"` // Completion percentage (0-100)
}

// ProjectProgressResp represents the HTTP response for phase-based project progress.
// This is the new format that provides separate progress for crawl and analyze phases.
type ProjectProgressResp struct {
	ProjectID              string            `json:"project_id"`               // Project unique identifier
	Status                 string            `json:"status"`                   // Current status: INITIALIZING, PROCESSING, DONE, FAILED
	Crawl                  PhaseProgressResp `json:"crawl"`                    // Crawl phase progress
	Analyze                PhaseProgressResp `json:"analyze"`                  // Analyze phase progress
	OverallProgressPercent float64           `json:"overall_progress_percent"` // Overall progress (0-100)
}

// newProjectProgressResp converts ProjectProgressOutput to ProjectProgressResp
func (h handler) newProjectProgressResp(o project.ProjectProgressOutput) ProjectProgressResp {
	return ProjectProgressResp{
		ProjectID: o.ProjectID,
		Status:    o.Status,
		Crawl: PhaseProgressResp{
			Total:           o.Crawl.Total,
			Done:            o.Crawl.Done,
			Errors:          o.Crawl.Errors,
			ProgressPercent: o.Crawl.ProgressPercent,
		},
		Analyze: PhaseProgressResp{
			Total:           o.Analyze.Total,
			Done:            o.Analyze.Done,
			Errors:          o.Analyze.Errors,
			ProgressPercent: o.Analyze.ProgressPercent,
		},
		OverallProgressPercent: o.OverallProgressPercent,
	}
}
