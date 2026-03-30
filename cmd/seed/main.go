package main

import (
	"crypto/rand"
	"encoding/base32"
	"fmt"
	"log"
	"math"
	mrand "math/rand"
	"time"

	"github.com/smart-attendance/smart-attendance/internal/config"
	"github.com/smart-attendance/smart-attendance/internal/database"
	"github.com/smart-attendance/smart-attendance/internal/models"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	cfg := config.Load()
	db, err := database.Connect(cfg)
	if err != nil {
		log.Fatalf("database connection failed: %v", err)
	}

	if err := database.AutoMigrate(db, &models.User{}, &models.Branch{}, &models.BranchIPWhitelist{}, &models.BranchLocation{}, &models.Attendance{}); err != nil {
		log.Fatalf("auto-migrate failed: %v", err)
	}

	// Check if already seeded
	var branchCount int64
	db.Model(&models.Branch{}).Count(&branchCount)
	if branchCount >= 100 {
		log.Println("Database already has 100+ branches. Skipping seed.")
		return
	}

	log.Println("=== Starting large seed: 100 branches, 5000 employees, attendance records ===")
	start := time.Now()

	hash, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	pw := string(hash)

	// --- 1. Create 100 branches ---
	log.Println("[1/4] Creating 100 branches...")
	branches := make([]models.Branch, 0, 100)
	for i := 0; i < 100; i++ {
		city := cities[i%len(cities)]
		lat := city.lat + (mrand.Float64()-0.5)*0.02
		lng := city.lng + (mrand.Float64()-0.5)*0.02

		secret := make([]byte, 20)
		rand.Read(secret)

		branch := models.Branch{
			Name:           fmt.Sprintf("%s — Chi nhánh %d", city.name, i+1),
			Address:        fmt.Sprintf("%d %s, %s", mrand.Intn(500)+1, streets[i%len(streets)], city.name),
			Lat:            floatPtr(lat),
			Lng:            floatPtr(lng),
			RadiusM:        200 + mrand.Intn(300),
			TOTPSecret:     base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(secret),
			AllowedMethods: "qr_totp,ip,location",
			WorkStartTime:  workStarts[mrand.Intn(len(workStarts))],
			WorkEndTime:    workEnds[mrand.Intn(len(workEnds))],
			IsActive:       true,
		}
		if err := db.Create(&branch).Error; err != nil {
			log.Fatalf("create branch %d: %v", i+1, err)
		}
		branches = append(branches, branch)

		// Add IP whitelist
		db.Create(&models.BranchIPWhitelist{BranchID: branch.ID, IPCIDR: "127.0.0.1/32", Label: "Localhost"})
		db.Create(&models.BranchIPWhitelist{BranchID: branch.ID, IPCIDR: fmt.Sprintf("10.%d.%d.0/24", i/10, i%10), Label: "Office LAN"})

		// Add location
		db.Create(&models.BranchLocation{BranchID: branch.ID, Lat: lat, Lng: lng, RadiusM: branch.RadiusM, Label: "Main Office"})
	}
	log.Printf("  ✓ %d branches created", len(branches))

	// --- 2. Create admin + 100 managers + ~4900 employees = ~5000 users ---
	log.Println("[2/4] Creating 5000 users (1 admin + 100 managers + 4899 employees)...")

	// Admin
	admin := &models.User{
		Email:        "admin@smartattendance.com",
		PasswordHash: pw,
		FullName:     "System Admin",
		Role:         models.RoleAdmin,
		IsActive:     true,
	}
	db.Create(admin)

	// 1 manager per branch
	managers := make([]models.User, 0, 100)
	for i, branch := range branches {
		bid := branch.ID
		mgr := models.User{
			Email:        fmt.Sprintf("manager%d@smartattendance.com", i+1),
			PasswordHash: pw,
			FullName:     fmt.Sprintf("%s %s", lastNames[mrand.Intn(len(lastNames))], firstNames[mrand.Intn(len(firstNames))]),
			Role:         models.RoleManager,
			BranchID:     &bid,
			IsActive:     true,
		}
		db.Create(&mgr)
		managers = append(managers, mgr)
	}

	// ~4899 employees distributed across branches
	employeeCount := 4899
	employees := make([]models.User, 0, employeeCount)
	for i := 0; i < employeeCount; i++ {
		branchIdx := i % len(branches)
		bid := branches[branchIdx].ID
		emp := models.User{
			Email:        fmt.Sprintf("emp%04d@smartattendance.com", i+1),
			PasswordHash: pw,
			FullName:     fmt.Sprintf("%s %s %s", lastNames[mrand.Intn(len(lastNames))], middleNames[mrand.Intn(len(middleNames))], firstNames[mrand.Intn(len(firstNames))]),
			Role:         models.RoleEmployee,
			BranchID:     &bid,
			IsActive:     i%50 != 0, // ~2% inactive
		}
		db.Create(&emp)
		employees = append(employees, emp)
	}
	log.Printf("  ✓ %d users created (1 admin + %d managers + %d employees)", 1+len(managers)+len(employees), len(managers), len(employees))

	// --- 3. Generate attendance records (30 days) ---
	log.Println("[3/4] Generating attendance records for past 30 days...")
	allUsers := append(managers, employees...)
	now := time.Now()
	totalAtt := 0

	// Batch insert for performance
	batch := make([]models.Attendance, 0, 1000)
	flushBatch := func() {
		if len(batch) > 0 {
			db.CreateInBatches(&batch, 500)
			totalAtt += len(batch)
			batch = batch[:0]
		}
	}

	for day := 29; day >= 0; day-- {
		date := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).AddDate(0, 0, -day)
		weekday := date.Weekday()

		// Skip weekends
		if weekday == time.Saturday || weekday == time.Sunday {
			continue
		}

		for _, user := range allUsers {
			if !user.IsActive {
				continue
			}
			// ~85% attendance rate
			if mrand.Float64() > 0.85 {
				continue
			}

			branchID := ""
			if user.BranchID != nil {
				branchID = *user.BranchID
			}
			if branchID == "" {
				continue
			}

			// Determine status
			var status models.AttendanceStatus
			roll := mrand.Float64()
			if roll < 0.75 {
				status = models.StatusOnTime
			} else if roll < 0.92 {
				status = models.StatusLate
			} else {
				status = models.StatusAbsent
			}

			// Parse work start time for this branch
			branchIdx := 0
			for idx, b := range branches {
				if b.ID == branchID {
					branchIdx = idx
					break
				}
			}
			startHour := 8
			fmt.Sscanf(branches[branchIdx].WorkStartTime, "%d", &startHour)

			var checkInMinute int
			switch status {
			case models.StatusOnTime:
				checkInMinute = -mrand.Intn(15) // 0-15 min early
			case models.StatusLate:
				checkInMinute = 5 + mrand.Intn(55) // 5-60 min late
			case models.StatusAbsent:
				checkInMinute = 0
			}

			checkIn := date.Add(time.Duration(startHour)*time.Hour + time.Duration(checkInMinute)*time.Minute + time.Duration(mrand.Intn(60))*time.Second)

			// Check-out: 8-10 hours after check-in
			workHours := 8 + mrand.Float64()*2
			checkOut := checkIn.Add(time.Duration(workHours * float64(time.Hour)))

			lat := branches[branchIdx].Lat
			lng := branches[branchIdx].Lng

			methods := []string{"qr_totp", "ip", "location"}
			method := methods[mrand.Intn(len(methods))]

			att := models.Attendance{
				UserID:       user.ID,
				BranchID:     branchID,
				CheckInAt:    &checkIn,
				CheckOutAt:   &checkOut,
				Status:       status,
				Method:       method,
				IPAddress:    fmt.Sprintf("10.%d.%d.%d", branchIdx/10, branchIdx%10, mrand.Intn(254)+1),
				Lat:          lat,
				Lng:          lng,
				TOTPVerified: method == "qr_totp",
				IPVerified:   method == "ip" || mrand.Float64() > 0.3,
				LocVerified:  method == "location" || mrand.Float64() > 0.3,
			}
			batch = append(batch, att)

			if len(batch) >= 1000 {
				flushBatch()
				if totalAtt%10000 == 0 {
					log.Printf("  ... %d records created", totalAtt)
				}
			}
		}
	}
	flushBatch()
	log.Printf("  ✓ %d attendance records created", totalAtt)

	// --- 4. Summary ---
	log.Println("[4/4] Seed complete!")
	log.Printf("  Branches:   100")
	log.Printf("  Users:      %d", 1+len(managers)+len(employees))
	log.Printf("  Attendance: %d", totalAtt)
	log.Printf("  Duration:   %s", time.Since(start).Round(time.Millisecond))
	log.Printf("  Credentials: admin@smartattendance.com / password123")
	log.Printf("               manager1@smartattendance.com / password123")
	log.Printf("               emp0001@smartattendance.com / password123")
}

func floatPtr(f float64) *float64 {
	return &f
}

// suppress unused import
var _ = math.Pi

type cityInfo struct {
	name string
	lat  float64
	lng  float64
}

var cities = []cityInfo{
	{"TP. Hồ Chí Minh", 10.7769, 106.7009},
	{"Hà Nội", 21.0285, 105.8542},
	{"Đà Nẵng", 16.0544, 108.2022},
	{"Hải Phòng", 20.8449, 106.6881},
	{"Cần Thơ", 10.0452, 105.7469},
	{"Biên Hòa", 10.9574, 106.8429},
	{"Nha Trang", 12.2388, 109.1967},
	{"Huế", 16.4637, 107.5909},
	{"Vũng Tàu", 10.3460, 107.0843},
	{"Đà Lạt", 11.9404, 108.4583},
	{"Quy Nhơn", 13.7829, 109.2197},
	{"Buôn Ma Thuột", 12.6680, 108.0377},
	{"Thái Nguyên", 21.5928, 105.8442},
	{"Nam Định", 20.4388, 106.1621},
	{"Vinh", 18.6796, 105.6813},
	{"Rạch Giá", 10.0125, 105.0808},
	{"Long Xuyên", 10.3860, 105.4350},
	{"Phan Thiết", 10.9289, 108.1022},
	{"Thanh Hóa", 19.8067, 105.7852},
	{"Bắc Ninh", 21.1861, 106.0763},
}

var streets = []string{
	"Nguyễn Huệ", "Lê Lợi", "Trần Hưng Đạo", "Hai Bà Trưng", "Lý Thường Kiệt",
	"Pasteur", "Điện Biên Phủ", "Võ Văn Tần", "Nam Kỳ Khởi Nghĩa", "Cách Mạng Tháng 8",
	"Nguyễn Trãi", "Phạm Ngũ Lão", "Bùi Viện", "Lê Duẩn", "Tôn Đức Thắng",
	"Hùng Vương", "Ba Tháng Hai", "Nguyễn Văn Cừ", "Trường Chinh", "Quang Trung",
}

var lastNames = []string{
	"Nguyễn", "Trần", "Lê", "Phạm", "Hoàng", "Huỳnh", "Phan", "Vũ", "Võ", "Đặng",
	"Bùi", "Đỗ", "Hồ", "Ngô", "Dương", "Lý", "Đào", "Đinh", "Lâm", "Tạ",
}

var middleNames = []string{
	"Văn", "Thị", "Đức", "Minh", "Hoàng", "Thanh", "Quốc", "Ngọc", "Hữu", "Thành",
	"Phương", "Anh", "Tuấn", "Hồng", "Kim", "Bảo", "Xuân", "Thu", "Quang", "Trung",
}

var firstNames = []string{
	"An", "Bình", "Châu", "Dũng", "Em", "Giang", "Hải", "Khoa", "Linh", "Minh",
	"Nam", "Phúc", "Quân", "Sơn", "Tâm", "Uy", "Vinh", "Yến", "Hưng", "Đạt",
	"Hà", "Lan", "Mai", "Ngân", "Oanh", "Phượng", "Quyên", "Trang", "Uyên", "Vân",
}

var workStarts = []string{"07:30", "08:00", "08:30", "09:00"}
var workEnds = []string{"16:30", "17:00", "17:30", "18:00"}
