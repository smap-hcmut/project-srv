package postgres

import (
	"database/sql"
	"time"

	"smap-project/internal/project/repository"
	"smap-project/pkg/log"
)

type implRepository struct {
	db    *sql.DB
	l     log.Logger
	clock func() time.Time
}

// New creates a new PostgreSQL repository for projects
func New(db *sql.DB, l log.Logger) repository.Repository {
	return &implRepository{
		db:    db,
		l:     l,
		clock: time.Now,
	}
}
