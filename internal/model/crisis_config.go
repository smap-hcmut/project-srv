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
	CrisisStatusWarning  CrisisStatus = "WARNING"
	CrisisStatusCritical CrisisStatus = "CRITICAL"
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

// --- Crisis Config ---

// CrisisConfig represents the crisis detection configuration for a project.
type CrisisConfig struct {
	ProjectID         string            `json:"project_id"`
	Status            CrisisStatus      `json:"status"`
	KeywordsTrigger   KeywordsTrigger   `json:"keywords_trigger"`
	VolumeTrigger     VolumeTrigger     `json:"volume_trigger"`
	SentimentTrigger  SentimentTrigger  `json:"sentiment_trigger"`
	InfluencerTrigger InfluencerTrigger `json:"influencer_trigger"`
	CreatedAt         time.Time         `json:"created_at"`
	UpdatedAt         time.Time         `json:"updated_at"`

	// Relations
	Project *Project `json:"project,omitempty"`
}

// NewCrisisConfigFromDB converts a sqlboiler ProjectsCrisisConfig to a domain CrisisConfig.
func NewCrisisConfigFromDB(db *sqlboiler.ProjectsCrisisConfig) *CrisisConfig {
	if db == nil {
		return nil
	}

	c := &CrisisConfig{
		ProjectID: db.ProjectID,
		Status:    CrisisStatusNormal, // Default
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

	// Load relation if eagerly loaded
	if db.R != nil && db.R.GetProject() != nil {
		c.Project = NewProjectFromDB(db.R.GetProject())
	}

	return c
}
