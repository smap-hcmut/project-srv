package producer

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"project-srv/internal/project"

	"github.com/smap-hcmut/shared-libs/go/kafka"
	"github.com/smap-hcmut/shared-libs/go/log"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func newTestProducer(t *testing.T) (*implProducer, *kafka.MockIProducer) {
	t.Helper()
	l := log.NewZapLogger(log.ZapConfig{
		Level:        log.LevelFatal,
		Mode:         log.ModeProduction,
		Encoding:     log.EncodingJSON,
		ColorEnabled: false,
	})
	kafkaProducer := kafka.NewMockIProducer(t)
	return New(l, kafkaProducer).(*implProducer), kafkaProducer
}

func TestNew(t *testing.T) {
	tcs := map[string]struct {
		input  kafka.IProducer
		mock   struct{}
		output bool
		err    error
	}{
		"success": {
			input:  kafka.NewMockIProducer(t),
			output: true,
		},
		"nil_producer": {
			output: true,
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			l := log.NewZapLogger(log.ZapConfig{Level: log.LevelFatal, Mode: log.ModeProduction, Encoding: log.EncodingJSON})

			p := New(l, tc.input)

			require.Equal(t, tc.output, p != nil)
		})
	}
}

func TestPublishLifecycleEvent(t *testing.T) {
	ctx := context.Background()
	event := project.LifecycleEvent{
		EventName:  project.ProjectLifecycleEventActivated,
		ProjectID:  "project-1",
		CampaignID: "campaign-1",
		Status:     "ACTIVE",
	}

	type mockPublish struct {
		isCalled bool
		err      error
	}
	type mockData struct {
		publish     mockPublish
		nilProducer bool
	}

	tcs := map[string]struct {
		input  project.LifecycleEvent
		mock   mockData
		output struct{}
		err    error
	}{
		"success": {
			input: event,
			mock:  mockData{publish: mockPublish{isCalled: true}},
		},
		"nil_producer": {
			input: event,
			mock:  mockData{nilProducer: true},
		},
		"publish_error": {
			input: event,
			mock:  mockData{publish: mockPublish{isCalled: true, err: errors.New("kafka error")}},
			err:   errors.New("kafka error"),
		},
		"marshal_error": {
			input: event,
			err:   errors.New("marshal error"),
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			p, kafkaProducer := newTestProducer(t)
			if tc.mock.nilProducer {
				p.producer = nil
			}
			if name == "marshal_error" {
				original := marshalLifecycleEvent
				marshalLifecycleEvent = func(any) ([]byte, error) {
					return nil, tc.err
				}
				t.Cleanup(func() { marshalLifecycleEvent = original })
			}
			if tc.mock.publish.isCalled {
				kafkaProducer.EXPECT().PublishWithContext(ctx, []byte(tc.input.ProjectID), mock.MatchedBy(func(payload []byte) bool {
					var got project.LifecycleEvent
					if err := json.Unmarshal(payload, &got); err != nil {
						return false
					}
					return got.EventName == tc.input.EventName &&
						got.ProjectID == tc.input.ProjectID &&
						got.CampaignID == tc.input.CampaignID &&
						got.Status == tc.input.Status
				})).Return(tc.mock.publish.err)
			}

			err := p.PublishLifecycleEvent(ctx, tc.input)

			if tc.err != nil {
				require.ErrorContains(t, err, tc.err.Error())
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestMockProducerGeneratedMethods(t *testing.T) {
	ctx := context.Background()
	event := project.LifecycleEvent{ProjectID: "project-1"}

	tcs := map[string]struct {
		input project.LifecycleEvent
		mock  struct {
			useRunAndReturn bool
			expectRun       bool
		}
		output error
		err    error
	}{
		"return": {
			input: event,
		},
		"run": {
			input: event,
			mock: struct {
				useRunAndReturn bool
				expectRun       bool
			}{expectRun: true},
		},
		"run_and_return": {
			input: event,
			mock: struct {
				useRunAndReturn bool
				expectRun       bool
			}{useRunAndReturn: true},
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			mockProducer := NewMockProducer(t)
			expecter := mockProducer.EXPECT()
			call := expecter.PublishLifecycleEvent(ctx, tc.input)
			ran := false
			if tc.mock.useRunAndReturn {
				call.RunAndReturn(func(context.Context, project.LifecycleEvent) error {
					return tc.err
				})
			} else {
				if tc.mock.expectRun {
					call.Run(func(context.Context, project.LifecycleEvent) {
						ran = true
					})
				}
				call.Return(tc.err)
			}

			err := mockProducer.PublishLifecycleEvent(ctx, tc.input)

			require.NoError(t, err)
			require.Equal(t, tc.mock.expectRun, ran)
		})
	}
}
