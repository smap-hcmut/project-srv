package campaign

import "errors"

// Domain errors — returned by UseCase layer.
var (
	ErrNotFound         = errors.New("campaign not found")
	ErrNameRequired     = errors.New("campaign name is required")
	ErrInvalidStatus    = errors.New("invalid campaign status")
	ErrInvalidSort      = errors.New("invalid campaign sort")
	ErrInvalidDateRange = errors.New("invalid date range")
	ErrCreateFailed     = errors.New("failed to create campaign")
	ErrUpdateFailed     = errors.New("failed to update campaign")
	ErrDeleteFailed     = errors.New("failed to delete campaign")
	ErrPauseFailed      = errors.New("failed to pause campaign")
	ErrResumeFailed     = errors.New("failed to resume campaign")
	ErrListFailed       = errors.New("failed to list campaigns")
	ErrArchiveFailed    = errors.New("failed to archive campaign")
)
