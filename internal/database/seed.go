package database

import (
	"crypto/rand"
	"encoding/base32"
	"log"

	"github.com/smart-attendance/smart-attendance/internal/models"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func Seed(db *gorm.DB) error {
	var userCount int64
	db.Model(&models.User{}).Count(&userCount)
	if userCount > 0 {
		return nil
	}

	hash, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	pw := string(hash)

	// 1. Create branch
	secret := make([]byte, 20)
	rand.Read(secret)

	branch := &models.Branch{
		Name:           "HQ - Ho Chi Minh",
		Address:        "123 Nguyen Hue, Quan 1, TP.HCM",
		Lat:            floatPtr(10.773889),
		Lng:            floatPtr(106.701944),
		RadiusM:        500,
		TOTPSecret:     base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(secret),
		AllowedMethods: "qr_totp,ip,location",
		WorkStartTime:  "08:00",
		WorkEndTime:    "17:00",
		IsActive:       true,
	}
	if err := db.Create(branch).Error; err != nil {
		return err
	}

	// Add IP whitelist
	db.Create(&models.BranchIPWhitelist{BranchID: branch.ID, IPCIDR: "127.0.0.1/32", Label: "Localhost"})
	db.Create(&models.BranchIPWhitelist{BranchID: branch.ID, IPCIDR: "192.168.1.0/24", Label: "Office LAN"})

	// Add location
	db.Create(&models.BranchLocation{BranchID: branch.ID, Lat: 10.773889, Lng: 106.701944, RadiusM: 500, Label: "Main Office"})

	log.Printf("[seed] created branch: %s (%s)", branch.Name, branch.ID)

	// 2. Admin
	admin := &models.User{
		Email:        "admin@smartattendance.com",
		PasswordHash: pw,
		FullName:     "System Admin",
		Role:         models.RoleAdmin,
		IsActive:     true,
	}
	db.Create(admin)
	log.Printf("[seed] admin: admin@smartattendance.com / password123")

	// 3. Manager — generates QR code for branch
	manager := &models.User{
		Email:        "manager@smartattendance.com",
		PasswordHash: pw,
		FullName:     "Branch Manager",
		Role:         models.RoleManager,
		BranchID:     &branch.ID,
		IsActive:     true,
	}
	db.Create(manager)
	log.Printf("[seed] manager (QR generator): manager@smartattendance.com / password123 → branch: %s", branch.Name)

	// 4. Employee — scans QR to check-in
	employee := &models.User{
		Email:        "employee@smartattendance.com",
		PasswordHash: pw,
		FullName:     "Nguyen Van A",
		Role:         models.RoleEmployee,
		BranchID:     &branch.ID,
		IsActive:     true,
	}
	db.Create(employee)
	log.Printf("[seed] employee (QR scanner): employee@smartattendance.com / password123 → branch: %s", branch.Name)

	return nil
}

func floatPtr(f float64) *float64 {
	return &f
}
