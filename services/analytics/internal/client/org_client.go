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

// ListBranches fetches all branches from the Organization Service.
func (c *OrgClient) ListBranches() ([]dto.Branch, error) {
	url := fmt.Sprintf("%s/api/internal/branches?limit=10000", c.baseURL)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("[client][org] list branches request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("[client][org] list branches returned status %d", resp.StatusCode)
	}

	var apiResp struct {
		response.Response
		Data []dto.Branch `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("[client][org] decode response: %w", err)
	}

	if !apiResp.Success {
		return nil, fmt.Errorf("[client][org] list branches: API returned failure")
	}

	return apiResp.Data, nil
}

// GetBranch fetches a single branch by ID from the Organization Service.
func (c *OrgClient) GetBranch(id string) (*dto.Branch, error) {
	url := fmt.Sprintf("%s/api/internal/branches/%s", c.baseURL, id)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("[client][org] get branch request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("[client][org] branch %s not found (status %d)", id, resp.StatusCode)
	}

	var apiResp struct {
		response.Response
		Data dto.Branch `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("[client][org] decode response: %w", err)
	}

	if !apiResp.Success {
		return nil, fmt.Errorf("[client][org] branch %s: API returned failure", id)
	}

	return &apiResp.Data, nil
}
