package service

import (
	"fmt"
	"log"
	"time"

	"github.com/smart-attendance/smart-attendance/internal/cache"
	"github.com/smart-attendance/smart-attendance/internal/models"
	"github.com/smart-attendance/smart-attendance/internal/repository"
	"github.com/smart-attendance/smart-attendance/internal/timezone"
	"gorm.io/gorm"
)

type DashboardService struct {
	attendanceRepo *repository.AttendanceRepository
	branchRepo     *repository.BranchRepository
	userRepo       *repository.UserRepository
	cache          *cache.Cache
	db             *gorm.DB
}

func NewDashboardService(
	attendanceRepo *repository.AttendanceRepository,
	branchRepo *repository.BranchRepository,
	userRepo *repository.UserRepository,
	cache *cache.Cache,
	db *gorm.DB,
) *DashboardService {
	return &DashboardService{
		attendanceRepo: attendanceRepo,
		branchRepo:     branchRepo,
		userRepo:       userRepo,
		cache:          cache,
		db:             db,
	}
}

const dashboardCacheTTL = 5 * time.Minute

// GetStats returns summary statistics for the dashboard.
func (s *DashboardService) GetStats(branchID string) (*repository.DashboardStats, error) {
	cacheKey := fmt.Sprintf("dashboard:stats:%s", branchID)
	if cached, ok := s.cache.Get(cacheKey); ok {
		return cached.(*repository.DashboardStats), nil
	}

	stats := &repository.DashboardStats{}

	// Total employees
	employees, err := s.attendanceRepo.CountActiveEmployees(s.db, branchID)
	if err != nil {
		log.Printf("[service][dashboard] ERROR counting employees: %v", err)
		return nil, err
	}
	stats.TotalEmployees = employees

	// Total branches
	if branchID == "" {
		branches, err := s.branchRepo.Count()
		if err != nil {
			log.Printf("[service][dashboard] ERROR counting branches: %v", err)
			return nil, err
		}
		stats.TotalBranches = branches
	} else {
		stats.TotalBranches = 1
	}

	// Today's check-ins
	todayTotal, err := s.attendanceRepo.CountTodayCheckIns(branchID)
	if err != nil {
		log.Printf("[service][dashboard] ERROR counting today check-ins: %v", err)
		return nil, err
	}
	stats.TodayCheckIns = todayTotal

	// Today's on-time
	onTime, err := s.attendanceRepo.CountTodayByStatus(branchID, models.StatusOnTime)
	if err != nil {
		log.Printf("[service][dashboard] ERROR counting on-time: %v", err)
		return nil, err
	}
	stats.TodayOnTime = onTime

	// Today's late
	late, err := s.attendanceRepo.CountTodayByStatus(branchID, models.StatusLate)
	if err != nil {
		log.Printf("[service][dashboard] ERROR counting late: %v", err)
		return nil, err
	}
	stats.TodayLate = late

	// Today's absent
	absent, err := s.attendanceRepo.CountTodayByStatus(branchID, models.StatusAbsent)
	if err != nil {
		log.Printf("[service][dashboard] ERROR counting absent: %v", err)
		return nil, err
	}
	stats.TodayAbsent = absent

	// Rates
	if todayTotal > 0 {
		stats.OnTimeRate = float64(onTime) / float64(todayTotal) * 100
	}
	if employees > 0 {
		stats.CheckedInRate = float64(todayTotal) / float64(employees) * 100
		if stats.CheckedInRate > 100 {
			stats.CheckedInRate = 100
		}
	}

	s.cache.Set(cacheKey, stats, dashboardCacheTTL)
	return stats, nil
}

// GetChartData returns daily attendance data for the last N days.
func (s *DashboardService) GetChartData(branchID string, days int) ([]repository.DailyAttendance, error) {
	cacheKey := fmt.Sprintf("dashboard:chart:%s:%d", branchID, days)
	if cached, ok := s.cache.Get(cacheKey); ok {
		return cached.([]repository.DailyAttendance), nil
	}

	data, err := s.attendanceRepo.DailyStats(branchID, days)
	if err != nil {
		log.Printf("[service][dashboard] ERROR getting chart data: %v", err)
		return nil, err
	}

	s.cache.Set(cacheKey, data, dashboardCacheTTL)
	return data, nil
}

// GetTopLate returns top late users for this month.
func (s *DashboardService) GetTopLate(branchID string, limit int) ([]repository.TopLateUser, error) {
	cacheKey := fmt.Sprintf("dashboard:toplate:%s:%d", branchID, limit)
	if cached, ok := s.cache.Get(cacheKey); ok {
		return cached.([]repository.TopLateUser), nil
	}

	now := timezone.Now()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	monthEnd := monthStart.AddDate(0, 1, 0)

	data, err := s.attendanceRepo.TopLateUsers(branchID, limit, monthStart, monthEnd)
	if err != nil {
		log.Printf("[service][dashboard] ERROR getting top late: %v", err)
		return nil, err
	}

	s.cache.Set(cacheKey, data, dashboardCacheTTL)
	return data, nil
}

// GetRecentActivity returns recent check-in records.
func (s *DashboardService) GetRecentActivity(branchID string, limit int) ([]repository.RecentCheckIn, error) {
	cacheKey := fmt.Sprintf("dashboard:recent:%s:%d", branchID, limit)
	if cached, ok := s.cache.Get(cacheKey); ok {
		return cached.([]repository.RecentCheckIn), nil
	}

	data, err := s.attendanceRepo.RecentCheckIns(branchID, limit)
	if err != nil {
		log.Printf("[service][dashboard] ERROR getting recent activity: %v", err)
		return nil, err
	}

	s.cache.Set(cacheKey, data, dashboardCacheTTL)
	return data, nil
}

// InvalidateCache clears all dashboard cache entries.
func (s *DashboardService) InvalidateCache() {
	s.cache.Flush()
	log.Printf("[service][dashboard] cache invalidated")
}
