package producer

import (
	context "context"
	"encoding/json"

	rabb "smap-project/internal/project/delivery/rabbitmq"
	"smap-project/pkg/rabbitmq"
)

// PublishDryRunTask publishes a dry-run task to the collector.inbound exchange.
func (p implProducer) PublishDryRunTask(ctx context.Context, msg rabb.DryRunCrawlRequest) error {
	body, err := json.Marshal(msg)
	if err != nil {
		p.l.Errorf(ctx, "producer.PublishDryRunTask.Marshal: %v", err)
		return err
	}

	err = p.dryRunWriter.Publish(ctx, rabbitmq.PublishArgs{
		Exchange:   rabb.CollectorInboundExchangeName,
		RoutingKey: rabb.DryRunKeywordRoutingKey,
		Mandatory:  false,
		Immediate:  false,
		Msg: rabbitmq.Publishing{
			ContentType: "application/json",
			Body:        body,
		},
	})

	if err != nil {
		p.l.Errorf(ctx, "producer.PublishDryRunTask.Publish: %v", err)
		return err
	}

	p.l.Infof(ctx, "Published dry-run task to RabbitMQ: job_id=%s", msg.JobID)
	return nil
}

// PublishProjectCreated publishes a project.created event to smap.events exchange.
func (p implProducer) PublishProjectCreated(ctx context.Context, event rabb.ProjectCreatedEvent) error {
	body, err := json.Marshal(event)
	if err != nil {
		p.l.Errorf(ctx, "producer.PublishProjectCreated.Marshal: %v", err)
		return err
	}

	err = p.smapEventsWriter.Publish(ctx, rabbitmq.PublishArgs{
		Exchange:   rabb.SMAPEventsExchangeName,
		RoutingKey: rabb.ProjectCreatedRoutingKey,
		Mandatory:  false,
		Immediate:  false,
		Msg: rabbitmq.Publishing{
			ContentType: "application/json",
			Body:        body,
		},
	})

	if err != nil {
		p.l.Errorf(ctx, "producer.PublishProjectCreated.Publish: %v", err)
		return err
	}

	return nil
}
