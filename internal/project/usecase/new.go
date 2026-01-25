package usecase

import (
	"time"

	"smap-project/internal/keyword"
	"smap-project/internal/project"
	"smap-project/internal/project/delivery/rabbitmq/producer"
	"smap-project/internal/project/repository"
	"smap-project/internal/sampling"
	"smap-project/internal/state"
	"smap-project/internal/webhook"
	pkgLog "smap-project/pkg/log"
)

type usecase struct {
	l         pkgLog.Logger
	repo      repository.Repository
	clock     func() time.Time
	keywordUC keyword.UseCase
	producer  producer.Producer
	webhookUC webhook.UseCase
	stateUC   state.UseCase
	sampler   sampling.UseCase
}

// New creates a new project usecase
func New(l pkgLog.Logger, repo repository.Repository, keywordUC keyword.UseCase, producer producer.Producer, webhookUC webhook.UseCase, stateUC state.UseCase, sampler sampling.UseCase) project.UseCase {
	return &usecase{
		l:         l,
		repo:      repo,
		clock:     time.Now,
		keywordUC: keywordUC,
		producer:  producer,
		webhookUC: webhookUC,
		stateUC:   stateUC,
		sampler:   sampler,
	}
}
