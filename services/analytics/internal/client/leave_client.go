package client

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// LeaveClient calls the Leave Service via HTTP.
type LeaveClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewLeaveClient(baseURL string) *LeaveClient {
	return &LeaveClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// GetPendingCount returns the count of pending leave requests for a branch.
func (c *LeaveClient) GetPendingCount(branchID string) (int64, error) {
	url := fmt.Sprintf("%s/api/internal/leave/pending-count?branch_id=%s", c.baseURL, branchID)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return 0, fmt.Errorf("[client][leave] pending-count request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("[client][leave] pending-count returned status %d", resp.StatusCode)
	}

	var apiResp struct {
		Success bool `json:"success"`
		Data    struct {
			Count int64 `json:"count"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return 0, fmt.Errorf("[client][leave] decode response: %w", err)
	}

	return apiResp.Data.Count, nil
}
