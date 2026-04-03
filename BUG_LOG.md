# Bug Log — Smart Attendance

> Ghi nhận các bug phát hiện, phân tích nguyên nhân, và fix đã áp dụng.

---

## BUG-001: Camera không mở được cho role Employee

| Field | Detail |
|---|---|
| **Ngày phát hiện** | 2026-03-31 |
| **Mức độ** | Critical |
| **Trang** | `/attendance` (Check-in page) |
| **Triệu chứng** | Employee mở trang check-in, camera không khởi động. Hiển thị "Không thể mở camera" hoặc mở camera trước (selfie) thay vì camera sau |
| **Nguyên nhân** | Typo trong `html5-qrcode` API: `faceMode` thay vì `facingMode`. Library không nhận constraint camera, fallback fail hoặc mở camera sai |
| **File** | `web/templates/pages/attendance.html:176` |
| **Before** | `html5QrCode.start({ faceMode: "environment" }, ...)` |
| **After** | `html5QrCode.start({ facingMode: "environment" }, ...)` |
| **Status** | **FIXED** |

---

## BUG-002: Geolocation TypeError khi đã check-in

| Field | Detail |
|---|---|
| **Ngày phát hiện** | 2026-03-31 |
| **Mức độ** | Low |
| **Trang** | `/attendance` (Check-in page — sau khi đã check-in) |
| **Triệu chứng** | Console JS báo `TypeError: Cannot set property 'value' of null` liên tục |
| **Nguyên nhân** | `navigator.geolocation.watchPosition` chạy unconditionally. Khi user đã check-in, form không render → `getElementById('lat')` return `null` → `.value = ...` crash |
| **File** | `web/templates/pages/attendance.html:109-134` |
| **Before** | Geolocation watcher chạy luôn, không guard null |
| **After** | Check `latField && lngField` tồn tại trước khi start watcher |
| **Status** | **FIXED** |

---

## BUG-003: Bottom nav thiếu menu items trên QR Display page

| Field | Detail |
|---|---|
| **Ngày phát hiện** | 2026-03-31 |
| **Mức độ** | Medium |
| **Trang** | `/attendance/qr/{branchID}` (Manager QR Display) |
| **Triệu chứng** | Mobile bottom nav chỉ hiển thị 2 items: Home + Lịch sử. Thiếu Dashboard, QR |
| **Nguyên nhân** | `QRDisplayPage` handler không truyền `UserRole` và `UserBranch` vào template data. Nav dùng `{{if eq .UserRole "manager"}}` → empty string → menu items bị ẩn |
| **File** | `internal/handler/attendance.go:100-107` |
| **Before** | Template data chỉ có `Branch`, `BranchID`, `TOTPCode`, `Remaining` |
| **After** | Thêm `"UserRole": middleware.GetUserRole(r)` và `"UserBranch": middleware.GetBranchID(r)` |
| **Status** | **FIXED** |

---

## BUG-004: Empty IP/Location whitelist = allow all

| Field | Detail |
|---|---|
| **Ngày phát hiện** | 2026-03-31 |
| **Mức độ** | Medium (security design) |
| **Trang** | Check-in flow (backend) |
| **Triệu chứng** | Branch chưa config IP whitelist hoặc Location list → validator return `true` (allow all) |
| **File** | `internal/service/ip_validator.go:18`, `internal/service/location_validator.go:19` |
| **Đánh giá** | By design — cho phép branch chỉ dùng 1-2 methods. Nếu method enabled mà whitelist rỗng → pass |
| **Fix** | Thêm `WARNING` log khi method enabled nhưng whitelist rỗng, giúp admin phát hiện misconfiguration qua server log |
| **Status** | **FIXED** — Warning log added |

---

## BUG-005: OR logic cho multi-method check-in

| Field | Detail |
|---|---|
| **Ngày phát hiện** | 2026-03-31 |
| **Mức độ** | Low (design decision) |
| **Triệu chứng** | Branch config 3 methods nhưng user chỉ cần pass 1 để check-in |
| **File** | `internal/service/attendance_service.go:138` |
| **Đánh giá** | `AllowedMethods` = "phương thức được phép" (OR), không phải "bắt buộc" (AND). OR logic phù hợp UX |
| **Status** | **BY DESIGN** |

---

---

## BUG-006: Admin bị 403 Forbidden trên mọi API/page

| Field | Detail |
|---|---|
| **Ngày phát hiện** | 2026-03-31 |
| **Mức độ** | Critical |
| **Trang** | Tất cả trang/API sau khi login (dashboard, branches, users, attendance...) |
| **Triệu chứng** | Role admin gọi bất kỳ endpoint nào đều trả về 403 Forbidden. Manager và Employee cũng bị |
| **Nguyên nhân** | `Seed()` function có guard `if userCount > 0 { return nil }` ở đầu. DB đã có users từ trước khi thêm permission system → `seedPermissions()` nằm cuối `Seed()` không bao giờ được gọi → bảng `permissions` và `role_permissions` rỗng → `RequirePermission` middleware query DB, tìm 0 permissions → deny tất cả |
| **File** | `internal/database/seed.go:13-97` |
| **Before** | `seedPermissions()`, `seedLeaveTypes()`, `seedHolidays()` nằm sau user guard → chỉ chạy khi DB trống hoàn toàn |
| **After** | Di chuyển 3 seed functions lên TRƯỚC user guard. Mỗi function tự kiểm tra `count > 0` nên idempotent |
| **Status** | **FIXED** |

---

---

## BUG-007: Admin UI hiển thị tính năng Mã QR không cần thiết

| Field | Detail |
|---|---|
| **Ngày phát hiện** | 2026-03-31 |
| **Mức độ** | Low (UX) |
| **Trang** | Nav bar (desktop + mobile), Home page |
| **Triệu chứng** | Admin thấy nút "Mã QR" ở 3 vị trí: desktop nav, mobile bottom nav, home page. Admin không quản lý QR trực tiếp — đó là việc của Manager tại chi nhánh |
| **Nguyên nhân** | Template dùng `{{if or (eq .UserRole "admin") (eq .UserRole "manager")}}` cho mục QR thay vì chỉ `manager` |
| **File** | `web/templates/components/nav.html`, `web/templates/pages/home.html` |
| **Fix** | QR chỉ hiển thị cho `manager`. Admin home thay QR bằng "Báo cáo" + "Chi nhánh" (primary) |
| **Status** | **FIXED** |

---

## BUG-008: /users và /branches load trang trắng từ admin

| Field | Detail |
|---|---|
| **Ngày phát hiện** | 2026-03-31 |
| **Mức độ** | Critical |
| **Trang** | `/users`, `/branches` (và mọi page navigate qua hx-boost links) |
| **Triệu chứng** | Click link từ home page vào /users hoặc /branches → trang trắng |
| **Nguyên nhân** | 2 lỗi chồng nhau: (1) `hx-boost="true"` trên `<body>` khiến HTMX gửi `HX-Request: true` khi click link → handler trả partial thay vì full page. (2) Partial files dùng `{{define "user_table"}}` wrapper → Go template `Execute` chạy template chính (filename) = empty content |
| **File** | `internal/handler/user.go`, `internal/handler/branch.go`, `internal/renderer/renderer.go`, `web/templates/partials/*.html`, `web/templates/pages/*.html` |
| **Fix** | (1) Handler kiểm tra `HX-Boosted` header: boosted requests → full page, non-boosted HTMX → partial. (2) Partials bỏ `{{define}}` wrapper, page templates reference by filename `{{template "user_list.html" .}}` |
| **Status** | **FIXED** |

---

## BUG-009: Docker build Go version mismatch

| Field | Detail |
|---|---|
| **Ngày phát hiện** | 2026-04-02 |
| **Mức độ** | Medium |
| **Trang** | CI/CD (Docker Build) |
| **Triệu chứng** | Docker build failed với lỗi `go.mod` require Go 1.25.1 nhưng image builder dùng 1.22 |
| **Nguyên nhân** | `Dockerfile` chưa update image `golang:alpine` lên version mới nhất để khớp với `go.mod` |
| **Fix** | Update `Dockerfile` sang `golang:1.25-alpine` |
| **Status** | **FIXED** |

---

## BUG-010: SQLite error: table user_credentials has no column named authenticator_aa_guid

| Field | Detail |
|---|---|
| **Ngày phát hiện** | 2026-04-02 |
| **Mức độ** | High |
| **Trang** | WebAuthn Registration/Login |
| **Triệu chứng** | GORM báo lỗi không tìm thấy cột `authenticator_aa_guid` khi migrate hoặc query |
| **Nguyên nhân** | Naming mismatch giữa GORM (tự sinh name) và thực tế schema. GORM parse `AuthenticatorAAGUID` thành `authenticator_aa_guid` thay vì `authenticator_aaguid` |
| **Fix** | Thêm explicit column tag: `gorm:"column:authenticator_aaguid"` vào model |
| **Status** | **FIXED** |

---

## BUG-011: WebAuthn "Backup Eligible flag inconsistency"

| Field | Detail |
|---|---|
| **Ngày phát hiện** | 2026-04-02 |
| **Mức độ** | High |
| **Trang** | WebAuthn Login |
| **Triệu chứng** | Login thất bại với lỗi: `Backup Eligible flag inconsistency detected during login validation` |
| **Nguyên nhân** | Model thiếu `BackupEligible` flag. DB lưu mặc định là `false` (0) nhưng thiết bị (iPhone/Mac) báo là `true`. Library detect sự khác biệt và từ chối login |
| **Fix** | (1) Thêm backup flags vào model. (2) Chạy migration UPDATE `backup_eligible = 1` cho user cũ. (3) Map flags trong `ToWebAuthn()` |
| **Status** | **FIXED** |

---

## BUG-012: Panic in PasswordCheckinPage (Type Assertion)

| Field | Detail |
|---|---|
| **Ngày phát hiện** | 2026-04-02 |
| **Mức độ** | Critical |
| **Trang** | /attendance/password-checkin |
| **Triệu chứng** | Server panic: `interface conversion: interface {} is models.Role, not string` |
| **Nguyên nhân** | `userContext` trả về `models.Role`, nhưng handler ép kiểu (type assertion) sang `string`. Go không cho phép ép kiểu trực tiếp giữa named type và underlying type trong interface assertion. |
| **Fix** | Cập nhật type assertion sang `models.Role` và sửa logic so sánh trong `HomeHandler`. |
| **Status** | **FIXED** |

| # | Bug | Severity | Status |
|---|---|---|---|
| BUG-001 | Camera `faceMode` typo | Critical | FIXED |
| BUG-002 | Geolocation TypeError | Low | FIXED |
| BUG-003 | Bottom nav thiếu trên QR page | Medium | FIXED |
| BUG-004 | Empty whitelist = allow all | Medium | FIXED (warning log) |
| BUG-005 | OR logic multi-method | Low | BY DESIGN |
| BUG-006 | Admin 403 trên mọi endpoint | Critical | FIXED |
| BUG-007 | Admin UI hiển thị Mã QR | Low | FIXED |
| BUG-008 | /users, /branches trang trắng | Critical | FIXED |
| BUG-009 | Docker Go version mismatch | Medium | FIXED |
| BUG-010 | SQLite column naming error | High | FIXED |
| BUG-011 | WebAuthn flag inconsistency | High | FIXED |
| BUG-012 | Panic in PasswordCheckinPage | Critical | FIXED |
