package usecase

import (
	"smap-project/internal/state"
	"smap-project/internal/state/repository"
	pkgLog "smap-project/pkg/log"
)

type stateUseCase struct {
	repo   repository.StateRepository
	logger pkgLog.Logger
}

// New creates a new state usecase with business logic.
func New(repo repository.StateRepository, logger pkgLog.Logger) state.UseCase {
	return &stateUseCase{
		repo:   repo,
		logger: logger,
	}
}
