package client

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/smart-attendance/shared/dto"
	"github.com/smart-attendance/shared/response"
)

// OrgClient calls the Organization Service via HTTP.
type OrgClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewOrgClient(baseURL string) *OrgClient {
	return &OrgClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// GetBranch fetches branch details (including IP whitelist, locations) from the Org Service.
func (c *OrgClient) GetBranch(branchID string) (*dto.Branch, error) {
	url := fmt.Sprintf("%s/api/internal/branches/%s", c.baseURL, branchID)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("[client][org] request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("[client][org] branch %s not found (status %d)", branchID, resp.StatusCode)
	}

	var apiResp struct {
		response.Response
		Data dto.Branch `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("[client][org] decode response: %w", err)
	}

	if !apiResp.Success {
		return nil, fmt.Errorf("[client][org] branch %s: API returned failure", branchID)
	}

	return &apiResp.Data, nil
}

// UserShift represents a shift assignment for a user.
type UserShift struct {
	ShiftID            string `json:"shift_id"`
	StartTime          string `json:"start_time"`
	GracePeriodMinutes int    `json:"grace_period_minutes"`
}

// GetUserShift fetches the shift assigned to a user at a branch on a given date.
func (c *OrgClient) GetUserShift(userID, branchID, workDate string) (*UserShift, error) {
	url := fmt.Sprintf("%s/api/internal/shifts/user?user_id=%s&branch_id=%s&work_date=%s",
		c.baseURL, userID, branchID, workDate)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("[client][org] shift request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("[client][org] shift not found (status %d)", resp.StatusCode)
	}

	var apiResp struct {
		response.Response
		Data UserShift `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("[client][org] decode shift response: %w", err)
	}

	if !apiResp.Success {
		return nil, fmt.Errorf("[client][org] shift: API returned failure")
	}

	return &apiResp.Data, nil
}
