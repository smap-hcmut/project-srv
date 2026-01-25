package identity

import (
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
	userEndpoint         = "/users/internal/%s"
	subscriptionEndpoint = "/subscriptions/internal/users/%s"
)

type httpClient struct {
	client      *http.Client
	l           log.Logger
	baseURL     string
	internalKey string
}

func newHTTPClient(cfg config.IdentityConfig, l log.Logger) *httpClient {
	return &httpClient{
		client: &http.Client{
			Timeout: time.Duration(cfg.Timeout) * time.Second,
		},
		l:           l,
		baseURL:     cfg.BaseURL,
		internalKey: cfg.InternalAPIKey,
	}
}

func (c *httpClient) GetUser(ctx context.Context, id string) (User, error) {
	url := fmt.Sprintf(c.baseURL+userEndpoint, id)

	var resp userResponse
	if err := c.doRequest(ctx, "GET", url, &resp); err != nil {
		return User{}, err
	}

	return resp.Data, nil
}

func (c *httpClient) GetUserSubscription(ctx context.Context, userID string) (Subscription, error) {
	url := fmt.Sprintf(c.baseURL+subscriptionEndpoint, userID)

	var resp subscriptionResponse
	if err := c.doRequest(ctx, "GET", url, &resp); err != nil {
		return Subscription{}, err
	}

	return resp.Data, nil
}

func (c *httpClient) doRequest(ctx context.Context, method, url string, result interface{}) error {
	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	// Add internal authentication header
	req.Header.Set("Authorization", c.internalKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrIdentityUnavailable, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return c.handleErrorResponse(resp)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("%w: failed to read response body: %v", ErrIdentityInvalidResponse, err)
	}

	// If result is nil, we don't need to unmarshal the response
	if result == nil {
		return nil
	}

	if err := json.Unmarshal(body, result); err != nil {
		return fmt.Errorf("%w: failed to unmarshal response: %v", ErrIdentityInvalidResponse, err)
	}

	return nil
}

func (c *httpClient) handleErrorResponse(resp *http.Response) error {
	switch resp.StatusCode {
	case http.StatusNotFound:
		return ErrUserNotFound
	case http.StatusUnauthorized:
		return ErrUnauthorized
	case http.StatusRequestTimeout:
		return ErrIdentityTimeout
	default:
		return fmt.Errorf("%w: status %d", ErrIdentityUnavailable, resp.StatusCode)
	}
}
