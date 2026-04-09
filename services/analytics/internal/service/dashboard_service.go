package service

import (
	"fmt"
	"log"
	"time"

	gocache "github.com/patrickmn/go-cache"
	"github.com/smart-attendance/analytics-service/internal/client"
)

// DashboardStats holds summary statistics for the dashboard.
type DashboardStats struct {
	TotalEmployees int64   `json:"total_employees"`
	TotalBranches  int64   `json:"total_branches"`
	TodayCheckIns  int64   `json:"today_check_ins"`
	TodayOnTime    int64   `json:"today_on_time"`
	TodayLate      int64   `json:"today_late"`
	TodayAbsent    int64   `json:"today_absent"`
	PendingLeave   int64   `json:"pending_leave"`
	OnTimeRate     float64 `json:"on_time_rate"`
	CheckedInRate  float64 `json:"checked_in_rate"`
}

type DashboardService struct {
	authClient       *client.AuthClient
	attendanceClient *client.AttendanceClient
	leaveClient      *client.LeaveClient
	orgClient        *client.OrgClient
	cache            *gocache.Cache
}

func NewDashboardService(
	authClient *client.AuthClient,
	attendanceClient *client.AttendanceClient,
	leaveClient *client.LeaveClient,
	orgClient *client.OrgClient,
) *DashboardService {
	return &DashboardService{
		authClient:       authClient,
		attendanceClient: attendanceClient,
		leaveClient:      leaveClient,
		orgClient:        orgClient,
		cache:            gocache.New(1*time.Minute, 2*time.Minute),
	}
}

// GetStats aggregates data from multiple services into dashboard statistics.
func (s *DashboardService) GetStats(branchID string) (*DashboardStats, error) {
	cacheKey := fmt.Sprintf("dashboard:stats:%s", branchID)
	if cached, ok := s.cache.Get(cacheKey); ok {
		return cached.(*DashboardStats), nil
	}

	stats := &DashboardStats{}

	// Total employees from Auth Service
	userCount, err := s.authClient.CountUsers(branchID)
	if err != nil {
		log.Printf("[service][dashboard] ERROR counting users: %v", err)
		// Non-fatal: continue with zero
	} else {
		stats.TotalEmployees = userCount
	}

	// Total branches from Org Service
	if branchID == "" {
		branches, err := s.orgClient.ListBranches()
		if err != nil {
			log.Printf("[service][dashboard] ERROR listing branches: %v", err)
		} else {
			stats.TotalBranches = int64(len(branches))
		}
	} else {
		stats.TotalBranches = 1
	}

	// Today's attendance data from Attendance Service
	dashData, err := s.attendanceClient.GetDashboardData(branchID)
	if err != nil {
		log.Printf("[service][dashboard] ERROR getting dashboard data: %v", err)
	} else {
		stats.TodayCheckIns = dashData.TodayCheckIns
		stats.TodayOnTime = dashData.TodayOnTime
		stats.TodayLate = dashData.TodayLate
		stats.TodayAbsent = dashData.TodayAbsent
	}

	// Pending leave from Leave Service
	if branchID != "" {
		pendingCount, err := s.leaveClient.GetPendingCount(branchID)
		if err != nil {
			log.Printf("[service][dashboard] ERROR getting pending leave: %v", err)
		} else {
			stats.PendingLeave = pendingCount
		}
	}

	// Calculate rates
	if stats.TodayCheckIns > 0 {
		stats.OnTimeRate = float64(stats.TodayOnTime) / float64(stats.TodayCheckIns) * 100
	}
	if stats.TotalEmployees > 0 {
		stats.CheckedInRate = float64(stats.TodayCheckIns) / float64(stats.TotalEmployees) * 100
		if stats.CheckedInRate > 100 {
			stats.CheckedInRate = 100
		}
	}

	s.cache.Set(cacheKey, stats, gocache.DefaultExpiration)
	return stats, nil
}

// GetChartData returns daily attendance data for the last N days.
func (s *DashboardService) GetChartData(branchID string, days int) ([]client.DailyAttendance, error) {
	cacheKey := fmt.Sprintf("dashboard:chart:%s:%d", branchID, days)
	if cached, ok := s.cache.Get(cacheKey); ok {
		return cached.([]client.DailyAttendance), nil
	}

	data, err := s.attendanceClient.GetChartData(branchID, days)
	if err != nil {
		log.Printf("[service][dashboard] ERROR getting chart data: %v", err)
		return nil, err
	}

	s.cache.Set(cacheKey, data, gocache.DefaultExpiration)
	return data, nil
}

// GetRecentActivity returns recent check-in records.
func (s *DashboardService) GetRecentActivity(branchID string, limit int) ([]client.RecentCheckIn, error) {
	cacheKey := fmt.Sprintf("dashboard:recent:%s:%d", branchID, limit)
	if cached, ok := s.cache.Get(cacheKey); ok {
		return cached.([]client.RecentCheckIn), nil
	}

	data, err := s.attendanceClient.GetRecentCheckIns(branchID, limit)
	if err != nil {
		log.Printf("[service][dashboard] ERROR getting recent activity: %v", err)
		return nil, err
	}

	s.cache.Set(cacheKey, data, gocache.DefaultExpiration)
	return data, nil
}

// InvalidateCache clears all dashboard cache entries.
func (s *DashboardService) InvalidateCache() {
	s.cache.Flush()
	log.Printf("[service][dashboard] cache invalidated")
}
