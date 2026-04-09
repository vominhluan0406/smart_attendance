// ─── API Response wrappers ──────────────────────────────────────────
export interface ApiResponse<T> {
  success: boolean;
  data: T;
  error?: { code: string; message: string };
  meta?: PaginationMeta;
}

export interface PaginationMeta {
  page: number;
  limit: number;
  total: number;
}

export interface PaginatedResponse<T> {
  success: boolean;
  data: T[];
  meta: PaginationMeta;
}

// ─── Auth / Session ─────────────────────────────────────────────────
export type UserRole = "admin" | "manager" | "manager_device" | "employee";

export interface Session {
  userId: string;
  email: string;
  fullName: string;
  role: UserRole;
  branchId?: string;
  branchName?: string;
}

export interface LoginRequest {
  email: string;
  password: string;
}

export interface LoginResponse {
  access_token: string;
  refresh_token: string;
  user: User;
}

// ─── User ───────────────────────────────────────────────────────────
export interface User {
  id: string;
  employee_code?: string;
  email: string;
  full_name: string;
  phone?: string;
  role: UserRole;
  branch_id?: string;
  department_id?: string;
  position?: string;
  join_date?: string;
  is_active: boolean;
  oauth_provider?: string;
  department?: Department;
  created_at: string;
  updated_at: string;
}

export interface Department {
  id: string;
  name: string;
  branch_id: string;
}

// ─── Branch ─────────────────────────────────────────────────────────
export interface Branch {
  id: string;
  name: string;
  address: string;
  lat?: number;
  lng?: number;
  radius_m: number;
  beacon_uuid?: string;
  allowed_methods: string;
  require_biometric: boolean;
  work_start_time: string;
  work_end_time: string;
  is_active: boolean;
  ip_whitelist?: IPWhitelist[];
  locations?: BranchLocation[];
  shifts?: WorkShift[];
  departments?: Department[];
  created_at: string;
  updated_at: string;
}

export interface IPWhitelist {
  id: string;
  branch_id: string;
  ip_cidr: string;
  label: string;
}

export interface BranchLocation {
  id: string;
  branch_id: string;
  label: string;
  lat: number;
  lng: number;
  radius_m: number;
}

export interface WorkShift {
  id: string;
  branch_id: string;
  name: string;
  start_time: string;
  end_time: string;
  is_default: boolean;
}

// ─── Attendance ─────────────────────────────────────────────────────
export type AttendanceStatus =
  | "on_time"
  | "late"
  | "absent"
  | "leave"
  | "invalidated";

export interface Attendance {
  id: string;
  user_id: string;
  branch_id: string;
  shift_id?: string;
  work_date: string;
  check_in_at?: string;
  check_out_at?: string;
  status: AttendanceStatus;
  method: string;
  ip_address: string;
  lat?: number;
  lng?: number;
  totp_verified: boolean;
  ip_verified: boolean;
  loc_verified: boolean;
  face_verified: boolean;
  nfc_verified: boolean;
  password_verified: boolean;
  biometric_verified: boolean;
  note?: string;
  is_adjusted: boolean;
  user?: User;
  branch?: Branch;
  shift?: WorkShift;
  created_at: string;
  updated_at: string;
}

export interface AttendanceLog {
  logged_at: string;
  method: string;
  totp_verified: boolean;
  ip_verified: boolean;
  loc_verified: boolean;
  face_verified: boolean;
  password_verified: boolean;
}

// ─── Fraud Alert ────────────────────────────────────────────────────
export type FraudAlertType =
  | "gps_accuracy"
  | "totp_reuse"
  | "impossible_travel"
  | "new_device"
  | "ip_location_mismatch"
  | "cloned_authenticator"
  | "anomaly_time"
  | "concurrent_session";

export type Severity = "warning" | "critical";

export interface FraudAlert {
  id: string;
  user_id: string;
  branch_id: string;
  alert_type: FraudAlertType;
  severity: Severity;
  description: string;
  details: string;
  ip_address: string;
  lat?: number;
  lng?: number;
  is_reviewed: boolean;
  reviewed_at?: string;
  reviewed_by?: string;
  user?: User;
  created_at: string;
  updated_at: string;
}

// ─── Leave ──────────────────────────────────────────────────────────
export type LeaveRequestStatus =
  | "pending"
  | "approved"
  | "rejected"
  | "cancelled";

export interface LeaveType {
  id: string;
  name: string;
  code: string;
  max_days_per_year: number;
  is_paid: boolean;
  requires_approval: boolean;
  color: string;
  is_active: boolean;
}

export interface LeaveRequest {
  id: string;
  user_id: string;
  leave_type_id: string;
  start_date: string;
  end_date: string;
  total_days: number;
  reason: string;
  status: LeaveRequestStatus;
  reviewer_id?: string;
  reviewed_at?: string;
  reviewer_note?: string;
  user?: User;
  leave_type?: LeaveType;
  reviewer?: User;
  created_at: string;
  updated_at: string;
}

// ─── Adjustment ─────────────────────────────────────────────────────
export type AdjustmentStatus = "pending" | "approved" | "rejected";

export interface AdjustmentRequest {
  id: string;
  user_id: string;
  attendance_id?: string;
  work_date: string;
  requested_check_in?: string;
  requested_check_out?: string;
  reason: string;
  status: AdjustmentStatus;
  reviewer_id?: string;
  reviewed_at?: string;
  reviewer_note?: string;
  user?: User;
  attendance?: Attendance;
  reviewer?: User;
  created_at: string;
  updated_at: string;
}

// ─── Dashboard ──────────────────────────────────────────────────────
export interface DashboardStats {
  total_employees: number;
  today_checkins: number;
  on_time_rate: number;
  late_count: number;
  absent_count: number;
  pending_leave: number;
  pending_adjustments: number;
  fraud_alerts_today: number;
}

export interface DashboardChartData {
  labels: string[];
  on_time: number[];
  late: number[];
  absent: number[];
}

export interface TopLateUser {
  full_name: string;
  email: string;
  late_count: number;
}

export interface RecentActivity {
  id: string;
  user_name: string;
  user_email: string;
  action: string;
  time: string;
  status: AttendanceStatus;
  method: string;
}
