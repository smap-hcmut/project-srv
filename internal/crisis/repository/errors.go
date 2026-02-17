package repository

import "errors"

var (
	ErrFailedToInsert = errors.New("crisis: failed to insert")
	ErrFailedToGet    = errors.New("crisis: failed to get")
	ErrFailedToUpdate = errors.New("crisis: failed to update")
	ErrFailedToDelete = errors.New("crisis: failed to delete")
)
