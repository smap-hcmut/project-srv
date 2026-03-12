package http

import (
	"project-srv/internal/campaign"

	pkgErrors "github.com/smap-hcmut/shared-libs/go/errors"
)

// Delivery-layer HTTP errors — all known errors are 400-level.
var (
	errNotFound          = pkgErrors.NewHTTPError(150001, "Campaign not found")
	errNameRequired      = pkgErrors.NewHTTPError(150002, "Campaign name is required")
	errInvalidStatus     = pkgErrors.NewHTTPError(150003, "Invalid campaign status")
	errInvalidDateRange  = pkgErrors.NewHTTPError(150004, "Invalid date range: start_date must be before end_date")
	errCreateFailed      = pkgErrors.NewHTTPError(150005, "Failed to create campaign")
	errUpdateFailed      = pkgErrors.NewHTTPError(150006, "Failed to update campaign")
	errDeleteFailed      = pkgErrors.NewHTTPError(150007, "Failed to delete campaign")
	errListFailed        = pkgErrors.NewHTTPError(150008, "Failed to list campaigns")
	errWrongBody         = pkgErrors.NewHTTPError(150009, "Wrong request body")
	errWrongQuery        = pkgErrors.NewHTTPError(150010, "Wrong query parameters")
	errInvalidDateFormat = pkgErrors.NewHTTPError(150011, "Invalid date format, expected RFC3339 (e.g. 2026-01-01T00:00:00Z)")
)

// mapError maps UseCase domain errors to delivery HTTP errors.
func (h *handler) mapError(err error) error {
	switch err {
	case campaign.ErrNotFound:
		return errNotFound
	case campaign.ErrNameRequired:
		return errNameRequired
	case campaign.ErrInvalidStatus:
		return errInvalidStatus
	case campaign.ErrInvalidDateRange:
		return errInvalidDateRange
	case campaign.ErrCreateFailed:
		return errCreateFailed
	case campaign.ErrUpdateFailed:
		return errUpdateFailed
	case campaign.ErrDeleteFailed:
		return errDeleteFailed
	case campaign.ErrListFailed:
		return errListFailed
	default:
		panic(err)
	}
}
