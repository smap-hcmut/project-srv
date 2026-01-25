package repository

import (
	"smap-project/internal/model"
	"smap-project/pkg/paginator"
	"time"
)

type CreateOptions struct {
	Name               string
	Description        *string
	Status             string
	FromDate           time.Time
	ToDate             time.Time
	BrandName          string
	CompetitorNames    []string
	BrandKeywords      []string
	CompetitorKeywords []model.CompetitorKeyword // Array of competitor keywords
	CreatedBy          string
}

type UpdateOptions struct {
	ID                 string
	Description        *string
	Status             *string
	FromDate           *time.Time
	ToDate             *time.Time
	BrandKeywords      []string
	CompetitorKeywords []model.CompetitorKeyword // Array of competitor keywords
}

type GetOptions struct {
	IDs           []string
	Statuses      []string
	CreatedBy     *string
	SearchName    *string
	PaginateQuery paginator.PaginateQuery
}

type ListOptions struct {
	IDs        []string
	Statuses   []string
	CreatedBy  *string
	SearchName *string
}

type GetOneOptions struct {
	ID string
}
