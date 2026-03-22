package usecase

import (
	"context"
	"errors"
	"project-srv/internal/model"
	"project-srv/internal/project"
	"time"

	"github.com/smap-hcmut/shared-libs/go/auth"
)

func (uc *implUseCase) publishLifecycleEvent(ctx context.Context, eventName project.LifecycleEventName, p model.Project) error {
	if uc.publisher == nil {
		return errors.New("project lifecycle event publisher is nil")
	}

	triggeredBy, success := auth.GetUserIDFromContext(ctx)
	if !success {
		uc.l.Warnf(ctx, "project.usecase.publishLifecycleEvent: failed to get user from context, using empty string as triggered_by")
	}

	return uc.publisher.PublishLifecycleEvent(ctx, project.LifecycleEvent{
		EventName:   eventName,
		ProjectID:   p.ID,
		CampaignID:  p.CampaignID,
		Status:      string(p.Status),
		TriggeredBy: triggeredBy,
		OccurredAt:  time.Now().UTC(),
	})
}
