package usecase

import (
	"project-srv/internal/ontology"
	repo "project-srv/internal/ontology/repository"
	"project-srv/internal/project"

	"github.com/smap-hcmut/shared-libs/go/log"
)

type implUseCase struct {
	l         log.Logger
	repo      repo.Repository
	projectUC project.UseCase
}

func New(l log.Logger, repo repo.Repository, projectUC project.UseCase) ontology.UseCase {
	return &implUseCase{l: l, repo: repo, projectUC: projectUC}
}
