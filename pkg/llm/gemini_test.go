package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"smap-project/config"
	"smap-project/pkg/log"

	"github.com/stretchr/testify/assert"
)

func newTestGeminiProvider(serverURL string) *geminiProvider {
	cfg := config.LLMConfig{
		Provider:   "gemini",
		APIKey:     "test-key",
		Model:      "gemini-1.5-flash",
		Timeout:    5,
		MaxRetries: 3,
	}

	p := newGeminiProvider(cfg, log.NewNopLogger())
	p.baseURL = serverURL + "/v1beta/models/%s:generateContent?key=%s"
	return p
}

func TestGeminiProvider_SuggestKeywords_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)

		var req GeminiRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		assert.NoError(t, err)

		resp := GeminiResponse{
			Candidates: []GeminiCandidate{
				{
					Content: GeminiContent{
						Parts: []GeminiPart{
							{Text: `{"niche": ["kw1", "kw2"], "negative": ["nkw1", "nkw2"]}`},
						},
					},
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	provider := newTestGeminiProvider(server.URL)
	niche, negative, err := provider.SuggestKeywords(context.Background(), "test brand")

	assert.NoError(t, err)
	assert.Equal(t, []string{"kw1", "kw2"}, niche)
	assert.Equal(t, []string{"nkw1", "nkw2"}, negative)
}

func TestGeminiProvider_CheckAmbiguity_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)

		var req GeminiRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		assert.NoError(t, err)

		resp := GeminiResponse{
			Candidates: []GeminiCandidate{
				{
					Content: GeminiContent{
						Parts: []GeminiPart{
							{Text: `{"ambiguous": true, "context": "it can be a fruit or a tech company"}`},
						},
					},
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	provider := newTestGeminiProvider(server.URL)
	isAmbiguous, context, err := provider.CheckAmbiguity(context.Background(), "apple")

	assert.NoError(t, err)
	assert.True(t, isAmbiguous)
	assert.Equal(t, "it can be a fruit or a tech company", context)
}

func TestGeminiProvider_Retry(t *testing.T) {
	attempt := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if attempt < 2 {
			w.WriteHeader(http.StatusInternalServerError)
			attempt++
			return
		}
		resp := GeminiResponse{
			Candidates: []GeminiCandidate{
				{
					Content: GeminiContent{
						Parts: []GeminiPart{
							{Text: `{"niche": ["kw1"], "negative": []}`},
						},
					},
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	provider := newTestGeminiProvider(server.URL)
	niche, _, err := provider.SuggestKeywords(context.Background(), "test brand")

	assert.NoError(t, err)
	assert.Equal(t, []string{"kw1"}, niche)
	assert.Equal(t, 2, attempt)
}

func TestGeminiProvider_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(10 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	provider := newTestGeminiProvider(server.URL)
	provider.client.Timeout = 5 * time.Millisecond

	_, _, err := provider.SuggestKeywords(context.Background(), "test brand")
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrLLMUnavailable)
}

func TestGeminiProvider_InvalidResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"invalid": "json"}`)
	}))
	defer server.Close()

	provider := newTestGeminiProvider(server.URL)
	_, _, err := provider.SuggestKeywords(context.Background(), "test brand")
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrLLMInvalidResponse)
}
