package project

import (
	"errors"

	"project-srv/pkg/microservice"
)

// Domain errors — returned by UseCase layer.
var (
	ErrNotFound                     = errors.New("project not found")
	ErrNameRequired                 = errors.New("project name is required")
	ErrCampaignRequired             = errors.New("campaign_id is required")
	ErrCampaignNotFound             = errors.New("campaign not found")
	ErrInvalidStatus                = errors.New("invalid project status")
	ErrInvalidEntity                = errors.New("invalid entity type")
	ErrInvalidTransition            = errors.New("invalid project lifecycle transition")
	ErrActivateNotAllowed           = errors.New("project cannot be activated in its current state")
	ErrPauseNotAllowed              = errors.New("project cannot be paused in its current state")
	ErrResumeNotAllowed             = errors.New("project cannot be resumed in its current state")
	ErrUnarchiveNotAllowed          = errors.New("project cannot be unarchived in its current state")
	ErrReadinessFailed              = errors.New("project activation readiness failed")
	ErrLifecycleManagerFailed       = errors.New("project lifecycle manager failed")
	ErrLifecycleManagerRejected     = errors.New("project lifecycle manager rejected request")
	ErrLifecycleManagerUnauthorized = errors.New("project lifecycle manager unauthorized")
	ErrLifecycleManagerForbidden    = errors.New("project lifecycle manager forbidden")
	ErrDeleteRequiresArchived       = errors.New("project must be archived before delete")
	ErrCreateFailed                 = errors.New("failed to create project")
	ErrDetailFailed                 = errors.New("failed to get project detail")
	ErrUpdateFailed                 = errors.New("failed to update project")
	ErrDeleteFailed                 = errors.New("failed to delete project")
	ErrListFailed                   = errors.New("failed to list projects")
)

func MapLifecycleClientError(err error) error {
	switch {
	case errors.Is(err, microservice.ErrBadRequest):
		return ErrLifecycleManagerRejected
	case errors.Is(err, microservice.ErrUnauthorized):
		return ErrLifecycleManagerUnauthorized
	case errors.Is(err, microservice.ErrForbidden):
		return ErrLifecycleManagerForbidden
	default:
		return ErrLifecycleManagerFailed
	}
}
