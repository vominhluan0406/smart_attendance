package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/smart-attendance/shared/dto"
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
			Timeout: 10 * time.Second,
		},
	}
}

// SyncLeave calls POST /api/internal/attendance/sync-leave on the Attendance Service
// to create attendance records with status "leave" for the approved leave period.
func (c *AttendanceClient) SyncLeave(userID, branchID, startDate, endDate, note string) error {
	url := fmt.Sprintf("%s/api/internal/attendance/sync-leave", c.baseURL)

	reqBody := dto.SyncLeaveRequest{
		UserID:    userID,
		BranchID:  branchID,
		StartDate: startDate,
		EndDate:   endDate,
		Note:      note,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("[client][attendance] marshal request: %w", err)
	}

	resp, err := c.httpClient.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("[client][attendance] sync-leave request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("[client][attendance] sync-leave returned status %d", resp.StatusCode)
	}

	log.Printf("[client][attendance] sync-leave success: user=%s branch=%s dates=%s~%s",
		userID, branchID, startDate, endDate)

	return nil
}
