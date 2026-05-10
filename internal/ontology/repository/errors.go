package repository

import "errors"

var (
	ErrFailedToGet    = errors.New("failed to get ontology rules")
	ErrFailedToInsert = errors.New("failed to insert ontology rules")
	ErrFailedToUpdate = errors.New("failed to update ontology rules")
	ErrFailedToDelete = errors.New("failed to delete ontology rules")
)
