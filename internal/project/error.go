package project

import "errors"

var (
	ErrProjectNotFound         = errors.New("project not found")
	ErrProjectExists           = errors.New("project already exists")
	ErrInvalidProject          = errors.New("invalid project")
	ErrFieldRequired           = errors.New("field required")
	ErrInvalidDateRange        = errors.New("invalid date range: to_date must be after from_date")
	ErrUnauthorized            = errors.New("unauthorized to access this project")
	ErrInvalidStatus           = errors.New("invalid project status")
	ErrInvalidStatusTransition = errors.New("invalid status transition")
	ErrInvalidUUID             = errors.New("invalid UUID format")
	ErrInvalidKeywordsMap      = errors.New("invalid competitor keywords map")
	ErrInvalidKeywords         = errors.New("invalid keywords")
	ErrProjectAlreadyExecuting = errors.New("project is already executing")
)
