package service

import (
	"fmt"
	"log"
	"time"

	"github.com/smart-attendance/smart-attendance/internal/models"
	"github.com/smart-attendance/smart-attendance/internal/repository"
	"github.com/xuri/excelize/v2"
)

type ReportService struct {
	attendanceRepo *repository.AttendanceRepository
}

func NewReportService(attendanceRepo *repository.AttendanceRepository) *ReportService {
	return &ReportService{
		attendanceRepo: attendanceRepo,
	}
}

// GetUserHistory returns the attendance history for a specific user.
func (s *ReportService) GetUserHistory(userID string, page, limit int, dateFrom, dateTo *time.Time, status string) (*repository.AttendanceListResult, error) {
	params := repository.AttendanceListParams{
		Page:     page,
		Limit:    limit,
		UserID:   userID,
		DateFrom: dateFrom,
		DateTo:   dateTo,
		Status:   status,
	}
	return s.attendanceRepo.List(params)
}

// GetBranchReport returns the attendance history for a specific branch.
func (s *ReportService) GetBranchReport(branchID string, page, limit int, dateFrom, dateTo *time.Time, status string) (*repository.AttendanceListResult, error) {
	params := repository.AttendanceListParams{
		Page:     page,
		Limit:    limit,
		BranchID: branchID,
		DateFrom: dateFrom,
		DateTo:   dateTo,
		Status:   status,
	}
	return s.attendanceRepo.List(params)
}

// ExportUserHistoryExcel generates an Excel file containing the user's attendance history.
func (s *ReportService) ExportUserHistoryExcel(userID string, dateFrom, dateTo *time.Time, status string) ([]byte, error) {
	// Fetch all records without pagination (or with a large limit)
	params := repository.AttendanceListParams{
		Page:     1,
		Limit:    10000,
		UserID:   userID,
		DateFrom: dateFrom,
		DateTo:   dateTo,
		Status:   status,
	}
	result, err := s.attendanceRepo.List(params)
	if err != nil {
		log.Printf("[service][report] ERROR fetching user history for export: %v", err)
		return nil, err
	}

	log.Printf("[service][report] ExportUserHistoryExcel: user=%s records=%d", userID, len(result.Records))
	return s.generateExcel(result.Records)
}

// ExportBranchReportExcel generates an Excel file containing the branch's attendance history.
func (s *ReportService) ExportBranchReportExcel(branchID string, dateFrom, dateTo *time.Time, status string) ([]byte, error) {
	params := repository.AttendanceListParams{
		Page:     1,
		Limit:    10000,
		BranchID: branchID,
		DateFrom: dateFrom,
		DateTo:   dateTo,
		Status:   status,
	}
	result, err := s.attendanceRepo.List(params)
	if err != nil {
		log.Printf("[service][report] ERROR fetching branch history for export: %v", err)
		return nil, err
	}

	log.Printf("[service][report] ExportBranchReportExcel: branch=%s records=%d", branchID, len(result.Records))
	return s.generateExcel(result.Records)
}

func (s *ReportService) generateExcel(records []models.Attendance) ([]byte, error) {
	f := excelize.NewFile()
	defer func() {
		if err := f.Close(); err != nil {
			log.Printf("[service][report] excel close error: %v", err)
		}
	}()

	sheetName := "Sheet1"
	// Ensure the sheet exists and is active
	f.NewSheet(sheetName)
	index, _ := f.GetSheetIndex(sheetName)
	f.SetActiveSheet(index)

	// Set Headers
	headers := []string{"ID", "Họ Tên", "Chi Nhánh", "Ngày", "Giờ Vào", "Giờ Ra", "Trạng Thái", "Phương Thức", "Ghi Chú"}
	for i, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheetName, cell, header)
	}

	// Make Header bold
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
		if r.User != nil {
			f.SetCellValue(sheetName, fmt.Sprintf("B%d", row), r.User.FullName)
		} else {
			f.SetCellValue(sheetName, fmt.Sprintf("B%d", row), r.UserID)
		}

		if r.Branch != nil {
			f.SetCellValue(sheetName, fmt.Sprintf("C%d", row), r.Branch.Name)
		} else {
			f.SetCellValue(sheetName, fmt.Sprintf("C%d", row), r.BranchID)
		}

		if r.CheckInAt != nil {
			f.SetCellValue(sheetName, fmt.Sprintf("D%d", row), r.CheckInAt.Format("2006-01-02"))
			f.SetCellValue(sheetName, fmt.Sprintf("E%d", row), r.CheckInAt.Format("15:04:05"))
		}
		
		if r.CheckOutAt != nil {
			f.SetCellValue(sheetName, fmt.Sprintf("F%d", row), r.CheckOutAt.Format("15:04:05"))
		}

		f.SetCellValue(sheetName, fmt.Sprintf("G%d", row), string(r.Status))
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
