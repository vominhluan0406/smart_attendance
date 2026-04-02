package service

import (
	"log"
	"time"

	"github.com/smart-attendance/smart-attendance/internal/models"
	"github.com/smart-attendance/smart-attendance/internal/timezone"
	"gorm.io/gorm"
)

type GamificationService struct {
	db *gorm.DB
}

func NewGamificationService(db *gorm.DB) *GamificationService {
	return &GamificationService{db: db}
}

// UpdateStreak increments the user's streak if they check in on a new day.
func (s *GamificationService) UpdateStreak(userID string) (*models.UserStreak, error) {
	var streak models.UserStreak
	now := timezone.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	err := s.db.Where("user_id = ?", userID).First(&streak).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// Create first streak
			streak = models.UserStreak{
				UserID:       userID,
				CurrentCount: 1,
				MaxCount:     1,
				LastCheckIn:  now,
			}
			if err := s.db.Create(&streak).Error; err != nil {
				return nil, err
			}
			return &streak, nil
		}
		return nil, err
	}

	lastCheckInDay := time.Date(streak.LastCheckIn.Year(), streak.LastCheckIn.Month(), streak.LastCheckIn.Day(), 0, 0, 0, 0, now.Location())

	if today.Equal(lastCheckInDay) {
		// Already checked in today, no streak change
		return &streak, nil
	}

	yesterday := today.AddDate(0, 0, -1)
	if lastCheckInDay.Equal(yesterday) {
		// Progressive streak
		streak.CurrentCount++
		if streak.CurrentCount > streak.MaxCount {
			streak.MaxCount = streak.CurrentCount
		}
	} else {
		// Reset streak
		streak.CurrentCount = 1
	}

	streak.LastCheckIn = now
	if err := s.db.Save(&streak).Error; err != nil {
		return nil, err
	}

	// Logic for awarding badges based on streak
	s.CheckAndAwardBadges(userID, &streak)

	return &streak, nil
}

func (s *GamificationService) CheckAndAwardBadges(userID string, streak *models.UserStreak) {
	if streak.CurrentCount == 7 {
		s.AwardBadge(userID, models.BadgePerfectWeek, "7-day check-in streak!")
	} else if streak.CurrentCount == 30 {
		s.AwardBadge(userID, models.BadgePerfectMonth, "30-day check-in streak! Amazing consistency.")
	}
}

func (s *GamificationService) AwardBadge(userID string, badgeType models.BadgeType, reason string) {
	var existing int64
	// Check if already earned today/recently to avoid duplicates
	s.db.Model(&models.UserBadge{}).Where("user_id = ? AND badge_type = ?", userID, badgeType).Count(&existing)
	
	if existing == 0 {
		badge := models.UserBadge{
			UserID:    userID,
			BadgeType: badgeType,
			EarnedAt:  timezone.Now(),
			Reason:    reason,
		}
		if err := s.db.Create(&badge).Error; err != nil {
			log.Printf("[gamification] failed to award badge: %v", err)
		}
	}
}
