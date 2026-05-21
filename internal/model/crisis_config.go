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

// KeywordsTrigger stores marketing issue keyword groups for project-level brand risk.
//
// [RESERVED] The current analysis crisis scorer does not use this field as a
// direct CRISIS_ALERT gate. The groups are still saved for issue taxonomy,
// presets, and future keyword-driven crisis scoring.
type KeywordsTrigger struct {
	Enabled bool           `json:"enabled"`
	Logic   string         `json:"logic"` // "OR" | "AND"
	Groups  []KeywordGroup `json:"groups"`
}

// --- Volume Trigger ---

// VolumeRule defines a single volume-based rule at a specific crisis level.
type VolumeRule struct {
	Level                  string  `json:"level"` // [WIRED] "WARNING" | "CRITICAL"
	ThresholdPercentGrowth float64 `json:"threshold_percent_growth"`
	ComparisonWindowHours  int     `json:"comparison_window_hours"` // [RESERVED] kept for future historical baseline scoring
	Baseline               string  `json:"baseline"`                // [RESERVED] e.g. "average_last_7_days"
}

// VolumeTrigger tunes the runtime issue_pressure crisis signal.
//
// [WIRED] rules[].threshold_percent_growth is mapped to issue_pressure
// thresholds derived from analysis top_issues_report. Metric, comparison
// window, and baseline are persisted but not consumed by the current scorer.
type VolumeTrigger struct {
	Enabled bool         `json:"enabled"`
	Metric  string       `json:"metric"` // [RESERVED] e.g. "mentions_count"
	Rules   []VolumeRule `json:"rules"`
}

// --- Sentiment Trigger ---

// SentimentRule defines a single sentiment-based rule.
type SentimentRule struct {
	Type                     string   `json:"type"`                                 // [WIRED for NEGATIVE_SPIKE] "NEGATIVE_SPIKE" | "ASPECT_NEGATIVE"
	ThresholdPercent         float64  `json:"threshold_percent,omitempty"`          // [WIRED for NEGATIVE_SPIKE]
	CriticalAspects          []string `json:"critical_aspects,omitempty"`           // [RESERVED]
	NegativeThresholdPercent float64  `json:"negative_threshold_percent,omitempty"` // [RESERVED]
}

// SentimentTrigger tunes the runtime sentiment_collapse proxy.
//
// [WIRED] NEGATIVE_SPIKE.threshold_percent changes the sentiment collapse
// threshold. MinSampleSize and ASPECT_NEGATIVE fields are reserved until
// mart-level sentiment/aspect joins are evaluated directly by analysis-srv.
type SentimentTrigger struct {
	Enabled       bool            `json:"enabled"`
	MinSampleSize int             `json:"min_sample_size"` // [RESERVED]
	Rules         []SentimentRule `json:"rules"`
}

// --- Influencer Trigger ---

// InfluencerRule defines a single influencer-based rule.
type InfluencerRule struct {
	Type              string `json:"type"`                         // [WIRED for VIRAL_NEGATIVE] "HIGH_REACH" | "VIRAL_NEGATIVE"
	MinFollowers      int    `json:"min_followers,omitempty"`      // [RESERVED]
	RequiredSentiment string `json:"required_sentiment,omitempty"` // [RESERVED]
	MinShares         int    `json:"min_shares,omitempty"`         // [RESERVED]
	MinComments       int    `json:"min_comments,omitempty"`       // [WIRED for VIRAL_NEGATIVE]
}

// InfluencerTrigger tunes the runtime controversy_spike signal.
//
// [WIRED] VIRAL_NEGATIVE.min_comments changes controversy thresholds derived
// from thread_controversy_report. HIGH_REACH, followers, shares, sentiment,
// and logic are persisted but not consumed by the current scorer.
type InfluencerTrigger struct {
	Enabled bool             `json:"enabled"`
	Logic   string           `json:"logic"` // [RESERVED] "OR" | "AND"
	Rules   []InfluencerRule `json:"rules"`
}

// --- Response Policy ---

// AdaptiveCrawlPolicy controls when analysis-srv asks project-srv/ingest-srv
// to accelerate or normalize crawl mode based on runtime crisis level.
type AdaptiveCrawlPolicy struct {
	Enabled         bool   `json:"enabled"`
	TriggerLevel    string `json:"trigger_level"`    // "WATCH" | "WARNING" | "CRITICAL"
	CooldownMinutes int    `json:"cooldown_minutes"` // minimum time between runtime changes
}

// NotificationPolicy controls when crisis assessments become user and ops
// notifications.
type NotificationPolicy struct {
	Enabled               bool   `json:"enabled"`
	TriggerLevel          string `json:"trigger_level"` // "WARNING" | "CRITICAL"
	RepeatCooldownMinutes int    `json:"repeat_cooldown_minutes"`
	OpsAlertOnCritical    bool   `json:"ops_alert_on_critical"`
}

// CrisisResponsePolicy defines runtime reactions after analysis computes a
// crisis level for a project.
type CrisisResponsePolicy struct {
	AdaptiveCrawl AdaptiveCrawlPolicy `json:"adaptive_crawl"`
	Notification  NotificationPolicy  `json:"notification"`
}

// DefaultCrisisResponsePolicy returns the default runtime reaction policy used
// when a project has not customized adaptive crawling or notifications.
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

// WithDefaults fills missing response policy fields with runtime-safe defaults.
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
