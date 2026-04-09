package client

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/smart-attendance/shared/dto"
	"github.com/smart-attendance/shared/response"
)

// AuthClient calls the Auth Service via HTTP.
type AuthClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewAuthClient(baseURL string) *AuthClient {
	return &AuthClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// GetUser fetches user info from the Auth Service by user ID.
func (c *AuthClient) GetUser(userID string) (*dto.User, error) {
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
