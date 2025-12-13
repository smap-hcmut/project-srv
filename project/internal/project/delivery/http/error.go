package http

import (
	"smap-project/internal/project"
	pkgErrors "smap-project/pkg/errors"
)

var (
	errWrongQuery              = pkgErrors.NewHTTPError(30001, "Wrong query")
	errWrongBody               = pkgErrors.NewHTTPError(30002, "Wrong body")
	errNotFound                = pkgErrors.NewHTTPError(30004, "Project not found")
	errUnauthorized            = pkgErrors.NewHTTPError(30005, "Unauthorized")
	errInvalidStatus           = pkgErrors.NewHTTPError(30006, "Invalid project status")
	errInvalidDateRange        = pkgErrors.NewHTTPError(30007, "Invalid date range")
	errAlreadyExecuting        = pkgErrors.NewHTTPError(30008, "Project is already executing")
	errInvalidStatusTransition = pkgErrors.NewHTTPError(30009, "Invalid status transition")
)

var NotFound = []error{
	errNotFound,
}

func (h handler) mapErrorCode(err error) error {
	switch err {
	case project.ErrProjectNotFound:
		return errNotFound
	case project.ErrUnauthorized:
		return errUnauthorized
	case project.ErrInvalidStatus:
		return errInvalidStatus
	case project.ErrInvalidDateRange:
		return errInvalidDateRange
	case project.ErrProjectAlreadyExecuting:
		return errAlreadyExecuting
	case project.ErrInvalidStatusTransition:
		return errInvalidStatusTransition
	default:
		return err
	}
}
