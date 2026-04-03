# Smart Attendance - Context File

## Project Overview

Hệ thống chấm công thông minh (Smart Attendance) cho doanh nghiệp quy mô **100 chi nhánh**, **5.000 nhân viên**. Xác định vị trí bằng WiFi và/hoặc GPS. Chống gian lận (fake GPS, VPN).

## Tech Stack

### Backend
- **Language**: Go (latest stable)
- **Framework**: Chi router (lightweight, idiomatic Go)
- **Database**: SQLite (file-based, zero config, embedded)
- **Cache**: In-process cache (`github.com/patrickmn/go-cache` hoặc sync.Map)
- **ORM**: GORM (với `glebarez/sqlite` — pure-Go driver, no CGO required)
- **Auth**: JWT + Refresh Token (`golang-jwt/jwt`) + WebAuthn (Passkeys/Biometrics)
- **WebAuthn**: `github.com/go-webauthn/webauthn` (FIDO2/U2F)
- **API**: RESTful, pagination + filtering on all list endpoints

### Frontend
- **HTMX**: Dynamic UI without JavaScript framework
- **Template Engine**: Go `html/template` (server-side rendering)
- **CSS**: Tailwind CSS (via CDN hoặc standalone CLI)
- **Charts**: Chart.js (via CDN, triggered by HTMX)

### Infrastructure
- **Container**: Docker + docker-compose (one command deploy)
- **Dockerfile**: Multi-stage build (builder -> production)
- **Env**: `.env.example` provided, never commit `.env` or secrets
- **Single binary**: Go compiles thành 1 binary, bao gồm cả templates + static files (embed)

## Architecture

```
smart_attendance/
├── cmd/
│   └── server/
│       └── main.go          # Entry point
├── internal/
│   ├── config/              # App configuration (env, defaults)
│   ├── models/              # GORM models (User, Branch, Attendance...)
│   ├── handler/             # HTTP handlers (grouped by feature)
│   │   ├── auth.go
│   │   ├── branch.go
│   │   ├── attendance.go
│   │   ├── report.go
│   │   └── dashboard.go
│   ├── service/             # Business logic layer
│   │   ├── auth.go
│   │   ├── branch.go
│   │   ├── attendance.go
│   │   ├── report.go
│   │   └── dashboard.go
│   ├── repository/          # Data access layer (GORM queries)
│   ├── middleware/           # JWT auth, RBAC, rate limiting, logging
│   ├── cache/               # In-memory cache wrapper (go-cache)
│   └── validator/           # Input validation helpers
├── web/
│   ├── templates/           # Go html/template files
│   │   ├── layouts/         # Base layouts (admin, auth)
│   │   ├── partials/        # HTMX partial fragments
│   │   ├── pages/           # Full page templates
│   │   └── components/      # Reusable template components
│   └── static/              # CSS, JS, images
│       ├── css/
│       └── js/
├── migrations/              # SQLite migration SQL files
├── data/                    # SQLite database file (gitignored)
├── docker-compose.yml       # Full stack orchestration
├── Dockerfile               # Multi-stage build
├── go.mod
├── go.sum
├── .env.example             # Environment template
├── CLAUDE.md                # This file - AI context
├── PROMPT_LOG.md            # AI prompt & result log
└── README.md                # Setup guide & scaling strategy
```

## Core Features

### 1. Check-in / Check-out
- **4 phương thức xác thực song song** (multi-factor):
  - **Camera QR Scanner**: Nhân viên sử dụng camera điện thoại/máy tính quét mã QR tại chi nhánh. Mã QR chứa TOTP code reset mỗi 15 giây. Tự động bóc tách và gửi lệnh điểm danh.
  - **IP Whitelist**: Mỗi chi nhánh cấu hình danh sách IP được phép (mạng nội bộ công ty). Request check-in phải từ IP trong whitelist.
  - **Location Whitelist**: Mỗi chi nhánh cấu hình tọa độ (lat, lng) + bán kính. GPS của nhân viên phải nằm trong vùng cho phép (haversine distance).
  - **Biometric (WebAuthn)**: Nhân viên sử dụng Passkeys (FaceID/TouchID/Windows Hello) để điểm danh. Yêu cầu admin phê duyệt thiết bị trước khi sử dụng.
- Chống gian lận: TOTP expire 15s, IP verify, GPS geofencing, detect mock location, Biometric hardware-backed security
- Mỗi nhân viên chỉ check-in được tại chi nhánh được gán
- Hỗ trợ check-in bằng 1 hoặc kết hợp nhiều phương thức (configurable per branch)

### 2. Branch Management (Quản lý chi nhánh)
- CRUD chi nhánh với cấu hình check-in riêng cho từng địa điểm
- Gán nhân viên vào chi nhánh
- Mỗi chi nhánh cấu hình:
  - **QR/TOTP**: secret key riêng, interval 15s, hiển thị QR trên màn hình tại chi nhánh
  - **IP Whitelist**: danh sách IP/CIDR được phép check-in (e.g., `192.168.1.0/24`)
  - **Location Whitelist**: tọa độ trung tâm (lat, lng) + bán kính cho phép (meters)
  - **Allowed methods**: cấu hình phương thức nào bắt buộc (QR / IP / Location / kết hợp)

### 3. History & Reports (Lịch sử & Báo cáo)
- Xem theo ngày/tuần/tháng
- Trạng thái: đúng giờ / trễ / vắng
- Tổng giờ làm, overtime
- Export báo cáo (CSV/Excel)

### 4. Dashboard
- Thống kê tổng hợp toàn hệ thống
- Lọc theo chi nhánh / phòng ban
- Xuất báo cáo

### 5. Authorization (Phân quyền)
- **Admin**: Toàn hệ thống - quản lý tất cả chi nhánh, nhân viên, báo cáo
- **Manager**: Quản lý chi nhánh được gán - xem báo cáo chi nhánh
- **Employee**: Xem thông tin cá nhân, check-in/check-out

## Database Design Conventions

- **SQLite** single file database tại `data/smart_attendance.db`
- Schema hỗ trợ multi-branch: mọi bảng liên quan đều có `branch_id`
- Sử dụng UUID (TEXT) cho primary key
- Soft delete (`deleted_at` timestamp) cho dữ liệu quan trọng
- Index trên các cột filter phổ biến: `branch_id`, `user_id`, `created_at`
- Pagination bắt buộc trên mọi list endpoint (offset-based, cursor-based cho large datasets)
- WAL mode enabled cho concurrent read performance
- GORM AutoMigrate cho development, SQL migration files cho production

## API Conventions

- RESTful naming: `GET /api/v1/branches`, `POST /api/v1/attendance/check-in`
- Response format thống nhất:
  ```json
  {
    "success": true,
    "data": {},
    "meta": { "page": 1, "limit": 20, "total": 100 }
  }
  ```
- Error format:
  ```json
  {
    "success": false,
    "error": { "code": "INVALID_LOCATION", "message": "..." }
  }
  ```
- Tất cả endpoint cần auth phải qua JWT middleware
- Rate limiting trên check-in endpoint (chống spam)
- HTMX endpoints trả về HTML fragments (partial), API endpoints trả về JSON
- HTMX routes: `GET /branches` (page), `GET /branches/list` (partial fragment)
- API routes: `GET /api/v1/branches` (JSON)

## Code Conventions

### General
- Language: Go (idiomatic, gofmt formatted)
- Naming: Go conventions — camelCase unexported, PascalCase exported
- File naming: snake_case (e.g., `check_in.go`, `branch_handler.go`)
- Proper error handling: luôn check và return error, không panic
- **Logging rõ ràng để trace bug**:
  - Mỗi log phải có context: `[layer][module][action]` — ví dụ: `[handler][auth] login failed: invalid credentials, email=user@example.com`
  - Log level: `log.Printf` cho info, `log.Printf("ERROR: ...")` cho errors
  - Handler layer: log request errors (400/500) kèm request method, path, error message
  - Service layer: log business logic errors kèm input parameters liên quan
  - Repository layer: GORM đã tự log SQL queries (development mode)
  - Middleware: log mỗi request (method, path, status, duration) — đã có Logger middleware
  - Không log sensitive data: password, JWT token, password_hash

### Backend (Go)
- **Handler → Service → Repository** pattern (3-layer)
- Handler chỉ parse request + render response, business logic ở Service
- Repository layer wrap GORM queries, không dùng GORM trực tiếp trong Service
- Dependency injection qua constructor (không dùng global variables)
- Middleware chain: Logger → Recovery → RateLimit → JWT Auth → RBAC
- Input validation tại Handler layer trước khi gọi Service

### Frontend (HTMX + Go Templates)
- Server-side rendering với Go `html/template`
- HTMX attributes cho dynamic behavior (hx-get, hx-post, hx-swap, hx-target)
- Partial templates cho HTMX fragment responses (không reload full page)
- Tailwind CSS cho styling (responsive, mobile-first)
- Minimal JavaScript — chỉ dùng khi HTMX không đủ (e.g., Chart.js, geolocation API)
- Template inheritance: `layouts/base.html` → `pages/*.html` → `partials/*.html`

## Git Flow

- **Branches**: `main` → `develop` → `feature/*` | `release/*` | `hotfix/*`
- **Mỗi feature** = 1 branch + 1 PR + review
- **Commit messages**: Conventional Commits format
  - `feat:` new feature
  - `fix:` bug fix
  - `docs:` documentation
  - `refactor:` code refactoring
  - `test:` adding tests
  - `chore:` maintenance
- Ví dụ: `feat(attendance): add WiFi-based check-in validation`

## Scaling Strategy

- **SQLite WAL mode**: concurrent reads không block nhau, write serialized
- **In-memory cache** (go-cache): cache branch config, dashboard aggregation, giảm DB reads
- **Single binary deployment**: Go compile thành 1 binary, embed templates + static files
- API: stateless design, có thể horizontal scale bằng cách chuyển sang PostgreSQL khi cần
- Database indexing strategy cho 100 branches x 5.000 employees
- Goroutine-based concurrency cho xử lý peak-hour check-in

## Docker

- `docker-compose up` chạy toàn bộ stack (single Go binary + SQLite volume mount)
- Dockerfile sử dụng multi-stage build: `golang:alpine` build → `alpine` runtime (~15MB)
- `.env.example` có sẵn, copy thành `.env` trước khi chạy
- Không commit secrets vào repo

## AI IDE Workflow

- **Workflow**: Spec → AI generate → Review & refine → Test → Commit
- **Review 100%** code AI sinh ra trước khi commit
- Ghi log prompt + kết quả vào `PROMPT_LOG.md`
