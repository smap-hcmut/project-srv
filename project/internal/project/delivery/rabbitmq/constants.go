package rabbitmq

import "smap-project/pkg/rabbitmq"

const (
	// Legacy exchange for dry-run events (will be deprecated)
	CollectorInboundExchangeName = "collector.inbound"
	DryRunKeywordRoutingKey      = "crawler.dryrun_keyword"

	// New standardized SMAP events exchange
	SMAPEventsExchangeName = "smap.events"

	// Routing keys for SMAP events
	ProjectCreatedRoutingKey = "project.created"
)

var (
	// CollectorInboundExchange is the legacy exchange for collector events
	CollectorInboundExchange = rabbitmq.ExchangeArgs{
		Name:       CollectorInboundExchangeName,
		Type:       "topic",
		Durable:    true,
		AutoDelete: false,
		Internal:   false,
		NoWait:     false,
	}

	// SMAPEventsExchange is the new standardized exchange for all SMAP events
	SMAPEventsExchange = rabbitmq.ExchangeArgs{
		Name:       SMAPEventsExchangeName,
		Type:       "topic",
		Durable:    true,
		AutoDelete: false,
		Internal:   false,
		NoWait:     false,
	}
)
