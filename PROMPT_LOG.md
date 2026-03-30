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
| **Prompt** | `Tạo context file CLAUDE.md Mô tả kiến trúc, conventions, tech stack từ ảnh đề bài` |
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
| **Prompt** | `Thay đổi tech stack: BE dùng Go, DB dùng SQLite, dùng internal cache (go-cache local), dùng HTMX làm FE cho Go` |
| **Output** | Cập nhật 6 sections trong CLAUDE.md: Tech Stack, Architecture (Go project layout cmd/internal/web), DB conventions (SQLite WAL, GORM), Code conventions (Go idiomatic, Handler→Service→Repository), Scaling strategy (in-memory cache, goroutine concurrency), Docker (alpine ~15MB) |
| **Review** | **Accepted** — Stack nhẹ, deploy đơn giản, phù hợp single binary + docker-compose |
| **Changes** | Không |
| **Files** | `CLAUDE.md` |
| **Commit** | — |
