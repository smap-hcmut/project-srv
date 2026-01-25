package usecase

import (
	"smap-project/internal/keyword"
	"smap-project/pkg/llm"
	pkgLog "smap-project/pkg/log"
	"time"
)

type usecase struct {
	l           pkgLog.Logger
	llmProvider llm.Provider
	clock       func() time.Time
}

// New creates a new keyword use case
func New(l pkgLog.Logger, llmProvider llm.Provider) keyword.UseCase {
	return &usecase{
		l:           l,
		llmProvider: llmProvider,
		clock:       time.Now,
	}
}
