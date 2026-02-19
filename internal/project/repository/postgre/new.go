package postgre

import (
	"database/sql"

	repo "project-srv/internal/project/repository"
	"project-srv/pkg/log"
)

type implRepository struct {
	db *sql.DB
	l  log.Logger
}

// New creates a new PostgreSQL project repository.
func New(db *sql.DB, l log.Logger) repo.Repository {
	return &implRepository{db: db, l: l}
}
