package repository

import (
	"time"

	"github.com/smart-attendance/smart-attendance/internal/models"
	"github.com/smart-attendance/smart-attendance/internal/timezone"
	"gorm.io/gorm"
)

// DashboardStats holds summary statistics for the dashboard.
type DashboardStats struct {
	TotalEmployees   int64
	TotalBranches    int64
	TodayCheckIns    int64
	TodayOnTime      int64
	TodayLate        int64
	TodayAbsent      int64
	TodayLeave       int64
	OnTimeRate       float64
	CheckedInRate    float64
}

// DailyAttendance holds aggregated attendance for one day.
type DailyAttendance struct {
	Date    string
	Total   int64
	OnTime  int64
	Late    int64
	Absent  int64
}

// TopLateUser holds a user and their late count.
type TopLateUser struct {
	UserID   string
	FullName string
	Email    string
	LateCount int64
}

// RecentCheckIn holds a recent check-in record for display.
type RecentCheckIn struct {
	UserName  string
	BranchName string
	CheckInAt  *time.Time
	Status     models.AttendanceStatus
	Method     string
}

// CountTodayCheckIns counts today's check-ins, optionally filtered by branch.
// Uses work_date index for fast lookup.
func (r *AttendanceRepository) CountTodayCheckIns(branchID string) (int64, error) {
	today := todayStr()
	query := r.db.Model(&models.Attendance{}).
		Where("work_date = ?", today)
	if branchID != "" {
		query = query.Where("branch_id = ?", branchID)
	}
	var count int64
	return count, query.Count(&count).Error
}

// CountTodayByStatus counts today's check-ins by status, optionally filtered by branch.
// Uses composite index (work_date, status).
func (r *AttendanceRepository) CountTodayByStatus(branchID string, status models.AttendanceStatus) (int64, error) {
	today := todayStr()
	query := r.db.Model(&models.Attendance{}).
		Where("work_date = ? AND status = ?", today, status)
	if branchID != "" {
		query = query.Where("branch_id = ?", branchID)
	}
	var count int64
	return count, query.Count(&count).Error
}

// DailyStats returns aggregated daily attendance for the last N days, optionally filtered by branch.
// Uses work_date column (index-friendly) instead of strftime(check_in_at).
func (r *AttendanceRepository) DailyStats(branchID string, days int) ([]DailyAttendance, error) {
	now := timezone.Now()
	startDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).AddDate(0, 0, -(days-1))
	startStr := startDate.Format("2006-01-02")

	type row struct {
		Day    string
		Status string
		Cnt    int64
	}

	query := r.db.Model(&models.Attendance{}).
		Select("work_date as day, status, count(*) as cnt").
		Where("work_date >= ?", startStr).
		Group("work_date, status").
		Order("work_date ASC")

	if branchID != "" {
		query = query.Where("branch_id = ?", branchID)
	}

	var rows []row
	if err := query.Find(&rows).Error; err != nil {
		return nil, err
	}

	// Build map for easy lookup
	type dayKey struct {
		Day    string
		Status string
	}
	countMap := make(map[dayKey]int64)
	for _, r := range rows {
		countMap[dayKey{r.Day, r.Status}] = r.Cnt
	}

	// Build result for each day
	result := make([]DailyAttendance, 0, days)
	for i := 0; i < days; i++ {
		d := startDate.AddDate(0, 0, i)
		dayStr := d.Format("2006-01-02")
		da := DailyAttendance{
			Date:   dayStr,
			OnTime: countMap[dayKey{dayStr, string(models.StatusOnTime)}],
			Late:   countMap[dayKey{dayStr, string(models.StatusLate)}],
			Absent: countMap[dayKey{dayStr, string(models.StatusAbsent)}],
		}
		da.Total = da.OnTime + da.Late + da.Absent
		result = append(result, da)
	}

	return result, nil
}

// TopLateUsers returns users with the most late check-ins in the given date range.
// Uses work_date for index-friendly date filtering.
func (r *AttendanceRepository) TopLateUsers(branchID string, limit int, dateFrom, dateTo time.Time) ([]TopLateUser, error) {
	fromStr := dateFrom.Format("2006-01-02")
	toStr := dateTo.Format("2006-01-02")

	query := r.db.Model(&models.Attendance{}).
		Select("attendances.user_id, users.full_name, users.email, count(*) as late_count").
		Joins("JOIN users ON users.id = attendances.user_id").
		Where("attendances.status = ? AND attendances.work_date >= ? AND attendances.work_date < ?",
			models.StatusLate, fromStr, toStr).
		Group("attendances.user_id").
		Order("late_count DESC").
		Limit(limit)

	if branchID != "" {
		query = query.Where("attendances.branch_id = ?", branchID)
	}

	var results []TopLateUser
	return results, query.Find(&results).Error
}

// RecentCheckIns returns the most recent check-in records, optionally filtered by branch.
func (r *AttendanceRepository) RecentCheckIns(branchID string, limit int) ([]RecentCheckIn, error) {
	query := r.db.Model(&models.Attendance{}).
		Select("users.full_name as user_name, branches.name as branch_name, attendances.check_in_at, attendances.status, attendances.method").
		Joins("JOIN users ON users.id = attendances.user_id").
		Joins("JOIN branches ON branches.id = attendances.branch_id").
		Order("attendances.check_in_at DESC").
		Limit(limit)

	if branchID != "" {
		query = query.Where("attendances.branch_id = ?", branchID)
	}

	var results []RecentCheckIn
	return results, query.Find(&results).Error
}

// CountActiveEmployees counts active employees, optionally by branch.
func (r *AttendanceRepository) CountActiveEmployees(db *gorm.DB, branchID string) (int64, error) {
	query := db.Model(&models.User{}).Where("is_active = ?", true)
	if branchID != "" {
		query = query.Where("branch_id = ?", branchID)
	}
	var count int64
	return count, query.Count(&count).Error
}

func todayRange() (time.Time, time.Time) {
	now := timezone.Now()
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	end := start.Add(24 * time.Hour)
	return start, end
}

func todayStr() string {
	return timezone.Now().Format("2006-01-02")
}
