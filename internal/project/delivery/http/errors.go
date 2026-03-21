package http

import (
	"errors"
	"net/http"

	"project-srv/internal/project"

	pkgErrors "github.com/smap-hcmut/shared-libs/go/errors"
)

// Delivery-layer HTTP errors for project lifecycle and CRUD.
var (
	errNotFound                     = &pkgErrors.HTTPError{Code: 160001, Message: "Project not found", StatusCode: http.StatusBadRequest}
	errNameRequired                 = &pkgErrors.HTTPError{Code: 160002, Message: "Project name is required", StatusCode: http.StatusBadRequest}
	errCampaignRequired             = &pkgErrors.HTTPError{Code: 160003, Message: "Campaign ID is required", StatusCode: http.StatusBadRequest}
	errCampaignNotFound             = &pkgErrors.HTTPError{Code: 160004, Message: "Campaign not found", StatusCode: http.StatusBadRequest}
	errInvalidStatus                = &pkgErrors.HTTPError{Code: 160005, Message: "Invalid project status", StatusCode: http.StatusBadRequest}
	errInvalidEntity                = &pkgErrors.HTTPError{Code: 160006, Message: "Invalid entity type", StatusCode: http.StatusBadRequest}
	errCreateFailed                 = &pkgErrors.HTTPError{Code: 160007, Message: "Failed to create project", StatusCode: http.StatusInternalServerError}
	errUpdateFailed                 = &pkgErrors.HTTPError{Code: 160008, Message: "Failed to update project", StatusCode: http.StatusInternalServerError}
	errDeleteFailed                 = &pkgErrors.HTTPError{Code: 160009, Message: "Failed to delete project", StatusCode: http.StatusInternalServerError}
	errListFailed                   = &pkgErrors.HTTPError{Code: 160010, Message: "Failed to list projects", StatusCode: http.StatusInternalServerError}
	errWrongBody                    = &pkgErrors.HTTPError{Code: 160011, Message: "Wrong request body", StatusCode: http.StatusBadRequest}
	errEntityTypeRequired           = &pkgErrors.HTTPError{Code: 160012, Message: "Entity type is required", StatusCode: http.StatusBadRequest}
	errEntityNameRequired           = &pkgErrors.HTTPError{Code: 160013, Message: "Entity name is required", StatusCode: http.StatusBadRequest}
	errInvalidTransition            = &pkgErrors.HTTPError{Code: 160014, Message: "Invalid project lifecycle transition", StatusCode: http.StatusBadRequest}
	errActivateNotAllowed           = &pkgErrors.HTTPError{Code: 160015, Message: "Project cannot be activated in its current state", StatusCode: http.StatusBadRequest}
	errPauseNotAllowed              = &pkgErrors.HTTPError{Code: 160016, Message: "Project cannot be paused in its current state", StatusCode: http.StatusBadRequest}
	errResumeNotAllowed             = &pkgErrors.HTTPError{Code: 160017, Message: "Project cannot be resumed in its current state", StatusCode: http.StatusBadRequest}
	errUnarchiveNotAllowed          = &pkgErrors.HTTPError{Code: 160018, Message: "Project cannot be unarchived in its current state", StatusCode: http.StatusBadRequest}
	errReadinessFailed              = &pkgErrors.HTTPError{Code: 160019, Message: "Project activation readiness failed", StatusCode: http.StatusBadRequest}
	errReadinessDatasourceRequired  = &pkgErrors.HTTPError{Code: 160026, Message: "Project must have at least one datasource before lifecycle activation", StatusCode: http.StatusBadRequest}
	errReadinessPassiveUnconfirmed  = &pkgErrors.HTTPError{Code: 160027, Message: "Project has passive datasource that is not confirmed", StatusCode: http.StatusBadRequest}
	errReadinessTargetDryrunMissing = &pkgErrors.HTTPError{Code: 160028, Message: "Project has crawl target that has never been dry-run", StatusCode: http.StatusBadRequest}
	errReadinessTargetDryrunFailed  = &pkgErrors.HTTPError{Code: 160029, Message: "Project has crawl target whose latest dry-run failed", StatusCode: http.StatusBadRequest}
	errReadinessActiveTargetMissing = &pkgErrors.HTTPError{Code: 160030, Message: "Project has crawl datasource without active target", StatusCode: http.StatusBadRequest}
	errReadinessDatasourceStatus    = &pkgErrors.HTTPError{Code: 160031, Message: "Project has datasource in invalid status for lifecycle command", StatusCode: http.StatusBadRequest}
	errLifecycleManagerFailed       = &pkgErrors.HTTPError{Code: 160020, Message: "Project lifecycle manager failed", StatusCode: http.StatusInternalServerError}
	errDeleteRequiresArchived       = &pkgErrors.HTTPError{Code: 160021, Message: "Project must be archived before delete", StatusCode: http.StatusBadRequest}
	errDetailFailed                 = &pkgErrors.HTTPError{Code: 160022, Message: "Failed to get project detail", StatusCode: http.StatusInternalServerError}
	errLifecycleManagerRejected     = &pkgErrors.HTTPError{Code: 160023, Message: "Project lifecycle manager rejected request", StatusCode: http.StatusBadRequest}
	errLifecycleManagerUnauthorized = &pkgErrors.HTTPError{Code: 160024, Message: "Project lifecycle manager unauthorized", StatusCode: http.StatusUnauthorized}
	errLifecycleManagerForbidden    = &pkgErrors.HTTPError{Code: 160025, Message: "Project lifecycle manager forbidden", StatusCode: http.StatusForbidden}
)

// mapError maps UseCase domain errors to delivery HTTP errors.
func (h *handler) mapError(err error) error {
	switch {
	case errors.Is(err, project.ErrNotFound):
		return errNotFound
	case errors.Is(err, project.ErrNameRequired):
		return errNameRequired
	case errors.Is(err, project.ErrCampaignRequired):
		return errCampaignRequired
	case errors.Is(err, project.ErrCampaignNotFound):
		return errCampaignNotFound
	case errors.Is(err, project.ErrInvalidStatus):
		return errInvalidStatus
	case errors.Is(err, project.ErrInvalidEntity):
		return errInvalidEntity
	case errors.Is(err, project.ErrCreateFailed):
		return errCreateFailed
	case errors.Is(err, project.ErrDetailFailed):
		return errDetailFailed
	case errors.Is(err, project.ErrUpdateFailed):
		return errUpdateFailed
	case errors.Is(err, project.ErrDeleteFailed):
		return errDeleteFailed
	case errors.Is(err, project.ErrListFailed):
		return errListFailed
	case errors.Is(err, project.ErrInvalidTransition):
		return errInvalidTransition
	case errors.Is(err, project.ErrActivateNotAllowed):
		return errActivateNotAllowed
	case errors.Is(err, project.ErrPauseNotAllowed):
		return errPauseNotAllowed
	case errors.Is(err, project.ErrResumeNotAllowed):
		return errResumeNotAllowed
	case errors.Is(err, project.ErrUnarchiveNotAllowed):
		return errUnarchiveNotAllowed
	case errors.Is(err, project.ErrReadinessDatasourceRequired):
		return errReadinessDatasourceRequired
	case errors.Is(err, project.ErrReadinessPassiveUnconfirmed):
		return errReadinessPassiveUnconfirmed
	case errors.Is(err, project.ErrReadinessTargetDryrunMissing):
		return errReadinessTargetDryrunMissing
	case errors.Is(err, project.ErrReadinessTargetDryrunFailed):
		return errReadinessTargetDryrunFailed
	case errors.Is(err, project.ErrReadinessActiveTargetMissing):
		return errReadinessActiveTargetMissing
	case errors.Is(err, project.ErrReadinessDatasourceStatus):
		return errReadinessDatasourceStatus
	case errors.Is(err, project.ErrReadinessFailed):
		return errReadinessFailed
	case errors.Is(err, project.ErrLifecycleManagerFailed):
		return errLifecycleManagerFailed
	case errors.Is(err, project.ErrLifecycleManagerRejected):
		return errLifecycleManagerRejected
	case errors.Is(err, project.ErrLifecycleManagerUnauthorized):
		return errLifecycleManagerUnauthorized
	case errors.Is(err, project.ErrLifecycleManagerForbidden):
		return errLifecycleManagerForbidden
	case errors.Is(err, project.ErrDeleteRequiresArchived):
		return errDeleteRequiresArchived
	default:
		panic(err)
	}
}
