package http

import (
	"errors"
	"testing"
	"time"

	"project-srv/internal/crisis"
	"project-srv/internal/model"

	"github.com/stretchr/testify/require"
)

func TestMapError(t *testing.T) {
	h, _ := newTestHandler(t)

	tcs := map[string]struct {
		input  error
		mock   struct{}
		output error
		err    error
	}{
		"not_found":       {input: crisis.ErrNotFound, output: errNotFound},
		"project_invalid": {input: crisis.ErrProjectInvalid, output: errProjectInvalid},
		"upsert_failed":   {input: crisis.ErrUpsertFailed, output: errUpsertFailed},
		"delete_failed":   {input: crisis.ErrDeleteFailed, output: errDeleteFailed},
		"invalid_status":  {input: crisis.ErrInvalidStatus, output: errInvalidCrisisStatus},
		"apply_failed":    {input: crisis.ErrApplyFailed, output: errApplyFailed},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			output := h.mapError(tc.input)

			require.Equal(t, tc.output, output)
		})
	}
}

func TestMapErrorDefault(t *testing.T) {
	h, _ := newTestHandler(t)

	tcs := map[string]struct {
		input  error
		mock   struct{}
		output struct{}
		err    error
	}{
		"panic_unknown_error": {input: errors.New("unknown")},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			require.Panics(t, func() {
				_ = h.mapError(tc.input)
			})
		})
	}
}

func TestUpsertReqValidate(t *testing.T) {
	validKeyword := keywordsTriggerReq{Enabled: true, Logic: "AND", Groups: []keywordGroupReq{{Name: "Pin", Keywords: []string{"pin"}, Weight: 10}}}
	validVolume := volumeTriggerReq{Enabled: true, Metric: "MENTIONS", Rules: []volumeRuleReq{{Level: "CRITICAL", ThresholdPercentGrowth: 150, ComparisonWindowHours: 1, Baseline: "PREVIOUS_PERIOD"}}}
	validSentiment := sentimentTriggerReq{Enabled: true, MinSampleSize: 10, Rules: []sentimentRuleReq{{Type: "NEGATIVE_SPIKE"}}}
	validInfluencer := influencerTriggerReq{Enabled: true, Logic: "OR", Rules: []influencerRuleReq{{Type: "HIGH_REACH"}}}

	tcs := map[string]struct {
		input  upsertReq
		mock   struct{}
		output struct{}
		err    error
	}{
		"success_all": {
			input: upsertReq{KeywordsTrigger: &validKeyword, VolumeTrigger: &validVolume, SentimentTrigger: &validSentiment, InfluencerTrigger: &validInfluencer},
		},
		"success_with_status": {
			input: upsertReq{Status: " warning ", KeywordsTrigger: &validKeyword},
		},
		"invalid_status": {
			input: upsertReq{Status: "BAD", KeywordsTrigger: &validKeyword},
			err:   errInvalidCrisisStatus,
		},
		"no_trigger": {
			err: errNoTrigger,
		},
		"keyword_groups_required": {
			input: upsertReq{KeywordsTrigger: &keywordsTriggerReq{Enabled: true}},
			err:   errKeywordGroupsRequired,
		},
		"invalid_keyword_group": {
			input: upsertReq{KeywordsTrigger: &keywordsTriggerReq{Enabled: true, Groups: []keywordGroupReq{{Name: "", Weight: 0}}}},
			err:   errInvalidKeywordGroup,
		},
		"volume_rules_required": {
			input: upsertReq{VolumeTrigger: &volumeTriggerReq{Enabled: true}},
			err:   errVolumeRulesRequired,
		},
		"invalid_volume_rule": {
			input: upsertReq{VolumeTrigger: &volumeTriggerReq{Enabled: true, Rules: []volumeRuleReq{{Level: "CRITICAL"}}}},
			err:   errInvalidVolumeRule,
		},
		"sentiment_rules_required": {
			input: upsertReq{SentimentTrigger: &sentimentTriggerReq{Enabled: true}},
			err:   errSentimentRulesRequired,
		},
		"invalid_sentiment_rule": {
			input: upsertReq{SentimentTrigger: &sentimentTriggerReq{Enabled: true, Rules: []sentimentRuleReq{{}}}},
			err:   errInvalidSentimentRule,
		},
		"influencer_rules_required": {
			input: upsertReq{InfluencerTrigger: &influencerTriggerReq{Enabled: true}},
			err:   errInfluencerRulesRequired,
		},
		"invalid_influencer_rule": {
			input: upsertReq{InfluencerTrigger: &influencerTriggerReq{Enabled: true, Rules: []influencerRuleReq{{}}}},
			err:   errInvalidInfluencerRule,
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			err := tc.input.validate()

			if tc.err != nil {
				require.Equal(t, tc.err, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestUpsertReqToInput(t *testing.T) {
	tcs := map[string]struct {
		input  upsertReq
		mock   struct{}
		output crisis.UpsertInput
		err    error
	}{
		"all_triggers": {
			input: upsertReq{
				Status:            "warning",
				KeywordsTrigger:   &keywordsTriggerReq{Enabled: true, Logic: "AND", Groups: []keywordGroupReq{{Name: "Pin", Keywords: []string{"pin"}, Weight: 10}}},
				VolumeTrigger:     &volumeTriggerReq{Enabled: true, Metric: "MENTIONS", Rules: []volumeRuleReq{{Level: "CRITICAL", ThresholdPercentGrowth: 150, ComparisonWindowHours: 1, Baseline: "PREVIOUS_PERIOD"}}},
				SentimentTrigger:  &sentimentTriggerReq{Enabled: true, MinSampleSize: 10, Rules: []sentimentRuleReq{{Type: "NEGATIVE_SPIKE", ThresholdPercent: 25, CriticalAspects: []string{"quality"}, NegativeThresholdPercent: 50}}},
				InfluencerTrigger: &influencerTriggerReq{Enabled: true, Logic: "OR", Rules: []influencerRuleReq{{Type: "HIGH_REACH", MinFollowers: 1000, RequiredSentiment: "NEGATIVE", MinShares: 10, MinComments: 20}}},
			},
			output: crisis.UpsertInput{ProjectID: "project-1"},
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			output := tc.input.toInput("project-1")

			require.Equal(t, tc.output.ProjectID, output.ProjectID)
			require.NotNil(t, output.Status)
			require.NotNil(t, output.KeywordsTrigger)
			require.NotNil(t, output.VolumeTrigger)
			require.NotNil(t, output.SentimentTrigger)
			require.NotNil(t, output.InfluencerTrigger)
		})
	}
}

func TestApplyRuntimeReqValidate(t *testing.T) {
	tcs := map[string]struct {
		input  applyRuntimeReq
		mock   struct{}
		output struct{}
		err    error
	}{
		"empty_status": {},
		"normal":       {input: applyRuntimeReq{Status: " normal "}},
		"warning":      {input: applyRuntimeReq{Status: "WARNING"}},
		"critical":     {input: applyRuntimeReq{Status: "CRITICAL"}},
		"invalid":      {input: applyRuntimeReq{Status: "BAD"}, err: errInvalidCrisisStatus},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			err := tc.input.validate()

			if tc.err != nil {
				require.Equal(t, tc.err, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestApplyRuntimeReqToInput(t *testing.T) {
	status := model.CrisisStatusCritical
	tcs := map[string]struct {
		input  applyRuntimeReq
		mock   struct{}
		output crisis.ApplyRuntimeInput
		err    error
	}{
		"with_status": {
			input:  applyRuntimeReq{Status: " critical ", Reason: " reason ", EventRef: " event-1 "},
			output: crisis.ApplyRuntimeInput{ProjectID: "project-1", Status: &status, Reason: "reason", EventRef: "event-1"},
		},
		"without_status": {
			input:  applyRuntimeReq{Reason: " reason "},
			output: crisis.ApplyRuntimeInput{ProjectID: "project-1", Reason: "reason"},
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			output := tc.input.toInput(" project-1 ")

			require.Equal(t, tc.output, output)
		})
	}
}

func TestNewApplyRuntimeResp(t *testing.T) {
	h, _ := newTestHandler(t)
	tcs := map[string]struct {
		input  crisis.ApplyRuntimeOutput
		mock   struct{}
		output applyRuntimeResp
		err    error
	}{
		"success": {
			input:  crisis.ApplyRuntimeOutput{ProjectID: "project-1", CrisisStatus: model.CrisisStatusCritical, AppliedCrawlMode: "CRISIS", AffectedDataSourceCount: 2},
			output: applyRuntimeResp{ProjectID: "project-1", CrisisStatus: "CRITICAL", AppliedCrawlMode: "CRISIS", AffectedDataSourceCount: 2},
		},
	}
	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, tc.output, h.newApplyRuntimeResp(tc.input))
		})
	}
}

func TestToCrisisConfigResp(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	tcs := map[string]struct {
		input  model.CrisisConfig
		mock   struct{}
		output crisisConfigResp
		err    error
	}{
		"all_triggers": {
			input: model.CrisisConfig{
				ProjectID: "project-1",
				Status:    model.CrisisStatusNormal,
				KeywordsTrigger: model.KeywordsTrigger{
					Enabled: true,
					Logic:   "AND",
					Groups:  []model.KeywordGroup{{Name: "Pin", Keywords: []string{"pin"}, Weight: 10}},
				},
				VolumeTrigger: model.VolumeTrigger{
					Enabled: true,
					Metric:  "MENTIONS",
					Rules:   []model.VolumeRule{{Level: "CRITICAL", ThresholdPercentGrowth: 150, ComparisonWindowHours: 1, Baseline: "PREVIOUS_PERIOD"}},
				},
				SentimentTrigger: model.SentimentTrigger{
					Enabled:       true,
					MinSampleSize: 10,
					Rules:         []model.SentimentRule{{Type: "NEGATIVE_SPIKE", ThresholdPercent: 25, CriticalAspects: []string{"quality"}, NegativeThresholdPercent: 50}},
				},
				InfluencerTrigger: model.InfluencerTrigger{
					Enabled: true,
					Logic:   "OR",
					Rules:   []model.InfluencerRule{{Type: "HIGH_REACH", MinFollowers: 1000, RequiredSentiment: "NEGATIVE", MinShares: 10, MinComments: 20}},
				},
				CreatedAt: now,
				UpdatedAt: now,
			},
			output: crisisConfigResp{
				ProjectID: "project-1",
				Status:    "NORMAL",
				KeywordsTrigger: keywordsTriggerResp{
					Enabled: true,
					Logic:   "AND",
					Groups:  []keywordGroupResp{{Name: "Pin", Keywords: []string{"pin"}, Weight: 10}},
				},
				VolumeTrigger: volumeTriggerResp{
					Enabled: true,
					Metric:  "MENTIONS",
					Rules:   []volumeRuleResp{{Level: "CRITICAL", ThresholdPercentGrowth: 150, ComparisonWindowHours: 1, Baseline: "PREVIOUS_PERIOD"}},
				},
				SentimentTrigger: sentimentTriggerResp{
					Enabled:       true,
					MinSampleSize: 10,
					Rules:         []sentimentRuleResp{{Type: "NEGATIVE_SPIKE", ThresholdPercent: 25, CriticalAspects: []string{"quality"}, NegativeThresholdPercent: 50}},
				},
				InfluencerTrigger: influencerTriggerResp{
					Enabled: true,
					Logic:   "OR",
					Rules:   []influencerRuleResp{{Type: "HIGH_REACH", MinFollowers: 1000, RequiredSentiment: "NEGATIVE", MinShares: 10, MinComments: 20}},
				},
				CreatedAt: "2026-01-01T00:00:00Z",
				UpdatedAt: "2026-01-01T00:00:00Z",
			},
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			output := toCrisisConfigResp(tc.input)

			require.Equal(t, tc.output, output)
		})
	}
}
