package usecase

import (
	"project-srv/internal/campaign"
	crisisrepo "project-srv/internal/crisis/repository"
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
	crisisRepo crisisrepo.Repository
}

// New creates a new Project use case.
func New(
	l log.Logger,
	r repo.Repository,
	domainRepo domain.Repository,
	campaignUC campaign.UseCase,
	ingest microservice.IngestUseCase,
	publisher projectproducer.Producer,
	crisisRepo crisisrepo.Repository,
) project.UseCase {
	return &implUseCase{
		l:          l,
		repo:       r,
		domainRepo: domainRepo,
		campaignUC: campaignUC,
		ingest:     ingest,
		publisher:  publisher,
		crisisRepo: crisisRepo,
	}
}
