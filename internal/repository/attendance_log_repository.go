package repository

import (
	"github.com/smart-attendance/smart-attendance/internal/models"
	"gorm.io/gorm"
)

type AttendanceLogRepository struct {
	db *gorm.DB
}

func NewAttendanceLogRepository(db *gorm.DB) *AttendanceLogRepository {
	return &AttendanceLogRepository{db: db}
}

func (r *AttendanceLogRepository) Create(log *models.AttendanceLog) error {
	return r.db.Create(log).Error
}

// FindTodayLogs returns all time logs for a user on a given work date, ordered by logged_at ASC.
func (r *AttendanceLogRepository) FindTodayLogs(userID, workDate string) ([]models.AttendanceLog, error) {
	var logs []models.AttendanceLog
	err := r.db.Where("user_id = ? AND work_date = ?", userID, workDate).
		Order("logged_at ASC").
		Find(&logs).Error
	return logs, err
}

// CountTodayLogs returns how many logs a user has on a given work date.
func (r *AttendanceLogRepository) CountTodayLogs(userID, workDate string) (int64, error) {
	var count int64
	err := r.db.Model(&models.AttendanceLog{}).
		Where("user_id = ? AND work_date = ?", userID, workDate).
		Count(&count).Error
	return count, err
}
