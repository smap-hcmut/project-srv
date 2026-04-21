package producer

import (
	"context"
	"encoding/json"
	"fmt"

	"project-srv/internal/project"
)

// PublishLifecycleEvent publishes one project lifecycle event to Kafka.
func (p *implProducer) PublishLifecycleEvent(ctx context.Context, event project.LifecycleEvent) error {
	if p.producer == nil {
		p.logger.Errorf(ctx, "project.delivery.kafka.producer.PublishLifecycleEvent: producer is nil")
		return nil
	}

	payload, err := json.Marshal(event)
	if err != nil {
		p.logger.Errorf(ctx, "project.delivery.kafka.producer.PublishLifecycleEvent: marshal event=%s err=%v", event.EventName, err)
		return fmt.Errorf("marshal lifecycle event: %w", err)
	}

	if err := p.producer.PublishWithContext(ctx, []byte(event.ProjectID), payload); err != nil {
		p.logger.Errorf(ctx, "project.delivery.kafka.producer.PublishLifecycleEvent: event=%s project_id=%s err=%v", event.EventName, event.ProjectID, err)
		return err
	}

	return nil
}
