// seed tool — tạo dữ liệu demo cho tất cả services
// Chạy: go run scripts/seed/main.go
// Yêu cầu: PostgreSQL đang chạy, các service đã migrate schema

package main

import (
	"database/sql"
	"log"
	"os"
	"time"

	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://app:secret@localhost:5432/smart_attendance?sslmode=disable"
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("connect failed: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("ping failed: %v", err)
	}

	log.Println("=== Smart Attendance Demo Seed ===")

	// 1. Get branch IDs from org_service
	branches := getBranchIDs(db)
	if len(branches) < 3 {
		log.Fatalf("expected at least 3 branches in org_service.branches, got %d — run org service first", len(branches))
	}

	hcmID := branches[0]
	hnID := branches[1]
	dnID := branches[2]

	log.Printf("branches: HCM=%s HN=%s DN=%s", hcmID[:8], hnID[:8], dnID[:8])

	// 2. Seed demo users in auth_service
	hash := hashPassword("password123")
	now := time.Now()
	joinDate := time.Date(2025, 6, 1, 0, 0, 0, 0, time.Local)

	users := []struct {
		code     string
		email    string
		name     string
		phone    string
		role     string
		branchID *string
		position string
	}{
		// Managers
		{"MGR-HCM", "manager.hcm@demo.com", "Trần Minh Quản Lý", "0901000010", "manager", &hcmID, "Quản lý chi nhánh HCM"},
		{"MGR-HN", "manager.hn@demo.com", "Nguyễn Thị Hà Nội", "0901000011", "manager", &hnID, "Quản lý chi nhánh HN"},
		{"MGR-DN", "manager.dn@demo.com", "Lê Văn Đà Nẵng", "0901000012", "manager", &dnID, "Quản lý chi nhánh ĐN"},

		// Kiosk devices (manager_device)
		{"DEV-HCM", "device.hcm@demo.com", "Kiosk HCM - Quận 1", "0901000020", "manager_device", &hcmID, "Thiết bị chấm công"},
		{"DEV-HN", "device.hn@demo.com", "Kiosk HN - Cầu Giấy", "0901000021", "manager_device", &hnID, "Thiết bị chấm công"},
		{"DEV-DN", "device.dn@demo.com", "Kiosk ĐN - Hải Châu", "0901000022", "manager_device", &dnID, "Thiết bị chấm công"},

		// Employees - HCM (5)
		{"EMP-HCM01", "emp1.hcm@demo.com", "Nguyễn Văn An", "0909100001", "employee", &hcmID, "Lập trình viên"},
		{"EMP-HCM02", "emp2.hcm@demo.com", "Trần Thị Bình", "0909100002", "employee", &hcmID, "Thiết kế UI/UX"},
		{"EMP-HCM03", "emp3.hcm@demo.com", "Lê Hoàng Cường", "0909100003", "employee", &hcmID, "Tester"},
		{"EMP-HCM04", "emp4.hcm@demo.com", "Phạm Thị Dung", "0909100004", "employee", &hcmID, "Nhân sự"},
		{"EMP-HCM05", "emp5.hcm@demo.com", "Hoàng Minh Đức", "0909100005", "employee", &hcmID, "Kinh doanh"},

		// Employees - HN (5)
		{"EMP-HN01", "emp1.hn@demo.com", "Vũ Thị Giang", "0909200001", "employee", &hnID, "Lập trình viên"},
		{"EMP-HN02", "emp2.hn@demo.com", "Đỗ Văn Hải", "0909200002", "employee", &hnID, "DevOps"},
		{"EMP-HN03", "emp3.hn@demo.com", "Bùi Thị Hương", "0909200003", "employee", &hnID, "BA"},
		{"EMP-HN04", "emp4.hn@demo.com", "Ngô Quốc Khánh", "0909200004", "employee", &hnID, "Nhân sự"},
		{"EMP-HN05", "emp5.hn@demo.com", "Trương Thị Lan", "0909200005", "employee", &hnID, "Kế toán"},

		// Employees - DN (5)
		{"EMP-DN01", "emp1.dn@demo.com", "Phan Văn Minh", "0909300001", "employee", &dnID, "Lập trình viên"},
		{"EMP-DN02", "emp2.dn@demo.com", "Lý Thị Ngọc", "0909300002", "employee", &dnID, "Tester"},
		{"EMP-DN03", "emp3.dn@demo.com", "Huỳnh Quốc Phong", "0909300003", "employee", &dnID, "Kinh doanh"},
		{"EMP-DN04", "emp4.dn@demo.com", "Võ Thị Quỳnh", "0909300004", "employee", &dnID, "Nhân sự"},
		{"EMP-DN05", "emp5.dn@demo.com", "Đặng Văn Sơn", "0909300005", "employee", &dnID, "Kỹ thuật"},
	}

	created := 0
	skipped := 0
	for _, u := range users {
		exists := userExists(db, u.email)
		if exists {
			skipped++
			continue
		}

		_, err := db.Exec(`
			INSERT INTO auth_service.users (id, employee_code, email, password_hash, full_name, phone, role, branch_id, position, join_date, is_active, created_at, updated_at)
			VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, $6, $7, $8, $9, true, $10, $10)
		`, u.code, u.email, hash, u.name, u.phone, u.role, u.branchID, u.position, joinDate, now)

		if err != nil {
			log.Printf("  ERROR creating %s: %v", u.email, err)
		} else {
			created++
		}
	}

	log.Printf("[auth] users: %d created, %d skipped (already exist)", created, skipped)

	// 3. Seed sample attendance data (last 7 days)
	seedAttendanceData(db, hcmID, hnID, dnID)

	// 4. Seed sample fraud alerts
	seedFraudAlerts(db, hcmID)

	log.Println("=== Seed Complete ===")
	log.Println("")
	log.Println("Demo accounts (password: password123):")
	log.Println("  Admin:    admin@smartattendance.com")
	log.Println("  Manager:  manager.hcm@demo.com / manager.hn@demo.com / manager.dn@demo.com")
	log.Println("  Kiosk:    device.hcm@demo.com / device.hn@demo.com")
	log.Println("  Employee: emp1.hcm@demo.com ~ emp5.hcm@demo.com")
}

func getBranchIDs(db *sql.DB) []string {
	rows, err := db.Query("SELECT id FROM org_service.branches ORDER BY created_at ASC LIMIT 10")
	if err != nil {
		log.Fatalf("query branches: %v", err)
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		rows.Scan(&id)
		ids = append(ids, id)
	}
	return ids
}

func userExists(db *sql.DB, email string) bool {
	var count int
	db.QueryRow("SELECT COUNT(*) FROM auth_service.users WHERE email = $1", email).Scan(&count)
	return count > 0
}

func hashPassword(pw string) string {
	h, _ := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
	return string(h)
}

func seedAttendanceData(db *sql.DB, hcmID, hnID, dnID string) {
	// Get employee IDs
	rows, err := db.Query("SELECT id, branch_id FROM auth_service.users WHERE role = 'employee' AND branch_id IS NOT NULL")
	if err != nil {
		log.Printf("[attendance] ERROR query employees: %v", err)
		return
	}
	defer rows.Close()

	type emp struct {
		id       string
		branchID string
	}
	var employees []emp
	for rows.Next() {
		var e emp
		rows.Scan(&e.id, &e.branchID)
		employees = append(employees, e)
	}

	if len(employees) == 0 {
		log.Printf("[attendance] no employees found, skipping attendance seed")
		return
	}

	now := time.Now()
	created := 0
	statuses := []string{"on_time", "on_time", "on_time", "late", "on_time"} // 80% on-time

	for dayOffset := 7; dayOffset >= 1; dayOffset-- {
		date := now.AddDate(0, 0, -dayOffset)
		if date.Weekday() == time.Saturday || date.Weekday() == time.Sunday {
			continue
		}
		workDate := date.Format("2006-01-02")

		for i, e := range employees {
			// Check if already exists
			var count int
			db.QueryRow("SELECT COUNT(*) FROM attendance_service.attendances WHERE user_id = $1 AND work_date = $2", e.id, workDate).Scan(&count)
			if count > 0 {
				continue
			}

			status := statuses[i%len(statuses)]
			checkInHour := 8
			checkInMin := i*3 + 1 // stagger check-in times
			if status == "late" {
				checkInHour = 8
				checkInMin = 30 + i*2
			}
			checkIn := time.Date(date.Year(), date.Month(), date.Day(), checkInHour, checkInMin, 0, 0, time.Local)
			checkOut := time.Date(date.Year(), date.Month(), date.Day(), 17, 5+i*2, 0, 0, time.Local)

			_, err := db.Exec(`
				INSERT INTO attendance_service.attendances (id, user_id, branch_id, work_date, check_in_at, check_out_at, status, method, ip_address, created_at, updated_at)
				VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, $6, 'qr_totp', '192.168.1.100', $7, $7)
			`, e.id, e.branchID, workDate, checkIn, checkOut, status, now)

			if err != nil {
				log.Printf("[attendance] ERROR insert: %v", err)
			} else {
				created++
			}
		}
	}

	log.Printf("[attendance] created %d attendance records (last 7 working days)", created)
}

func seedFraudAlerts(db *sql.DB, hcmBranchID string) {
	// Get 2 employees from HCM
	rows, err := db.Query("SELECT id FROM auth_service.users WHERE role = 'employee' AND branch_id = $1 LIMIT 2", hcmBranchID)
	if err != nil {
		return
	}
	defer rows.Close()

	var empIDs []string
	for rows.Next() {
		var id string
		rows.Scan(&id)
		empIDs = append(empIDs, id)
	}

	if len(empIDs) < 2 {
		return
	}

	now := time.Now()
	alerts := []struct {
		userID    string
		alertType string
		severity  string
		desc      string
	}{
		{empIDs[0], "gps_accuracy", "warning", "GPS accuracy 3.2m — nghi ngờ fake GPS"},
		{empIDs[0], "new_device", "warning", "Thiết bị mới: Samsung Galaxy S24"},
		{empIDs[1], "impossible_travel", "critical", "Di chuyển 1200km trong 15 phút (HCM → HN)"},
		{empIDs[1], "ip_location_mismatch", "warning", "IP từ Đức, GPS tại HCM — nghi VPN"},
	}

	created := 0
	for _, a := range alerts {
		var count int
		db.QueryRow("SELECT COUNT(*) FROM attendance_service.fraud_alerts WHERE user_id = $1 AND alert_type = $2", a.userID, a.alertType).Scan(&count)
		if count > 0 {
			continue
		}

		_, err := db.Exec(`
			INSERT INTO attendance_service.fraud_alerts (id, user_id, branch_id, alert_type, severity, description, ip_address, is_reviewed, created_at, updated_at)
			VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, '185.220.101.42', false, $6, $6)
		`, a.userID, hcmBranchID, a.alertType, a.severity, a.desc, now)

		if err != nil {
			log.Printf("[fraud] ERROR insert: %v", err)
		} else {
			created++
		}
	}

	log.Printf("[fraud_alerts] created %d sample fraud alerts", created)
}
