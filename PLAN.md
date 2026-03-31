# BA: Database Schema Redesign for Smart Attendance

## 1. Bug Fix: Bottom Nav on QR Display Page

**Problem:** Trang `/attendance/qr/{branchID}` (Manager QR Display) chỉ hiển thị 2 item trong bottom menu (Home + Lịch sử) vì handler không truyền `UserRole` và `UserBranch` vào template data.

**Fix:** Đã thêm `UserRole` và `UserBranch` vào `QRDisplayPage` handler (attendance.go:105-106). Nav template dùng `.UserRole` để quyết định hiển thị menu items.

---

## 2. Current Schema Analysis

### 2.1 Existing Tables (5 tables)

| Table | Purpose | Records (seed) |
|---|---|---|
| `users` | Nhân viên, manager, admin | 5,000 |
| `branches` | Chi nhánh | 100 |
| `branch_ip_whitelists` | IP whitelist per branch | ~200 |
| `branch_locations` | GPS zones per branch | ~100 |
| `attendances` | Check-in/check-out records | ~170K |

### 2.2 Issues Found

| # | Issue | Impact | Severity |
|---|---|---|---|
| 1 | **No shift system** — `WorkStartTime`/`WorkEndTime` hardcoded as string in Branch | Không hỗ trợ ca sáng/chiều/đêm, không flexible | HIGH |
| 2 | **Grace period hardcoded 15 phút** trong `calculateStatus()` | Không config per branch/shift | MEDIUM |
| 3 | **No composite indexes** trên `(user_id, check_in_at)` và `(branch_id, check_in_at)` | Query chậm khi data lớn, mọi dashboard/report query đều dùng pattern này | HIGH |
| 4 | **`Branch.RadiusM` redundant** — `BranchLocation` đã có `RadiusM` riêng, `Branch.RadiusM` không ai dùng | Confusing schema | LOW |
| 5 | **`StatusAbsent` never written by app** — chỉ seed script tạo | Không có job đánh vắng mặt cuối ngày | MEDIUM |
| 6 | **No leave management** — không xin nghỉ phép | Thiếu feature core | HIGH |
| 7 | **No attendance adjustment** — không bổ sung công khi quên chấm | Thiếu feature core | HIGH |
| 8 | **No department/team** — chỉ có branch level | Không phân nhóm nhân viên trong 1 chi nhánh | MEDIUM |
| 9 | **No holiday calendar** — không biết ngày lễ/nghỉ | Tính công sai vào ngày lễ | MEDIUM |
| 10 | **No overtime tracking** — không quản lý OT | Thiếu feature | MEDIUM |
| 11 | **Attendance thiếu `work_date`** — phải parse từ `check_in_at` timestamp | GROUP BY ngày phải dùng `strftime()`, không index-friendly | MEDIUM |

---

## 3. Proposed Schema (14 tables)

### 3.1 Modified Tables

#### `users` — Thêm fields

```go
type User struct {
    BaseModel
    EmployeeCode  string  `gorm:"type:text;uniqueIndex"`     // NEW: Mã nhân viên (VD: NV001)
    Email         string  `gorm:"type:text;uniqueIndex;not null"`
    PasswordHash  string  `gorm:"type:text" json:"-"`
    FullName      string  `gorm:"type:text;not null"`
    Phone         string  `gorm:"type:text"`                  // NEW
    Role          Role    `gorm:"type:text;not null;default:employee"`
    BranchID      *string `gorm:"type:text;index"`
    DepartmentID  *string `gorm:"type:text;index"`            // NEW: FK → departments
    Position      string  `gorm:"type:text"`                  // NEW: Chức vụ
    JoinDate      *time.Time                                  // NEW: Ngày vào làm
    IsActive      bool    `gorm:"default:true"`
    OAuthProvider string  `gorm:"type:text"`
    OAuthID       string  `gorm:"type:text;index"`
}
```

#### `branches` — Bỏ fields thừa

```go
type Branch struct {
    BaseModel
    Name           string   `gorm:"type:text;not null"`
    Address        string   `gorm:"type:text"`
    Lat            *float64 `gorm:"type:real"`
    Lng            *float64 `gorm:"type:real"`
    // REMOVED: RadiusM (redundant — BranchLocation has its own)
    TOTPSecret     string   `gorm:"type:text" json:"-"`
    AllowedMethods string   `gorm:"type:text;not null;default:'qr_totp,ip,location'"`
    // REMOVED: WorkStartTime, WorkEndTime (moved to WorkShift)
    IsActive       bool     `gorm:"default:true"`

    IPWhitelist []BranchIPWhitelist `gorm:"foreignKey:BranchID"`
    Locations   []BranchLocation    `gorm:"foreignKey:BranchID"`
    Shifts      []WorkShift         `gorm:"foreignKey:BranchID"` // NEW relation
}
```

#### `attendances` — Thêm shift reference + work_date + indexes

```go
type Attendance struct {
    BaseModel
    UserID       string           `gorm:"type:text;not null;index:idx_att_user_date"`
    BranchID     string           `gorm:"type:text;not null;index:idx_att_branch_date"`
    ShiftID      *string          `gorm:"type:text;index"`          // NEW: FK → work_shifts
    WorkDate     string           `gorm:"type:text;not null;index:idx_att_user_date;index:idx_att_branch_date"` // NEW: "2006-01-02"
    CheckInAt    *time.Time
    CheckOutAt   *time.Time
    Status       AttendanceStatus `gorm:"type:text;not null;default:'on_time'"`
    Method       string           `gorm:"type:text"`
    IPAddress    string           `gorm:"type:text"`
    Lat          *float64         `gorm:"type:real"`
    Lng          *float64         `gorm:"type:real"`
    TOTPVerified bool             `gorm:"default:false"`
    IPVerified   bool             `gorm:"default:false"`
    LocVerified  bool             `gorm:"default:false"`
    Note         string           `gorm:"type:text"`
    IsAdjusted   bool             `gorm:"default:false"`            // NEW: Đã bổ sung công
    AdjustedByID *string          `gorm:"type:text"`                // NEW: Ai duyệt bổ sung

    User   *User      `gorm:"foreignKey:UserID"`
    Branch *Branch    `gorm:"foreignKey:BranchID"`
    Shift  *WorkShift `gorm:"foreignKey:ShiftID"`                  // NEW relation
}
// Composite indexes: (user_id, work_date), (branch_id, work_date)
```

### 3.2 New Tables

#### `departments` — Phòng ban trong chi nhánh

```go
type Department struct {
    BaseModel
    BranchID  string  `gorm:"type:text;not null;index"`
    Name      string  `gorm:"type:text;not null"`
    Code      string  `gorm:"type:text"`                // VD: "IT", "HR", "SALES"
    ManagerID *string `gorm:"type:text"`                // FK → users (trưởng phòng)
    IsActive  bool    `gorm:"default:true"`
}
```

#### `work_shifts` — Ca làm việc

```go
type WorkShift struct {
    BaseModel
    BranchID            string `gorm:"type:text;not null;index"`
    Name                string `gorm:"type:text;not null"`           // "Ca sáng", "Ca chiều"
    Code                string `gorm:"type:text"`                    // "MORNING", "AFTERNOON"
    StartTime           string `gorm:"type:text;not null"`           // "08:00"
    EndTime             string `gorm:"type:text;not null"`           // "17:00"
    GracePeriodMinutes  int    `gorm:"default:15"`                   // Thay cho hardcode 15 phút
    LateThresholdMinutes int   `gorm:"default:0"`                    // Trễ thêm bao nhiêu phút mới tính là trễ
    IsOvernight         bool   `gorm:"default:false"`                // Ca đêm (qua ngày)
    BreakDurationMinutes int   `gorm:"default:60"`                   // Nghỉ trưa
    WorkingDays         string `gorm:"type:text;default:'1,2,3,4,5'"` // 1=Mon,...,7=Sun
    Color               string `gorm:"type:text;default:'#3B82F6'"`  // UI color
    IsDefault           bool   `gorm:"default:false"`                // Ca mặc định của branch
    IsActive            bool   `gorm:"default:true"`
}
```

#### `user_shift_assignments` — Phân ca cho nhân viên

```go
type UserShiftAssignment struct {
    BaseModel
    UserID        string     `gorm:"type:text;not null;index"`
    ShiftID       string     `gorm:"type:text;not null;index"`
    EffectiveFrom string     `gorm:"type:text;not null"`          // "2026-04-01"
    EffectiveTo   *string    `gorm:"type:text"`                   // null = vô thời hạn
    // Unique: (user_id, effective_from) — 1 user chỉ 1 ca/ngày bắt đầu
}
```

#### `holidays` — Ngày lễ / ngày nghỉ

```go
type Holiday struct {
    BaseModel
    Name        string  `gorm:"type:text;not null"`
    Date        string  `gorm:"type:text;not null;index"`     // "2026-01-01"
    BranchID    *string `gorm:"type:text;index"`              // null = toàn công ty
    HolidayType string  `gorm:"type:text;default:'company'"`  // national, company, branch
    IsRecurring bool    `gorm:"default:false"`                 // Lặp lại hàng năm
    IsActive    bool    `gorm:"default:true"`
}
```

#### `leave_types` — Loại phép

```go
type LeaveType struct {
    BaseModel
    Name             string `gorm:"type:text;not null"`        // "Nghỉ phép năm"
    Code             string `gorm:"type:text;uniqueIndex"`     // "ANNUAL", "SICK", "PERSONAL"
    MaxDaysPerYear   int    `gorm:"default:12"`
    IsPaid           bool   `gorm:"default:true"`
    RequiresApproval bool   `gorm:"default:true"`
    Color            string `gorm:"type:text;default:'#10B981'"`
    IsActive         bool   `gorm:"default:true"`
}
```

#### `leave_requests` — Đơn xin nghỉ phép

```go
type LeaveRequestStatus string
const (
    LeaveStatusPending   LeaveRequestStatus = "pending"
    LeaveStatusApproved  LeaveRequestStatus = "approved"
    LeaveStatusRejected  LeaveRequestStatus = "rejected"
    LeaveStatusCancelled LeaveRequestStatus = "cancelled"
)

type LeaveRequest struct {
    BaseModel
    UserID       string             `gorm:"type:text;not null;index"`
    LeaveTypeID  string             `gorm:"type:text;not null;index"`
    StartDate    string             `gorm:"type:text;not null"`       // "2026-04-01"
    EndDate      string             `gorm:"type:text;not null"`       // "2026-04-03"
    TotalDays    float64            `gorm:"type:real;not null"`       // 2.5 (hỗ trợ nửa ngày)
    Reason       string             `gorm:"type:text"`
    Status       LeaveRequestStatus `gorm:"type:text;not null;default:'pending'"`
    ReviewerID   *string            `gorm:"type:text"`                // Người duyệt
    ReviewedAt   *time.Time
    ReviewerNote string             `gorm:"type:text"`

    User      *User      `gorm:"foreignKey:UserID"`
    LeaveType *LeaveType `gorm:"foreignKey:LeaveTypeID"`
    Reviewer  *User      `gorm:"foreignKey:ReviewerID"`
}
```

#### `leave_balances` — Số ngày phép còn lại

```go
type LeaveBalance struct {
    BaseModel
    UserID      string  `gorm:"type:text;not null;uniqueIndex:idx_lb_unique"`
    LeaveTypeID string  `gorm:"type:text;not null;uniqueIndex:idx_lb_unique"`
    Year        int     `gorm:"not null;uniqueIndex:idx_lb_unique"`
    TotalDays   float64 `gorm:"type:real;default:12"`
    UsedDays    float64 `gorm:"type:real;default:0"`
    // RemainingDays = TotalDays - UsedDays (computed, not stored)
}
```

#### `attendance_adjustments` — Đơn bổ sung công

```go
type AdjustmentStatus string
const (
    AdjustStatusPending  AdjustmentStatus = "pending"
    AdjustStatusApproved AdjustmentStatus = "approved"
    AdjustStatusRejected AdjustmentStatus = "rejected"
)

type AttendanceAdjustment struct {
    BaseModel
    UserID             string           `gorm:"type:text;not null;index"`
    AttendanceID       *string          `gorm:"type:text"`             // null nếu chưa có record (quên check-in hoàn toàn)
    WorkDate           string           `gorm:"type:text;not null"`    // "2026-03-28"
    RequestedCheckIn   *time.Time                                      // Giờ check-in đề nghị
    RequestedCheckOut  *time.Time                                      // Giờ check-out đề nghị
    Reason             string           `gorm:"type:text;not null"`
    Status             AdjustmentStatus `gorm:"type:text;not null;default:'pending'"`
    ReviewerID         *string          `gorm:"type:text"`
    ReviewedAt         *time.Time
    ReviewerNote       string           `gorm:"type:text"`

    User       *User       `gorm:"foreignKey:UserID"`
    Attendance *Attendance `gorm:"foreignKey:AttendanceID"`
    Reviewer   *User       `gorm:"foreignKey:ReviewerID"`
}
```

#### `overtime_requests` — Đơn xin làm thêm giờ

```go
type OvertimeStatus string
const (
    OTStatusPending  OvertimeStatus = "pending"
    OTStatusApproved OvertimeStatus = "approved"
    OTStatusRejected OvertimeStatus = "rejected"
)

type OvertimeRequest struct {
    BaseModel
    UserID       string         `gorm:"type:text;not null;index"`
    WorkDate     string         `gorm:"type:text;not null"`
    PlannedStart string         `gorm:"type:text;not null"`         // "18:00"
    PlannedEnd   string         `gorm:"type:text;not null"`         // "21:00"
    PlannedHours float64        `gorm:"type:real"`
    Reason       string         `gorm:"type:text"`
    Status       OvertimeStatus `gorm:"type:text;not null;default:'pending'"`
    ReviewerID   *string        `gorm:"type:text"`
    ReviewedAt   *time.Time
    ReviewerNote string         `gorm:"type:text"`

    User     *User `gorm:"foreignKey:UserID"`
    Reviewer *User `gorm:"foreignKey:ReviewerID"`
}
```

---

### 3.3 Permission Tables (RBAC)

#### `permissions` — Danh sách quyền hạn

```go
type Permission struct {
    BaseModel
    Code        string `gorm:"type:text;uniqueIndex;not null"` // "attendance.check_in"
    Name        string `gorm:"type:text;not null"`             // "Check In"
    Description string `gorm:"type:text"`
    Module      string `gorm:"type:text;not null;index"`       // "attendance", "branch", "user"
    IsActive    bool   `gorm:"default:true"`
}
```

#### `role_permissions` — Mapping Role → Permission

```go
type RolePermission struct {
    BaseModel
    Role         Role   `gorm:"type:text;not null;uniqueIndex:idx_rp_role_perm"`
    PermissionID string `gorm:"type:text;not null;uniqueIndex:idx_rp_role_perm"`
}
```

#### Permission Modules (31 permissions, 10 modules)

| Module | Permission Codes | Employee | Manager | Admin |
|---|---|:---:|:---:|:---:|
| attendance | check_in, check_out, view_own | x | x | x |
| attendance | view_branch, view_all | | x | x |
| attendance | adjustment_request | x | x | x |
| attendance | adjustment_approve | | x | x |
| branch | view, manage | | view | x |
| user | view_branch, manage | | view | x |
| report | view_own | x | x | x |
| report | view_branch, view_all, export | | x | x |
| dashboard | view, view_all | | x | x |
| leave | request, view_own | x | x | x |
| leave | view_branch, approve | | x | x |
| overtime | request, view_own | x | x | x |
| overtime | view_branch, approve | | x | x |
| shift | view | x | x | x |
| shift | manage | | | x |
| holiday | view | x | x | x |
| holiday | manage | | | x |
| department | view | x | x | x |
| department | manage | | | x |

---

## 4. Index Strategy

### Composite Indexes (Critical for Performance)

```
attendances:    (user_id, work_date)      — FindTodayByUser, user history
attendances:    (branch_id, work_date)    — Dashboard stats, branch reports
attendances:    (work_date, status)       — Daily aggregation queries
leave_requests: (user_id, status)         — Pending requests per user
leave_balances: (user_id, leave_type_id, year) — UNIQUE, balance lookup
```

### Why These Matter
- `FindTodayByUser` chạy MỌI lần user check-in → phải < 1ms
- Dashboard aggregate queries chạy cho 5,000 users → composite index giảm từ full-scan → index-seek
- `work_date` as TEXT "2006-01-02" cho phép range query bằng string comparison, không cần `strftime()`

---

## 5. Entity Relationship Overview

```
Branch ─1:N─ Department ─1:N─ User
Branch ─1:N─ WorkShift ─1:N─ UserShiftAssignment ─N:1─ User
Branch ─1:N─ BranchIPWhitelist
Branch ─1:N─ BranchLocation
Branch ─1:N─ Holiday (nullable branch_id = company-wide)

User ─1:N─ Attendance (has ShiftID → WorkShift)
User ─1:N─ LeaveRequest → LeaveType
User ─1:N─ LeaveBalance (per year, per type)
User ─1:N─ AttendanceAdjustment
User ─1:N─ OvertimeRequest

Role ─N:M─ Permission (via RolePermission)
```

---

## 6. Migration Strategy

Vì dùng **SQLite + GORM AutoMigrate**, migration an toàn:

1. **Phase A — Add new columns to existing tables** (non-breaking)
   - `users`: add `employee_code`, `phone`, `department_id`, `position`, `join_date`
   - `attendances`: add `shift_id`, `work_date`, `is_adjusted`, `adjusted_by_id`
   - Backfill `work_date` from existing `check_in_at` timestamps

2. **Phase B — Create new tables** (non-breaking)
   - `departments`, `work_shifts`, `user_shift_assignments`
   - `holidays`, `leave_types`, `leave_requests`, `leave_balances`
   - `attendance_adjustments`, `overtime_requests`

3. **Phase C — Seed default data**
   - Tạo default shift per branch từ `WorkStartTime`/`WorkEndTime` cũ
   - Tạo default leave types (Annual 12d, Sick 30d, Personal 3d)
   - Tạo default holidays (Tết, 30/4, 1/5, 2/9...)

4. **Phase D — Update services** (breaking changes contained in code)
   - `calculateStatus()` đọc grace period từ `WorkShift` thay vì hardcode
   - `CheckIn()` resolve shift cho user, ghi `shift_id` + `work_date`
   - Dashboard queries dùng `work_date` thay vì `strftime(check_in_at)`

5. **Phase E — Remove deprecated** (cleanup)
   - Branch: drop `WorkStartTime`, `WorkEndTime`, `RadiusM` (sau khi migrate xong)

---

## 7. Implementation Plan

| Step | Task | Files | Size |
|---|---|---|---|
| 1 | Create new models (9 files) | `internal/models/department.go, work_shift.go, leave_type.go, leave_request.go, leave_balance.go, attendance_adjustment.go, overtime_request.go, holiday.go, user_shift_assignment.go` | M |
| 2 | Update existing models (3 files) | `internal/models/user.go, branch.go, attendance.go` | S |
| 3 | Add composite indexes + AutoMigrate update | `internal/database/database.go, cmd/server/main.go` | S |
| 4 | Backfill migration: `work_date` + default shifts | `internal/database/migrate.go` (new) | M |
| 5 | Seed default leave types + holidays | `internal/database/seed.go` | S |
| 6 | Update AttendanceService: shift-aware check-in | `internal/service/attendance_service.go` | M |
| 7 | Update Dashboard queries: use `work_date` | `internal/repository/dashboard_queries.go` | M |
