package llm

import (
	"fmt"

	"smap-project/config"
	"smap-project/pkg/log"
)

// NewProvider creates a new LLM provider based on the configuration.
func NewProvider(cfg config.LLMConfig, l log.Logger) (Provider, error) {
	switch cfg.Provider {
	case "gemini":
		return newGeminiProvider(cfg, l), nil
	// case "openai":
	// 	return newOpenAIProvider(cfg, l), nil
	default:
		return nil, fmt.Errorf("unsupported LLM provider: %s", cfg.Provider)
	}
}
