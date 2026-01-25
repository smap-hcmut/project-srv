package llm

import "context"

// Provider defines the interface for an LLM provider.
type Provider interface {
	// SuggestKeywords suggests niche and negative keywords for a given brand name.
	SuggestKeywords(ctx context.Context, brandName string) (niche []string, negative []string, err error)
	// CheckAmbiguity checks if a keyword is ambiguous.
	CheckAmbiguity(ctx context.Context, keyword string) (isAmbiguous bool, context string, err error)
}
