package usecase

import (
	"context"
	"time"

	"smap-project/config"
	"smap-project/internal/sampling"
	"smap-project/pkg/log"
)

type usecase struct {
	config   parsedConfig
	logger   log.Logger
	selector keywordSelector
}

// parsedConfig holds the parsed configuration with time.Duration values
type parsedConfig struct {
	Percentage          float64
	MinKeywords         int
	MaxKeywords         int
	KeywordTimeEstimate time.Duration
	DefaultStrategy     sampling.StrategyType
	EmergencyThreshold  time.Duration
	EmergencyKeywords   int
}

// defaultParsedConfig returns default configuration values
func defaultParsedConfig() parsedConfig {
	return parsedConfig{
		Percentage:          10.0,
		MinKeywords:         3,
		MaxKeywords:         5,
		KeywordTimeEstimate: 16 * time.Second,
		DefaultStrategy:     sampling.PercentageStrategy,
		EmergencyThreshold:  70 * time.Second,
		EmergencyKeywords:   3,
	}
}

// NewStrategy creates a new sampling strategy based on the configuration
// If config validation fails, it falls back to default settings
func NewStrategy(cfg config.DryRunSamplingConfig, logger log.Logger) sampling.UseCase {
	parsed, err := parseConfig(cfg)
	if err != nil {
		logger.Warnf(context.Background(), "Invalid sampling config, falling back to defaults: %v", err)
		parsed = defaultParsedConfig()
	}

	// Validate parsed config, fallback to defaults if invalid
	if err := validateParsedConfig(parsed); err != nil {
		logger.Warnf(context.Background(), "Invalid parsed config, falling back to defaults: %v", err)
		parsed = defaultParsedConfig()
	}

	return &usecase{
		config:   parsed,
		logger:   logger,
		selector: newRandomSelector(),
	}
}

// parseConfig converts config.DryRunSamplingConfig to parsedConfig
func parseConfig(cfg config.DryRunSamplingConfig) (parsedConfig, error) {
	keywordTimeEstimate, err := time.ParseDuration(cfg.KeywordTimeEstimate)
	if err != nil {
		return parsedConfig{}, err
	}

	emergencyThreshold, err := time.ParseDuration(cfg.EmergencyThreshold)
	if err != nil {
		return parsedConfig{}, err
	}

	return parsedConfig{
		Percentage:          cfg.Percentage,
		MinKeywords:         cfg.MinKeywords,
		MaxKeywords:         cfg.MaxKeywords,
		KeywordTimeEstimate: keywordTimeEstimate,
		DefaultStrategy:     sampling.StrategyType(cfg.DefaultStrategy),
		EmergencyThreshold:  emergencyThreshold,
		EmergencyKeywords:   cfg.EmergencyKeywords,
	}, nil
}

// validateParsedConfig validates the parsed configuration
func validateParsedConfig(cfg parsedConfig) error {
	if cfg.Percentage <= 0 || cfg.Percentage > 100 {
		return sampling.ErrInvalidPercentage
	}
	if cfg.MinKeywords < 1 {
		return sampling.ErrInvalidMinKeywords
	}
	if cfg.MaxKeywords < cfg.MinKeywords {
		return sampling.ErrInvalidMaxKeywords
	}
	if cfg.KeywordTimeEstimate <= 0 {
		return sampling.ErrInvalidTimeEstimate
	}
	if cfg.EmergencyKeywords < 1 {
		return sampling.ErrInvalidMinKeywords
	}

	switch cfg.DefaultStrategy {
	case sampling.PercentageStrategy, sampling.FixedStrategy, sampling.TieredStrategy:
		// Valid
	default:
		return sampling.ErrInvalidStrategy
	}

	return nil
}
