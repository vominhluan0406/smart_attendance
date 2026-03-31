# Smart Attendance — Task Breakdown

> Phân rã công việc theo phase. Mỗi task map với 1 feature branch theo git flow.
> Ước lượng theo **T-shirt size**: S (< 2h), M (2–4h), L (4–8h), XL (> 8h)

## Phase 0 — Project Skeleton & Infrastructure

| # | Task | Branch | Size | Depends | Status |
|---|------|--------|------|---------|--------|
| 0.1 | Init Go module, cài dependencies (chi, gorm, jwt, go-cache) | `feature/init-project` | S | — | DONE |
| 0.2 | Cấu trúc thư mục theo CLAUDE.md (cmd, internal, web, migrations, data) | `feature/init-project` | S | — | DONE |
| 0.3 | Config module: load .env, struct Config, defaults | `feature/init-project` | S | — | DONE |
| 0.4 | SQLite connection + GORM setup (pure-Go driver) + WAL mode + AutoMigrate | `feature/init-project` | S | 0.3 | DONE |
| 0.5 | Base template layout (layouts/base.html, app.html, auth.html) + Tailwind CDN + HTMX CDN | `feature/init-project` | S | — | DONE |
| 0.6 | HTTP server bootstrap (Chi router + middleware + renderer + static files) | `feature/init-project` | M | 0.3, 0.4 | DONE |
| 0.7 | Dockerfile multi-stage + docker-compose.yml + .env.example + .dockerignore | `feature/docker-setup` | M | 0.6 | DONE |
| 0.8 | .gitignore (data/, .env, binary) + README.md skeleton | `feature/init-project` | S | — | DONE |

## Phase 1 — Auth & User Management

| # | Task | Branch | Size | Depends | Status |
|---|------|--------|------|---------|--------|
| 1.1 | Models: User (id, email, password_hash, full_name, role, branch_id, is_active, timestamps) | `feature/auth` | S | 0.4 | DONE |
| 1.2 | Repository: UserRepository (Create, FindByEmail, FindByID, List w/ pagination+filter, Update, Delete, Count) | `feature/auth` | M | 1.1 | DONE |
| 1.3 | Service: AuthService (Register, Login, RefreshToken, ValidateToken, bcrypt hash/verify, BranchID in JWT) | `feature/auth` | M | 1.2 | DONE |
| 1.4 | JWT middleware: extract token (header + cookie), validate, inject claims into context | `feature/auth` | M | 1.3 | DONE |
| 1.5 | RBAC middleware: RequireRoles, AdminOnly, ManagerOrAdmin | `feature/auth` | S | 1.4 | DONE |
| 1.6 | Handler: Auth API (login, register, refresh) + HTMX form handlers + cookie management | `feature/auth` | M | 1.3, 1.4 | DONE |
| 1.7 | Templates: login page, register page (HTMX form submit + loading indicator + error partial) | `feature/auth` | M | 1.6, 0.5 | DONE |
| 1.8 | Handler: User CRUD API + HTMX pages (list w/ search+filter, create, edit, delete) — Admin only | `feature/user-management` | L | 1.5, 1.6 | DONE |
| 1.9 | Seed data: tạo admin mặc định (admin@smartattendance.com / admin123) khi DB trống | `feature/auth` | S | 1.2 | DONE |

## Phase 2 — Branch Management

| # | Task | Branch | Size | Depends | Status |
|---|------|--------|------|---------|--------|
| 2.1 | Models: Branch (name, address, lat, lng, radius_m, totp_secret, allowed_methods, work times) | `feature/branch-management` | S | 0.4 | DONE |
| 2.2 | Models: BranchIPWhitelist (ip_cidr, label) + BranchLocation (lat, lng, radius_m, label) | `feature/branch-management` | S | 2.1 | DONE |
| 2.3 | Repository: BranchRepository (CRUD, paginated list, FindByID w/ Preload IPs+Locations, Replace IPs/Locations) | `feature/branch-management` | M | 2.1, 2.2 | DONE |
| 2.4 | Service: BranchService (CRUD + TOTP secret gen + IP/Location replace + assign/unassign employees + cache) | `feature/branch-management` | M | 2.3 | DONE |
| 2.5 | Handler: CRUD /api/v1/branches (Admin) + HTMX pages (list, create, edit with IP/Location/Methods config) | `feature/branch-management` | L | 2.4, 1.5 | DONE |
| 2.6 | Cache: GetByIDCached with go-cache, invalidate on update/delete | `feature/branch-management` | S | 2.4 | DONE |
| 2.7 | Assign/unassign employees to branch (service layer, wired in BranchService) | `feature/branch-management` | M | 2.4, 1.8 | DONE |

## Phase 3 — Check-in / Check-out (Core)

| # | Task | Branch | Size | Depends | Status |
|---|------|--------|------|---------|--------|
| 3.1 | Models: Attendance (id, user_id, branch_id, check_in_at, check_out_at, status, method, ip_address, lat, lng, totp_verified, location_data JSON) | `feature/attendance` | S | 0.4 | DONE |
| 3.2 | Repository: AttendanceRepository (Create, Update, FindTodayByUser, ListByDateRange) | `feature/attendance` | M | 3.1 | DONE |
| 3.3 | Service: QRTOTPService — generate TOTP secret per branch, produce QR code image, validate TOTP code (15s interval) | `feature/attendance` | L | 2.6 | DONE |
| 3.4 | Service: IPValidator — check request IP against branch IP whitelist (support CIDR notation) | `feature/attendance` | M | 2.6 | DONE |
| 3.5 | Service: LocationValidator — verify GPS coords against branch location whitelist (haversine distance) | `feature/attendance` | M | 2.6 | DONE |
| 3.6 | Service: AttendanceService (CheckIn, CheckOut — orchestrate multi-method validation per branch config) | `feature/attendance` | L | 3.2, 3.3, 3.4, 3.5 | DONE |
| 3.7 | Handler: POST /api/v1/attendance/check-in, /check-out + GET /api/v1/attendance/qr/:branch_id (QR image) | `feature/attendance` | M | 3.6, 1.4 | DONE |
| 3.8 | Rate limiting middleware trên check-in endpoint | `feature/attendance` | S | 3.7 | DONE |
| 3.9 | Templates: check-in page (QR scanner via camera + geolocation API + HTMX submit + real-time TOTP display for managers) | `feature/attendance` | L | 3.7, 0.5 | DONE |
| 3.10 | Auto status calculation: đúng giờ / trễ / vắng (dựa trên cấu hình giờ làm) | `feature/attendance` | M | 3.6 | DONE |
| 3.11 | Manager QR display page: show live QR code (auto-refresh 15s) for branch check-in | `feature/attendance` | M | 3.3 | DONE |

## Phase 4 — History & Reports

| # | Task | Branch | Size | Depends | Status |
|---|------|--------|------|---------|--------|
| 4.1 | Repository: queries cho attendance history (by user, branch, date range, status) | `feature/reports` | M | 3.1 | DONE |
| 4.2 | Service: ReportService (GetUserHistory, GetBranchReport, GetSummary, CalcOvertime) | `feature/reports` | L | 4.1 | DONE |
| 4.3 | Handler: GET /api/v1/reports/user/:id, GET /api/v1/reports/branch/:id | `feature/reports` | M | 4.2, 1.5 | DONE |
| 4.4 | Templates: attendance history page (filter ngày/tuần/tháng, HTMX partial reload) | `feature/reports` | L | 4.3 | DONE |
| 4.5 | Export CSV/Excel (go library: excelize) | `feature/reports` | M | 4.2 | DONE |
| 4.6 | Cache: cache report aggregation, invalidate khi có check-in mới | `feature/reports` | S | 4.2 | DONE |
| 4.7 | RBAC Enforcement: Branch-level security for Managers, JWT BranchID injection | `feature/rbac-polish` | M | P1, 4.2 | DONE |
| 4.8 | Camera Scanner: QR scanning via html5-qrcode for user check-in (replacing manual input) | `feature/attendance` | M | 3.9 | DONE |

## Phase 5 — Dashboard

| # | Task | Branch | Size | Depends | Status |
|---|------|--------|------|---------|--------|
| 5.1 | Service: DashboardService (tổng nhân viên, tổng check-in hôm nay, tỉ lệ đúng giờ, top trễ) | `feature/dashboard` | M | 4.2, 3.2 | DONE |
| 5.2 | Handler: GET /api/v1/dashboard/stats, GET /api/v1/dashboard/charts | `feature/dashboard` | M | 5.1 | DONE |
| 5.3 | Templates: dashboard page (stat cards + Chart.js charts + filter chi nhánh) | `feature/dashboard` | L | 5.2, 0.5 | DONE |
| 5.4 | HTMX: dynamic filter by branch/department (partial reload stats + charts) | `feature/dashboard` | M | 5.3 | DONE |
| 5.5 | Cache: cache dashboard stats (TTL 5 min), invalidate on new check-in | `feature/dashboard` | S | 5.1 | DONE |

## Phase 6 — Polish & Delivery

| # | Task | Branch | Size | Depends | Status |
|---|------|--------|------|---------|--------|
| 6.1 | Responsive UI: test & fix mobile layout cho tất cả pages | `feature/ui-polish` | M | P1–P5 | DONE |
| 6.2 | Error pages: 404, 403, 500 (template) | `feature/ui-polish` | S | 0.5 | DONE |
| 6.3 | Seed data: script tạo 100 branches + 5.000 employees + attendance records | `feature/seed-data` | M | P1–P3 | DONE |
| 6.4 | README.md: setup guide, architecture diagram, scaling strategy, screenshots | `docs/readme` | L | P0–P5 | DONE |
| 6.5 | Docker: test docker-compose up from scratch, verify 1-click deploy | `feature/docker-setup` | M | 0.7, P1–P5 | DONE |
| 6.6 | PROMPT_LOG.md: hoàn thiện log tất cả sessions | `docs/prompt-log` | S | — | DONE |

---

## Dependency Graph (Critical Path)

```
P0 (Skeleton) ──→ P1 (Auth) ──→ P2 (Branch) ──→ P3 (Attendance) ──→ P4 (Reports) ──→ P5 (Dashboard)
                                                                                            │
                                                                                            ▼
                                                                                      P6 (Polish)
```

**Critical path**: P0 → P1 → P2 → P3 → P4 → P5 → P6

**Parallelizable**:
- P2 (Branch) và P1.8 (User CRUD) có thể song song sau khi P1.5 (RBAC) done
- P4 (Reports) và P5 (Dashboard) có thể song song sau P3
- P6.4 (README) có thể viết dần xuyên suốt
- Docker (0.7) có thể setup sớm, test lại cuối

## Tiêu chí đánh giá mapping

| Tiêu chí | Tỷ trọng | Tasks liên quan |
|---|---|---|
| Tính năng & UX | 25% | P1–P5 (all features), 6.1 (responsive) |
| Kiến trúc & khả năng mở rộng | 20% | 0.4 (DB setup), 2.6/4.6/5.5 (cache), 6.3 (seed 5K) |
| Git Flow & Docker | 15% | 0.7 (Docker), 6.5 (verify), all branches + PRs |
| AI IDE workflow & Prompt Log | 15% | 6.6 (PROMPT_LOG.md), CLAUDE.md |
| Sáng tạo & khác biệt | 25% | 3.3 (QR TOTP 15s), 3.4 (IP whitelist), 3.5 (location whitelist), 3.11 (live QR display), HTMX approach, Go single binary |
