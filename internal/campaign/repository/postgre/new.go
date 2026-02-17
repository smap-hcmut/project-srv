package postgre

import (
	"database/sql"

	repo "project-srv/internal/campaign/repository"
	"project-srv/pkg/log"
)

type implRepository struct {
	db *sql.DB
	l  log.Logger
}

// New creates a new PostgreSQL campaign repository.
func New(db *sql.DB, l log.Logger) repo.Repository {
	return &implRepository{
		db: db,
		l:  l,
	}
}
