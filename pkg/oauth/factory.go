package oauth

import (
	"fmt"
)

// NewProvider creates an identity provider based on configuration
func NewProvider(cfg Config) (Provider, error) {
	switch cfg.ProviderType {
	case "google":
		return NewGoogleProvider(cfg), nil
	case "azure":
		return NewAzureProvider(cfg), nil
	case "okta":
		if cfg.OktaDomain == "" {
			return nil, fmt.Errorf("okta_domain is required for Okta provider")
		}
		return NewOktaProvider(cfg, cfg.OktaDomain), nil
	default:
		return nil, fmt.Errorf("unsupported provider type: %s (supported: google, azure, okta)", cfg.ProviderType)
	}
}
