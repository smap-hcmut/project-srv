package repository

import "errors"

// Repository-level errors.
var (
	ErrFailedToInsert = errors.New("failed to insert project")
	ErrNotFound       = errors.New("project not found")
	ErrFailedToGet    = errors.New("failed to get project")
	ErrFailedToUpdate = errors.New("failed to update project")
	ErrFailedToDelete = errors.New("failed to delete project")
	ErrFailedToList   = errors.New("failed to list projects")
)
