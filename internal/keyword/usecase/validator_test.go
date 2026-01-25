package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"smap-project/pkg/llm"
	"smap-project/pkg/log"

	"github.com/stretchr/testify/assert"
)

type mockAmbiguousLLMProvider struct {
	mockLLMProvider
	ambiguousKeyword string
	ambiguousContext string
}

func (m *mockAmbiguousLLMProvider) CheckAmbiguity(ctx context.Context, keyword string) (bool, string, error) {
	if keyword == m.ambiguousKeyword {
		return true, m.ambiguousContext, nil
	}
	return false, "", nil
}

// mockFailingLLMProvider simulates LLM service failures
type mockFailingLLMProvider struct {
	err error
}

func (m *mockFailingLLMProvider) SuggestKeywords(ctx context.Context, brandName string) ([]string, []string, error) {
	return nil, nil, m.err
}

func (m *mockFailingLLMProvider) CheckAmbiguity(ctx context.Context, keyword string) (bool, string, error) {
	return false, "", m.err
}

func TestValidator_validateOne_AmbiguityCheck(t *testing.T) {
	llmProvider := &mockAmbiguousLLMProvider{
		ambiguousKeyword: "apple",
		ambiguousContext: "fruit or tech",
	}
	uc := &usecase{
		l:           log.NewNopLogger(),
		llmProvider: llmProvider,
		clock:       time.Now,
	}

	// Test with ambiguous keyword
	_, err := uc.validateOne(context.Background(), "apple")
	assert.NoError(t, err) // Should not return an error, just a warning

	// Test with non-ambiguous keyword
	_, err = uc.validateOne(context.Background(), "vinfast")
	assert.NoError(t, err)

	// Test with multi-word keyword (should skip ambiguity check)
	_, err = uc.validateOne(context.Background(), "vinfast vf9")
	assert.NoError(t, err)
}


func TestValidator_validateOne_LLMFailure_ShouldContinue(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		keyword  string
		expected string
	}{
		{
			name:     "LLM unavailable - should return keyword",
			err:      llm.ErrLLMUnavailable,
			keyword:  "Apple",
			expected: "apple",
		},
		{
			name:     "LLM timeout - should return keyword",
			err:      llm.ErrLLMTimeout,
			keyword:  "Samsung",
			expected: "samsung",
		},
		{
			name:     "LLM invalid response - should return keyword",
			err:      llm.ErrLLMInvalidResponse,
			keyword:  "Google",
			expected: "google",
		},
		{
			name:     "LLM invalid API key - should return keyword",
			err:      llm.ErrLLMInvalidAPIKey,
			keyword:  "Microsoft",
			expected: "microsoft",
		},
		{
			name:     "Generic error - should return keyword",
			err:      errors.New("network error"),
			keyword:  "Tesla",
			expected: "tesla",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := &usecase{
				l:           log.NewNopLogger(),
				llmProvider: &mockFailingLLMProvider{err: tt.err},
				clock:       time.Now,
			}

			// LLM failure should NOT cause validation to fail
			result, err := uc.validateOne(context.Background(), tt.keyword)

			assert.NoError(t, err, "LLM failure should not return error")
			assert.Equal(t, tt.expected, result, "Should return normalized keyword even when LLM fails")
		})
	}
}

func TestValidator_validate_LLMFailure_AllKeywordsReturned(t *testing.T) {
	uc := &usecase{
		l:           log.NewNopLogger(),
		llmProvider: &mockFailingLLMProvider{err: llm.ErrLLMUnavailable},
		clock:       time.Now,
	}

	keywords := []string{"Apple", "Samsung", "Google"}

	// All keywords should be validated and returned even when LLM fails
	result, err := uc.validate(context.Background(), keywords)

	assert.NoError(t, err, "LLM failure should not cause validation to fail")
	assert.Equal(t, []string{"apple", "samsung", "google"}, result, "All keywords should be returned normalized")
}
