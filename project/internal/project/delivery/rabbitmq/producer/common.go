package producer

import (
	"fmt"

	rabb "smap-project/internal/project/delivery/rabbitmq"
	rmqPkg "smap-project/pkg/rabbitmq"
)

// Run runs the producer
func (p *implProducer) Run() (err error) {
	// Initialize dry-run writer (legacy collector.inbound exchange)
	p.dryRunWriter, err = p.getWriter(rabb.CollectorInboundExchange)
	if err != nil {
		fmt.Println("Error when getting dry-run writer")
		return
	}

	// Initialize smap.events writer (new standardized exchange)
	p.smapEventsWriter, err = p.getWriter(rabb.SMAPEventsExchange)
	if err != nil {
		fmt.Println("Error when getting smap.events writer")
		return
	}

	return
}

// Close closes the producer
func (p *implProducer) Close() {
	if p.dryRunWriter != nil {
		p.dryRunWriter.Close()
	}
	if p.smapEventsWriter != nil {
		p.smapEventsWriter.Close()
	}
}

func (p implProducer) getWriter(exchange rmqPkg.ExchangeArgs) (*rmqPkg.Channel, error) {
	ch, err := p.conn.Channel()
	if err != nil {
		fmt.Println("Error when getting channel")
		return nil, err
	}

	err = ch.ExchangeDeclare(exchange)
	if err != nil {
		return nil, err
	}

	return ch, nil
}
