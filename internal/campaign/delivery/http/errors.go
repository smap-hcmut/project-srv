package http

import (
	"net/http"

	"project-srv/internal/campaign"

	pkgErrors "github.com/smap-hcmut/shared-libs/go/errors"
)

// Delivery-layer HTTP errors.
var (
	errNotFound          = &pkgErrors.HTTPError{Code: 150001, Message: "Campaign not found", StatusCode: http.StatusNotFound}
	errNameRequired      = &pkgErrors.HTTPError{Code: 150002, Message: "Campaign name is required", StatusCode: http.StatusBadRequest}
	errInvalidStatus     = &pkgErrors.HTTPError{Code: 150003, Message: "Invalid campaign status", StatusCode: http.StatusBadRequest}
	errInvalidSort       = &pkgErrors.HTTPError{Code: 150012, Message: "Invalid campaign sort", StatusCode: http.StatusBadRequest}
	errInvalidDateRange  = &pkgErrors.HTTPError{Code: 150004, Message: "Invalid date range: start_date must be before end_date", StatusCode: http.StatusBadRequest}
	errCreateFailed      = &pkgErrors.HTTPError{Code: 150005, Message: "Failed to create campaign", StatusCode: http.StatusInternalServerError}
	errUpdateFailed      = &pkgErrors.HTTPError{Code: 150006, Message: "Failed to update campaign", StatusCode: http.StatusInternalServerError}
	errDeleteFailed      = &pkgErrors.HTTPError{Code: 150007, Message: "Failed to delete campaign", StatusCode: http.StatusInternalServerError}
	errListFailed        = &pkgErrors.HTTPError{Code: 150008, Message: "Failed to list campaigns", StatusCode: http.StatusInternalServerError}
	errWrongBody         = &pkgErrors.HTTPError{Code: 150009, Message: "Wrong request body", StatusCode: http.StatusBadRequest}
	errWrongQuery        = &pkgErrors.HTTPError{Code: 150010, Message: "Wrong query parameters", StatusCode: http.StatusBadRequest}
	errInvalidDateFormat = &pkgErrors.HTTPError{Code: 150011, Message: "Invalid date format, expected RFC3339 (e.g. 2026-01-01T00:00:00Z)", StatusCode: http.StatusBadRequest}
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
	case campaign.ErrInvalidSort:
		return errInvalidSort
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
