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

// ListUsers fetches users for a branch (or all users if branchID is empty).
func (c *AuthClient) ListUsers(branchID string) ([]dto.User, error) {
	url := fmt.Sprintf("%s/api/internal/users?limit=10000", c.baseURL)
	if branchID != "" {
		url += "&branch_id=" + branchID
	}

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("[client][auth] list users request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("[client][auth] list users returned status %d", resp.StatusCode)
	}

	var apiResp struct {
		response.Response
		Data []dto.User `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("[client][auth] decode response: %w", err)
	}

	if !apiResp.Success {
		return nil, fmt.Errorf("[client][auth] list users: API returned failure")
	}

	return apiResp.Data, nil
}

// CountUsers returns the count of active users for a branch (or all if branchID is empty).
func (c *AuthClient) CountUsers(branchID string) (int64, error) {
	users, err := c.ListUsers(branchID)
	if err != nil {
		return 0, err
	}
	var count int64
	for _, u := range users {
		if u.IsActive {
			count++
		}
	}
	return count, nil
}
