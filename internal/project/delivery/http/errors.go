package http

import (
	"project-srv/internal/project"

	pkgErrors "github.com/smap-hcmut/shared-libs/go/errors"
)

// Delivery-layer HTTP errors — all known errors are 400-level.
var (
	errNotFound           = pkgErrors.NewHTTPError(160001, "Project not found")
	errNameRequired       = pkgErrors.NewHTTPError(160002, "Project name is required")
	errCampaignRequired   = pkgErrors.NewHTTPError(160003, "Campaign ID is required")
	errCampaignNotFound   = pkgErrors.NewHTTPError(160004, "Campaign not found")
	errInvalidStatus      = pkgErrors.NewHTTPError(160005, "Invalid project status")
	errInvalidEntity      = pkgErrors.NewHTTPError(160006, "Invalid entity type")
	errCreateFailed       = pkgErrors.NewHTTPError(160007, "Failed to create project")
	errUpdateFailed       = pkgErrors.NewHTTPError(160008, "Failed to update project")
	errDeleteFailed       = pkgErrors.NewHTTPError(160009, "Failed to delete project")
	errListFailed         = pkgErrors.NewHTTPError(160010, "Failed to list projects")
	errWrongBody          = pkgErrors.NewHTTPError(160011, "Wrong request body")
	errEntityTypeRequired = pkgErrors.NewHTTPError(160012, "Entity type is required")
	errEntityNameRequired = pkgErrors.NewHTTPError(160013, "Entity name is required")
)

// mapError maps UseCase domain errors to delivery HTTP errors.
func (h *handler) mapError(err error) error {
	switch err {
	case project.ErrNotFound:
		return errNotFound
	case project.ErrNameRequired:
		return errNameRequired
	case project.ErrCampaignRequired:
		return errCampaignRequired
	case project.ErrCampaignNotFound:
		return errCampaignNotFound
	case project.ErrInvalidStatus:
		return errInvalidStatus
	case project.ErrInvalidEntity:
		return errInvalidEntity
	case project.ErrCreateFailed:
		return errCreateFailed
	case project.ErrUpdateFailed:
		return errUpdateFailed
	case project.ErrDeleteFailed:
		return errDeleteFailed
	case project.ErrListFailed:
		return errListFailed
	default:
		panic(err)
	}
}
