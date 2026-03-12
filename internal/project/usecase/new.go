package usecase

import (
	"project-srv/internal/campaign"
	"project-srv/internal/project"
	repo "project-srv/internal/project/repository"

	"github.com/smap-hcmut/shared-libs/go/log"
)

type implUseCase struct {
	l          log.Logger
	repo       repo.Repository
	campaignUC campaign.UseCase
}

// New creates a new Project use case.
func New(l log.Logger, r repo.Repository, campaignUC campaign.UseCase) project.UseCase {
	return &implUseCase{l: l, repo: r, campaignUC: campaignUC}
}
