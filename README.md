# Smart Attendance — Chấm Công Thông Minh

Hệ thống chấm công thông minh cho doanh nghiệp quy mô **100 chi nhánh**, **5.000 nhân viên**.
Xác thực đa phương thức: QR/TOTP, IP Whitelist, GPS Geofencing. Chống gian lận (fake GPS, VPN).

## Tech Stack

| Layer | Technology |
|---|---|
| Backend | Go 1.22+, Chi router, GORM |
| Database | SQLite (WAL mode, pure-Go driver) |
| Cache | go-cache (in-memory, TTL 5 min) |
| Frontend | HTMX 2.0 + Go html/template + Tailwind CSS |
| Charts | Chart.js 4.x |
| Auth | JWT + Refresh Token + Microsoft OAuth2 |
| Icons | Lucide Icons |
| Deploy | Docker multi-stage build (~15MB image) |

## Quick Start

### Prerequisites

- Go 1.22+ (no CGO required)
- Docker & Docker Compose (optional)

### Run locally

```bash
# 1. Clone & setup
git clone <repo-url> && cd smart_attendance
cp .env.example .env

# 2. Run server
go run cmd/server/main.go

# 3. Open browser
# http://localhost:8080
```

### Default credentials

| Role | Email | Password |
|------|-------|----------|
| Admin | admin@smartattendance.com | password123 |
| Manager | manager@smartattendance.com | password123 |
| Employee | employee@smartattendance.com | password123 |

### Run with Docker

```bash
cp .env.example .env
docker-compose up --build
```

### Seed large dataset (100 branches + 5000 users)

```bash
go run cmd/seed/main.go
```

## Architecture

```
┌─────────────────────────────────────────────────────┐
│                    Browser / Mobile                   │
│         HTMX + Tailwind CSS + Chart.js               │
└─────────────────────┬───────────────────────────────┘
                      │ HTTP/HTMX/JSON
┌─────────────────────▼───────────────────────────────┐
│                   Chi Router                         │
│  ┌──────────┬──────────┬───────────┬──────────────┐ │
│  │  Logger  │ Recovery │ RateLimit │  JWT + RBAC  │ │
│  └──────────┴──────────┴───────────┴──────────────┘ │
├─────────────────────────────────────────────────────┤
│                  Handler Layer                       │
│  Auth │ Users │ Branches │ Attendance │ Dashboard    │
│  Reports │ OAuth │ Error Pages                       │
├─────────────────────────────────────────────────────┤
│                  Service Layer                       │
│  AuthService │ UserService │ BranchService           │
│  AttendanceService │ DashboardService │ ReportService│
│  TOTPService │ IPValidator │ LocationValidator        │
├─────────────────────────────────────────────────────┤
│                 Repository Layer                     │
│  UserRepo │ BranchRepo │ AttendanceRepo              │
├─────────────────────────────────────────────────────┤
│              SQLite (WAL mode) + go-cache            │
└─────────────────────────────────────────────────────┘
```

## Project Structure

```
smart_attendance/
├── cmd/
│   ├── server/main.go       # Entry point
│   └── seed/main.go         # Large dataset seeder (100 branches, 5K users)
├── internal/
│   ├── config/              # Environment configuration
│   ├── database/            # SQLite connection, migration, seed
│   ├── models/              # GORM models (User, Branch, Attendance)
│   ├── repository/          # Data access layer (queries, aggregations)
│   ├── service/             # Business logic (auth, attendance, dashboard)
│   ├── handler/             # HTTP handlers (HTMX pages + JSON API)
│   ├── middleware/          # JWT auth, RBAC, rate limit, recovery, logger
│   ├── cache/               # In-memory cache wrapper (go-cache)
│   ├── renderer/            # Go template renderer (pages + partials)
│   └── validator/           # Input validation helpers
├── web/
│   ├── templates/
│   │   ├── layouts/         # base.html (Tailwind + HTMX + Lucide)
│   │   ├── pages/           # Full page templates (14 pages)
│   │   ├── partials/        # HTMX fragment responses (10 partials)
│   │   └── components/      # Reusable components (nav, toast, alert)
│   └── static/css/          # Custom CSS (responsive tables, skeleton)
├── Dockerfile               # Multi-stage build (golang:alpine → alpine)
├── docker-compose.yml       # One-command deploy
├── .env.example             # Environment template
└── CLAUDE.md                # AI context file
```

## Features

### Check-in / Check-out (3 phương thức)
- **QR/TOTP**: Camera quét mã QR chứa TOTP code, reset mỗi 15 giây
- **IP Whitelist**: Verify request IP thuộc CIDR range của chi nhánh
- **GPS Geofencing**: Haversine distance check trong bán kính cho phép
- Chống gian lận: TOTP expire 15s, multi-factor validation
- Auto status: đúng giờ / trễ / vắng (dựa trên giờ làm)

### Branch Management
- CRUD chi nhánh với config riêng (QR/IP/Location)
- IP Whitelist entries (CIDR notation)
- Location Whitelist (lat, lng, radius)
- Assign/unassign employees
- Live QR display cho Manager (auto-refresh 15s)

### Dashboard
- 4 stat cards: tổng NV, check-in hôm nay, tỉ lệ đúng giờ, đi trễ
- Chart.js stacked bar chart: 14-day trend
- Top late leaderboard (tháng)
- Recent activity feed
- HTMX filter by branch (Admin)
- Cache 5 min TTL

### Reports & History
- Filter ngày/tuần/tháng + trạng thái
- HTMX partial reload (không reload page)
- Export Excel (.xlsx)
- Branch report (Manager/Admin)

### Authentication & Authorization
- JWT access + refresh token (cookie-based)
- Microsoft OAuth2 login (optional)
- 3 roles: Admin / Manager / Employee
- RBAC middleware: AdminOnly, ManagerOrAdmin
- Manager chỉ xem data chi nhánh mình

### UI/UX
- HTMX-powered SPA-like experience (no JS framework)
- Tailwind CSS responsive (mobile-first)
- Bottom navigation (mobile)
- Responsive tables → card view on mobile
- Toast notifications for all error states
- Error pages: 404, 403, 500
- Lucide icons

## Configuration

| Variable | Default | Description |
|---|---|---|
| `PORT` | 8080 | Server port |
| `ENV` | development | development / production |
| `DB_PATH` | data/smart_attendance.db | SQLite file path |
| `JWT_SECRET` | (change me) | JWT signing secret |
| `JWT_EXPIRE_MINUTES` | 60 | Access token TTL |
| `JWT_REFRESH_HOURS` | 168 | Refresh token TTL (7 days) |
| `RATE_LIMIT_PER_MIN` | 10 | Check-in rate limit |
| `MICROSOFT_CLIENT_ID` | (empty) | Microsoft OAuth client ID |
| `MICROSOFT_CLIENT_SECRET` | (empty) | Microsoft OAuth secret |
| `MICROSOFT_REDIRECT_URI` | http://localhost:8080/auth/oauth/microsoft/callback | OAuth redirect |
| `MICROSOFT_TENANT_ID` | common | Azure AD tenant |

## API Endpoints

### Public
- `POST /api/v1/auth/login` — Login (JSON)
- `POST /api/v1/auth/register` — Register (JSON)
- `POST /api/v1/auth/refresh` — Refresh token

### Protected (JWT required)
- `GET /api/v1/dashboard/stats` — Dashboard statistics
- `GET /api/v1/dashboard/charts` — Chart data
- `POST /api/v1/attendance/check-in` — Check in
- `POST /api/v1/attendance/check-out` — Check out
- `GET /api/v1/attendance/status` — Today's status
- `GET /api/v1/branches` — List branches (Admin)
- `POST /api/v1/branches` — Create branch (Admin)
- `GET /api/v1/users` — List users (Admin)

## Scaling Strategy

- **SQLite WAL mode** — concurrent reads, serialized writes
- **In-memory cache** — branch config, dashboard stats (go-cache, 5 min TTL)
- **Single binary** — Go compiles to 1 binary, templates via filesystem
- **Indexed queries** — composite indexes on `(branch_id, user_id, created_at)`
- **Batch inserts** — seed data uses `CreateInBatches(500)` for performance
- **Horizontal scaling path** — swap SQLite → PostgreSQL via GORM driver change
- **Goroutine concurrency** — handles peak-hour check-in load

## License

MIT
