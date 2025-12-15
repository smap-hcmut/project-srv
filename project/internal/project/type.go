package project

import (
	"time"

	"smap-project/internal/model"
	"smap-project/pkg/paginator"
)

// CreateInput represents input for creating a new project
type CompetitorKeywordOpts struct {
	Name     string
	Keywords []string
}

type CreateInput struct {
	Name               string
	Description        *string
	FromDate           time.Time
	ToDate             time.Time
	BrandName          string
	BrandKeywords      []string
	CompetitorKeywords []CompetitorKeywordOpts
}

// UpdateInput represents input for updating a project
type PatchInput struct {
	ID                 string
	Description        *string
	Status             *string
	FromDate           *time.Time
	ToDate             *time.Time
	BrandKeywords      []string
	CompetitorKeywords []CompetitorKeywordOpts
}

// ProjectOutput represents output for a single project
type ProjectOutput struct {
	Project   model.Project
	CreatedBy model.User
}

// GetProjectOutput represents output for multiple projects with pagination
type GetProjectOutput struct {
	Projects  []model.Project
	Paginator paginator.Paginator
}

// GetOneInput represents input for getting a single project
type GetOneInput struct {
	ID string
}

// ListInput represents input for listing projects
type ListInput struct {
	Filter Filter
}

// GetInput represents input for getting projects with pagination
type GetInput struct {
	Filter        Filter
	PaginateQuery paginator.PaginateQuery
}

// Filter represents filtering options for projects
type Filter struct {
	IDs        []string
	Statuses   []string
	CreatedBy  *string // User ID who created the projects
	SearchName *string // Search by project name
}

// DeleteInput represents input for deleting projects
type DeleteInput struct {
	IDs []string
}

// DryRunKeywordsInput represents input for dry-run keywords request
type DryRunKeywordsInput struct {
	Keywords []string
}

// DryRunKeywordsOutput represents output for dry-run keywords request
type DryRunKeywordsOutput struct {
	JobID             string        `json:"job_id"`
	Status            string        `json:"status"` // "processing"
	SampledKeywords   []string      `json:"sampled_keywords"`
	TotalKeywords     int           `json:"total_keywords"`
	SamplingRatio     float64       `json:"sampling_ratio"`
	EstimatedDuration time.Duration `json:"estimated_duration"`
}

// ProgressOutput represents output for project progress tracking (old flat format)
type ProgressOutput struct {
	Project         model.Project
	Status          string
	TotalItems      int64
	ProcessedItems  int64
	FailedItems     int64
	ProgressPercent float64
}

// PhaseProgressOutput represents progress data for a single processing phase.
type PhaseProgressOutput struct {
	Total           int64
	Done            int64
	Errors          int64
	ProgressPercent float64
}

// ProjectProgressOutput represents phase-based project progress output.
// This is the new format that provides separate progress for crawl and analyze phases.
type ProjectProgressOutput struct {
	ProjectID              string
	Status                 string
	Crawl                  PhaseProgressOutput
	Analyze                PhaseProgressOutput
	OverallProgressPercent float64
}
