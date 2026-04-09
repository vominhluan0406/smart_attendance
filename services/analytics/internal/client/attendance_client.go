package client

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/smart-attendance/shared/dto"
	"github.com/smart-attendance/shared/response"
)

// AttendanceClient calls the Attendance Service via HTTP.
type AttendanceClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewAttendanceClient(baseURL string) *AttendanceClient {
	return &AttendanceClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// AttendanceListResult holds paginated attendance records.
type AttendanceListResult struct {
	Records []dto.Attendance `json:"records"`
	Page    int              `json:"page"`
	Limit   int              `json:"limit"`
	Total   int64            `json:"total"`
}

// DashboardData holds aggregated stats returned by the attendance service.
type DashboardData struct {
	TodayCheckIns int64   `json:"today_check_ins"`
	TodayOnTime   int64   `json:"today_on_time"`
	TodayLate     int64   `json:"today_late"`
	TodayAbsent   int64   `json:"today_absent"`
	OnTimeRate    float64 `json:"on_time_rate"`
}

// DailyAttendance holds aggregated attendance for one day.
type DailyAttendance struct {
	Date   string `json:"date"`
	Total  int64  `json:"total"`
	OnTime int64  `json:"on_time"`
	Late   int64  `json:"late"`
	Absent int64  `json:"absent"`
}

// RecentCheckIn holds a recent check-in record.
type RecentCheckIn struct {
	UserName   string `json:"user_name"`
	BranchName string `json:"branch_name"`
	CheckInAt  string `json:"check_in_at"`
	Status     string `json:"status"`
	Method     string `json:"method"`
}

// ListAttendance fetches paginated attendance records from the Attendance Service.
func (c *AttendanceClient) ListAttendance(branchID, userID string, page, limit int, dateFrom, dateTo, status string) (*AttendanceListResult, error) {
	url := fmt.Sprintf("%s/api/internal/attendance?page=%d&limit=%d", c.baseURL, page, limit)
	if branchID != "" {
		url += "&branch_id=" + branchID
	}
	if userID != "" {
		url += "&user_id=" + userID
	}
	if dateFrom != "" {
		url += "&date_from=" + dateFrom
	}
	if dateTo != "" {
		url += "&date_to=" + dateTo
	}
	if status != "" {
		url += "&status=" + status
	}

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("[client][attendance] list request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("[client][attendance] list returned status %d", resp.StatusCode)
	}

	var apiResp struct {
		Success bool             `json:"success"`
		Data    []dto.Attendance `json:"data"`
		Meta    *response.Meta   `json:"meta,omitempty"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("[client][attendance] decode response: %w", err)
	}

	if !apiResp.Success {
		return nil, fmt.Errorf("[client][attendance] list: API returned failure")
	}

	result := &AttendanceListResult{
		Records: apiResp.Data,
		Page:    page,
		Limit:   limit,
	}
	if apiResp.Meta != nil {
		result.Total = apiResp.Meta.Total
		result.Page = apiResp.Meta.Page
		result.Limit = apiResp.Meta.Limit
	}

	return result, nil
}

// GetDashboardData fetches aggregated dashboard stats from the Attendance Service.
func (c *AttendanceClient) GetDashboardData(branchID string) (*DashboardData, error) {
	url := fmt.Sprintf("%s/api/internal/attendance/dashboard", c.baseURL)
	if branchID != "" {
		url += "?branch_id=" + branchID
	}

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("[client][attendance] dashboard request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("[client][attendance] dashboard returned status %d", resp.StatusCode)
	}

	var apiResp struct {
		Success bool          `json:"success"`
		Data    DashboardData `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("[client][attendance] decode dashboard: %w", err)
	}

	if !apiResp.Success {
		return nil, fmt.Errorf("[client][attendance] dashboard: API returned failure")
	}

	return &apiResp.Data, nil
}

// GetChartData fetches daily attendance aggregation for chart display.
func (c *AttendanceClient) GetChartData(branchID string, days int) ([]DailyAttendance, error) {
	url := fmt.Sprintf("%s/api/internal/attendance/chart?days=%d", c.baseURL, days)
	if branchID != "" {
		url += "&branch_id=" + branchID
	}

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("[client][attendance] chart request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("[client][attendance] chart returned status %d", resp.StatusCode)
	}

	var apiResp struct {
		Success bool              `json:"success"`
		Data    []DailyAttendance `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("[client][attendance] decode chart: %w", err)
	}

	if !apiResp.Success {
		return nil, fmt.Errorf("[client][attendance] chart: API returned failure")
	}

	return apiResp.Data, nil
}

// GetRecentCheckIns fetches recent check-in records.
func (c *AttendanceClient) GetRecentCheckIns(branchID string, limit int) ([]RecentCheckIn, error) {
	url := fmt.Sprintf("%s/api/internal/attendance/recent?limit=%d", c.baseURL, limit)
	if branchID != "" {
		url += "&branch_id=" + branchID
	}

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("[client][attendance] recent request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("[client][attendance] recent returned status %d", resp.StatusCode)
	}

	var apiResp struct {
		Success bool            `json:"success"`
		Data    []RecentCheckIn `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("[client][attendance] decode recent: %w", err)
	}

	if !apiResp.Success {
		return nil, fmt.Errorf("[client][attendance] recent: API returned failure")
	}

	return apiResp.Data, nil
}
