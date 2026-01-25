package llm

import "errors"

var (
	// ErrLLMUnavailable is returned when the LLM service is unavailable.
	ErrLLMUnavailable = errors.New("LLM service unavailable")
	// ErrLLMTimeout is returned when the LLM request times out.
	ErrLLMTimeout = errors.New("LLM request timeout")
	// ErrLLMInvalidResponse is returned when the LLM returns an invalid response.
	ErrLLMInvalidResponse = errors.New("LLM returned invalid response")
	// ErrLLMInvalidAPIKey is returned when the LLM API key is invalid.
	ErrLLMInvalidAPIKey = errors.New("LLM API key is invalid")
)
