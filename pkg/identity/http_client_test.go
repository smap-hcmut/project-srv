package identity

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"smap-project/config"
	"smap-project/pkg/log"
	"testing"

	"github.com/stretchr/testify/assert"
)

func newTestHTTPClient(serverURL string) *httpClient {
	cfg := config.IdentityConfig{
		BaseURL:        serverURL,
		Timeout:        5,
		InternalAPIKey: "test-key",
	}
	return newHTTPClient(cfg, log.NewNopLogger())
}

func TestHTTPClient_GetUser_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/users/internal/user-1", r.URL.Path)
		assert.Equal(t, "test-key", r.Header.Get("Authorization"))

		resp := userResponse{
			Data: User{
				ID:       "user-1",
				Username: "testuser",
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := newTestHTTPClient(server.URL)
	user, err := client.GetUser(context.Background(), "user-1")

	assert.NoError(t, err)
	assert.Equal(t, "user-1", user.ID)
	assert.Equal(t, "testuser", user.Username)
}

func TestHTTPClient_GetUserSubscription_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/subscriptions/internal/users/user-1", r.URL.Path)
		assert.Equal(t, "test-key", r.Header.Get("Authorization"))

		resp := subscriptionResponse{
			Data: Subscription{
				ID:     "sub-1",
				UserID: "user-1",
				Status: "active",
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := newTestHTTPClient(server.URL)
	sub, err := client.GetUserSubscription(context.Background(), "user-1")

	assert.NoError(t, err)
	assert.Equal(t, "sub-1", sub.ID)
	assert.Equal(t, "user-1", sub.UserID)
	assert.Equal(t, "active", sub.Status)
}

func TestHTTPClient_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := newTestHTTPClient(server.URL)
	_, err := client.GetUser(context.Background(), "user-1")
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrUserNotFound)
}

func TestHTTPClient_Unauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	client := newTestHTTPClient(server.URL)
	_, err := client.GetUser(context.Background(), "user-1")
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrUnauthorized)
}
