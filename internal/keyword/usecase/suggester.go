package usecase

import (
	"context"
)

func (uc *usecase) suggestProcessing(ctx context.Context, brandName string) ([]string, []string, error) {
	niche, negative, err := uc.llmProvider.SuggestKeywords(ctx, brandName)
	if err != nil {
		uc.l.Warnf(ctx, "LLM suggestion failed, using fallback: %v", err)
		return uc.fallbackSuggestions(brandName), nil, nil
	}

	// Post-validation and deduplication can be added here
	return niche, negative, nil
}

func (uc *usecase) fallbackSuggestions(brandName string) []string {
	return []string{
		brandName + " review",
		brandName + " price",
		brandName + " specs",
		brandName + " problems",
	}
}
