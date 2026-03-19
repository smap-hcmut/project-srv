package producer

import (
	"context"
	"project-srv/internal/project"

	"github.com/smap-hcmut/shared-libs/go/kafka"
	"github.com/smap-hcmut/shared-libs/go/log"
)

// Producer publishes project lifecycle events.
type Producer interface {
	PublishLifecycleEvent(ctx context.Context, event project.LifecycleEvent) error
}

type implProducer struct {
	logger   log.Logger
	producer kafka.IProducer
}

var _ Producer = (*implProducer)(nil)

// New creates a new Kafka lifecycle event publisher.
func New(logger log.Logger, producer kafka.IProducer) Producer {
	return &implProducer{
		logger:   logger,
		producer: producer,
	}
}
