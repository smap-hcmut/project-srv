package usecase

import (
	"context"
	"time"

	"smap-project/internal/sampling"
)

// Sample implements the sampling.UseCase interface
func (uc *usecase) Sample(ctx context.Context, input sampling.SampleInput) (sampling.SampleOutput, error) {
	// Validate input
	if err := uc.validateInput(input); err != nil {
		return sampling.SampleOutput{}, err
	}

	if len(input.Keywords) == 0 {
		return sampling.SampleOutput{
			SelectedKeywords: []string{},
			TotalKeywords:    0,
			SamplingRatio:    0,
			SelectionMethod:  "empty",
			EstimatedTime:    0,
		}, nil
	}

	// Calculate target count
	targetCount := uc.calculateTargetCount(len(input.Keywords))

	// Apply emergency fallback if needed
	if uc.isEmergencyFallbackNeeded(targetCount) {
		uc.logger.Warnf(ctx, "Emergency fallback applied for user %s: estimated time %v exceeds threshold %v",
			input.UserID, uc.estimateExecutionTime(targetCount), uc.config.EmergencyThreshold)
		targetCount = uc.config.EmergencyKeywords
		if targetCount > len(input.Keywords) {
			targetCount = len(input.Keywords)
		}
	}

	// Select keywords using the configured selector
	selectedKeywords, selectionMethod := uc.selector.Select(input.Keywords, targetCount)

	// Calculate sampling ratio
	samplingRatio := float64(len(selectedKeywords)) / float64(len(input.Keywords))

	// Estimate execution time
	estimatedTime := uc.estimateExecutionTime(len(selectedKeywords))

	// Log sampling decision
	uc.logger.Infof(ctx, "Keyword sampling completed for user %s: %d/%d keywords selected (%.1f%%), method=%s, estimated_time=%v",
		input.UserID, len(selectedKeywords), len(input.Keywords),
		samplingRatio*100, selectionMethod, estimatedTime)

	return sampling.SampleOutput{
		SelectedKeywords: selectedKeywords,
		TotalKeywords:    len(input.Keywords),
		SamplingRatio:    samplingRatio,
		SelectionMethod:  selectionMethod,
		EstimatedTime:    estimatedTime,
	}, nil
}

// Private helper methods

func (uc *usecase) validateInput(input sampling.SampleInput) error {
	if input.UserID == "" {
		return sampling.ErrMissingUserID
	}

	if input.Keywords == nil {
		return sampling.ErrNilKeywords
	}

	return nil
}

func (uc *usecase) calculateTargetCount(totalKeywords int) int {
	if totalKeywords == 0 {
		return 0
	}

	switch uc.config.DefaultStrategy {
	case sampling.PercentageStrategy:
		return uc.calculatePercentageTarget(totalKeywords)
	case sampling.FixedStrategy:
		return uc.config.MaxKeywords
	case sampling.TieredStrategy:
		return uc.calculateTieredTarget(totalKeywords)
	default:
		return uc.calculatePercentageTarget(totalKeywords)
	}
}

func (uc *usecase) calculatePercentageTarget(totalKeywords int) int {
	// Calculate percentage-based count
	percentageCount := int(float64(totalKeywords) * uc.config.Percentage / 100.0)

	// Apply min/max constraints
	targetCount := percentageCount
	if targetCount < uc.config.MinKeywords {
		targetCount = uc.config.MinKeywords
	}
	if targetCount > uc.config.MaxKeywords {
		targetCount = uc.config.MaxKeywords
	}

	// Cannot exceed available keywords
	if targetCount > totalKeywords {
		targetCount = totalKeywords
	}

	return targetCount
}

func (uc *usecase) calculateTieredTarget(totalKeywords int) int {
	// Simple tiered approach
	switch {
	case totalKeywords <= 10:
		return min(totalKeywords, uc.config.MinKeywords)
	case totalKeywords <= 50:
		return min(totalKeywords, (uc.config.MinKeywords+uc.config.MaxKeywords)/2)
	default:
		return min(totalKeywords, uc.config.MaxKeywords)
	}
}

func (uc *usecase) estimateExecutionTime(keywordCount int) time.Duration {
	return time.Duration(keywordCount) * uc.config.KeywordTimeEstimate
}

func (uc *usecase) isEmergencyFallbackNeeded(keywordCount int) bool {
	estimatedTime := uc.estimateExecutionTime(keywordCount)
	return estimatedTime > uc.config.EmergencyThreshold
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
