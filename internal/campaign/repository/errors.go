package repository

import "errors"

var (
	ErrFailedToInsert = errors.New("campaign: failed to insert")
	ErrFailedToGet    = errors.New("campaign: failed to get")
	ErrFailedToList   = errors.New("campaign: failed to list")
	ErrFailedToUpdate = errors.New("campaign: failed to update")
	ErrFailedToDelete = errors.New("campaign: failed to delete")
)
