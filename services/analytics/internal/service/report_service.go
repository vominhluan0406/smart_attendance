package service

import (
	"fmt"
	"log"

	"github.com/smart-attendance/analytics-service/internal/client"
	"github.com/smart-attendance/shared/dto"
	"github.com/xuri/excelize/v2"
)

type ReportService struct {
	attendanceClient *client.AttendanceClient
}

func NewReportService(attendanceClient *client.AttendanceClient) *ReportService {
	return &ReportService{
		attendanceClient: attendanceClient,
	}
}

// GetBranchReport returns paginated attendance records for a branch.
func (s *ReportService) GetBranchReport(branchID string, page, limit int, dateFrom, dateTo, status string) (*client.AttendanceListResult, error) {
	result, err := s.attendanceClient.ListAttendance(branchID, "", page, limit, dateFrom, dateTo, status)
	if err != nil {
		log.Printf("[service][report] ERROR GetBranchReport branch=%s: %v", branchID, err)
		return nil, err
	}
	return result, nil
}

// GetUserHistory returns paginated attendance records for a user.
func (s *ReportService) GetUserHistory(userID string, page, limit int, dateFrom, dateTo, status string) (*client.AttendanceListResult, error) {
	result, err := s.attendanceClient.ListAttendance("", userID, page, limit, dateFrom, dateTo, status)
	if err != nil {
		log.Printf("[service][report] ERROR GetUserHistory user=%s: %v", userID, err)
		return nil, err
	}
	return result, nil
}

// ExportBranchExcel generates an Excel file with all attendance records for a branch.
func (s *ReportService) ExportBranchExcel(branchID, dateFrom, dateTo, status string) ([]byte, error) {
	result, err := s.attendanceClient.ListAttendance(branchID, "", 1, 10000, dateFrom, dateTo, status)
	if err != nil {
		log.Printf("[service][report] ERROR fetching branch records for export: %v", err)
		return nil, err
	}

	log.Printf("[service][report] ExportBranchExcel: branch=%s records=%d", branchID, len(result.Records))
	return generateExcel(result.Records)
}

// ExportUserExcel generates an Excel file with all attendance records for a user.
func (s *ReportService) ExportUserExcel(userID, dateFrom, dateTo, status string) ([]byte, error) {
	result, err := s.attendanceClient.ListAttendance("", userID, 1, 10000, dateFrom, dateTo, status)
	if err != nil {
		log.Printf("[service][report] ERROR fetching user records for export: %v", err)
		return nil, err
	}

	log.Printf("[service][report] ExportUserExcel: user=%s records=%d", userID, len(result.Records))
	return generateExcel(result.Records)
}

func generateExcel(records []dto.Attendance) ([]byte, error) {
	f := excelize.NewFile()
	defer func() {
		if err := f.Close(); err != nil {
			log.Printf("[service][report] excel close error: %v", err)
		}
	}()

	sheetName := "Sheet1"
	f.NewSheet(sheetName)
	index, _ := f.GetSheetIndex(sheetName)
	f.SetActiveSheet(index)

	// Set headers
	headers := []string{"ID", "Ho Ten", "Chi Nhanh", "Ngay", "Gio Vao", "Gio Ra", "Trang Thai", "Phuong Thuc", "Ghi Chu"}
	for i, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheetName, cell, header)
	}

	// Make header bold
	style, err := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#E2E8F0"}, Pattern: 1},
	})
	if err == nil {
		f.SetRowStyle(sheetName, 1, 1, style)
	}

	// Fill data
	for i, r := range records {
		row := i + 2

		f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), r.ID)

		userName := r.UserName
		if userName == "" {
			userName = r.UserID
		}
		f.SetCellValue(sheetName, fmt.Sprintf("B%d", row), userName)

		branchName := r.BranchName
		if branchName == "" {
			branchName = r.BranchID
		}
		f.SetCellValue(sheetName, fmt.Sprintf("C%d", row), branchName)

		if r.WorkDate != "" {
			f.SetCellValue(sheetName, fmt.Sprintf("D%d", row), r.WorkDate)
		} else if r.CheckInAt != nil {
			f.SetCellValue(sheetName, fmt.Sprintf("D%d", row), r.CheckInAt.Format("2006-01-02"))
		}

		if r.CheckInAt != nil {
			f.SetCellValue(sheetName, fmt.Sprintf("E%d", row), r.CheckInAt.Format("15:04:05"))
		}

		if r.CheckOutAt != nil {
			f.SetCellValue(sheetName, fmt.Sprintf("F%d", row), r.CheckOutAt.Format("15:04:05"))
		}

		statusStr := r.Status
		switch r.Status {
		case "on_time":
			statusStr = "Dung gio"
		case "late":
			statusStr = "Di tre"
		case "absent":
			statusStr = "Vang mat"
		case "leave":
			statusStr = "Nghi phep"
		}
		f.SetCellValue(sheetName, fmt.Sprintf("G%d", row), statusStr)
		f.SetCellValue(sheetName, fmt.Sprintf("H%d", row), r.Method)
		f.SetCellValue(sheetName, fmt.Sprintf("I%d", row), r.Note)
	}

	// Auto-fit columns roughly
	f.SetColWidth(sheetName, "A", "A", 36) // UUID
	f.SetColWidth(sheetName, "B", "C", 20)
	f.SetColWidth(sheetName, "D", "G", 15)

	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, fmt.Errorf("write excel to buffer: %w", err)
	}

	return buf.Bytes(), nil
}
