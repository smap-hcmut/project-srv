package project

import "errors"

// Domain errors — returned by UseCase layer.
var (
	ErrNotFound         = errors.New("project not found")
	ErrNameRequired     = errors.New("project name is required")
	ErrCampaignRequired = errors.New("campaign_id is required")
	ErrCampaignNotFound = errors.New("campaign not found")
	ErrInvalidStatus    = errors.New("invalid project status")
	ErrInvalidEntity    = errors.New("invalid entity type")
	ErrCreateFailed     = errors.New("failed to create project")
	ErrUpdateFailed     = errors.New("failed to update project")
	ErrDeleteFailed     = errors.New("failed to delete project")
	ErrListFailed       = errors.New("failed to list projects")
)
