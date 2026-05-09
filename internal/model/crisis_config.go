package model

import (
	"encoding/json"
	"time"

	"project-srv/internal/sqlboiler"
)

// CrisisStatus represents the status of crisis detection.
type CrisisStatus string

const (
	CrisisStatusNormal   CrisisStatus = "NORMAL"
	CrisisStatusWatch    CrisisStatus = "WATCH"
	CrisisStatusWarning  CrisisStatus = "WARNING"
	CrisisStatusCritical CrisisStatus = "CRITICAL"
)

// CrisisRuntimeLevel includes the transient NONE level used by analysis runtime.
type CrisisRuntimeLevel string

const (
	CrisisRuntimeLevelNone     CrisisRuntimeLevel = "NONE"
	CrisisRuntimeLevelNormal   CrisisRuntimeLevel = "NORMAL"
	CrisisRuntimeLevelWatch    CrisisRuntimeLevel = "WATCH"
	CrisisRuntimeLevelWarning  CrisisRuntimeLevel = "WARNING"
	CrisisRuntimeLevelCritical CrisisRuntimeLevel = "CRITICAL"
)

// --- Keywords Trigger ---

// KeywordGroup represents a named group of keywords with a weight.
type KeywordGroup struct {
	Name     string   `json:"name"`
	Keywords []string `json:"keywords"`
	Weight   int      `json:"weight"`
}

// KeywordsTrigger defines crisis triggers based on keyword matching.
type KeywordsTrigger struct {
	Enabled bool           `json:"enabled"`
	Logic   string         `json:"logic"` // "OR" | "AND"
	Groups  []KeywordGroup `json:"groups"`
}

// --- Volume Trigger ---

// VolumeRule defines a single volume-based rule at a specific crisis level.
type VolumeRule struct {
	Level                  string  `json:"level"` // "WARNING" | "CRITICAL"
	ThresholdPercentGrowth float64 `json:"threshold_percent_growth"`
	ComparisonWindowHours  int     `json:"comparison_window_hours"`
	Baseline               string  `json:"baseline"` // e.g. "average_last_7_days"
}

// VolumeTrigger defines crisis triggers based on mention volume spikes.
type VolumeTrigger struct {
	Enabled bool         `json:"enabled"`
	Metric  string       `json:"metric"` // e.g. "mentions_count"
	Rules   []VolumeRule `json:"rules"`
}

// --- Sentiment Trigger ---

// SentimentRule defines a single sentiment-based rule.
type SentimentRule struct {
	Type                     string   `json:"type"` // "negative_ratio" | "absa_aspect_alert"
	ThresholdPercent         float64  `json:"threshold_percent,omitempty"`
	CriticalAspects          []string `json:"critical_aspects,omitempty"`
	NegativeThresholdPercent float64  `json:"negative_threshold_percent,omitempty"`
}

// SentimentTrigger defines crisis triggers based on negative sentiment.
type SentimentTrigger struct {
	Enabled       bool            `json:"enabled"`
	MinSampleSize int             `json:"min_sample_size"`
	Rules         []SentimentRule `json:"rules"`
}

// --- Influencer Trigger ---

// InfluencerRule defines a single influencer-based rule.
type InfluencerRule struct {
	Type              string `json:"type"` // "macro_influencer" | "viral_post"
	MinFollowers      int    `json:"min_followers,omitempty"`
	RequiredSentiment string `json:"required_sentiment,omitempty"`
	MinShares         int    `json:"min_shares,omitempty"`
	MinComments       int    `json:"min_comments,omitempty"`
}

// InfluencerTrigger defines crisis triggers based on author influence.
type InfluencerTrigger struct {
	Enabled bool             `json:"enabled"`
	Logic   string           `json:"logic"` // "OR" | "AND"
	Rules   []InfluencerRule `json:"rules"`
}

// --- Response Policy ---

type AdaptiveCrawlPolicy struct {
	Enabled         bool   `json:"enabled"`
	TriggerLevel    string `json:"trigger_level"`    // "WATCH" | "WARNING" | "CRITICAL"
	CooldownMinutes int    `json:"cooldown_minutes"` // minimum time between runtime changes
}

type NotificationPolicy struct {
	Enabled               bool   `json:"enabled"`
	TriggerLevel          string `json:"trigger_level"` // "WARNING" | "CRITICAL"
	RepeatCooldownMinutes int    `json:"repeat_cooldown_minutes"`
	OpsAlertOnCritical    bool   `json:"ops_alert_on_critical"`
}

type CrisisResponsePolicy struct {
	AdaptiveCrawl AdaptiveCrawlPolicy `json:"adaptive_crawl"`
	Notification  NotificationPolicy  `json:"notification"`
}

func DefaultCrisisResponsePolicy() CrisisResponsePolicy {
	return CrisisResponsePolicy{
		AdaptiveCrawl: AdaptiveCrawlPolicy{
			Enabled:         true,
			TriggerLevel:    string(CrisisRuntimeLevelWatch),
			CooldownMinutes: 30,
		},
		Notification: NotificationPolicy{
			Enabled:               true,
			TriggerLevel:          string(CrisisRuntimeLevelWarning),
			RepeatCooldownMinutes: 60,
			OpsAlertOnCritical:    true,
		},
	}
}

func (p CrisisResponsePolicy) WithDefaults() CrisisResponsePolicy {
	defaults := DefaultCrisisResponsePolicy()
	if p == (CrisisResponsePolicy{}) {
		return defaults
	}
	if p.AdaptiveCrawl.TriggerLevel == "" {
		p.AdaptiveCrawl.TriggerLevel = defaults.AdaptiveCrawl.TriggerLevel
	}
	if p.AdaptiveCrawl.CooldownMinutes <= 0 {
		p.AdaptiveCrawl.CooldownMinutes = defaults.AdaptiveCrawl.CooldownMinutes
	}
	if p.Notification.TriggerLevel == "" {
		p.Notification.TriggerLevel = defaults.Notification.TriggerLevel
	}
	if p.Notification.RepeatCooldownMinutes <= 0 {
		p.Notification.RepeatCooldownMinutes = defaults.Notification.RepeatCooldownMinutes
	}
	return p
}

// --- Crisis Config ---

// CrisisConfig represents the crisis detection configuration for a project.
type CrisisConfig struct {
	ProjectID         string               `json:"project_id"`
	Status            CrisisStatus         `json:"status"`
	KeywordsTrigger   KeywordsTrigger      `json:"keywords_trigger"`
	VolumeTrigger     VolumeTrigger        `json:"volume_trigger"`
	SentimentTrigger  SentimentTrigger     `json:"sentiment_trigger"`
	InfluencerTrigger InfluencerTrigger    `json:"influencer_trigger"`
	ResponsePolicy    CrisisResponsePolicy `json:"response_policy"`
	CreatedAt         time.Time            `json:"created_at"`
	UpdatedAt         time.Time            `json:"updated_at"`

	// Relations
	Project *Project `json:"project,omitempty"`
}

// NewCrisisConfigFromDB converts a sqlboiler ProjectsCrisisConfig to a domain CrisisConfig.
func NewCrisisConfigFromDB(db *sqlboiler.ProjectsCrisisConfig) *CrisisConfig {
	if db == nil {
		return nil
	}

	c := &CrisisConfig{
		ProjectID:      db.ProjectID,
		Status:         CrisisStatusNormal, // Default
		ResponsePolicy: DefaultCrisisResponsePolicy(),
	}

	if db.Status.Valid {
		c.Status = CrisisStatus(db.Status.Val)
	}

	if db.CreatedAt.Valid {
		c.CreatedAt = db.CreatedAt.Time
	}
	if db.UpdatedAt.Valid {
		c.UpdatedAt = db.UpdatedAt.Time
	}

	// Unmarshal JSONB fields
	if db.KeywordsRules.Valid {
		_ = json.Unmarshal(db.KeywordsRules.JSON, &c.KeywordsTrigger)
	}
	if db.VolumeRules.Valid {
		_ = json.Unmarshal(db.VolumeRules.JSON, &c.VolumeTrigger)
	}
	if db.SentimentRules.Valid {
		_ = json.Unmarshal(db.SentimentRules.JSON, &c.SentimentTrigger)
	}
	if db.InfluencerRules.Valid {
		_ = json.Unmarshal(db.InfluencerRules.JSON, &c.InfluencerTrigger)
	}
	if db.ResponsePolicy.Valid {
		_ = json.Unmarshal(db.ResponsePolicy.JSON, &c.ResponsePolicy)
		c.ResponsePolicy = c.ResponsePolicy.WithDefaults()
	}

	// Load relation if eagerly loaded
	if db.R != nil && db.R.GetProject() != nil {
		c.Project = NewProjectFromDB(db.R.GetProject())
	}

	return c
}
