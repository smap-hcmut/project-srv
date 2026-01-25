package usecase

import (
	"context"

	"smap-project/internal/keyword"
)

// Suggest returns niche and negative keyword suggestions based on brand name
func (uc *usecase) Suggest(ctx context.Context, brandName string) (keyword.SuggestOutput, error) {
	niche, negative, err := uc.suggestProcessing(ctx, brandName)
	if err != nil {
		uc.l.Errorf(ctx, "internal.keyword.usecase.Suggest: %v", err)
		return keyword.SuggestOutput{}, err
	}

	return keyword.SuggestOutput{
		Niche:    niche,
		Negative: negative,
	}, nil
}

// Validate validates and normalizes keywords
func (uc *usecase) Validate(ctx context.Context, input keyword.ValidateInput) (keyword.ValidateOutput, error) {
	validKeywords, err := uc.validate(ctx, input.Keywords)
	if err != nil {
		uc.l.Errorf(ctx, "internal.keyword.usecase.Validate: %v", err)
		return keyword.ValidateOutput{}, err
	}

	return keyword.ValidateOutput{
		ValidKeywords: validKeywords,
	}, nil
}
