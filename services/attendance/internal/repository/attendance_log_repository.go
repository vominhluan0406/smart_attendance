package repository

import (
	"fmt"

	"github.com/smart-attendance/attendance-service/internal/model"
	"gorm.io/gorm"
)

type AttendanceLogRepository struct {
	db *gorm.DB
}

func NewAttendanceLogRepository(db *gorm.DB) *AttendanceLogRepository {
	return &AttendanceLogRepository{db: db}
}

func (r *AttendanceLogRepository) Create(log *model.AttendanceLog) error {
	return r.db.Create(log).Error
}

// FindByID returns a single attendance log by ID (used for WAL idempotency check).
func (r *AttendanceLogRepository) FindByID(id string) (*model.AttendanceLog, error) {
	var log model.AttendanceLog
	if err := r.db.First(&log, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &log, nil
}

// FindTodayLogs returns all time logs for a user on a given work date, ordered by logged_at ASC.
func (r *AttendanceLogRepository) FindTodayLogs(userID, workDate string) ([]model.AttendanceLog, error) {
	var logs []model.AttendanceLog
	err := r.db.Where("user_id = ? AND work_date = ?", userID, workDate).
		Order("logged_at ASC").
		Find(&logs).Error
	return logs, err
}

// CountTodayLogs returns how many logs a user has on a given work date.
func (r *AttendanceLogRepository) CountTodayLogs(userID, workDate string) (int64, error) {
	var count int64
	err := r.db.Model(&model.AttendanceLog{}).
		Where("user_id = ? AND work_date = ?", userID, workDate).
		Count(&count).Error
	return count, err
}

// FindLastWithLocation returns the most recent log for a user that has GPS coordinates.
func (r *AttendanceLogRepository) FindLastWithLocation(userID string) (*model.AttendanceLog, error) {
	var log model.AttendanceLog
	err := r.db.Where("user_id = ? AND lat IS NOT NULL AND lng IS NOT NULL", userID).
		Order("logged_at DESC").First(&log).Error
	if err != nil {
		return nil, err
	}
	return &log, nil
}

// FindRecentFirstLogs returns the first log of each day for the last N days (for anomaly detection).
func (r *AttendanceLogRepository) FindRecentFirstLogs(userID string, days int) ([]model.AttendanceLog, error) {
	var logs []model.AttendanceLog
	// Get the earliest log per work_date for the last N days (PostgreSQL syntax)
	err := r.db.Raw(`
		SELECT al.* FROM attendance_logs al
		INNER JOIN (
			SELECT work_date, MIN(logged_at) as min_logged_at
			FROM attendance_logs
			WHERE user_id = ? AND work_date >= (CURRENT_DATE - INTERVAL '1 day' * ?)::text
			GROUP BY work_date
		) sub ON al.work_date = sub.work_date AND al.logged_at = sub.min_logged_at
		WHERE al.user_id = ?
		ORDER BY al.work_date DESC
	`, userID, fmt.Sprintf("%d", days), userID).Scan(&logs).Error
	return logs, err
}
