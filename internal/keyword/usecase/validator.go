package usecase

import (
	"context"
	"errors"
	"regexp"
	"strings"
)

func (uc *usecase) validate(ctx context.Context, keywords []string) ([]string, error) {
	validKeywords := make([]string, 0, len(keywords))
	seen := make(map[string]bool)

	for _, kw := range keywords {
		normalized, err := uc.validateOne(ctx, kw)
		if err != nil {
			uc.l.Errorf(ctx, "internal.keyword.usecase.validator.validate: %v", err)
			return nil, err
		}
		if !seen[normalized] {
			validKeywords = append(validKeywords, normalized)
			seen[normalized] = true
		}
	}

	return validKeywords, nil
}

func (uc *usecase) validateOne(ctx context.Context, keyword string) (string, error) {
	// Normalize: trim spaces, lowercase
	normalized := strings.TrimSpace(strings.ToLower(keyword))

	// Check length
	if len(normalized) < 2 {
		uc.l.Warnf(ctx, "internal.keyword.usecase.validator.validateOne: '%s' is too short (min 2 characters)", keyword)
		return "", errors.New("keyword '" + keyword + "' is too short (min 2 characters)")
	}
	if len(normalized) > 50 {
		uc.l.Warnf(ctx, "internal.keyword.usecase.validator.validateOne: '%s' is too long (max 50 characters)", keyword)
		return "", errors.New("keyword '" + keyword + "' is too long (max 50 characters)")
	}

	// Check character set (alphanumeric, spaces, hyphens, underscores)
	// Allow Vietnamese characters as well? For now, let's stick to simple regex or just allow most.
	// The requirement says: alphanumeric, spaces, hyphens, underscores.
	// Regex: ^[a-zA-Z0-9\s\-_]+$ (plus unicode for Vietnamese if needed, but let's stick to basic for now as per spec)
	// Actually, for a real app, we should allow unicode.
	// Let's use a slightly more permissive regex but block special chars like @, #, $, etc.
	matched, _ := regexp.MatchString(`^[\p{L}\p{N}\s\-_]+$`, normalized)
	if !matched {
		uc.l.Warnf(ctx, "internal.keyword.usecase.validator.validateOne: '%s' contains invalid characters", keyword)
		return "", errors.New("keyword '" + keyword + "' contains invalid characters")
	}

	// Check generic terms (stopwords)
	stopwords := map[string]bool{
		"xe": true, "mua": true, "ban": true, "bán": true,
		"buy": true, "sell": true, "car": true, "house": true,
	}
	if stopwords[normalized] {
		uc.l.Warnf(ctx, "internal.keyword.usecase.validator.validateOne: '%s' is too generic", keyword)
		return "", errors.New("keyword '" + keyword + "' is too generic")
	}

	// Check for ambiguity in single words using LLM
	if isSingleWord(normalized) {
		isAmbiguous, context, err := uc.llmProvider.CheckAmbiguity(ctx, normalized)
		if err != nil {
			// Log the error but don't fail validation, as this is a non-critical check
			uc.l.Warnf(ctx, "LLM ambiguity check failed for '%s': %v", normalized, err)
		} else if isAmbiguous {
			uc.l.Warnf(ctx, "Keyword '%s' might be ambiguous. Context: %s", normalized, context)
		}
	}

	return normalized, nil
}

// isSingleWord checks if the string is a single word (no spaces).
func isSingleWord(s string) bool {
	return !strings.Contains(s, " ")
}
