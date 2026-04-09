package database

import (
	"log"

	"github.com/smart-attendance/leave-service/internal/model"
	"gorm.io/gorm"
)

// SeedLeaveTypes seeds the 7 default leave types if they don't already exist.
func SeedLeaveTypes(db *gorm.DB) error {
	defaultTypes := []model.LeaveType{
		{
			Name:             "Nghỉ phép năm",
			Code:             "ANNUAL",
			MaxDaysPerYear:   12,
			IsPaid:           true,
			RequiresApproval: true,
			Color:            "#10B981",
			IsActive:         true,
		},
		{
			Name:             "Nghỉ ốm",
			Code:             "SICK",
			MaxDaysPerYear:   30,
			IsPaid:           true,
			RequiresApproval: true,
			Color:            "#EF4444",
			IsActive:         true,
		},
		{
			Name:             "Nghỉ việc riêng",
			Code:             "PERSONAL",
			MaxDaysPerYear:   3,
			IsPaid:           false,
			RequiresApproval: true,
			Color:            "#F59E0B",
			IsActive:         true,
		},
		{
			Name:             "Nghỉ cưới",
			Code:             "WEDDING",
			MaxDaysPerYear:   3,
			IsPaid:           true,
			RequiresApproval: true,
			Color:            "#EC4899",
			IsActive:         true,
		},
		{
			Name:             "Nghỉ tang",
			Code:             "BEREAVEMENT",
			MaxDaysPerYear:   3,
			IsPaid:           true,
			RequiresApproval: true,
			Color:            "#6B7280",
			IsActive:         true,
		},
		{
			Name:             "Nghỉ thai sản",
			Code:             "MATERNITY",
			MaxDaysPerYear:   180,
			IsPaid:           true,
			RequiresApproval: true,
			Color:            "#8B5CF6",
			IsActive:         true,
		},
		{
			Name:             "Nghỉ không lương",
			Code:             "UNPAID",
			MaxDaysPerYear:   365,
			IsPaid:           false,
			RequiresApproval: true,
			Color:            "#9CA3AF",
			IsActive:         true,
		},
	}

	for _, lt := range defaultTypes {
		var existing model.LeaveType
		result := db.Where("code = ?", lt.Code).First(&existing)
		if result.Error == nil {
			// Already exists, skip
			continue
		}

		if err := db.Create(&lt).Error; err != nil {
			log.Printf("[leave][seed] ERROR creating leave type %s: %v", lt.Code, err)
			return err
		}
		log.Printf("[leave][seed] created leave type: %s (%s)", lt.Name, lt.Code)
	}

	log.Printf("[leave][seed] leave types seeding completed")
	return nil
}
