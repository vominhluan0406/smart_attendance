package repository

import (
	"time"

	"github.com/smart-attendance/attendance-service/internal/model"
	"gorm.io/gorm"
)

type AttendanceRepository struct {
	db *gorm.DB
}

func NewAttendanceRepository(db *gorm.DB) *AttendanceRepository {
	return &AttendanceRepository{db: db}
}

func (r *AttendanceRepository) Create(att *model.Attendance) error {
	return r.db.Create(att).Error
}

func (r *AttendanceRepository) Update(att *model.Attendance) error {
	return r.db.Save(att).Error
}

func (r *AttendanceRepository) FindByID(id string) (*model.Attendance, error) {
	var att model.Attendance
	if err := r.db.First(&att, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &att, nil
}

// FindTodayByUser returns today's attendance record for a user (if any).
func (r *AttendanceRepository) FindTodayByUser(userID string) (*model.Attendance, error) {
	today := time.Now().Format("2006-01-02")
	return r.FindTodayByUserAndDate(userID, today)
}

func (r *AttendanceRepository) FindTodayByUserAndDate(userID string, dateStr string) (*model.Attendance, error) {
	var att model.Attendance
	err := r.db.Where("user_id = ? AND work_date = ?", userID, dateStr).
		First(&att).Error
	if err != nil {
		return nil, err
	}
	return &att, nil
}

type AttendanceListParams struct {
	Page     int
	Limit    int
	UserID   string
	BranchID string
	Status   string
	DateFrom *time.Time
	DateTo   *time.Time
}

type AttendanceListResult struct {
	Records []model.Attendance
	Total   int64
	Page    int
	Limit   int
}

func (r *AttendanceRepository) List(params AttendanceListParams) (*AttendanceListResult, error) {
	if params.Page < 1 {
		params.Page = 1
	}
	if params.Limit < 1 || params.Limit > 10000 {
		params.Limit = 20
	}

	query := r.db.Model(&model.Attendance{})

	if params.UserID != "" {
		query = query.Where("user_id = ?", params.UserID)
	}
	if params.BranchID != "" {
		query = query.Where("branch_id = ?", params.BranchID)
	}
	if params.Status != "" {
		query = query.Where("status = ?", params.Status)
	}
	if params.DateFrom != nil {
		query = query.Where("work_date >= ?", params.DateFrom.Format("2006-01-02"))
	}
	if params.DateTo != nil {
		query = query.Where("work_date <= ?", params.DateTo.Format("2006-01-02"))
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	// Build fresh query for Find to avoid side-effects from Count
	findQuery := r.db.Model(&model.Attendance{})
	if params.UserID != "" {
		findQuery = findQuery.Where("user_id = ?", params.UserID)
	}
	if params.BranchID != "" {
		findQuery = findQuery.Where("branch_id = ?", params.BranchID)
	}
	if params.Status != "" {
		findQuery = findQuery.Where("status = ?", params.Status)
	}
	if params.DateFrom != nil {
		findQuery = findQuery.Where("work_date >= ?", params.DateFrom.Format("2006-01-02"))
	}
	if params.DateTo != nil {
		findQuery = findQuery.Where("work_date <= ?", params.DateTo.Format("2006-01-02"))
	}

	var records []model.Attendance
	offset := (params.Page - 1) * params.Limit
	if err := findQuery.Offset(offset).Limit(params.Limit).
		Order("work_date DESC, check_in_at DESC").
		Find(&records).Error; err != nil {
		return nil, err
	}

	return &AttendanceListResult{
		Records: records,
		Total:   total,
		Page:    params.Page,
		Limit:   params.Limit,
	}, nil
}

// InvalidateByUserAndDate sets the attendance status to "invalidated" for a user on a specific date.
// Returns the number of affected rows.
func (r *AttendanceRepository) InvalidateByUserAndDate(userID, workDate, note string) (int64, error) {
	result := r.db.Model(&model.Attendance{}).
		Where("user_id = ? AND work_date = ? AND status != ?", userID, workDate, model.StatusInvalidated).
		Updates(map[string]interface{}{
			"status": model.StatusInvalidated,
			"note":   note,
		})
	return result.RowsAffected, result.Error
}

// FindLatestByUser returns the most recent non-invalidated attendance record for a user.
func (r *AttendanceRepository) FindLatestByUser(userID string) (*model.Attendance, error) {
	var att model.Attendance
	err := r.db.Where("user_id = ? AND status != ?", userID, model.StatusInvalidated).
		Order("work_date DESC").First(&att).Error
	if err != nil {
		return nil, err
	}
	return &att, nil
}

// RecentCheckIn holds a recent check-in record for display.
type RecentCheckIn struct {
	UserID    string
	BranchID  string
	CheckInAt *time.Time
	Status    model.AttendanceStatus
	Method    string
}

// RecentCheckIns returns the most recent check-in records, optionally filtered by branch.
func (r *AttendanceRepository) RecentCheckIns(branchID string, limit int) ([]RecentCheckIn, error) {
	today := time.Now().Format("2006-01-02")
	query := r.db.Model(&model.Attendance{}).
		Select("user_id, branch_id, check_in_at, status, method").
		Where("work_date = ?", today)
	if branchID != "" {
		query = query.Where("branch_id = ?", branchID)
	}

	var results []RecentCheckIn
	err := query.Order("check_in_at DESC").Limit(limit).Find(&results).Error
	return results, err
}
