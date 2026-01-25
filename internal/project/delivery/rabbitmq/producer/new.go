package producer

import (
	context "context"

	"smap-project/internal/project/delivery/rabbitmq"
	"smap-project/pkg/log"
	rmq "smap-project/pkg/rabbitmq"
)

// Producer is an interface that represents a producer
//
//go:generate mockery --name=Producer
type Producer interface {
	PublishDryRunTask(ctx context.Context, msg rabbitmq.DryRunCrawlRequest) error
	// PublishProjectCreated publishes a project.created event to smap.events exchange
	PublishProjectCreated(ctx context.Context, event rabbitmq.ProjectCreatedEvent) error
	// Run runs the producer
	Run() error
	// Close closes the producer
	Close()
}

type implProducer struct {
	l                log.Logger
	conn             rmq.Connection
	dryRunWriter     *rmq.Channel
	smapEventsWriter *rmq.Channel
}

// New creates a new producer
func New(l log.Logger, conn rmq.Connection) Producer {
	return &implProducer{
		l:    l,
		conn: conn,
	}
}
