package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"smap-project/pkg/log"

	"github.com/stretchr/testify/assert"
)

type mockLLMProvider struct {
	shouldError bool
}

func (m *mockLLMProvider) SuggestKeywords(ctx context.Context, brandName string) ([]string, []string, error) {
	if m.shouldError {
		return nil, nil, errors.New("LLM error")
	}
	return []string{"llm kw1", "llm kw2"}, []string{"llm nkw1"}, nil
}

func (m *mockLLMProvider) CheckAmbiguity(ctx context.Context, keyword string) (bool, string, error) {
	return false, "", nil
}

func TestSuggester_suggestProcessing_LLMSuccess(t *testing.T) {
	uc := &usecase{
		l:           log.NewNopLogger(),
		llmProvider: &mockLLMProvider{shouldError: false},
		clock:       time.Now,
	}

	niche, negative, err := uc.suggestProcessing(context.Background(), "test brand")

	assert.NoError(t, err)
	assert.Equal(t, []string{"llm kw1", "llm kw2"}, niche)
	assert.Equal(t, []string{"llm nkw1"}, negative)
}

func TestSuggester_suggestProcessing_LLMFailure(t *testing.T) {
	uc := &usecase{
		l:           log.NewNopLogger(),
		llmProvider: &mockLLMProvider{shouldError: true},
		clock:       time.Now,
	}

	niche, negative, err := uc.suggestProcessing(context.Background(), "test brand")

	assert.NoError(t, err)
	assert.Equal(t, []string{
		"test brand review",
		"test brand price",
		"test brand specs",
		"test brand problems",
	}, niche)
	assert.Nil(t, negative)
}
