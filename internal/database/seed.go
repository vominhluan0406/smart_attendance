package database

import (
	"crypto/rand"
	"encoding/base32"
	"fmt"
	"log"
	"math"
	mrand "math/rand"
	"time"

	"github.com/smart-attendance/smart-attendance/internal/models"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func Seed(db *gorm.DB) error {
	// Silence logging during seeding to avoid massive SQL dumps in development mode
	originalLogger := db.Logger
	db.Logger = originalLogger.LogMode(logger.Warn)
	defer func() { db.Logger = originalLogger }()

	return db.Transaction(func(tx *gorm.DB) error {
		// These seeds are idempotent (each checks own count) — always run
		seedLeaveTypes(tx)
		seedHolidays(tx)
		seedPermissions(tx)

		// Users: only seed if DB is empty
		var userCount int64
		if err := tx.Model(&models.User{}).Count(&userCount).Error; err != nil {
			log.Printf("[seed] CRITICAL ERROR: count users failed: %v", err)
			return err
		}
		log.Printf("[seed] Initial users count in DB = %d", userCount)
		if userCount > 0 {
			log.Printf("[seed] users table has %d rows — skipping user/data seed", userCount)
			return nil
		}

		hash, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
		pw := string(hash)

		// ... (rest of the logic remains same but uses tx instead of db)
		// I'll need to update the entire function to use tx.
		// For brevity in this chunk, I'll just show the start and then use another chunk or bigger replacement.
		return seedAllData(tx, pw)
	})
}

func seedAllData(db *gorm.DB, pw string) error {
	// ============================================================
	// 1. Branches (3 branches)
	// ============================================================
	branches := []struct {
		Name    string
		Address string
		Lat     float64
		Lng     float64
	}{
		{"HQ - Hồ Chí Minh", "123 Nguyễn Huệ, Quận 1, TP.HCM", 10.773889, 106.701944},
		{"Chi nhánh Hà Nội", "45 Lý Thường Kiệt, Hoàn Kiếm, Hà Nội", 21.024800, 105.841171},
		{"Chi nhánh Đà Nẵng", "78 Bạch Đằng, Hải Châu, Đà Nẵng", 16.068083, 108.212028},
	}

	var branchIDs []string
	for _, b := range branches {
		secret := make([]byte, 20)
		rand.Read(secret)
		branch := &models.Branch{
			Name:           b.Name,
			Address:        b.Address,
			Lat:            floatPtr(b.Lat),
			Lng:            floatPtr(b.Lng),
			RadiusM:        500,
			TOTPSecret:     base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(secret),
			AllowedMethods: "qr_totp,ip,location,face",
			WorkStartTime:  "08:00",
			WorkEndTime:    "17:00",
			IsActive:       true,
		}
		if err := db.Create(branch).Error; err != nil {
			log.Printf("[seed] error creating branch %s: %v", b.Name, err)
			continue
		}
		branchIDs = append(branchIDs, branch.ID)

		db.Create(&models.BranchIPWhitelist{BranchID: branch.ID, IPCIDR: "127.0.0.1/32", Label: "Localhost"})
		db.Create(&models.BranchIPWhitelist{BranchID: branch.ID, IPCIDR: "192.168.1.0/24", Label: "Office LAN"})
		db.Create(&models.BranchLocation{BranchID: branch.ID, Lat: b.Lat, Lng: b.Lng, RadiusM: 500, Label: "Main Office"})
		log.Printf("[seed] branch created: %s", b.Name)
	}

	// ============================================================
	// 2. Departments (per branch)
	// ============================================================
	deptDefs := []struct {
		Name string
		Code string
	}{
		{"Phòng Kỹ thuật", "IT"},
		{"Phòng Nhân sự", "HR"},
		{"Phòng Kinh doanh", "SALES"},
	}

	deptMap := make(map[string][]string) // branchID → []deptID
	for _, bid := range branchIDs {
		for _, d := range deptDefs {
			dept := &models.Department{BranchID: bid, Name: d.Name, Code: d.Code, IsActive: true}
			if err := db.Create(dept).Error; err != nil {
				log.Printf("[seed] error creating department %s: %v", d.Name, err)
			}
			deptMap[bid] = append(deptMap[bid], dept.ID)
		}
	}
	log.Printf("[seed] created %d departments", len(branchIDs)*len(deptDefs))

	// ============================================================
	// 3. Work Shifts (per branch)
	// ============================================================
	shiftDefs := []struct {
		Name      string
		Code      string
		Start     string
		End       string
		Color     string
		IsDefault bool
	}{
		{"Ca sáng", "MORNING", "07:00", "15:00", "#F59E0B", false},
		{"Ca chính", "MAIN", "08:00", "17:00", "#3B82F6", true},
		{"Ca chiều", "AFTERNOON", "13:00", "21:00", "#8B5CF6", false},
	}

	shiftMap := make(map[string][]string) // branchID → []shiftID
	for _, bid := range branchIDs {
		// Remove default shift created by migration
		db.Where("branch_id = ? AND code = 'DEFAULT'", bid).Delete(&models.WorkShift{})
		for _, s := range shiftDefs {
			shift := &models.WorkShift{
				BranchID: bid, Name: s.Name, Code: s.Code,
				StartTime: s.Start, EndTime: s.End,
				GracePeriodMinutes: 15, BreakDurationMinutes: 60,
				WorkingDays: "1,2,3,4,5", Color: s.Color,
				IsDefault: s.IsDefault, IsActive: true,
			}
			db.Create(shift)
			shiftMap[bid] = append(shiftMap[bid], shift.ID)
		}
	}
	log.Printf("[seed] created %d shifts", len(branchIDs)*len(shiftDefs))

	// ============================================================
	// 4. Users (1 admin + 3 managers + 3 manager_devices + 15 employees = 22 users)
	// ============================================================
	joinDate := time.Date(2025, 6, 1, 0, 0, 0, 0, time.Local)

	// Admin
	admin := &models.User{
		EmployeeCode: "ADM001", Email: "admin@smartattendance.com", PasswordHash: pw,
		FullName: "Trần Quốc Bảo", Phone: "0901000001",
		Role: models.RoleAdmin, Position: "System Administrator",
		JoinDate: &joinDate, IsActive: true,
	}
	db.Create(admin)

	// Managers (1 per branch)
	managerDefs := []struct {
		Code  string
		Email string
		Name  string
		Phone string
	}{
		{"MGR001", "manager.hcm@demo.com", "Nguyễn Thị Hương", "0901000010"},
		{"MGR002", "manager.hn@demo.com", "Phạm Văn Đức", "0901000020"},
		{"MGR003", "manager.dn@demo.com", "Lê Thị Mai", "0901000030"},
	}

	var managerIDs []string
	for i, m := range managerDefs {
		user := &models.User{
			EmployeeCode: m.Code, Email: m.Email, PasswordHash: pw,
			FullName: m.Name, Phone: m.Phone,
			Role: models.RoleManager, BranchID: &branchIDs[i],
			DepartmentID: &deptMap[branchIDs[i]][0], // IT dept
			Position:     "Branch Manager", JoinDate: &joinDate, IsActive: true,
		}
		db.Create(user)
		managerIDs = append(managerIDs, user.ID)
	}

	// Manager Devices (1 per branch — kiosk accounts for QR display + password check-in)
	deviceDefs := []struct {
		Code  string
		Email string
		Name  string
	}{
		{"DEV001", "device.hcm@demo.com", "Kiosk HCM"},
		{"DEV002", "device.hn@demo.com", "Kiosk Hà Nội"},
		{"DEV003", "device.dn@demo.com", "Kiosk Đà Nẵng"},
	}
	for i, d := range deviceDefs {
		user := &models.User{
			EmployeeCode: d.Code, Email: d.Email, PasswordHash: pw,
			FullName: d.Name,
			Role: models.RoleManagerDevice, BranchID: &branchIDs[i],
			Position: "Kiosk Device", JoinDate: &joinDate, IsActive: true,
		}
		db.Create(user)
	}

	// Set managers on departments
	for i, bid := range branchIDs {
		for _, did := range deptMap[bid] {
			db.Model(&models.Department{}).Where("id = ?", did).Update("manager_id", managerIDs[i])
		}
	}

	// Employees (5 per branch = 15)
	empNames := []struct {
		Name     string
		Phone    string
		Position string
	}{
		{"Hoàng Văn An", "0912000001", "Developer"},
		{"Trần Thị Bích", "0912000002", "HR Specialist"},
		{"Ngô Quang Cường", "0912000003", "Sales Executive"},
		{"Đỗ Thị Dung", "0912000004", "QA Engineer"},
		{"Vũ Minh Hiếu", "0912000005", "Accountant"},
	}

	var allEmployeeIDs []string
	empIdx := 0
	for bIdx, bid := range branchIDs {
		depts := deptMap[bid]
		for eIdx, emp := range empNames {
			empIdx++
			deptID := depts[eIdx%len(depts)]
			user := &models.User{
				EmployeeCode: fmt.Sprintf("EMP%03d", empIdx),
				Email:        fmt.Sprintf("emp%d.%s@demo.com", empIdx, []string{"hcm", "hn", "dn"}[bIdx]),
				PasswordHash: pw, FullName: emp.Name, Phone: emp.Phone,
				Role: models.RoleEmployee, BranchID: &bid, DepartmentID: &deptID,
				Position: emp.Position, JoinDate: &joinDate, IsActive: true,
			}
			if err := db.Create(user).Error; err != nil {
				log.Printf("[seed] error creating employee %s: %v", user.Email, err)
			}
			allEmployeeIDs = append(allEmployeeIDs, user.ID)
		}
	}
	log.Printf("[seed] created 22 users (1 admin, 3 managers, 3 manager_devices, 15 employees)")
	log.Printf("[seed] login: admin@smartattendance.com / password123")
	log.Printf("[seed] login: manager.hcm@demo.com / password123 (Quản lý)")
	log.Printf("[seed] login: device.hcm@demo.com / password123 (Manager Máy)")
	log.Printf("[seed] login: emp1.hcm@demo.com / password123")

	// ============================================================
	// 5. Attendance + Logs (last 14 working days)
	// ============================================================
	seedAttendanceData(db, branchIDs, shiftMap, managerIDs, allEmployeeIDs)

	return nil
}

func seedAttendanceData(db *gorm.DB, branchIDs []string, shiftMap map[string][]string, managerIDs, employeeIDs []string) {
	var attCount int64
	db.Model(&models.Attendance{}).Count(&attCount)
	if attCount > 0 {
		return
	}

	now := time.Now()
	rng := mrand.New(mrand.NewSource(now.UnixNano()))

	// Build user→branch map
	type userInfo struct {
		ID       string
		BranchID string
		ShiftIdx int // 0=morning, 1=main(default), 2=afternoon
	}

	var users []userInfo
	for i, mid := range managerIDs {
		users = append(users, userInfo{mid, branchIDs[i], 1}) // managers on main shift
	}
	for i, eid := range employeeIDs {
		bIdx := i / 5 // 5 employees per branch
		shiftIdx := 1 // default to main shift
		if i%5 == 0 {
			shiftIdx = 0 // first employee per branch on morning shift
		} else if i%5 == 4 {
			shiftIdx = 2 // last employee per branch on afternoon shift
		}
		users = append(users, userInfo{eid, branchIDs[bIdx], shiftIdx})
	}

	shiftTimes := []struct{ Start, End int }{
		{7, 15},  // morning
		{8, 17},  // main
		{13, 21}, // afternoon
	}

	var attBatch []models.Attendance
	var logBatch []models.AttendanceLog

	for dayOffset := 14; dayOffset >= 1; dayOffset-- {
		day := now.AddDate(0, 0, -dayOffset)
		if day.Weekday() == time.Saturday || day.Weekday() == time.Sunday {
			continue
		}
		workDate := day.Format("2006-01-02")

		for _, u := range users {
			// 90% attendance rate
			if rng.Float64() > 0.90 {
				continue
			}

			shifts := shiftMap[u.BranchID]
			if len(shifts) == 0 {
				continue
			}
			shiftID := shifts[u.ShiftIdx%len(shifts)]
			st := shiftTimes[u.ShiftIdx%len(shiftTimes)]

			// Generate check-in time: 70% on-time, 30% late
			var checkInMinOffset int
			if rng.Float64() < 0.70 {
				checkInMinOffset = rng.Intn(15) - 5 // -5 to +10 min (on time)
			} else {
				checkInMinOffset = 15 + rng.Intn(30) // 15-45 min late
			}
			checkIn := time.Date(day.Year(), day.Month(), day.Day(), st.Start, checkInMinOffset, rng.Intn(60), 0, now.Location())

			// Check-out: 8-9 hours later
			checkOutMinOffset := rng.Intn(60) // 0-60 min past end
			checkOut := time.Date(day.Year(), day.Month(), day.Day(), st.End, checkOutMinOffset, rng.Intn(60), 0, now.Location())

			// Status
			gracePeriod := 15
			deadline := time.Date(day.Year(), day.Month(), day.Day(), st.Start, gracePeriod, 0, 0, now.Location())
			status := models.StatusOnTime
			if checkIn.After(deadline) {
				status = models.StatusLate
			}

			method := "qr_totp"
			att := models.Attendance{
				UserID: u.ID, BranchID: u.BranchID, ShiftID: &shiftID,
				WorkDate: workDate, CheckInAt: &checkIn, CheckOutAt: &checkOut,
				Status: status, Method: method,
				TOTPVerified: true, IPVerified: true, LocVerified: true,
			}
			attBatch = append(attBatch, att)

			// 2 logs per day (in + out)
			logBatch = append(logBatch,
				models.AttendanceLog{
					UserID: u.ID, BranchID: u.BranchID, ShiftID: &shiftID,
					WorkDate: workDate, LoggedAt: checkIn, Method: method,
					TOTPVerified: true, IPVerified: true, LocVerified: true,
				},
				models.AttendanceLog{
					UserID: u.ID, BranchID: u.BranchID, ShiftID: &shiftID,
					WorkDate: workDate, LoggedAt: checkOut, Method: method,
					TOTPVerified: true, IPVerified: true, LocVerified: true,
				},
			)
		}
	}

	// Batch insert
	batchSize := 100
	for i := 0; i < len(attBatch); i += batchSize {
		end := int(math.Min(float64(i+batchSize), float64(len(attBatch))))
		if err := db.Create(attBatch[i:end]).Error; err != nil {
			log.Printf("[seed] error creating attendance batch: %v", err)
		}
	}
	for i := 0; i < len(logBatch); i += batchSize {
		end := int(math.Min(float64(i+batchSize), float64(len(logBatch))))
		if err := db.Create(logBatch[i:end]).Error; err != nil {
			log.Printf("[seed] error creating log batch: %v", err)
		}
	}

	log.Printf("[seed] created %d attendance records + %d time logs (14 working days)", len(attBatch), len(logBatch))
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
	if count == 0 {
		// Fresh DB — create all permissions
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

		// Create role-permission mappings
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

	// Ensure any new permissions added to code are created in DB.
	ensureNewPermissions(db)

	// Always ensure every role in DefaultRolePermissions has its mappings.
	// This handles new roles added after initial seed (e.g., manager_device).
	ensureRolePermissions(db)
}

// ensureNewPermissions creates any permissions defined in code but missing from the DB.
func ensureNewPermissions(db *gorm.DB) {
	var existing []models.Permission
	db.Find(&existing)
	existingCodes := make(map[string]bool, len(existing))
	for _, p := range existing {
		existingCodes[p.Code] = true
	}

	var added int
	for _, p := range models.DefaultPermissions() {
		if existingCodes[p.Code] {
			continue
		}
		if err := db.Create(&p).Error; err == nil {
			added++
		}
	}
	if added > 0 {
		log.Printf("[seed] created %d new permissions", added)
	}
}

// ensureRolePermissions checks each role's expected permissions and inserts any missing ones.
func ensureRolePermissions(db *gorm.DB) {
	// Build code → ID map from DB
	var allPerms []models.Permission
	db.Find(&allPerms)
	permMap := make(map[string]string, len(allPerms))
	for _, p := range allPerms {
		permMap[p.Code] = p.ID
	}

	rolePerms := models.DefaultRolePermissions()
	for role, codes := range rolePerms {
		// Get existing permission IDs for this role
		var existingPermIDs []string
		db.Model(&models.RolePermission{}).
			Where("role = ?", role).
			Pluck("permission_id", &existingPermIDs)

		existingSet := make(map[string]bool, len(existingPermIDs))
		for _, pid := range existingPermIDs {
			existingSet[pid] = true
		}

		// Insert missing
		var added int
		for _, code := range codes {
			pid, ok := permMap[code]
			if !ok {
				continue
			}
			if existingSet[pid] {
				continue
			}
			rp := models.RolePermission{Role: role, PermissionID: pid}
			if err := db.Create(&rp).Error; err == nil {
				added++
			}
		}
		if added > 0 {
			log.Printf("[seed] added %d missing permissions for role %s", added, role)
		}
	}
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
