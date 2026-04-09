package client

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	gocache "github.com/patrickmn/go-cache"
	"github.com/smart-attendance/shared/dto"
	"github.com/smart-attendance/shared/response"
)

// AuthClient calls the Auth Service via HTTP with local cache.
type AuthClient struct {
	baseURL    string
	httpClient *http.Client
	cache      *gocache.Cache
}

func NewAuthClient(baseURL string) *AuthClient {
	return &AuthClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
		cache: gocache.New(5*time.Minute, 10*time.Minute),
	}
}

// GetUser fetches user info — cache first, then HTTP fallback.
func (c *AuthClient) GetUser(userID string) (*dto.User, error) {
	cacheKey := "user:" + userID

	// Cache hit
	if cached, found := c.cache.Get(cacheKey); found {
		return cached.(*dto.User), nil
	}

	// Cache miss → HTTP
	user, err := c.fetchUser(userID)
	if err != nil {
		return nil, err
	}

	c.cache.Set(cacheKey, user, 5*time.Minute)
	return user, nil
}

// InvalidateUser removes a user from local cache (called by NATS subscriber).
func (c *AuthClient) InvalidateUser(userID string) {
	c.cache.Delete("user:" + userID)
	log.Printf("[client][auth] cache invalidated: user=%s", userID)
}

// InvalidateAllUsers flushes all cached users.
func (c *AuthClient) InvalidateAllUsers() {
	c.cache.Flush()
	log.Printf("[client][auth] cache flushed: all users")
}

func (c *AuthClient) fetchUser(userID string) (*dto.User, error) {
	url := fmt.Sprintf("%s/api/internal/users/%s", c.baseURL, userID)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("[client][auth] request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("[client][auth] user %s not found (status %d)", userID, resp.StatusCode)
	}

	var apiResp struct {
		response.Response
		Data dto.User `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("[client][auth] decode response: %w", err)
	}

	if !apiResp.Success {
		return nil, fmt.Errorf("[client][auth] user %s: API returned failure", userID)
	}

	return &apiResp.Data, nil
}
