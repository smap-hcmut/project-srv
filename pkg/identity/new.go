package identity

import (
	"smap-project/config"
	"smap-project/pkg/log"
)

// NewClient creates a new Identity Service client.
func NewClient(cfg config.IdentityConfig, l log.Logger) Client {
	return newHTTPClient(cfg, l)
}
