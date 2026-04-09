# Smart Attendance - Context File

## Project Overview

Hệ thống chấm công thông minh (Smart Attendance) cho doanh nghiệp quy mô **100 chi nhánh**, **5.000 nhân viên**. Xác định vị trí bằng WiFi và/hoặc GPS. Chống gian lận (fake GPS, VPN).

**Kiến trúc: Microservices** — 5 Go backend services + 1 API Gateway + 1 Next.js frontend (7 containers).

## Tech Stack

### Backend (5 Go Microservices + Gateway)
- **Language**: Go (latest stable)
- **Framework**: Chi router (lightweight, idiomatic Go)
- **Database**: PostgreSQL (1 instance, schema-per-service isolation)
- **Cache**: In-process cache (`github.com/patrickmn/go-cache`)
- **ORM**: GORM (với `gorm.io/driver/postgres`)
- **Auth**: JWT + Refresh Token (`golang-jwt/jwt`) + WebAuthn (Passkeys/Biometrics)
- **WebAuthn**: `github.com/go-webauthn/webauthn` (FIDO2/U2F)
- **API**: RESTful JSON, pagination + filtering on all list endpoints
- **Inter-service**: Sync HTTP REST giữa các service (internal API)

### Frontend (Next.js)
- **Framework**: Next.js (App Router, SSR)
- **Language**: TypeScript
- **UI**: React components
- **CSS**: Tailwind CSS
- **Charts**: Chart.js (React wrapper)
- **Auth**: httpOnly cookie (JWT từ Gateway)

### Infrastructure
- **Container**: Docker + docker-compose (7 containers)
- **Dockerfile**: Multi-stage build per service (~15MB mỗi container Go)
- **Env**: `.env.example` provided, never commit `.env` or secrets
- **Gateway**: Go reverse proxy, JWT validation, rate limiting, CORS

## Architecture — Microservices

```
┌───────────────────┐
│  Next.js (3000)   │  ← SSR + React + Tailwind + TypeScript
└────────┬──────────┘
         │ HTTP/JSON
┌────────▼──────────┐
│  Gateway (8080)   │  ← JWT validate, rate limit, route, CORS
└──┬──┬──┬──┬──┬───┘
   │  │  │  │  │
   ▼  ▼  ▼  ▼  ▼
 Auth Attend Leave Analytics Org
 8081  8082  8083   8084    8085
```

### 5 Microservices

| Service | Port | Owns |
|---|---|---|
| **Auth** | 8081 | User, Session, Credential, Permission, RBAC |
| **Attendance** | 8082 | Attendance, AttendanceLog, FraudAlert, Device, Adjustment |
| **Leave** | 8083 | LeaveRequest, LeaveType, LeaveBalance |
| **Analytics** | 8084 | Read-only aggregation (Dashboard, Report, Export) |
| **Organization** | 8085 | Branch, IPWhitelist, Location, Shift, Department |

### Directory Structure

```
smart_attendance/
├── services/
│   ├── gateway/              # API Gateway
│   │   ├── cmd/main.go
│   │   └── internal/         # proxy, middleware, config
│   ├── auth/                 # Auth Service
│   │   ├── cmd/main.go
│   │   └── internal/         # handler, service, repository, model, database
│   ├── attendance/           # Attendance Service
│   │   ├── cmd/main.go
│   │   └── internal/         # + client/ (HTTP clients → Auth, Org)
│   ├── leave/                # Leave Service
│   │   ├── cmd/main.go
│   │   └── internal/         # + client/ (→ Auth, Attendance)
│   ├── analytics/            # Analytics Service
│   │   ├── cmd/main.go
│   │   └── internal/         # + client/ (→ all services)
│   └── organization/         # Organization Service
│       ├── cmd/main.go
│       └── internal/
├── frontend/                 # Next.js
│   ├── src/
│   │   ├── app/              # App Router (SSR pages)
│   │   ├── components/       # React components
│   │   └── lib/              # API client, auth, types
│   ├── tailwind.config.ts
│   └── package.json
├── shared/                   # Shared Go packages
│   ├── dto/                  # Inter-service DTOs
│   ├── middleware/            # Internal auth header parsing
│   └── response/             # Standard JSON response format
├── docker-compose.yml        # 7 containers
├── CLAUDE.md
└── README.md
```

### Monolith (legacy, still in repo root)

```
├── cmd/server/main.go        # Original monolith entry point
├── internal/                  # Original monolith code
├── web/templates/             # Original HTMX templates (reference)
```

## Core Features

### 1. Check-in / Check-out
- **4 phương thức xác thực song song** (multi-factor):
  - **Camera QR Scanner**: Nhân viên sử dụng camera điện thoại/máy tính quét mã QR tại chi nhánh. Mã QR chứa TOTP code reset mỗi 15 giây. Tự động bóc tách và gửi lệnh điểm danh.
  - **IP Whitelist**: Mỗi chi nhánh cấu hình danh sách IP được phép (mạng nội bộ công ty). Request check-in phải từ IP trong whitelist.
  - **Location Whitelist**: Mỗi chi nhánh cấu hình tọa độ (lat, lng) + bán kính. GPS của nhân viên phải nằm trong vùng cho phép (haversine distance).
  - **Biometric (WebAuthn)**: Nhân viên sử dụng Passkeys (FaceID/TouchID/Windows Hello) để điểm danh. Yêu cầu admin phê duyệt thiết bị trước khi sử dụng.
- **Chống gian lận (Anti-Fraud System)** — 9 lớp bảo vệ:
  1. **GPS Accuracy Check**: Reject GPS accuracy < 10m (fake GPS) hoặc > 150m (tín hiệu yếu)
  2. **TOTP Single-Use Nonce**: Mỗi mã QR chỉ dùng 1 lần trong 30s, chặn chụp ảnh/chia sẻ QR
  3. **Rate Limit per User**: Giới hạn 3 check-in / 5 phút / user (ngoài rate limit IP)
  4. **Impossible Travel Detection**: Phát hiện di chuyển > 150km/h giữa 2 lần check-in
  5. **Device Fingerprinting**: Thu thập fingerprint thiết bị (SHA-256), bind với user, alert thiết bị mới
  6. **IP-Location Cross-Check**: So sánh vị trí IP vs GPS, phát hiện VPN (warning)
  7. **WebAuthn Sign Count**: Detect authenticator bị clone qua sign count không tăng
  8. **Anomaly Detection**: Z-score trên thời gian check-in 30 ngày, flag nếu > 3.0 standard deviation
  9. **Concurrent Session**: Giới hạn 3 session / user, tự revoke session cũ nhất
- Hệ thống `FraudAlert` ghi nhận mọi cảnh báo gian lận cho admin review
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

### 5. Leave Management (Quản lý nghỉ phép)
- **Employee**: đăng ký nghỉ phép qua form (loại phép, ngày, lý do)
- **Manager**: duyệt/từ chối đơn phép của nhân viên chi nhánh
- 7 loại phép: Nghỉ phép năm, Nghỉ ốm, Nghỉ việc riêng, Nghỉ cưới, Nghỉ tang, Nghỉ thai sản, Nghỉ không lương
- Kiểm tra trùng lịch (overlap detection)
- Tự động tạo attendance record (status: "leave") khi phép được duyệt
- Leave balance tracking theo năm
- HTMX-powered UI: form đăng ký, bảng quản lý, filter theo trạng thái

### 6. Authorization (Phân quyền)
- **Admin**: Toàn hệ thống - quản lý tất cả chi nhánh, nhân viên, báo cáo
- **Manager**: Quản lý chi nhánh được gán - xem báo cáo chi nhánh
- **Employee**: Xem thông tin cá nhân, check-in/check-out

## Database Design Conventions

- **PostgreSQL** — 1 instance, mỗi service sở hữu schema riêng (schema-per-service pattern)
- Schema hỗ trợ multi-branch: mọi bảng liên quan đều có `branch_id`
- Sử dụng UUID (TEXT) cho primary key
- Soft delete (`deleted_at` timestamp) cho dữ liệu quan trọng
- Index trên các cột filter phổ biến: `branch_id`, `user_id`, `created_at`
- Pagination bắt buộc trên mọi list endpoint (offset-based, cursor-based cho large datasets)
- WAL mode enabled cho concurrent read performance
- GORM AutoMigrate cho development, SQL migration files cho production

## API Conventions

- RESTful naming: `GET /api/branches`, `POST /api/attendance/log`
- Tất cả endpoint trả về **JSON** (không còn HTML fragment — frontend là Next.js)
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
- **Gateway** validate JWT, inject user context headers → proxy to service
- **Internal API** (giữa services): prefix `/api/internal/`, không qua Gateway
- Rate limiting tại Gateway (per IP) + tại Attendance Service (per User)

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

### Frontend (Next.js + React + TypeScript)
- **Next.js App Router** với SSR (Server Components + Client Components)
- **TypeScript** strict mode
- **Tailwind CSS** cho styling (responsive, mobile-first)
- **React components** thay thế HTMX templates — reusable, typed
- **API client** (`lib/api.ts`) gọi Gateway, auto-attach JWT cookie
- **Middleware** (`middleware.ts`) redirect unauthenticated users
- **Chart.js** (React wrapper) cho dashboard charts
- Page structure mirror HTMX templates: `app/dashboard/`, `app/attendance/`, `app/leave/`, etc.

### Inter-Service Communication
- **Sync HTTP REST** giữa các service (internal API prefix `/api/internal/`)
- **Gateway** inject user context qua headers: `X-User-ID`, `X-User-Role`, `X-Branch-ID`
- **Service clients** (`internal/client/`) — typed HTTP wrappers cho cross-service calls
- **Async flow**: Leave approved → POST `/api/internal/attendance/sync-leave` (→ Attendance Service)

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

- **Microservices**: mỗi service scale độc lập (Attendance cần scale 10x lúc peak, Analytics không)
- **SQLite WAL mode** per service: concurrent reads không block nhau
- **In-memory cache** (go-cache): cache branch config, permissions, dashboard aggregation
- **Schema-per-service**: mỗi service sở hữu PostgreSQL schema riêng, không truy cập cross-schema
- Scale tiếp: tách thành separate PostgreSQL instances khi cần
- **Gateway** stateless → horizontal scale bằng load balancer
- Database indexing strategy cho 100 branches x 5.000 employees

## Docker

- `docker-compose up` chạy **8 containers**: 1 PostgreSQL + 5 Go services + 1 gateway + 1 Next.js
- Mỗi Go service: multi-stage build `golang:alpine` → `alpine` runtime (~15MB)
- Next.js: `node:alpine` build → `node:alpine` runtime
- PostgreSQL 16: 1 instance, 5 schemas (1 volume)
- `.env.example` có sẵn, copy thành `.env` trước khi chạy
- Không commit secrets vào repo
- Xem chi tiết: **[docs/MICROSERVICES.md](docs/MICROSERVICES.md)**

## AI IDE Workflow

- **Workflow**: Spec → AI generate → Review & refine → Test → Commit
- **Review 100%** code AI sinh ra trước khi commit
- Ghi log prompt + kết quả vào `PROMPT_LOG.md`
