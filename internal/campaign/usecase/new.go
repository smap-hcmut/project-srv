package usecase

import (
	"project-srv/internal/campaign"
	repo "project-srv/internal/campaign/repository"

	"github.com/smap-hcmut/shared-libs/go/log"
)

type implUseCase struct {
	l    log.Logger
	repo repo.Repository
}

// New creates a new Campaign UseCase.
func New(l log.Logger, repo repo.Repository) campaign.UseCase {
	return &implUseCase{
		l:    l,
		repo: repo,
	}
}
