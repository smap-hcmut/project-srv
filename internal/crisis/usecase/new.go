package usecase

import (
	"project-srv/internal/crisis"
	repo "project-srv/internal/crisis/repository"
	"project-srv/internal/project"

	"github.com/smap-hcmut/shared-libs/go/log"
)

type implUseCase struct {
	l         log.Logger
	repo      repo.Repository
	projectUC project.UseCase
}

// New creates a new Crisis Config UseCase.
func New(l log.Logger, repo repo.Repository, projectUC project.UseCase) crisis.UseCase {
	return &implUseCase{
		l:         l,
		repo:      repo,
		projectUC: projectUC,
	}
}
