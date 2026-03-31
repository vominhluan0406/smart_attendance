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
	// These seeds are idempotent (each checks own count) — always run
	seedLeaveTypes(db)
	seedHolidays(db)
	seedPermissions(db)

	// Users: only seed if DB is empty
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

func seedLeaveTypes(db *gorm.DB) {
	var count int64
	db.Model(&models.LeaveType{}).Count(&count)
	if count > 0 {
		return
	}

	types := []models.LeaveType{
		{Name: "Nghỉ phép năm", Code: "ANNUAL", MaxDaysPerYear: 12, IsPaid: true, RequiresApproval: true, Color: "#10B981"},
		{Name: "Nghỉ ốm", Code: "SICK", MaxDaysPerYear: 30, IsPaid: true, RequiresApproval: true, Color: "#F59E0B"},
		{Name: "Nghỉ việc riêng", Code: "PERSONAL", MaxDaysPerYear: 3, IsPaid: false, RequiresApproval: true, Color: "#8B5CF6"},
		{Name: "Nghỉ cưới", Code: "WEDDING", MaxDaysPerYear: 3, IsPaid: true, RequiresApproval: true, Color: "#EC4899"},
		{Name: "Nghỉ tang", Code: "FUNERAL", MaxDaysPerYear: 3, IsPaid: true, RequiresApproval: true, Color: "#6B7280"},
		{Name: "Nghỉ thai sản", Code: "MATERNITY", MaxDaysPerYear: 180, IsPaid: true, RequiresApproval: true, Color: "#F472B6"},
		{Name: "Nghỉ không lương", Code: "UNPAID", MaxDaysPerYear: 365, IsPaid: false, RequiresApproval: true, Color: "#9CA3AF"},
	}

	for i := range types {
		types[i].IsActive = true
	}

	if err := db.Create(&types).Error; err != nil {
		log.Printf("[seed] warning: failed to create leave types: %v", err)
		return
	}
	log.Printf("[seed] created %d leave types", len(types))
}

func seedPermissions(db *gorm.DB) {
	var count int64
	db.Model(&models.Permission{}).Count(&count)
	if count > 0 {
		return
	}

	// 1. Create all permissions
	perms := models.DefaultPermissions()
	if err := db.Create(&perms).Error; err != nil {
		log.Printf("[seed] warning: failed to create permissions: %v", err)
		return
	}
	log.Printf("[seed] created %d permissions", len(perms))

	// Build code → ID map
	permMap := make(map[string]string, len(perms))
	for _, p := range perms {
		permMap[p.Code] = p.ID
	}

	// 2. Create role-permission mappings
	rolePerms := models.DefaultRolePermissions()
	var rps []models.RolePermission
	for role, codes := range rolePerms {
		for _, code := range codes {
			if pid, ok := permMap[code]; ok {
				rps = append(rps, models.RolePermission{Role: role, PermissionID: pid})
			}
		}
	}

	if err := db.Create(&rps).Error; err != nil {
		log.Printf("[seed] warning: failed to create role permissions: %v", err)
		return
	}
	log.Printf("[seed] created %d role-permission mappings", len(rps))
}

func seedHolidays(db *gorm.DB) {
	var count int64
	db.Model(&models.Holiday{}).Count(&count)
	if count > 0 {
		return
	}

	holidays := []models.Holiday{
		{Name: "Tết Dương lịch", Date: "2026-01-01", HolidayType: "national", IsRecurring: true, IsActive: true},
		{Name: "Giỗ tổ Hùng Vương", Date: "2026-04-06", HolidayType: "national", IsRecurring: false, IsActive: true},
		{Name: "Ngày Giải phóng miền Nam", Date: "2026-04-30", HolidayType: "national", IsRecurring: true, IsActive: true},
		{Name: "Ngày Quốc tế Lao động", Date: "2026-05-01", HolidayType: "national", IsRecurring: true, IsActive: true},
		{Name: "Quốc khánh", Date: "2026-09-02", HolidayType: "national", IsRecurring: true, IsActive: true},
		{Name: "Nghỉ bù Quốc khánh", Date: "2026-09-03", HolidayType: "national", IsRecurring: false, IsActive: true},
		{Name: "Tết Nguyên Đán (28 Tết)", Date: "2026-02-15", HolidayType: "national", IsRecurring: false, IsActive: true},
		{Name: "Tết Nguyên Đán (29 Tết)", Date: "2026-02-16", HolidayType: "national", IsRecurring: false, IsActive: true},
		{Name: "Tết Nguyên Đán (30 Tết)", Date: "2026-02-17", HolidayType: "national", IsRecurring: false, IsActive: true},
		{Name: "Tết Nguyên Đán (Mùng 1)", Date: "2026-02-17", HolidayType: "national", IsRecurring: false, IsActive: true},
		{Name: "Tết Nguyên Đán (Mùng 2)", Date: "2026-02-18", HolidayType: "national", IsRecurring: false, IsActive: true},
		{Name: "Tết Nguyên Đán (Mùng 3)", Date: "2026-02-19", HolidayType: "national", IsRecurring: false, IsActive: true},
		{Name: "Tết Nguyên Đán (Mùng 4)", Date: "2026-02-20", HolidayType: "national", IsRecurring: false, IsActive: true},
	}

	if err := db.Create(&holidays).Error; err != nil {
		log.Printf("[seed] warning: failed to create holidays: %v", err)
		return
	}
	log.Printf("[seed] created %d holidays", len(holidays))
}

func floatPtr(f float64) *float64 {
	return &f
}
