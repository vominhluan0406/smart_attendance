# AI Prompt Log — Smart Attendance

> Tài liệu ghi nhận quy trình sử dụng AI IDE trong quá trình phát triển dự án.
> Workflow: **Spec → AI Generate → Review & Refine → Test → Commit**

## Conventions

| Field | Description |
|---|---|
| **Task** | Mục tiêu cần đạt được |
| **Spec** | Yêu cầu chi tiết, input/output mong muốn |
| **AI Tool** | Công cụ AI sử dụng (Claude Code, Copilot...) |
| **Prompt** | Prompt gửi cho AI |
| **Output** | Tóm tắt kết quả AI sinh ra |
| **Review** | Đánh giá: Accepted / Modified / Rejected. Lý do nếu Modified/Rejected |
| **Changes** | Các thay đổi thủ công sau review (nếu có) |
| **Files** | Danh sách file bị ảnh hưởng |
| **Commit** | Commit hash sau khi merge |

---

## Session 1 — Project Setup & Configuration (2026-03-30)

### 1.1 — Khởi tạo context file

| Field | Detail |
|---|---|
| **Task** | Tạo CLAUDE.md mô tả kiến trúc, conventions, tech stack cho AI IDE |
| **Spec** | Dựa trên đề bài Smart Attendance: 100 chi nhánh, 5.000 nhân viên, check-in WiFi/GPS, dashboard, RBAC |
| **AI Tool** | Claude Code (Opus) |
| **Prompt** | `Generate a CLAUDE.md context file for AI IDE. Include: project overview, tech stack, directory structure, core features (check-in/out, branch management, reports, dashboard, RBAC), database/API/code conventions, git flow, scaling strategy, and Docker setup. Base on the attached project specification document.` |
| **Output** | File CLAUDE.md hoàn chỉnh: project overview, tech stack (NestJS + Next.js + PostgreSQL + Redis), project structure, 5 core features, DB/API/code conventions, git flow, scaling strategy, docker setup |
| **Review** | **Modified** — Tech stack ban đầu (Node.js/React) quá nặng cho scope dự án, chuyển sang Go monolith |
| **Changes** | Xem mục 1.2 |
| **Files** | `CLAUDE.md` |
| **Commit** | — |

### 1.2 — Điều chỉnh tech stack

| Field | Detail |
|---|---|
| **Task** | Chuyển tech stack sang Go + SQLite + HTMX cho phù hợp yêu cầu lightweight, single binary |
| **Spec** | BE: Go (Chi router, GORM). DB: SQLite (WAL mode). Cache: go-cache in-memory. FE: HTMX + Go html/template + Tailwind CSS |
| **AI Tool** | Claude Code (Opus) |
| **Prompt** | `Refactor CLAUDE.md tech stack: replace Node.js/NestJS with Go (Chi router, GORM), PostgreSQL with SQLite (WAL mode), Redis with in-process go-cache, and Next.js frontend with HTMX + Go html/template + Tailwind CSS. Update all related sections: architecture, code conventions, scaling strategy, Docker config.` |
| **Output** | Cập nhật 6 sections trong CLAUDE.md: Tech Stack, Architecture (Go project layout cmd/internal/web), DB conventions (SQLite WAL, GORM), Code conventions (Go idiomatic, Handler→Service→Repository), Scaling strategy (in-memory cache, goroutine concurrency), Docker (alpine ~15MB) |
| **Review** | **Accepted** — Stack nhẹ, deploy đơn giản, phù hợp single binary + docker-compose |
| **Changes** | Không |
| **Files** | `CLAUDE.md` |
| **Commit** | — |

### 1.3 — Tạo task breakdown

| Field | Detail |
|---|---|
| **Task** | Phân rã toàn bộ dự án thành tasks có thể thực thi, theo phase |
| **Spec** | Map tasks theo feature branch (git flow), ước lượng T-shirt size, dependency graph, critical path |
| **AI Tool** | Claude Code (Opus) |
| **Prompt** | `Create TASKS.md: break down the entire project into phases and actionable tasks. Each task should map to a feature branch (git flow), include T-shirt size estimate, dependencies, and status. Add a dependency graph with critical path analysis and map tasks to evaluation criteria.` |
| **Output** | TASKS.md: 7 phases (P0–P6), ~40 tasks, dependency graph, tiêu chí đánh giá mapping |
| **Review** | **Accepted** |
| **Changes** | Không |
| **Files** | `TASKS.md` |
| **Commit** | — |

---

## Session 2 — Phase 0: Project Skeleton & Infrastructure (2026-03-30)

### 2.1 — Implement Phase 0 (all tasks)

| Field | Detail |
|---|---|
| **Task** | Triển khai toàn bộ Phase 0: init project, config, DB, templates, server, Docker |
| **Spec** | Go module + dependencies → directory structure → config (.env) → SQLite/GORM (WAL) → base templates (HTMX/Tailwind) → Chi router + middleware → Dockerfile + docker-compose |
| **AI Tool** | Claude Code (Opus) |
| **Prompt** | `Implement all Phase 0 tasks: initialize Go module with dependencies (chi, gorm, jwt, go-cache, uuid), create project directory structure per CLAUDE.md, build config module (.env loader), setup SQLite/GORM with WAL mode, create base HTML templates (layouts/base, app, auth) with HTMX + Tailwind CDN, bootstrap Chi HTTP server with logger/recovery middleware and template renderer, create Dockerfile (multi-stage alpine) + docker-compose.yml + .env.example, add .gitignore and README.md.` |
| **Output** | 8 tasks hoàn thành: go.mod (chi, gorm, jwt, go-cache, uuid), project structure (cmd/internal/web), config module, SQLite + GORM + WAL mode, 3 template layouts (base/app/auth) + home page + alert component, Chi router + logger/recovery middleware + renderer, Dockerfile multi-stage + docker-compose + .env.example |
| **Review** | **Modified** — `gorm.io/driver/sqlite` (CGO) không build được trên Windows → chuyển sang `glebarez/sqlite` (pure-Go, no CGO). Server tested: build OK, startup OK, WAL mode enabled |
| **Changes** | Thay SQLite driver: `gorm.io/driver/sqlite` → `glebarez/sqlite` (pure-Go). Cập nhật CLAUDE.md reflect driver change |
| **Files** | `cmd/server/main.go`, `internal/config/config.go`, `internal/database/database.go`, `internal/cache/cache.go`, `internal/models/base.go`, `internal/renderer/renderer.go`, `internal/middleware/logger.go`, `internal/middleware/recovery.go`, `internal/handler/home.go`, `internal/router/router.go`, `web/templates/layouts/*.html`, `web/templates/pages/home.html`, `web/templates/components/alert.html`, `Dockerfile`, `docker-compose.yml`, `.env.example`, `.dockerignore`, `.gitignore`, `README.md` |
| **Commit** | — |

---

## Session 3 — Phase 1: Auth & User Management (2026-03-30)

### 3.1 — Implement Phase 1 (all tasks 1.1–1.9)

| Field | Detail |
|---|---|
| **Task** | Implement complete authentication system and user management module |
| **Spec** | User model → UserRepository (CRUD + paginated list) → AuthService (bcrypt + JWT pair) → JWT middleware (header + cookie) → RBAC middleware → Auth handlers (API + HTMX) → User CRUD (API + HTMX pages) → Seed admin |
| **AI Tool** | Claude Code (Opus) |
| **Prompt** | `Implement all Phase 1 tasks: User GORM model with role enum/UUID/soft-delete/is_active. UserRepository with paginated+filterable List. AuthService with bcrypt+JWT pair generation+refresh+validation. JWT middleware (Authorization header + cookie, context injection, HTMX-aware redirect). RBAC middleware (RequireRoles, AdminOnly, ManagerOrAdmin). Auth handlers for API JSON + HTMX forms + cookie management + logout. Login/register templates with HTMX submit+spinner+error partial. User CRUD Admin pages (list+search+filter+pagination, create, edit, delete). Seed default admin on empty DB. Wire dependencies in main.go and router.go.` |
| **Output** | 19 files created/updated. User model (Role enum, BaseModel embed), UserRepository (paginated List with 4 filters), AuthService (bcrypt + HS256 JWT pair + refresh + validate), JWT middleware (header + cookie + HTMX-aware redirect), RBAC middleware (RequireRoles generic + 2 convenience wrappers), AuthHandler (3 API + 2 HTMX form + logout), UserHandler (4 API + 5 HTMX endpoints), UserService (GetByID, List, Update, Delete), 6 templates, seed admin, renderer FuncMap for pagination, Deps struct in router |
| **Review** | **Accepted** — `go build ./...` passes. Server starts: users table migrated (3 indexes), admin seeded (admin@smartattendance.com / admin123), all routes registered |
| **Changes** | Renderer refactored: added FuncMap (add/subtract/multiply/int/int64/divCeil), pages parsed with partials for shared defines |
| **Files** | `internal/models/user.go`, `internal/repository/user_repository.go`, `internal/service/auth_service.go`, `internal/service/user_service.go`, `internal/middleware/auth.go`, `internal/middleware/rbac.go`, `internal/handler/auth.go`, `internal/handler/user.go`, `internal/database/seed.go`, `internal/renderer/renderer.go`, `internal/router/router.go`, `cmd/server/main.go`, `web/templates/pages/{login,register,users,user_create,user_edit}.html`, `web/templates/partials/{user_list,auth_error}.html` |
| **Commit** | — |

---

## Session 4 — Phase 2: Branch Management (2026-03-30)

### 4.1 — Update BA: QR TOTP + IP/Location Whitelist

| Field | Detail |
|---|---|
| **Task** | Add new check-in methods to project spec |
| **Spec** | 3 methods: QR Code (TOTP 15s), IP Whitelist (CIDR), Location Whitelist (lat/lng + radius) |
| **AI Tool** | Claude Code (Opus) |
| **Prompt** | `Add business requirement: check-in via QR Code (TOTP reset every 15s) + IP whitelist per branch (CIDR) + Location whitelist per branch (lat/lng + radius). Update CLAUDE.md features, branch config. Update TASKS.md Phase 2 and Phase 3 with new tasks.` |
| **Output** | Updated CLAUDE.md (3 check-in methods, branch config), TASKS.md Phase 2 + Phase 3 (tasks 3.3–3.11) |
| **Review** | **Accepted** |
| **Files** | `CLAUDE.md`, `TASKS.md` |
| **Commit** | — |

### 4.2 — Implement Phase 2 (all tasks 2.1–2.7)

| Field | Detail |
|---|---|
| **Task** | Implement branch management with QR TOTP secret, IP whitelist, location whitelist |
| **Spec** | Branch + BranchIPWhitelist + BranchLocation models → BranchRepository (CRUD, Preload, Replace) → BranchService (TOTP gen, cache, employee assign) → BranchHandler (API + HTMX) → Templates → Wire router + main.go |
| **AI Tool** | Claude Code (Opus) |
| **Prompt** | `Implement all Phase 2 tasks: Branch model (totp_secret, allowed_methods, work times) + BranchIPWhitelist/BranchLocation. Repository with CRUD, paginated list, Preload, transactional Replace. Service with TOTP secret gen, GetByIDCached (go-cache), IP/Location update, employee assign/unassign. Handler with 5 API + 5 HTMX endpoints. Templates (list, create, edit with IP/Location/Methods config). Add containsMethod template func. Wire in router + main.go.` |
| **Output** | 10 files created/updated. 3 models, repository, service (TOTP + cache + assign), handler (API + HTMX), 4 templates, renderer FuncMap, router + main.go wired |
| **Review** | **Accepted** — `go vet ./...` passes clean |
| **Changes** | Renderer FuncMap: replaced arithmetic helpers with `containsMethod` for checkbox state |
| **Files** | `internal/models/branch.go`, `internal/repository/branch_repository.go`, `internal/service/branch_service.go`, `internal/handler/branch.go`, `internal/renderer/renderer.go`, `internal/router/router.go`, `cmd/server/main.go`, `web/templates/pages/{branches,branch_create,branch_edit}.html`, `web/templates/partials/branch_list.html` |
| **Commit** | — |
---

## Session 5 — Phase 4: History, Reports & QR Enhancements (2026-03-30)

### 5.1 — Implement Phase 4 (all tasks 4.1–4.6)

| Field | Detail |
|---|---|
| **Task** | Implement Attendance History, Excel Export, and QR Display enhancements |
| **Spec** | ReportRepository (Filter queries) → ReportService (History, Excelize export) → ReportHandler (HTMX partials + Export) → Templates (History page, Branch report) → QR UI (Countdown timer, auto-refresh) |
| **AI Tool** | Claude Code (Opus) |
| **Prompt** | `Implement all Phase 4 tasks: ReportService for user and branch attendance history with filtering (date, status). Add Excel export using github.com/xuri/excelize/v2. Create ReportHandler with HTMX partial rendering for tables. Update QR display with a live 15s countdown timer and CSS progress bar. Implement event-driven QR refresh using 'refresh-qr' custom event instead of polling. Standardize partials by removing 'define' blocks.` |
| **Output** | ReportService (Excelize integrated), ReportHandler (HTMX-aware), History templates, and refactored QR Display with JS countdown and custom events. |
| **Review** | **Accepted** — QR code refreshes perfectly at 0s. Excel files generated correctly with headers and status colors. |
| **Changes** | Fixed template rendering issue by removing `{{define}}` from partials intended for standalone rendering. |
| **Files** | `internal/service/report_service.go`, `internal/handler/report.go`, `web/templates/pages/{my_history,branch_report}.html`, `web/templates/partials/qr_code.html`, `web/templates/pages/qr_display.html` |
| **Commit** | — |

---

## Session 6 — RBAC & Camera Scanner Implementation (2026-03-30)

### 6.1 — Business Analysis & RBAC Enforcement

| Field | Detail |
|---|---|
| **Task** | Document and implement full RBAC as per BA requirements |
| **Spec** | Create business_analysis_rbac.md → Update JWT Claims with BranchID → Enforce branch-level access in handlers → Conditional UI (Nav/Home) based on role |
| **AI Tool** | Claude Code (Opus) |
| **Prompt** | `Create a BA document for RBAC detailing Employee/Manager/Admin permissions. Then implement it: Add BranchID to JWT, update middleware to inject it. Secure ReportHandler and AttendanceHandler so Managers only see their own branch. Update nav.html and home.html to hide unauthorized links/buttons. Also, replace manual check-in with a camera-based QR scanner using html5-qrcode.` |
| **Output** | business_analysis_rbac.md created. JWT claims updated. Secrity checks added to 6 handler methods. Navigation bar and home screen are now role-aware. |
| **Review** | **Accepted** — Camera scanner works fluently on mobile. Admin/Manager see only relevant data. |
| **Files** | `business_analysis_rbac.md`, `internal/service/auth_service.go`, `internal/middleware/auth.go`, `internal/handler/{report,attendance,home,user,branch}.go`, `web/templates/components/nav.html`, `web/templates/pages/attendance.html` |
| **Commit** | — |
