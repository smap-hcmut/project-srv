package sampling

import (
	"time"
)

// Input contains all parameters needed for keyword sampling
// This is the public type used by other modules
type SampleInput struct {
	Keywords  []string `json:"keywords"`
	UserID    string   `json:"user_id"`
	ProjectID string   `json:"project_id,omitempty"`
}

// Output contains the results of keyword sampling
// This is the public type used by other modules
type SampleOutput struct {
	SelectedKeywords []string      `json:"selected_keywords"`
	TotalKeywords    int           `json:"total_keywords"`
	SamplingRatio    float64       `json:"sampling_ratio"`
	SelectionMethod  string        `json:"selection_method"`
	EstimatedTime    time.Duration `json:"estimated_time"`
}

// StrategyType defines the available sampling strategy types
type StrategyType string

const (
	PercentageStrategy StrategyType = "percentage"
	FixedStrategy      StrategyType = "fixed"
	TieredStrategy     StrategyType = "tiered"
)
