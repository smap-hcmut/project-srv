package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"smap-project/config"
	"smap-project/pkg/log"
)

const (
	geminiAPIURL = "https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s"
)

type geminiProvider struct {
	client     *http.Client
	l          log.Logger
	cfg        config.LLMConfig
	baseURL    string
	suggester  string
	ambiguity  string
	maxRetries int
}

func newGeminiProvider(cfg config.LLMConfig, l log.Logger) *geminiProvider {
	if cfg.Model == "" {
		cfg.Model = "gemini-1.5-flash"
	}
	// Sanitize model name: remove "models/" prefix if present
	if len(cfg.Model) > 7 && cfg.Model[:7] == "models/" {
		cfg.Model = cfg.Model[7:]
	}
	l.Infof(context.Background(), "Initializing Gemini provider with model: %s", cfg.Model)

	return &geminiProvider{
		client: &http.Client{
			Timeout: time.Duration(cfg.Timeout) * time.Second,
		},
		l:          l,
		cfg:        cfg,
		baseURL:    "", // This will be empty for production use
		suggester:  "Suggest 5-10 niche keywords and 5-10 negative keywords for brand: %s. Return as JSON: {\"niche\": [...], \"negative\": [...]}",
		ambiguity:  "Is the keyword '%s' ambiguous? Return JSON: {\"ambiguous\": true/false, \"context\": \"explanation\"}",
		maxRetries: cfg.MaxRetries,
	}
}

type GeminiRequest struct {
	Contents         []GeminiContent  `json:"contents"`
	GenerationConfig GenerationConfig `json:"generationConfig"`
}

type GeminiContent struct {
	Parts []GeminiPart `json:"parts"`
}

type GeminiPart struct {
	Text string `json:"text"`
}

type GenerationConfig struct {
	ResponseMIMEType string `json:"responseMimeType"`
}

type GeminiResponse struct {
	Candidates []GeminiCandidate `json:"candidates"`
}

type GeminiCandidate struct {
	Content GeminiContent `json:"content"`
}

type SuggestionResponse struct {
	Niche    []string `json:"niche"`
	Negative []string `json:"negative"`
}

type AmbiguityResponse struct {
	Ambiguous bool   `json:"ambiguous"`
	Context   string `json:"context"`
}

func (p *geminiProvider) SuggestKeywords(ctx context.Context, brandName string) ([]string, []string, error) {
	prompt := fmt.Sprintf(p.suggester, brandName)
	reqBody := GeminiRequest{
		Contents: []GeminiContent{
			{Parts: []GeminiPart{{Text: prompt}}},
		},
		GenerationConfig: GenerationConfig{ResponseMIMEType: "application/json"},
	}

	var suggestion SuggestionResponse
	err := p.doRequest(ctx, reqBody, &suggestion)
	if err != nil {
		return nil, nil, err
	}

	return suggestion.Niche, suggestion.Negative, nil
}

func (p *geminiProvider) CheckAmbiguity(ctx context.Context, keyword string) (bool, string, error) {
	prompt := fmt.Sprintf(p.ambiguity, keyword)
	reqBody := GeminiRequest{
		Contents: []GeminiContent{
			{Parts: []GeminiPart{{Text: prompt}}},
		},
		GenerationConfig: GenerationConfig{ResponseMIMEType: "application/json"},
	}

	var ambiguity AmbiguityResponse
	err := p.doRequest(ctx, reqBody, &ambiguity)
	if err != nil {
		return false, "", err
	}

	return ambiguity.Ambiguous, ambiguity.Context, nil
}

func (p *geminiProvider) doRequest(ctx context.Context, reqBody GeminiRequest, respBody interface{}) error {
	var err error
	var resp *http.Response

	for i := 0; i < p.maxRetries; i++ {
		var req *http.Request
		req, err = p.buildRequest(ctx, reqBody)
		if err != nil {
			return err
		}

		resp, err = p.client.Do(req)
		if err == nil && resp.StatusCode == http.StatusOK {
			break
		}

		if resp != nil {
			p.l.Warnf(ctx, "LLM request failed, attempt %d/%d: status %d", i+1, p.maxRetries, resp.StatusCode)
			resp.Body.Close()
		} else {
			p.l.Warnf(ctx, "LLM request failed, attempt %d/%d: %v", i+1, p.maxRetries, err)
		}
		time.Sleep(time.Duration(i+1) * time.Second) // Exponential backoff
	}

	if err != nil {
		return fmt.Errorf("%w: %v", ErrLLMUnavailable, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return p.handleErrorResponse(resp)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("%w: failed to read response body: %v", ErrLLMInvalidResponse, err)
	}
	p.l.Infof(ctx, "LLM response body: %s", string(body))

	var geminiResp GeminiResponse
	if err := json.Unmarshal(body, &geminiResp); err != nil {
		return fmt.Errorf("%w: failed to unmarshal gemini response: %v", ErrLLMInvalidResponse, err)
	}

	if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return fmt.Errorf("%w: empty response from LLM", ErrLLMInvalidResponse)
	}

	jsonStr := geminiResp.Candidates[0].Content.Parts[0].Text
	if err := json.Unmarshal([]byte(jsonStr), respBody); err != nil {
		return fmt.Errorf("%w: failed to unmarshal nested JSON: %v", ErrLLMInvalidResponse, err)
	}

	return nil
}

func (p *geminiProvider) buildRequest(ctx context.Context, reqBody GeminiRequest) (*http.Request, error) {
	var url string
	if p.baseURL != "" {
		url = fmt.Sprintf(p.baseURL, p.cfg.Model, p.cfg.APIKey)
	} else {
		url = fmt.Sprintf(geminiAPIURL, p.cfg.Model, p.cfg.APIKey)
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %v", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	return req, nil
}

func (p *geminiProvider) handleErrorResponse(resp *http.Response) error {
	switch resp.StatusCode {
	case http.StatusUnauthorized, http.StatusForbidden:
		return ErrLLMInvalidAPIKey
	case http.StatusRequestTimeout:
		return ErrLLMTimeout
	default:
		return fmt.Errorf("%w: status %d", ErrLLMUnavailable, resp.StatusCode)
	}
}
