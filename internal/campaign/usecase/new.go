package usecase

import (
	"project-srv/internal/campaign"
	repo "project-srv/internal/campaign/repository"
	projectrepo "project-srv/internal/project/repository"
	"project-srv/pkg/microservice"

	"github.com/smap-hcmut/shared-libs/go/log"
)

type implUseCase struct {
	l           log.Logger
	repo        repo.Repository
	projectRepo projectrepo.Repository
	ingest      microservice.IngestUseCase
}

// New creates a new Campaign UseCase.
func New(l log.Logger, repo repo.Repository, projectRepo projectrepo.Repository, ingest microservice.IngestUseCase) campaign.UseCase {
	return &implUseCase{
		l:           l,
		repo:        repo,
		projectRepo: projectRepo,
		ingest:      ingest,
	}
}
