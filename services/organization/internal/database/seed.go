package database

import (
	"crypto/rand"
	"encoding/base32"
	"log"

	"github.com/smart-attendance/organization-service/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Seed seeds default branches, departments, and shifts.
func Seed(db *gorm.DB) error {
	// Silence logging during seeding
	originalLogger := db.Logger
	db.Logger = originalLogger.LogMode(logger.Warn)
	defer func() { db.Logger = originalLogger }()

	return db.Transaction(func(tx *gorm.DB) error {
		if err := seedBranches(tx); err != nil {
			return err
		}
		return nil
	})
}

func seedBranches(db *gorm.DB) error {
	var count int64
	if err := db.Model(&model.Branch{}).Count(&count).Error; err != nil {
		log.Printf("[org][seed] ERROR: count branches failed: %v", err)
		return err
	}

	if count > 0 {
		log.Printf("[org][seed] branches table has %d rows, skipping seed", count)
		return nil
	}

	latHCM := 10.7769
	lngHCM := 106.7009
	latHN := 21.0285
	lngHN := 105.8542
	latDN := 16.0544
	lngDN := 108.2022

	branches := []model.Branch{
		{
			Name:             "HCM - Quận 1",
			Address:          "123 Nguyễn Huệ, Quận 1, TP.HCM",
			Lat:              &latHCM,
			Lng:              &lngHCM,
			RadiusM:          200,
			TOTPSecret:       generateTOTPSecret(),
			AllowedMethods:   "qr_totp,ip,location",
			RequireBiometric: false,
			WorkStartTime:    "08:00",
			WorkEndTime:      "17:00",
			IsActive:         true,
		},
		{
			Name:             "Hà Nội - Cầu Giấy",
			Address:          "456 Trần Duy Hưng, Cầu Giấy, Hà Nội",
			Lat:              &latHN,
			Lng:              &lngHN,
			RadiusM:          250,
			TOTPSecret:       generateTOTPSecret(),
			AllowedMethods:   "qr_totp,ip,location,face",
			RequireBiometric: true,
			WorkStartTime:    "08:30",
			WorkEndTime:      "17:30",
			IsActive:         true,
		},
		{
			Name:             "Đà Nẵng - Hải Châu",
			Address:          "789 Bạch Đằng, Hải Châu, Đà Nẵng",
			Lat:              &latDN,
			Lng:              &lngDN,
			RadiusM:          150,
			TOTPSecret:       generateTOTPSecret(),
			AllowedMethods:   "qr_totp,location",
			RequireBiometric: false,
			WorkStartTime:    "08:00",
			WorkEndTime:      "17:00",
			IsActive:         true,
		},
	}

	for i := range branches {
		if err := db.Create(&branches[i]).Error; err != nil {
			log.Printf("[org][seed] ERROR: failed to create branch %s: %v", branches[i].Name, err)
			return err
		}
	}

	log.Printf("[org][seed] created %d default branches", len(branches))

	// Seed IP whitelists
	ipWhitelists := []model.BranchIPWhitelist{
		{BranchID: branches[0].ID, IPCIDR: "192.168.1.0/24", Label: "HCM Office LAN"},
		{BranchID: branches[0].ID, IPCIDR: "10.0.1.0/24", Label: "HCM Office VPN"},
		{BranchID: branches[1].ID, IPCIDR: "192.168.2.0/24", Label: "HN Office LAN"},
		{BranchID: branches[2].ID, IPCIDR: "192.168.3.0/24", Label: "DN Office LAN"},
	}

	for i := range ipWhitelists {
		if err := db.Create(&ipWhitelists[i]).Error; err != nil {
			log.Printf("[org][seed] warning: failed to create IP whitelist: %v", err)
		}
	}

	log.Printf("[org][seed] created %d IP whitelist entries", len(ipWhitelists))

	// Seed branch locations
	locations := []model.BranchLocation{
		{BranchID: branches[0].ID, Label: "HCM Main Entrance", Lat: 10.7769, Lng: 106.7009, RadiusM: 200},
		{BranchID: branches[0].ID, Label: "HCM Parking", Lat: 10.7771, Lng: 106.7011, RadiusM: 100},
		{BranchID: branches[1].ID, Label: "HN Main Entrance", Lat: 21.0285, Lng: 105.8542, RadiusM: 250},
		{BranchID: branches[2].ID, Label: "DN Main Entrance", Lat: 16.0544, Lng: 108.2022, RadiusM: 150},
	}

	for i := range locations {
		if err := db.Create(&locations[i]).Error; err != nil {
			log.Printf("[org][seed] warning: failed to create location: %v", err)
		}
	}

	log.Printf("[org][seed] created %d branch locations", len(locations))

	// Seed departments
	departments := []model.Department{
		{BranchID: branches[0].ID, Name: "Kỹ thuật", Code: "TECH-HCM", IsActive: true},
		{BranchID: branches[0].ID, Name: "Nhân sự", Code: "HR-HCM", IsActive: true},
		{BranchID: branches[0].ID, Name: "Kinh doanh", Code: "SALES-HCM", IsActive: true},
		{BranchID: branches[1].ID, Name: "Kỹ thuật", Code: "TECH-HN", IsActive: true},
		{BranchID: branches[1].ID, Name: "Nhân sự", Code: "HR-HN", IsActive: true},
		{BranchID: branches[2].ID, Name: "Kỹ thuật", Code: "TECH-DN", IsActive: true},
		{BranchID: branches[2].ID, Name: "Kinh doanh", Code: "SALES-DN", IsActive: true},
	}

	for i := range departments {
		if err := db.Create(&departments[i]).Error; err != nil {
			log.Printf("[org][seed] warning: failed to create department: %v", err)
		}
	}

	log.Printf("[org][seed] created %d departments", len(departments))

	// Seed work shifts
	shifts := []model.WorkShift{
		{
			BranchID:             branches[0].ID,
			Name:                 "Ca sáng",
			Code:                 "MORNING",
			StartTime:            "08:00",
			EndTime:              "17:00",
			GracePeriodMinutes:   15,
			LateThresholdMinutes: 0,
			BreakDurationMinutes: 60,
			WorkingDays:          "1,2,3,4,5",
			Color:                "#3B82F6",
			IsDefault:            true,
			IsActive:             true,
		},
		{
			BranchID:             branches[0].ID,
			Name:                 "Ca chiều",
			Code:                 "AFTERNOON",
			StartTime:            "13:00",
			EndTime:              "22:00",
			GracePeriodMinutes:   15,
			LateThresholdMinutes: 0,
			BreakDurationMinutes: 60,
			WorkingDays:          "1,2,3,4,5",
			Color:                "#F59E0B",
			IsDefault:            false,
			IsActive:             true,
		},
		{
			BranchID:             branches[1].ID,
			Name:                 "Ca hành chính",
			Code:                 "OFFICE",
			StartTime:            "08:30",
			EndTime:              "17:30",
			GracePeriodMinutes:   15,
			LateThresholdMinutes: 0,
			BreakDurationMinutes: 60,
			WorkingDays:          "1,2,3,4,5",
			Color:                "#3B82F6",
			IsDefault:            true,
			IsActive:             true,
		},
		{
			BranchID:             branches[2].ID,
			Name:                 "Ca sáng",
			Code:                 "MORNING",
			StartTime:            "08:00",
			EndTime:              "17:00",
			GracePeriodMinutes:   15,
			LateThresholdMinutes: 0,
			BreakDurationMinutes: 60,
			WorkingDays:          "1,2,3,4,5",
			Color:                "#3B82F6",
			IsDefault:            true,
			IsActive:             true,
		},
	}

	for i := range shifts {
		if err := db.Create(&shifts[i]).Error; err != nil {
			log.Printf("[org][seed] warning: failed to create shift: %v", err)
		}
	}

	log.Printf("[org][seed] created %d work shifts", len(shifts))

	return nil
}

func generateTOTPSecret() string {
	secret := make([]byte, 20)
	rand.Read(secret)
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(secret)
}
