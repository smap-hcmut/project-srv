package usecase

import (
	"project-srv/internal/campaign"
	"project-srv/internal/domain"
	"project-srv/internal/project"
	projectproducer "project-srv/internal/project/delivery/kafka/producer"
	repo "project-srv/internal/project/repository"
	"project-srv/pkg/microservice"

	"github.com/smap-hcmut/shared-libs/go/log"
)

type implUseCase struct {
	l          log.Logger
	repo       repo.Repository
	domainRepo domain.Repository
	campaignUC campaign.UseCase
	ingest     microservice.IngestUseCase
	publisher  projectproducer.Producer
}

// New creates a new Project use case.
func New(
	l log.Logger,
	r repo.Repository,
	domainRepo domain.Repository,
	campaignUC campaign.UseCase,
	ingest microservice.IngestUseCase,
	publisher projectproducer.Producer,
) project.UseCase {
	return &implUseCase{
		l:          l,
		repo:       r,
		domainRepo: domainRepo,
		campaignUC: campaignUC,
		ingest:     ingest,
		publisher:  publisher,
	}
}
