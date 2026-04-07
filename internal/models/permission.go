package models

// Permission represents a granular action a user can perform.
type Permission struct {
	BaseModel
	Code        string `gorm:"type:text;uniqueIndex;not null" json:"code"`        // e.g. "attendance.check_in"
	Name        string `gorm:"type:text;not null" json:"name"`                    // e.g. "Check In"
	Description string `gorm:"type:text" json:"description,omitempty"`
	Module      string `gorm:"type:text;not null;index" json:"module"`            // e.g. "attendance", "branch", "user"
	IsActive    bool   `gorm:"default:true" json:"is_active"`
}

// RolePermission maps a Role to a Permission.
// This allows fine-grained control over what each role can do.
type RolePermission struct {
	BaseModel
	Role         Role   `gorm:"type:text;not null;index:idx_rp_role_perm,unique" json:"role"`
	PermissionID string `gorm:"type:text;not null;index:idx_rp_role_perm,unique" json:"permission_id"`

	Permission *Permission `gorm:"foreignKey:PermissionID" json:"permission,omitempty"`
}

// --- Permission code constants ---

const (
	// Attendance
	PermAttendanceCheckIn           = "attendance.check_in"
	PermAttendanceCheckOut          = "attendance.check_out"
	PermAttendanceViewOwn           = "attendance.view_own"
	PermAttendanceViewBranch        = "attendance.view_branch"
	PermAttendanceViewAll           = "attendance.view_all"
	PermAttendanceAdjustmentRequest = "attendance.adjustment_request"
	PermAttendanceAdjustmentApprove = "attendance.adjustment_approve"

	// Branch
	PermBranchView   = "branch.view"
	PermBranchManage = "branch.manage"

	// User
	PermUserViewBranch = "user.view_branch"
	PermUserManage     = "user.manage"

	// Report
	PermReportViewOwn    = "report.view_own"
	PermReportViewBranch = "report.view_branch"
	PermReportViewAll    = "report.view_all"
	PermReportExport     = "report.export"

	// Dashboard
	PermDashboardView    = "dashboard.view"
	PermDashboardViewAll = "dashboard.view_all"

	// Leave
	PermLeaveRequest    = "leave.request"
	PermLeaveViewOwn    = "leave.view_own"
	PermLeaveViewBranch = "leave.view_branch"
	PermLeaveApprove    = "leave.approve"

	// Overtime
	PermOvertimeRequest    = "overtime.request"
	PermOvertimeViewOwn    = "overtime.view_own"
	PermOvertimeViewBranch = "overtime.view_branch"
	PermOvertimeApprove    = "overtime.approve"

	// Shift
	PermShiftView   = "shift.view"
	PermShiftManage = "shift.manage"

	// Holiday
	PermHolidayView   = "holiday.view"
	PermHolidayManage = "holiday.manage"

	// Department
	PermDepartmentView   = "department.view"
	PermDepartmentManage = "department.manage"

	// Fraud Alert
	PermFraudAlertView   = "fraud_alert.view"
	PermFraudAlertReview = "fraud_alert.review"
)

// DefaultPermissions returns all permission definitions for seeding.
func DefaultPermissions() []Permission {
	return []Permission{
		// Attendance
		{Code: PermAttendanceCheckIn, Name: "Check In", Module: "attendance", Description: "Chấm công vào", IsActive: true},
		{Code: PermAttendanceCheckOut, Name: "Check Out", Module: "attendance", Description: "Chấm công ra", IsActive: true},
		{Code: PermAttendanceViewOwn, Name: "Xem công cá nhân", Module: "attendance", Description: "Xem lịch sử chấm công của mình", IsActive: true},
		{Code: PermAttendanceViewBranch, Name: "Xem công chi nhánh", Module: "attendance", Description: "Xem chấm công toàn chi nhánh", IsActive: true},
		{Code: PermAttendanceViewAll, Name: "Xem công toàn hệ thống", Module: "attendance", Description: "Xem chấm công tất cả chi nhánh", IsActive: true},
		{Code: PermAttendanceAdjustmentRequest, Name: "Yêu cầu bổ sung công", Module: "attendance", Description: "Gửi đơn bổ sung công khi quên chấm", IsActive: true},
		{Code: PermAttendanceAdjustmentApprove, Name: "Duyệt bổ sung công", Module: "attendance", Description: "Phê duyệt đơn bổ sung công", IsActive: true},

		// Branch
		{Code: PermBranchView, Name: "Xem chi nhánh", Module: "branch", Description: "Xem danh sách chi nhánh", IsActive: true},
		{Code: PermBranchManage, Name: "Quản lý chi nhánh", Module: "branch", Description: "Thêm/sửa/xóa chi nhánh", IsActive: true},

		// User
		{Code: PermUserViewBranch, Name: "Xem nhân viên chi nhánh", Module: "user", Description: "Xem nhân viên trong chi nhánh", IsActive: true},
		{Code: PermUserManage, Name: "Quản lý nhân viên", Module: "user", Description: "Thêm/sửa/xóa nhân viên toàn hệ thống", IsActive: true},

		// Report
		{Code: PermReportViewOwn, Name: "Xem báo cáo cá nhân", Module: "report", Description: "Xem báo cáo chấm công của mình", IsActive: true},
		{Code: PermReportViewBranch, Name: "Xem báo cáo chi nhánh", Module: "report", Description: "Xem báo cáo chấm công chi nhánh", IsActive: true},
		{Code: PermReportViewAll, Name: "Xem báo cáo toàn hệ thống", Module: "report", Description: "Xem báo cáo tất cả chi nhánh", IsActive: true},
		{Code: PermReportExport, Name: "Xuất báo cáo", Module: "report", Description: "Export báo cáo Excel/CSV", IsActive: true},

		// Dashboard
		{Code: PermDashboardView, Name: "Xem Dashboard", Module: "dashboard", Description: "Xem dashboard chi nhánh", IsActive: true},
		{Code: PermDashboardViewAll, Name: "Xem Dashboard toàn hệ thống", Module: "dashboard", Description: "Xem dashboard tất cả chi nhánh", IsActive: true},

		// Leave
		{Code: PermLeaveRequest, Name: "Xin nghỉ phép", Module: "leave", Description: "Gửi đơn xin nghỉ phép", IsActive: true},
		{Code: PermLeaveViewOwn, Name: "Xem phép cá nhân", Module: "leave", Description: "Xem đơn phép của mình", IsActive: true},
		{Code: PermLeaveViewBranch, Name: "Xem phép chi nhánh", Module: "leave", Description: "Xem đơn phép nhân viên chi nhánh", IsActive: true},
		{Code: PermLeaveApprove, Name: "Duyệt phép", Module: "leave", Description: "Phê duyệt đơn nghỉ phép", IsActive: true},

		// Overtime
		{Code: PermOvertimeRequest, Name: "Đăng ký OT", Module: "overtime", Description: "Gửi đơn xin làm thêm giờ", IsActive: true},
		{Code: PermOvertimeViewOwn, Name: "Xem OT cá nhân", Module: "overtime", Description: "Xem đơn OT của mình", IsActive: true},
		{Code: PermOvertimeViewBranch, Name: "Xem OT chi nhánh", Module: "overtime", Description: "Xem đơn OT nhân viên chi nhánh", IsActive: true},
		{Code: PermOvertimeApprove, Name: "Duyệt OT", Module: "overtime", Description: "Phê duyệt đơn làm thêm giờ", IsActive: true},

		// Shift
		{Code: PermShiftView, Name: "Xem ca làm việc", Module: "shift", Description: "Xem danh sách ca", IsActive: true},
		{Code: PermShiftManage, Name: "Quản lý ca", Module: "shift", Description: "Thêm/sửa/xóa ca làm việc", IsActive: true},

		// Holiday
		{Code: PermHolidayView, Name: "Xem ngày lễ", Module: "holiday", Description: "Xem lịch ngày nghỉ", IsActive: true},
		{Code: PermHolidayManage, Name: "Quản lý ngày lễ", Module: "holiday", Description: "Thêm/sửa/xóa ngày lễ", IsActive: true},

		// Department
		{Code: PermDepartmentView, Name: "Xem phòng ban", Module: "department", Description: "Xem danh sách phòng ban", IsActive: true},
		{Code: PermDepartmentManage, Name: "Quản lý phòng ban", Module: "department", Description: "Thêm/sửa/xóa phòng ban", IsActive: true},

		// Fraud Alert
		{Code: PermFraudAlertView, Name: "Xem cảnh báo gian lận", Module: "fraud_alert", Description: "Xem danh sách cảnh báo gian lận chi nhánh", IsActive: true},
		{Code: PermFraudAlertReview, Name: "Xét duyệt cảnh báo", Module: "fraud_alert", Description: "Đánh dấu cảnh báo đã xem xét", IsActive: true},
	}
}

// DefaultRolePermissions returns the default permission mapping for each role.
func DefaultRolePermissions() map[Role][]string {
	return map[Role][]string{
		RoleEmployee: {
			PermAttendanceCheckIn, PermAttendanceCheckOut, PermAttendanceViewOwn,
			PermAttendanceAdjustmentRequest,
			PermReportViewOwn,
			PermLeaveRequest, PermLeaveViewOwn,
			PermOvertimeRequest, PermOvertimeViewOwn,
			PermShiftView, PermHolidayView, PermDepartmentView,
		},
		RoleManager: {
			// Inherits employee permissions
			PermAttendanceCheckIn, PermAttendanceCheckOut, PermAttendanceViewOwn,
			PermAttendanceAdjustmentRequest,
			PermReportViewOwn,
			PermLeaveRequest, PermLeaveViewOwn,
			PermOvertimeRequest, PermOvertimeViewOwn,
			PermShiftView, PermHolidayView, PermDepartmentView,
			// Manager-specific
			PermAttendanceViewBranch, PermAttendanceAdjustmentApprove,
			PermBranchView, PermUserViewBranch,
			PermReportViewBranch, PermReportExport,
			PermDashboardView,
			PermLeaveViewBranch, PermLeaveApprove,
			PermOvertimeViewBranch, PermOvertimeApprove,
			PermFraudAlertView, PermFraudAlertReview,
		},
		RoleManagerDevice: {
			// Minimal kiosk permissions — QR display + password check-in only
			PermAttendanceCheckIn, PermAttendanceCheckOut,
			PermAttendanceViewBranch,
			PermBranchView,
		},
		RoleAdmin: {
			// All permissions
			PermAttendanceCheckIn, PermAttendanceCheckOut, PermAttendanceViewOwn,
			PermAttendanceViewBranch, PermAttendanceViewAll,
			PermAttendanceAdjustmentRequest, PermAttendanceAdjustmentApprove,
			PermBranchView, PermBranchManage,
			PermUserViewBranch, PermUserManage,
			PermReportViewOwn, PermReportViewBranch, PermReportViewAll, PermReportExport,
			PermDashboardView, PermDashboardViewAll,
			PermLeaveRequest, PermLeaveViewOwn, PermLeaveViewBranch, PermLeaveApprove,
			PermOvertimeRequest, PermOvertimeViewOwn, PermOvertimeViewBranch, PermOvertimeApprove,
			PermShiftView, PermShiftManage,
			PermHolidayView, PermHolidayManage,
			PermDepartmentView, PermDepartmentManage,
			PermFraudAlertView, PermFraudAlertReview,
		},
	}
}
