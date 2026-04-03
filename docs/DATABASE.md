# Database Schema — Smart Attendance

## Entity Relationship Diagram (ERD)

```
┌─────────────────────────────────────────────────────────────────────────────────────────────┐
│                                    CORE ENTITIES                                             │
│                                                                                              │
│  ┌──────────────┐       ┌──────────────┐       ┌──────────────────┐                         │
│  │   branches   │──┐    │    users     │──┐    │   departments    │                         │
│  │──────────────│  │    │──────────────│  │    │──────────────────│                         │
│  │ id (PK)      │  │    │ id (PK)      │  │    │ id (PK)          │                         │
│  │ name         │  │    │ email        │  │    │ branch_id (FK)   │◄── branches              │
│  │ address      │  │    │ password_hash│  │    │ name             │                         │
│  │ lat, lng     │  │    │ full_name    │  │    │ manager_id (FK)  │◄── users                 │
│  │ totp_secret  │  │    │ role         │  │    │ is_active        │                         │
│  │ allowed_meth.│  │    │ branch_id(FK)│──┘    └──────────────────┘                         │
│  │ is_active    │  │    │ department_id│                                                     │
│  └──────┬───────┘  │    │ is_active    │                                                     │
│         │          │    └──────┬───────┘                                                     │
│         │          │           │                                                              │
└─────────┼──────────┼───────────┼────────────────────────────────────────────────────────────┘
          │          │           │
          │          │           │
┌─────────┼──────────┼───────────┼────────────────────────────────────────────────────────────┐
│         │  BRANCH CONFIG       │                                                             │
│         │          │           │                                                             │
│  ┌──────▼────────┐ │    ┌──────▼────────────┐    ┌──────────────────┐                       │
│  │ branch_ip_    │ │    │ branch_locations  │    │   work_shifts    │                       │
│  │ whitelists    │ │    │───────────────────│    │──────────────────│                       │
│  │───────────────│ │    │ branch_id (FK)    │    │ branch_id (FK)   │◄── branches            │
│  │ branch_id(FK) │ │    │ lat, lng          │    │ name, code       │                       │
│  │ ip_cidr       │ │    │ radius_m          │    │ start/end_time   │                       │
│  │ label         │ │    │ label             │    │ grace_period_min │                       │
│  └───────────────┘ │    └───────────────────┘    │ working_days     │                       │
│                    │                              └────────┬─────────┘                       │
└────────────────────┼───────────────────────────────────────┼────────────────────────────────┘
                     │                                       │
                     │                                       │
┌────────────────────┼───────────────────────────────────────┼────────────────────────────────┐
│                    │   ATTENDANCE (Chấm công)              │                                 │
│                    │                                       │                                 │
│  ┌─────────────────▼──────────────┐   ┌────────────────────▼────────┐                       │
│  │        attendances             │   │  user_shift_assignments     │                       │
│  │────────────────────────────────│   │─────────────────────────────│                       │
│  │ user_id (FK) ──► users         │   │ user_id (FK) ──► users      │                       │
│  │ branch_id (FK) ──► branches    │   │ shift_id (FK) ──► shifts    │                       │
│  │ shift_id (FK) ──► work_shifts  │   │ effective_from / to         │                       │
│  │ work_date                      │   └─────────────────────────────┘                       │
│  │ check_in_at, check_out_at     │                                                          │
│  │ status (on_time/late/absent/   │   ┌─────────────────────────────┐                       │
│  │         leave)                 │   │    attendance_logs           │                       │
│  │ method, ip_address             │   │─────────────────────────────│                       │
│  │ lat, lng                       │   │ user_id (FK) ──► users      │                       │
│  │ totp/ip/loc/face/pwd_verified  │   │ branch_id (FK)              │                       │
│  │ note, is_adjusted              │   │ work_date, logged_at        │                       │
│  └────────────────────────────────┘   │ method, ip, lat, lng        │                       │
│                                       │ accuracy_m, device_fp       │                       │
│  ┌────────────────────────────────┐   │ anomaly_flag, anomaly_score │                       │
│  │  attendance_adjustments        │   │ totp/ip/loc/face verified   │                       │
│  │────────────────────────────────│   └─────────────────────────────┘                       │
│  │ user_id (FK) ──► users         │                                                          │
│  │ attendance_id (FK)             │                                                          │
│  │ reviewer_id (FK) ──► users     │                                                          │
│  │ status (pending/approved/rej.) │                                                          │
│  └────────────────────────────────┘                                                          │
└──────────────────────────────────────────────────────────────────────────────────────────────┘

┌──────────────────────────────────────────────────────────────────────────────────────────────┐
│                              LEAVE MANAGEMENT (Nghỉ phép)                                    │
│                                                                                              │
│  ┌──────────────────┐    ┌────────────────────────┐    ┌──────────────────────┐              │
│  │   leave_types    │    │    leave_requests       │    │   leave_balances     │              │
│  │──────────────────│    │────────────────────────│    │──────────────────────│              │
│  │ name, code       │◄───│ leave_type_id (FK)     │    │ user_id (FK)         │              │
│  │ max_days_per_year│    │ user_id (FK) ──► users │    │ leave_type_id (FK)   │              │
│  │ is_paid          │    │ start_date, end_date   │    │ year                 │              │
│  │ color            │    │ total_days, reason     │    │ total_days, used_days│              │
│  └──────────────────┘    │ status (pending/       │    └──────────────────────┘              │
│                          │   approved/rejected)   │                                          │
│                          │ reviewer_id (FK)►users │                                          │
│                          └────────────────────────┘                                          │
│                                                                                              │
│  ┌────────────────────────┐    ┌──────────────────┐                                          │
│  │  overtime_requests     │    │     holidays      │                                          │
│  │────────────────────────│    │──────────────────│                                          │
│  │ user_id (FK) ──► users │    │ name, date        │                                          │
│  │ work_date              │    │ branch_id (FK)    │ (null = toàn công ty)                    │
│  │ planned_start/end      │    │ holiday_type      │                                          │
│  │ reviewer_id (FK)►users │    │ is_recurring      │                                          │
│  │ status                 │    └──────────────────┘                                          │
│  └────────────────────────┘                                                                  │
└──────────────────────────────────────────────────────────────────────────────────────────────┘

┌──────────────────────────────────────────────────────────────────────────────────────────────┐
│                         SECURITY & ANTI-FRAUD (Bảo mật)                                      │
│                                                                                              │
│  ┌──────────────────────┐   ┌──────────────────────┐   ┌──────────────────────┐              │
│  │  user_credentials    │   │    user_devices       │   │   user_sessions      │              │
│  │──────────────────────│   │──────────────────────│   │──────────────────────│              │
│  │ user_id (FK)►users   │   │ user_id (FK)►users   │   │ user_id (FK)►users   │              │
│  │ credential_id (FIDO) │   │ fingerprint_hash     │   │ token_hash (unique)  │              │
│  │ public_key           │   │ user_agent           │   │ ip_address           │              │
│  │ sign_count           │   │ is_trusted           │   │ expires_at           │              │
│  │ is_approved          │   │ is_blocked           │   │ is_revoked           │              │
│  └──────────────────────┘   └──────────────────────┘   └──────────────────────┘              │
│                                                                                              │
│  ┌──────────────────────┐   ┌────────────────────────────────┐                               │
│  │    fraud_alerts      │   │  permissions & role_permissions │                               │
│  │──────────────────────│   │────────────────────────────────│                               │
│  │ user_id (FK)►users   │   │ permissions: code, module      │                               │
│  │ branch_id (FK)       │   │ role_permissions:              │                               │
│  │ alert_type           │   │   role + permission_id (FK)    │                               │
│  │ severity             │   │ Roles: admin, manager,         │                               │
│  │ description, details │   │   manager_device, employee     │                               │
│  │ is_reviewed          │   └────────────────────────────────┘                               │
│  └──────────────────────┘                                                                    │
│                                                                                              │
│  ┌──────────────────────┐   ┌──────────────────────┐                                         │
│  │   user_streaks       │   │    user_badges        │  (Gamification)                         │
│  │──────────────────────│   │──────────────────────│                                         │
│  │ user_id (FK, unique) │   │ user_id (FK)►users   │                                         │
│  │ current/max_count    │   │ badge_type           │                                         │
│  │ last_check_in        │   │ earned_at, reason    │                                         │
│  └──────────────────────┘   └──────────────────────┘                                         │
└──────────────────────────────────────────────────────────────────────────────────────────────┘
```

## Table Descriptions (22 tables)

### Core Entities

| Table | Vai trò | Records ước tính |
|---|---|---|
| **branches** | Chi nhánh công ty — cấu hình check-in (TOTP secret, allowed methods, giờ làm) | ~100 |
| **users** | Nhân viên — 4 roles: admin, manager, manager_device, employee | ~5,000 |
| **departments** | Phòng ban trong mỗi chi nhánh, có manager_id riêng | ~300 |

### Branch Configuration

| Table | Vai trò | Quan hệ |
|---|---|---|
| **branch_ip_whitelists** | Danh sách IP/CIDR cho phép check-in từ mạng nội bộ | N:1 → branches |
| **branch_locations** | Vùng GPS cho phép (lat, lng, radius) — haversine geofencing | N:1 → branches |
| **work_shifts** | Ca làm việc (giờ vào/ra, grace period, working days) | N:1 → branches |

### Attendance (Chấm công)

| Table | Vai trò | Quan hệ |
|---|---|---|
| **attendances** | Bản ghi chấm công hàng ngày — 1 record/user/ngày (check-in sớm nhất, check-out muộn nhất) | N:1 → users, branches, work_shifts |
| **attendance_logs** | Mỗi lần quét (QR, GPS, Face, Password) — nhiều logs/user/ngày. Lưu chi tiết: IP, GPS, accuracy, device fingerprint, anomaly score | N:1 → users, branches |
| **attendance_adjustments** | Đơn bổ sung công khi quên chấm — workflow pending/approved/rejected | N:1 → users, attendances |
| **user_shift_assignments** | Gán nhân viên vào ca làm việc theo thời gian hiệu lực | N:1 → users, work_shifts |

### Leave Management (Nghỉ phép)

| Table | Vai trò | Quan hệ |
|---|---|---|
| **leave_types** | 7 loại phép: Nghỉ năm, Ốm, Việc riêng, Cưới, Tang, Thai sản, Không lương | Standalone |
| **leave_requests** | Đơn xin nghỉ — workflow: pending → approved/rejected. Khi approved, tự tạo attendance record | N:1 → users, leave_types |
| **leave_balances** | Số ngày phép còn lại theo năm + loại phép. Unique (user, type, year) | N:1 → users, leave_types |
| **overtime_requests** | Đơn xin OT — workflow tương tự leave | N:1 → users |
| **holidays** | Ngày lễ quốc gia/công ty/chi nhánh — branch_id null = toàn công ty | N:1 → branches (nullable) |

### Security & Anti-Fraud (Bảo mật)

| Table | Vai trò | Quan hệ |
|---|---|---|
| **user_credentials** | WebAuthn/FIDO2 credentials — Passkeys, FaceID, Touch ID. Lưu public key + sign count để detect clone | N:1 → users |
| **user_devices** | Device fingerprint (SHA-256 của browser info). Bind thiết bị với nhân viên, phát hiện thiết bị lạ | N:1 → users |
| **user_sessions** | JWT session tracking — max 3 sessions/user, auto-revoke oldest. Detect concurrent login | N:1 → users |
| **fraud_alerts** | Log cảnh báo gian lận: GPS giả, TOTP reuse, impossible travel, device mới, IP-location mismatch, anomaly | N:1 → users, branches |

### RBAC (Phân quyền)

| Table | Vai trò | Quan hệ |
|---|---|---|
| **permissions** | 31 quyền hạt nhỏ: attendance.check_in, leave.approve, report.export... | Standalone |
| **role_permissions** | Mapping role → permission. 4 roles × N permissions | N:1 → permissions |

### Gamification

| Table | Vai trò | Quan hệ |
|---|---|---|
| **user_streaks** | Chuỗi chấm công đúng giờ liên tiếp (current + max) | 1:1 → users |
| **user_badges** | Huy hiệu: early_bird, perfect_week, perfect_month, punctual | N:1 → users |

## Key Indexes

| Index | Table | Columns | Purpose |
|---|---|---|---|
| idx_att_user_date | attendances | (user_id, work_date) | Fast lookup: "User X đã chấm công ngày Y chưa?" |
| idx_att_branch_date | attendances | (branch_id, work_date) | Branch report: tất cả chấm công chi nhánh theo ngày |
| idx_att_date_status | attendances | (work_date, status) | Dashboard: đếm on_time/late/absent theo ngày |
| idx_log_user_date | attendance_logs | (user_id, work_date) | Lịch sử quét trong ngày của 1 user |
| idx_device_fp | user_devices | (fingerprint_hash) | Anti-fraud: lookup device nhanh |
| idx_fraud_type | fraud_alerts | (alert_type) | Filter alerts theo loại |
| idx_lb_unique | leave_balances | (user_id, leave_type_id, year) | Unique constraint: 1 balance/user/type/year |
| idx_rp_role_perm | role_permissions | (role, permission_id) | Unique constraint: không duplicate mapping |
