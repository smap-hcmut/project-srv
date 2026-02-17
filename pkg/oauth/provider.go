package oauth

import (
	"context"

	"golang.org/x/oauth2"
)

// Provider defines the interface for OAuth2 identity providers
type Provider interface {
	// GetAuthCodeURL returns the OAuth2 authorization URL
	GetAuthCodeURL(state string, opts ...oauth2.AuthCodeOption) string

	// ExchangeCode exchanges an authorization code for an access token
	ExchangeCode(ctx context.Context, code string) (*oauth2.Token, error)

	// GetUserInfo retrieves user information using the access token
	GetUserInfo(ctx context.Context, token *oauth2.Token) (*UserInfo, error)

	// GetProviderName returns the provider name (google, azure, okta)
	GetProviderName() string
}

// UserInfo represents normalized user information from any provider
type UserInfo struct {
	Email   string
	Name    string
	Picture string
}

// Config holds OAuth2 provider configuration
type Config struct {
	ClientID     string
	ClientSecret string
	RedirectURI  string
	Scopes       []string
	ProviderType string // "google", "azure", "okta"
	OktaDomain   string // Only for Okta
}
