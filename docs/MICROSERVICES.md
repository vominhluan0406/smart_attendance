# Microservices Architecture Plan

> Tách monolith Smart Attendance thành 5 Go microservices + 1 API Gateway + 1 Next.js frontend.

## Tổng quan

```
                        ┌──────────────────┐
                        │   Next.js (3000)  │
                        │   SSR + React     │
                        │   Tailwind + TS   │
                        └────────┬─────────┘
                                 │ HTTP/JSON
                        ┌────────▼─────────┐
                        │  API Gateway (8080)│
                        │  Go + Chi          │
                        │  JWT validate      │
                        │  Rate limit        │
                        │  Route + Aggregate │
                        └──┬──┬──┬──┬──┬───┘
           ┌───────────────┘  │  │  │  └───────────────┐
           ▼                  ▼  │  ▼                   ▼
   ┌──────────────┐ ┌──────────┐│┌──────────┐ ┌──────────────┐
   │ Auth (8081)  │ │Attend.   │││ Leave    │ │ Org (8085)   │
   │              │ │ (8082)   │││ (8083)   │ │              │
   │ • User CRUD │ │• Checkin  │││• Request │ │ • Branch     │
   │ • JWT       │ │• Anti-   │││• Approve │ │ • Shift      │
   │ • WebAuthn  │ │  Fraud   │││• Balance │ │ • Department │
   │ • RBAC      │ │• QR/TOTP │││• Type    │ │ • Holiday    │
   │ • OAuth     │ │• Alerts  │││          │ │ • IP/GPS cfg │
   │ • Session   │ │• Adjust. │││          │ │              │
   └──────┬──────┘ └────┬─────┘│└────┬─────┘ └──────┬───────┘
          │              │      │     │               │
          └──────────────┴──────┼─────┴───────────────┘
                                │
                      ┌─────────▼─────────┐
                      │ Analytics (8084)   │
                      │ • Dashboard        │
                      │ • Reports          │
                      │ • Excel export     │
                      │ • Gamification     │
                      └─────────┬──────────┘
                                │
               ┌────────────────▼────────────────┐
               │  PostgreSQL (5432)               │
               │  1 instance, 5 schemas:          │
               │  ┌─────┬──────┬─────┬─────┬───┐ │
               │  │auth │attend│leave│anal.│org │ │
               │  └─────┴──────┴─────┴─────┴───┘ │
               └─────────────────────────────────┘
```

## 1. Directory Structure

```
smart_attendance/
├── services/
│   ├── gateway/                  # API Gateway (port 8080)
│   │   ├── cmd/main.go
│   │   ├── internal/
│   │   │   ├── proxy/            # Reverse proxy + service registry
│   │   │   ├── middleware/       # JWT validation, rate limit, CORS
│   │   │   └── config/
│   │   ├── Dockerfile
│   │   └── go.mod
│   │
│   ├── auth/                     # Auth Service (port 8081)
│   │   ├── cmd/main.go
│   │   ├── internal/
│   │   │   ├── handler/          # auth, user, webauthn, oauth handlers
│   │   │   ├── service/          # auth, user, permission, webauthn services
│   │   │   ├── repository/       # user, session, credential, permission repos
│   │   │   ├── model/            # User, Session, Credential, Permission
│   │   │   ├── middleware/       # Internal auth middleware
│   │   │   ├── database/         # Migration + seed
│   │   │   └── config/
│   │   ├── Dockerfile
│   │   └── go.mod
│   │
│   ├── attendance/               # Attendance Service (port 8082)
│   │   ├── cmd/main.go
│   │   ├── internal/
│   │   │   ├── handler/          # attendance, fraud_alert, adjustment handlers
│   │   │   ├── service/          # attendance, antifraud, fraud_alert, adjustment, totp, ip, location services
│   │   │   ├── repository/       # attendance, attendance_log, fraud_alert, device, adjustment repos
│   │   │   ├── model/            # Attendance, AttendanceLog, FraudAlert, UserDevice, Adjustment
│   │   │   ├── client/           # HTTP clients → Auth, Organization services
│   │   │   ├── database/
│   │   │   └── config/
│   │   ├── Dockerfile
│   │   └── go.mod
│   │
│   ├── leave/                    # Leave Service (port 8083)
│   │   ├── cmd/main.go
│   │   ├── internal/
│   │   │   ├── handler/          # leave handler
│   │   │   ├── service/          # leave service
│   │   │   ├── repository/       # leave, leave_type repos
│   │   │   ├── model/            # LeaveRequest, LeaveType, LeaveBalance, OvertimeRequest
│   │   │   ├── client/           # HTTP clients → Auth, Attendance services
│   │   │   ├── database/
│   │   │   └── config/
│   │   ├── Dockerfile
│   │   └── go.mod
│   │
│   ├── analytics/                # Analytics Service (port 8084)
│   │   ├── cmd/main.go
│   │   ├── internal/
│   │   │   ├── handler/          # dashboard, report handlers
│   │   │   ├── service/          # dashboard, report, gamification services
│   │   │   ├── repository/       # dashboard_queries, report queries
│   │   │   ├── model/            # Read-only DTOs
│   │   │   ├── client/           # HTTP clients → Auth, Attendance, Leave, Org
│   │   │   ├── database/
│   │   │   └── config/
│   │   ├── Dockerfile
│   │   └── go.mod
│   │
│   └── organization/             # Organization Service (port 8085)
│       ├── cmd/main.go
│       ├── internal/
│       │   ├── handler/          # branch, shift, department, holiday handlers
│       │   ├── service/          # branch, shift services
│       │   ├── repository/       # branch, shift repos
│       │   ├── model/            # Branch, BranchIPWhitelist, BranchLocation, WorkShift, Department, Holiday
│       │   ├── database/
│       │   └── config/
│       ├── Dockerfile
│       └── go.mod
│
├── frontend/                     # Next.js Frontend (port 3000)
│   ├── src/
│   │   ├── app/                  # App Router pages
│   │   │   ├── layout.tsx        # Root layout (Tailwind, nav)
│   │   │   ├── page.tsx          # Home
│   │   │   ├── login/page.tsx
│   │   │   ├── dashboard/page.tsx
│   │   │   ├── attendance/
│   │   │   │   ├── page.tsx      # QR scan check-in
│   │   │   │   ├── wifi-gps/page.tsx
│   │   │   │   ├── password/page.tsx
│   │   │   │   └── qr/[branchId]/page.tsx  # QR display (kiosk)
│   │   │   ├── reports/
│   │   │   │   ├── page.tsx      # Admin report selection
│   │   │   │   ├── my-history/page.tsx
│   │   │   │   └── branch/[branchId]/page.tsx
│   │   │   ├── leave/
│   │   │   │   ├── my/page.tsx
│   │   │   │   └── manage/page.tsx
│   │   │   ├── adjustments/
│   │   │   │   ├── my/page.tsx
│   │   │   │   └── manage/page.tsx
│   │   │   ├── alerts/page.tsx
│   │   │   ├── branches/
│   │   │   │   ├── page.tsx
│   │   │   │   ├── create/page.tsx
│   │   │   │   └── [id]/edit/page.tsx
│   │   │   ├── users/
│   │   │   │   ├── page.tsx
│   │   │   │   ├── create/page.tsx
│   │   │   │   └── [id]/edit/page.tsx
│   │   │   └── profile/page.tsx
│   │   ├── components/           # Shared React components
│   │   │   ├── nav.tsx
│   │   │   ├── data-table.tsx
│   │   │   ├── status-badge.tsx
│   │   │   ├── pagination.tsx
│   │   │   ├── filter-form.tsx
│   │   │   └── ...
│   │   ├── lib/
│   │   │   ├── api.ts            # API client (fetch wrapper → Gateway)
│   │   │   ├── auth.ts           # Auth helpers (cookie, redirect)
│   │   │   └── types.ts          # TypeScript interfaces matching Go models
│   │   └── middleware.ts         # Next.js middleware (auth redirect)
│   ├── tailwind.config.ts
│   ├── next.config.ts
│   ├── package.json
│   ├── tsconfig.json
│   └── Dockerfile
│
├── shared/                       # Shared Go packages (imported by services)
│   ├── dto/                      # Shared DTOs for inter-service communication
│   │   ├── user.go
│   │   ├── branch.go
│   │   ├── attendance.go
│   │   └── ...
│   ├── middleware/                # Shared middleware (internal auth header parsing)
│   │   └── internal_auth.go
│   ├── response/                 # Standard JSON response format
│   │   └── response.go
│   └── go.mod
│
├── docker-compose.yml            # 7 containers orchestration
├── docker-compose.dev.yml        # Dev overrides (hot reload)
├── .env.example
├── CLAUDE.md
└── README.md
```

## 2. Inter-Service API Contracts

### Auth Service (8081)

```
# Public
POST   /api/auth/login              → { access_token, refresh_token }
POST   /api/auth/refresh             → { access_token }
POST   /api/auth/logout              → 204
GET    /api/auth/oauth/microsoft     → redirect
GET    /api/auth/oauth/callback      → { access_token }

# Internal (called by Gateway/other services)
GET    /api/internal/validate-token  → { user_id, email, role, branch_id }  (Header: Authorization)
GET    /api/internal/users/:id       → User
GET    /api/internal/users           → []User (with pagination)

# User CRUD (requires admin)
GET    /api/users                    → []User
POST   /api/users                    → User
GET    /api/users/:id                → User
PUT    /api/users/:id                → User
DELETE /api/users/:id                → 204

# WebAuthn
GET    /api/webauthn/register/begin  → PublicKeyCredentialCreationOptions
POST   /api/webauthn/register/finish → Credential
GET    /api/webauthn/login/begin     → PublicKeyCredentialRequestOptions
POST   /api/webauthn/login/finish    → { verified }

# Permissions
GET    /api/internal/permissions/check?role=X&code=Y → { allowed: bool }

# Profile
GET    /api/profile                  → User
```

### Attendance Service (8082)

```
# Check-in/out
POST   /api/attendance/log           → LogTimeResult
GET    /api/attendance/status         → Attendance (today)
GET    /api/attendance/logs/today     → []AttendanceLog

# QR / TOTP
GET    /api/attendance/qr/:branchId/code   → { code, expires_at }
GET    /api/attendance/qr/:branchId/image  → PNG image

# Fraud Alerts
GET    /api/alerts                   → []FraudAlert (pagination + filters)
POST   /api/alerts/:id/review        → 200
POST   /api/alerts/:id/invalidate    → 200

# Adjustments
GET    /api/adjustments/my           → []AttendanceAdjustment
POST   /api/adjustments/my           → AttendanceAdjustment
GET    /api/adjustments/manage       → []AttendanceAdjustment (branch)
POST   /api/adjustments/manage/:id/review → 200

# Internal
GET    /api/internal/attendance       → []Attendance (for Analytics)
POST   /api/internal/attendance/sync-leave → 200 (called by Leave service)
```

### Leave Service (8083)

```
GET    /api/leave/my                 → []LeaveRequest
POST   /api/leave/my                 → LeaveRequest
GET    /api/leave/manage             → []LeaveRequest (branch, filtered)
POST   /api/leave/manage/:id/review  → 200
GET    /api/leave/types              → []LeaveType
GET    /api/leave/balance            → []LeaveBalance

# Internal
GET    /api/internal/leave/pending-count?branch_id=X → { count }
```

### Analytics Service (8084)

```
GET    /api/dashboard/stats          → DashboardStats
GET    /api/dashboard/charts         → ChartData
GET    /api/dashboard/recent         → []RecentCheckIn
GET    /api/reports/branch/:id       → []Attendance (paginated)
GET    /api/reports/branch/:id/export → Excel file
GET    /api/reports/my-history       → []Attendance (paginated)
GET    /api/reports/my-history/export → Excel file
```

### Organization Service (8085)

```
# Branches
GET    /api/branches                 → []Branch
POST   /api/branches                 → Branch
GET    /api/branches/:id             → Branch (with IP whitelist, locations, shifts)
PUT    /api/branches/:id             → Branch
DELETE /api/branches/:id             → 204

# Internal (called by Attendance service)
GET    /api/internal/branches/:id    → Branch (cached, full config)
GET    /api/internal/branches/:id/methods → { allowed_methods, ip_whitelist, locations }

# Shifts
GET    /api/shifts?branch_id=X       → []WorkShift
POST   /api/shifts                   → WorkShift

# Departments
GET    /api/departments              → []Department

# Holidays
GET    /api/holidays                 → []Holiday
```

## 3. Gateway Routing Table

```
Gateway (8080) — routes incoming requests to services

Path Prefix                → Service         Port
─────────────────────────────────────────────────
/api/auth/**               → auth            8081
/api/users/**              → auth            8081
/api/webauthn/**           → auth            8081
/api/profile               → auth            8081
/api/attendance/**         → attendance      8082
/api/alerts/**             → attendance      8082
/api/adjustments/**        → attendance      8082
/api/leave/**              → leave           8083
/api/dashboard/**          → analytics       8084
/api/reports/**            → analytics       8084
/api/branches/**           → organization    8085
/api/shifts/**             → organization    8085
/api/departments/**        → organization    8085
/api/holidays/**           → organization    8085

Middleware chain:
1. CORS (allow Next.js origin)
2. Rate Limit (per IP)
3. JWT Validation (call auth /api/internal/validate-token)
   → Inject headers: X-User-ID, X-User-Role, X-Branch-ID, X-User-Email
4. Reverse Proxy → target service
```

## 4. Next.js Pages Mapping

```
Current HTMX Template          → Next.js Page
────────────────────────────────────────────────────
layouts/base.html              → app/layout.tsx (root layout)
components/nav.html            → components/nav.tsx
pages/login.html               → app/login/page.tsx
pages/home.html                → app/page.tsx
pages/dashboard.html           → app/dashboard/page.tsx
  partials/dashboard_stats     → components/dashboard-stats.tsx
  partials/dashboard_chart     → components/dashboard-chart.tsx (Chart.js)
  partials/dashboard_recent    → components/dashboard-recent.tsx
pages/attendance.html          → app/attendance/page.tsx (QR scanner)
pages/wifi_gps_checkin.html    → app/attendance/wifi-gps/page.tsx
pages/password_checkin.html    → app/attendance/password/page.tsx
pages/qr_display.html         → app/attendance/qr/[branchId]/page.tsx
pages/my_history.html          → app/reports/my-history/page.tsx
  partials/history_list        → components/history-table.tsx
pages/branch_report.html       → app/reports/branch/[branchId]/page.tsx
pages/report_branches.html     → app/reports/page.tsx
pages/my_leave.html            → app/leave/my/page.tsx
pages/manage_leave.html        → app/leave/manage/page.tsx
pages/my_adjustments.html      → app/adjustments/my/page.tsx
pages/manage_adjustments.html  → app/adjustments/manage/page.tsx
pages/fraud_alerts.html        → app/alerts/page.tsx
  partials/fraud_alerts_list   → components/fraud-alerts-table.tsx
pages/branches.html            → app/branches/page.tsx
pages/branch_create.html       → app/branches/create/page.tsx
pages/branch_edit.html         → app/branches/[id]/edit/page.tsx
pages/users.html               → app/users/page.tsx
pages/user_create.html         → app/users/create/page.tsx
pages/user_edit.html           → app/users/[id]/edit/page.tsx
pages/profile.html             → app/profile/page.tsx
pages/error.html               → app/not-found.tsx + app/error.tsx
```

## 5. Cross-Service Communication

```
Flow 1: Employee Check-in
──────────────────────────
Next.js → Gateway → Attendance Service
                     ├─ GET /api/internal/branches/:id (→ Org Service) — lấy config
                     ├─ Anti-fraud checks (internal)
                     ├─ Create AttendanceLog + Attendance
                     └─ Return result

Flow 2: Leave Approval → Create Attendance
───────────────────────────────────────────
Next.js → Gateway → Leave Service
                     ├─ Update LeaveRequest status = approved
                     └─ POST /api/internal/attendance/sync-leave (→ Attendance Service)
                          └─ Create Attendance records (status: "leave")

Flow 3: Dashboard Aggregation
──────────────────────────────
Next.js → Gateway → Analytics Service
                     ├─ GET /api/internal/attendance?branch_id=X (→ Attendance)
                     ├─ GET /api/internal/users?branch_id=X (→ Auth)
                     ├─ GET /api/internal/leave/pending-count (→ Leave)
                     └─ Aggregate + return stats

Flow 4: Fraud Alert → Invalidate Attendance
────────────────────────────────────────────
Next.js → Gateway → Attendance Service (self-contained, same DB)
```

## 6. Build Order

```
Phase 1: Foundation (ngày 1)
├─ shared/ package (DTOs, response format)
├─ Auth Service (core dependency — mọi service cần validate user)
└─ Organization Service (Attendance cần branch config)

Phase 2: Core Business (ngày 2-3)
├─ Attendance Service (phức tạp nhất — anti-fraud, QR, adjustments)
└─ Leave Service (depends on Attendance API for sync)

Phase 3: Read Layer (ngày 3)
├─ Analytics Service (read-only aggregation)
└─ API Gateway (routing + JWT proxy)

Phase 4: Frontend (ngày 4-6)
├─ Next.js project setup (Tailwind, auth, layout)
├─ Auth pages (login, profile)
├─ Attendance pages (check-in, QR, history)
├─ Management pages (branches, users, leave, adjustments)
├─ Dashboard + Reports
└─ Fraud alerts page

Phase 5: Integration (ngày 7)
├─ Docker-compose full stack
├─ End-to-end testing
└─ Documentation
```

## 7. Docker Compose

```yaml
services:
  # ─── Database ───
  postgres:
    image: postgres:16-alpine
    ports: ["5432:5432"]
    volumes: [pgdata:/var/lib/postgresql/data]
    environment:
      POSTGRES_DB: smart_attendance
      POSTGRES_USER: app
      POSTGRES_PASSWORD: secret
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U app -d smart_attendance"]
      interval: 5s
      retries: 5

  # ─── API Gateway ───
  gateway:
    build: ./services/gateway
    ports: ["8080:8080"]
    depends_on: [auth, attendance, leave, analytics, organization]
    environment:
      AUTH_SERVICE_URL: http://auth:8081
      ATTENDANCE_SERVICE_URL: http://attendance:8082
      LEAVE_SERVICE_URL: http://leave:8083
      ANALYTICS_SERVICE_URL: http://analytics:8084
      ORG_SERVICE_URL: http://organization:8085
      JWT_SECRET: ${JWT_SECRET}

  # ─── Microservices ───
  auth:
    build: ./services/auth
    ports: ["8081:8081"]
    depends_on:
      postgres: { condition: service_healthy }
    environment:
      DATABASE_URL: postgres://app:secret@postgres:5432/smart_attendance?search_path=auth_service
      JWT_SECRET: ${JWT_SECRET}

  attendance:
    build: ./services/attendance
    ports: ["8082:8082"]
    depends_on:
      postgres: { condition: service_healthy }
    environment:
      DATABASE_URL: postgres://app:secret@postgres:5432/smart_attendance?search_path=attendance_service
      AUTH_SERVICE_URL: http://auth:8081
      ORG_SERVICE_URL: http://organization:8085

  leave:
    build: ./services/leave
    ports: ["8083:8083"]
    depends_on:
      postgres: { condition: service_healthy }
    environment:
      DATABASE_URL: postgres://app:secret@postgres:5432/smart_attendance?search_path=leave_service
      AUTH_SERVICE_URL: http://auth:8081
      ATTENDANCE_SERVICE_URL: http://attendance:8082

  analytics:
    build: ./services/analytics
    ports: ["8084:8084"]
    depends_on:
      postgres: { condition: service_healthy }
    environment:
      DATABASE_URL: postgres://app:secret@postgres:5432/smart_attendance?search_path=analytics_service
      AUTH_SERVICE_URL: http://auth:8081
      ATTENDANCE_SERVICE_URL: http://attendance:8082
      LEAVE_SERVICE_URL: http://leave:8083
      ORG_SERVICE_URL: http://organization:8085

  organization:
    build: ./services/organization
    ports: ["8085:8085"]
    depends_on:
      postgres: { condition: service_healthy }
    environment:
      DATABASE_URL: postgres://app:secret@postgres:5432/smart_attendance?search_path=org_service

  # ─── Frontend ───
  frontend:
    build: ./frontend
    ports: ["3000:3000"]
    environment:
      NEXT_PUBLIC_API_URL: http://localhost:8080
      API_URL: http://gateway:8080
    depends_on: [gateway]

volumes:
  pgdata:
```

**Tổng: 8 containers** — 1 PostgreSQL + 5 Go services + 1 Gateway + 1 Next.js

## 8. Database — PostgreSQL Schema Isolation

**1 PostgreSQL instance**, mỗi service sở hữu **schema riêng** (database-per-service pattern trên shared instance):

```
PostgreSQL (5432)
├── Schema: auth_service
│   ├── users
│   ├── user_credentials
│   ├── user_sessions
│   ├── permissions
│   └── role_permissions
├── Schema: attendance_service
│   ├── attendances
│   ├── attendance_logs
│   ├── fraud_alerts
│   ├── user_devices
│   └── attendance_adjustments
├── Schema: leave_service
│   ├── leave_requests
│   ├── leave_types
│   ├── leave_balances
│   └── overtime_requests
├── Schema: analytics_service
│   └── (materialized views / cache tables)
└── Schema: org_service
    ├── branches
    ├── branch_ip_whitelists
    ├── branch_locations
    ├── work_shifts
    ├── user_shift_assignments
    ├── departments
    └── holidays
```

**GORM config** per service — chỉ cần set `search_path`:

```go
// Mỗi service kết nối cùng PostgreSQL, khác schema
dsn := "host=postgres user=app password=secret dbname=smart_attendance"
db.Exec("SET search_path TO auth_service")
```

**Ưu điểm so với SQLite:**
- Concurrent writes không bị lock
- Schema isolation = data boundary rõ ràng
- 1 PostgreSQL instance = 1 Docker container (không cần 5 volume)
- Connection pooling, LISTEN/NOTIFY cho event
- Production-ready, horizontal scale khi cần

## 9. Auth Flow

```
1. Login:
   Next.js → POST /api/auth/login (Gateway) → Auth Service
   ← { access_token, refresh_token }
   Next.js set httpOnly cookie

2. Authenticated request:
   Next.js (server component) → GET /api/dashboard/stats
   ├─ Attach cookie → Gateway
   ├─ Gateway extract JWT → call Auth /api/internal/validate-token
   ├─ Gateway inject X-User-ID, X-User-Role, X-Branch-ID headers
   └─ Proxy to Analytics Service

3. Token refresh:
   Next.js middleware detect 401 → POST /api/auth/refresh → retry
```

## 10. Production Path

| Component | Hiện tại | Scale tiếp |
|---|---|---|
| Database | **PostgreSQL** (schema per service, 1 instance) | Separate PostgreSQL instances per service |
| Cache | go-cache (in-process) | **Redis** (shared, session store) |
| Service discovery | Docker DNS (hardcoded URLs) | Consul / K8s service discovery |
| API Gateway | Custom Go proxy | Kong / Envoy / Traefik |
| Event bus | Sync HTTP calls | **NATS / RabbitMQ / Kafka** |
| Monitoring | Log stdout | Prometheus + Grafana + Jaeger (tracing) |
| Deploy | docker-compose | **Kubernetes** |
| CI/CD | Manual | GitHub Actions → Docker Registry → K8s deploy |
