package http

import (
	"project-srv/internal/crisis"
	"project-srv/internal/model"
)

// --- Request DTOs ---

// keywordGroupReq represents a keyword group in crisis detection
type keywordGroupReq struct {
	Name     string   `json:"name" example:"Pin bị lỗi"`           // Keyword group name
	Keywords []string `json:"keywords" example:"pin,sụt nhanh,hư"` // List of keywords
	Weight   int      `json:"weight" example:"10"`                 // Weight for scoring (1-100)
}

// keywordsTriggerReq represents keyword-based crisis trigger configuration
type keywordsTriggerReq struct {
	Enabled bool              `json:"enabled" example:"true"`             // Enable keyword trigger
	Logic   string            `json:"logic" example:"AND" enums:"AND,OR"` // Logic operator for groups
	Groups  []keywordGroupReq `json:"groups"`                             // Keyword groups
}

// volumeRuleReq represents volume anomaly detection rule
type volumeRuleReq struct {
	Level                  string  `json:"level" example:"CRITICAL" enums:"WARNING,CRITICAL"`                                 // Alert severity level
	ThresholdPercentGrowth float64 `json:"threshold_percent_growth" example:"150"`                                            // Percentage growth threshold (e.g., 150% means 2.5x baseline)
	ComparisonWindowHours  int     `json:"comparison_window_hours" example:"1"`                                               // Time window for comparison
	Baseline               string  `json:"baseline" example:"PREVIOUS_PERIOD" enums:"PREVIOUS_PERIOD,AVERAGE_7D,AVERAGE_30D"` // Baseline calculation method
}

// volumeTriggerReq represents volume-based crisis trigger configuration
type volumeTriggerReq struct {
	Enabled bool            `json:"enabled" example:"true"`                                      // Enable volume trigger
	Metric  string          `json:"metric" example:"MENTIONS" enums:"MENTIONS,ENGAGEMENT,REACH"` // Metric to monitor
	Rules   []volumeRuleReq `json:"rules"`                                                       // Volume rules
}

// sentimentRuleReq represents sentiment anomaly detection rule
type sentimentRuleReq struct {
	Type                     string   `json:"type" example:"NEGATIVE_SPIKE" enums:"NEGATIVE_SPIKE,ASPECT_NEGATIVE"` // Rule type
	ThresholdPercent         float64  `json:"threshold_percent,omitempty" example:"25"`                             // Negative sentiment threshold (for NEGATIVE_SPIKE)
	CriticalAspects          []string `json:"critical_aspects,omitempty" example:"Giá,Chất lượng,An toàn"`          // Critical aspects to monitor (for ASPECT_NEGATIVE)
	NegativeThresholdPercent float64  `json:"negative_threshold_percent,omitempty" example:"50"`                    // Negative threshold for critical aspects
}

// sentimentTriggerReq represents sentiment-based crisis trigger configuration
type sentimentTriggerReq struct {
	Enabled       bool               `json:"enabled" example:"true"`       // Enable sentiment trigger
	MinSampleSize int                `json:"min_sample_size" example:"10"` // Minimum sample size for reliable analysis
	Rules         []sentimentRuleReq `json:"rules"`                        // Sentiment rules
}

// influencerRuleReq represents influencer-based crisis trigger rule
type influencerRuleReq struct {
	Type              string `json:"type" example:"HIGH_REACH" enums:"HIGH_REACH,VIRAL_NEGATIVE"`              // Rule type
	MinFollowers      int    `json:"min_followers,omitempty" example:"100000"`                                 // Minimum follower count (for HIGH_REACH)
	RequiredSentiment string `json:"required_sentiment,omitempty" example:"NEGATIVE" enums:"NEGATIVE,NEUTRAL"` // Required sentiment (for HIGH_REACH)
	MinShares         int    `json:"min_shares,omitempty" example:"1000"`                                      // Minimum shares (for VIRAL_NEGATIVE)
	MinComments       int    `json:"min_comments,omitempty" example:"500"`                                     // Minimum comments (for VIRAL_NEGATIVE)
}

// influencerTriggerReq represents influencer-based crisis trigger configuration
type influencerTriggerReq struct {
	Enabled bool                `json:"enabled" example:"true"`            // Enable influencer trigger
	Logic   string              `json:"logic" example:"OR" enums:"AND,OR"` // Logic operator for rules
	Rules   []influencerRuleReq `json:"rules"`                             // Influencer rules
}

// upsertReq represents crisis config create/update request
type upsertReq struct {
	KeywordsTrigger   *keywordsTriggerReq   `json:"keywords_trigger,omitempty"`   // Keyword-based trigger (optional)
	VolumeTrigger     *volumeTriggerReq     `json:"volume_trigger,omitempty"`     // Volume-based trigger (optional)
	SentimentTrigger  *sentimentTriggerReq  `json:"sentiment_trigger,omitempty"`  // Sentiment-based trigger (optional)
	InfluencerTrigger *influencerTriggerReq `json:"influencer_trigger,omitempty"` // Influencer-based trigger (optional)
}

func (r upsertReq) validate() error {
	// At least one trigger must be provided
	if r.KeywordsTrigger == nil && r.VolumeTrigger == nil && r.SentimentTrigger == nil && r.InfluencerTrigger == nil {
		return errNoTrigger
	}

	// Validate keywords trigger
	if r.KeywordsTrigger != nil && r.KeywordsTrigger.Enabled {
		if len(r.KeywordsTrigger.Groups) == 0 {
			return errKeywordGroupsRequired
		}
		for _, g := range r.KeywordsTrigger.Groups {
			if g.Name == "" || len(g.Keywords) == 0 || g.Weight <= 0 {
				return errInvalidKeywordGroup
			}
		}
	}

	// Validate volume trigger
	if r.VolumeTrigger != nil && r.VolumeTrigger.Enabled {
		if len(r.VolumeTrigger.Rules) == 0 {
			return errVolumeRulesRequired
		}
		for _, rule := range r.VolumeTrigger.Rules {
			if rule.Level == "" || rule.ThresholdPercentGrowth <= 0 || rule.ComparisonWindowHours <= 0 {
				return errInvalidVolumeRule
			}
		}
	}

	// Validate sentiment trigger
	if r.SentimentTrigger != nil && r.SentimentTrigger.Enabled {
		if len(r.SentimentTrigger.Rules) == 0 {
			return errSentimentRulesRequired
		}
		for _, rule := range r.SentimentTrigger.Rules {
			if rule.Type == "" {
				return errInvalidSentimentRule
			}
		}
	}

	// Validate influencer trigger
	if r.InfluencerTrigger != nil && r.InfluencerTrigger.Enabled {
		if len(r.InfluencerTrigger.Rules) == 0 {
			return errInfluencerRulesRequired
		}
		for _, rule := range r.InfluencerTrigger.Rules {
			if rule.Type == "" {
				return errInvalidInfluencerRule
			}
		}
	}

	return nil
}

func (r upsertReq) toInput(projectID string) crisis.UpsertInput {
	input := crisis.UpsertInput{
		ProjectID: projectID,
	}

	if r.KeywordsTrigger != nil {
		groups := make([]model.KeywordGroup, len(r.KeywordsTrigger.Groups))
		for i, g := range r.KeywordsTrigger.Groups {
			groups[i] = model.KeywordGroup{
				Name:     g.Name,
				Keywords: g.Keywords,
				Weight:   g.Weight,
			}
		}
		input.KeywordsTrigger = &model.KeywordsTrigger{
			Enabled: r.KeywordsTrigger.Enabled,
			Logic:   r.KeywordsTrigger.Logic,
			Groups:  groups,
		}
	}

	if r.VolumeTrigger != nil {
		rules := make([]model.VolumeRule, len(r.VolumeTrigger.Rules))
		for i, rule := range r.VolumeTrigger.Rules {
			rules[i] = model.VolumeRule{
				Level:                  rule.Level,
				ThresholdPercentGrowth: rule.ThresholdPercentGrowth,
				ComparisonWindowHours:  rule.ComparisonWindowHours,
				Baseline:               rule.Baseline,
			}
		}
		input.VolumeTrigger = &model.VolumeTrigger{
			Enabled: r.VolumeTrigger.Enabled,
			Metric:  r.VolumeTrigger.Metric,
			Rules:   rules,
		}
	}

	if r.SentimentTrigger != nil {
		rules := make([]model.SentimentRule, len(r.SentimentTrigger.Rules))
		for i, rule := range r.SentimentTrigger.Rules {
			rules[i] = model.SentimentRule{
				Type:                     rule.Type,
				ThresholdPercent:         rule.ThresholdPercent,
				CriticalAspects:          rule.CriticalAspects,
				NegativeThresholdPercent: rule.NegativeThresholdPercent,
			}
		}
		input.SentimentTrigger = &model.SentimentTrigger{
			Enabled:       r.SentimentTrigger.Enabled,
			MinSampleSize: r.SentimentTrigger.MinSampleSize,
			Rules:         rules,
		}
	}

	if r.InfluencerTrigger != nil {
		rules := make([]model.InfluencerRule, len(r.InfluencerTrigger.Rules))
		for i, rule := range r.InfluencerTrigger.Rules {
			rules[i] = model.InfluencerRule{
				Type:              rule.Type,
				MinFollowers:      rule.MinFollowers,
				RequiredSentiment: rule.RequiredSentiment,
				MinShares:         rule.MinShares,
				MinComments:       rule.MinComments,
			}
		}
		input.InfluencerTrigger = &model.InfluencerTrigger{
			Enabled: r.InfluencerTrigger.Enabled,
			Logic:   r.InfluencerTrigger.Logic,
			Rules:   rules,
		}
	}

	return input
}

// --- Response DTOs ---

// keywordGroupResp represents keyword group in response
type keywordGroupResp struct {
	Name     string   `json:"name" example:"Pin bị lỗi"`
	Keywords []string `json:"keywords" example:"pin,sụt nhanh,hư"`
	Weight   int      `json:"weight" example:"10"`
}

// keywordsTriggerResp represents keyword trigger in response
type keywordsTriggerResp struct {
	Enabled bool               `json:"enabled" example:"true"`
	Logic   string             `json:"logic" example:"AND"`
	Groups  []keywordGroupResp `json:"groups"`
}

// volumeRuleResp represents volume rule in response
type volumeRuleResp struct {
	Level                  string  `json:"level" example:"CRITICAL"`
	ThresholdPercentGrowth float64 `json:"threshold_percent_growth" example:"150"`
	ComparisonWindowHours  int     `json:"comparison_window_hours" example:"1"`
	Baseline               string  `json:"baseline" example:"PREVIOUS_PERIOD"`
}

// volumeTriggerResp represents volume trigger in response
type volumeTriggerResp struct {
	Enabled bool             `json:"enabled" example:"true"`
	Metric  string           `json:"metric" example:"MENTIONS"`
	Rules   []volumeRuleResp `json:"rules"`
}

// sentimentRuleResp represents sentiment rule in response
type sentimentRuleResp struct {
	Type                     string   `json:"type" example:"NEGATIVE_SPIKE"`
	ThresholdPercent         float64  `json:"threshold_percent,omitempty" example:"25"`
	CriticalAspects          []string `json:"critical_aspects,omitempty" example:"Giá,Chất lượng"`
	NegativeThresholdPercent float64  `json:"negative_threshold_percent,omitempty" example:"50"`
}

// sentimentTriggerResp represents sentiment trigger in response
type sentimentTriggerResp struct {
	Enabled       bool                `json:"enabled" example:"true"`
	MinSampleSize int                 `json:"min_sample_size" example:"10"`
	Rules         []sentimentRuleResp `json:"rules"`
}

// influencerRuleResp represents influencer rule in response
type influencerRuleResp struct {
	Type              string `json:"type" example:"HIGH_REACH"`
	MinFollowers      int    `json:"min_followers,omitempty" example:"100000"`
	RequiredSentiment string `json:"required_sentiment,omitempty" example:"NEGATIVE"`
	MinShares         int    `json:"min_shares,omitempty" example:"1000"`
	MinComments       int    `json:"min_comments,omitempty" example:"500"`
}

// influencerTriggerResp represents influencer trigger in response
type influencerTriggerResp struct {
	Enabled bool                 `json:"enabled" example:"true"`
	Logic   string               `json:"logic" example:"OR"`
	Rules   []influencerRuleResp `json:"rules"`
}

// crisisConfigResp represents complete crisis config in API responses
type crisisConfigResp struct {
	ProjectID         string                `json:"project_id" example:"550e8400-e29b-41d4-a716-446655440002"`  // Project UUID
	Status            string                `json:"status" example:"ACTIVE" enums:"ACTIVE,INACTIVE,MONITORING"` // Config status
	KeywordsTrigger   keywordsTriggerResp   `json:"keywords_trigger"`                                           // Keywords trigger config
	VolumeTrigger     volumeTriggerResp     `json:"volume_trigger"`                                             // Volume trigger config
	SentimentTrigger  sentimentTriggerResp  `json:"sentiment_trigger"`                                          // Sentiment trigger config
	InfluencerTrigger influencerTriggerResp `json:"influencer_trigger"`                                         // Influencer trigger config
	CreatedAt         string                `json:"created_at" example:"2026-02-18T00:00:00Z"`                  // Creation timestamp
	UpdatedAt         string                `json:"updated_at" example:"2026-02-18T00:00:00Z"`                  // Last update timestamp
}

// upsertResp wraps crisis config upsert response
type upsertResp struct {
	CrisisConfig crisisConfigResp `json:"crisis_config"` // Upserted crisis config
}

// detailResp wraps crisis config detail response
type detailResp struct {
	CrisisConfig crisisConfigResp `json:"crisis_config"` // Crisis config details
}

// --- Response Mappers (receiver on handler) ---

func (h *handler) newUpsertResp(o crisis.UpsertOutput) upsertResp {
	return upsertResp{
		CrisisConfig: toCrisisConfigResp(o.CrisisConfig),
	}
}

func (h *handler) newDetailResp(o crisis.DetailOutput) detailResp {
	return detailResp{
		CrisisConfig: toCrisisConfigResp(o.CrisisConfig),
	}
}

// --- Internal Mapper ---

func toCrisisConfigResp(c model.CrisisConfig) crisisConfigResp {
	// Keywords
	kwGroups := make([]keywordGroupResp, len(c.KeywordsTrigger.Groups))
	for i, g := range c.KeywordsTrigger.Groups {
		kwGroups[i] = keywordGroupResp{
			Name:     g.Name,
			Keywords: g.Keywords,
			Weight:   g.Weight,
		}
	}

	// Volume
	volRules := make([]volumeRuleResp, len(c.VolumeTrigger.Rules))
	for i, r := range c.VolumeTrigger.Rules {
		volRules[i] = volumeRuleResp{
			Level:                  r.Level,
			ThresholdPercentGrowth: r.ThresholdPercentGrowth,
			ComparisonWindowHours:  r.ComparisonWindowHours,
			Baseline:               r.Baseline,
		}
	}

	// Sentiment
	sentRules := make([]sentimentRuleResp, len(c.SentimentTrigger.Rules))
	for i, r := range c.SentimentTrigger.Rules {
		sentRules[i] = sentimentRuleResp{
			Type:                     r.Type,
			ThresholdPercent:         r.ThresholdPercent,
			CriticalAspects:          r.CriticalAspects,
			NegativeThresholdPercent: r.NegativeThresholdPercent,
		}
	}

	// Influencer
	infRules := make([]influencerRuleResp, len(c.InfluencerTrigger.Rules))
	for i, r := range c.InfluencerTrigger.Rules {
		infRules[i] = influencerRuleResp{
			Type:              r.Type,
			MinFollowers:      r.MinFollowers,
			RequiredSentiment: r.RequiredSentiment,
			MinShares:         r.MinShares,
			MinComments:       r.MinComments,
		}
	}

	return crisisConfigResp{
		ProjectID: c.ProjectID,
		Status:    string(c.Status),
		KeywordsTrigger: keywordsTriggerResp{
			Enabled: c.KeywordsTrigger.Enabled,
			Logic:   c.KeywordsTrigger.Logic,
			Groups:  kwGroups,
		},
		VolumeTrigger: volumeTriggerResp{
			Enabled: c.VolumeTrigger.Enabled,
			Metric:  c.VolumeTrigger.Metric,
			Rules:   volRules,
		},
		SentimentTrigger: sentimentTriggerResp{
			Enabled:       c.SentimentTrigger.Enabled,
			MinSampleSize: c.SentimentTrigger.MinSampleSize,
			Rules:         sentRules,
		},
		InfluencerTrigger: influencerTriggerResp{
			Enabled: c.InfluencerTrigger.Enabled,
			Logic:   c.InfluencerTrigger.Logic,
			Rules:   infRules,
		},
		CreatedAt: c.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: c.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}
