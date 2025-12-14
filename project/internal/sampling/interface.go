package sampling

import (
	"context"
)

//go:generate mockery --name UseCase
type UseCase interface {
	Sample(ctx context.Context, input SampleInput) (SampleOutput, error)
}
