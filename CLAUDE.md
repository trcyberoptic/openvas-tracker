# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

OpenVAS-Tracker is a single-binary vulnerability management platform built in Go with an embedded React SPA. It integrates Nmap and OpenVAS scanning, tracks vulnerabilities with CVE enrichment, manages remediation tickets, and generates reports. Licensed under GPL v3.

## Build & Development Commands

```bash
# Backend only
go build ./cmd/openvas-tracker        # compile
go test ./... -v -count=1             # all tests
go test ./internal/scanner/ -v        # single package
go test ./internal/auth/ -v -run Token  # single test by name

# Frontend only
cd frontend && npm ci && npm run build  # production build
cd frontend && npm run dev              # dev server with HMR

# Combined
make dev          # backend + frontend dev servers (frontend proxies API to :8080)
make build        # production build (builds frontend, copies to cmd/openvas-tracker/static/, compiles Go binary)
make build-linux  # cross-compile for Linux amd64
make test         # go test ./...
make clean        # remove build artifacts

# Database
make migrate-up    # apply migrations (needs DATABASE_URL env)
make migrate-down  # rollback one migration
make sqlc          # regenerate query code from sql/queries/*.sql
```

## Architecture

**Layered Go backend:** `cmd/openvas-tracker/main.go` wires everything together.

```
handler (Echo HTTP) → service (business logic) → queries (database/sql) → MariaDB
                                                → scanner (nmap/openvas CLI)
                                                → report (html/pdf/excel/md)
```

- **`internal/handler/`** — Echo route handlers. Each has a `RegisterRoutes(*echo.Group)` method mounted in main.go.
- **`internal/service/`** — Business logic. Each takes `*sql.DB` in constructor (e.g., `NewUserService(db)`).
- **`internal/database/queries/`** — Hand-written query stubs (sqlc-style but manually maintained since no live DB during initial build). Uses `database/sql` with `go-sql-driver/mysql`.
- **`internal/scanner/`** — Nmap/OpenVAS XML parsers (`ParseNmapXML`, `ParseOpenVASXML`) and CLI wrappers. CVE enrichment via NVD API.
- **`internal/report/`** — Report generators: `GenerateHTML` (embed template), `GeneratePDF` (maroto v2), `GenerateExcel` (excelize), `GenerateMarkdown`.
- **`internal/worker/`** — Asynq (Redis-backed) async job handlers for scan execution.
- **`internal/websocket/`** — Hub + Client for real-time per-user push via gorilla/websocket.
- **`internal/auth/`** — JWT (golang-jwt) and bcrypt password utilities.
- **`internal/middleware/`** — Echo middleware: JWT auth, RBAC, rate limiting, security headers, audit logging.
- **`internal/plugin/`** — Go plugin interface for custom scanners (`.so` files loaded at runtime).

**Frontend:** React 19 + Vite + Tailwind + shadcn/ui, embedded in the Go binary via `//go:embed all:static` in `cmd/openvas-tracker/frontend.go`. The Makefile copies `frontend/dist/` → `cmd/openvas-tracker/static/` before Go build.

## Configuration

All config via environment variables with `OT_` prefix (Viper, `internal/config/config.go`):

| Variable | Default | Purpose |
|----------|---------|---------|
| `OT_SERVER_PORT` | 8080 | HTTP listen port |
| `OT_DATABASE_DSN` | `...@tcp(localhost:3306)/openvas-tracker?parseTime=true` | MariaDB DSN |
| `OT_REDIS_ADDR` | `localhost:6379` | Redis for Asynq job queue |
| `OT_JWT_SECRET` | `change-me-in-production` | JWT signing key |
| `OT_SCANNER_NMAPPATH` | `nmap` | Path to nmap binary |
| `OT_SCANNER_OPENVASPATH` | `gvm-cli` | Path to GVM CLI binary |

## Database

- **MariaDB** with `database/sql` + `go-sql-driver/mysql`
- 12 migrations in `sql/migrations/` (golang-migrate format, numbered 001-012)
- UUIDs are `CHAR(36)`, generated in Go code (`uuid.New().String()`), not DB-side
- Query files in `sql/queries/` — sqlc config targets MySQL engine

## Key Patterns

- **Auth flow:** JWT Bearer tokens. Public routes under `/api/auth/*`, everything else behind `middleware.JWTAuth`. User ID extracted via `middleware.GetUserID(c)` (returns `string`).
- **Scan execution:** Handler creates scan record → enqueues Asynq task → worker picks it up → runs nmap/openvas CLI → parses XML → stores results.
- **Report generation:** Synchronous — handler calls `ReportService.Generate()` which aggregates vulns from scan IDs and dispatches to the requested format generator.
- **SPA routing:** `cmd/openvas-tracker/frontend.go` serves embedded static files with fallback to `index.html` for client-side routing.

## Deployment

Single binary targets Debian Trixie as a systemd service. Deploy files in `deploy/`:
- `openvas-tracker.service` — systemd unit with security hardening
- `install.sh` — creates user, installs binary, copies config, enables service
- `Dockerfile` — multi-stage build (node → go → alpine runtime)
