# Smart Attendance — Chấm Công Thông Minh

Hệ thống chấm công thông minh cho doanh nghiệp quy mô **100 chi nhánh**, **5.000 nhân viên**.
Xác định vị trí bằng WiFi SSID/BSSID và GPS geofencing. Chống gian lận (fake GPS, VPN).

## Tech Stack

| Layer | Technology |
|---|---|
| Backend | Go, Chi router, GORM |
| Database | SQLite (WAL mode) |
| Cache | go-cache (in-memory) |
| Frontend | HTMX + Go html/template + Tailwind CSS |
| Charts | Chart.js |
| Auth | JWT + Refresh Token |
| Deploy | Docker multi-stage build |

## Quick Start

### Prerequisites

- Go 1.22+
- Docker & Docker Compose (optional)

### Run locally

```bash
cp .env.example .env
go run cmd/server/main.go
```

Server starts at `http://localhost:8080`

### Run with Docker

```bash
cp .env.example .env
docker-compose up --build
```

## Project Structure

```
├── cmd/server/          # Entry point
├── internal/
│   ├── config/          # App configuration
│   ├── models/          # GORM models
│   ├── handler/         # HTTP handlers
│   ├── service/         # Business logic
│   ├── repository/      # Data access layer
│   ├── middleware/       # JWT, RBAC, rate limit
│   ├── cache/           # In-memory cache
│   └── validator/       # Input validation
├── web/
│   ├── templates/       # Go html/template
│   └── static/          # CSS, JS, images
├── migrations/          # SQL migration files
├── Dockerfile
├── docker-compose.yml
└── .env.example
```

## Features

- **Check-in/Check-out**: WiFi SSID/BSSID + GPS geofencing, anti-fraud detection
- **Branch Management**: CRUD branches, WiFi/GPS config per location
- **History & Reports**: Filter by day/week/month, export CSV/Excel
- **Dashboard**: Real-time stats, Chart.js visualizations, filter by branch
- **Authorization**: Admin / Manager / Employee roles (RBAC)

## Scaling Strategy

- **SQLite WAL mode** — concurrent reads, serialized writes
- **In-memory cache** — branch config, dashboard stats (go-cache with TTL)
- **Single binary** — Go embed templates + static files, ~15MB Docker image
- **Indexed queries** — composite indexes on `(branch_id, user_id, created_at)`
- **Horizontal scaling path** — swap SQLite → PostgreSQL via GORM driver change

## License

MIT
