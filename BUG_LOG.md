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

## Summary

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
