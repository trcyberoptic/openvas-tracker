# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

OpenVAS-Tracker is a vulnerability management dashboard built in Go with an embedded React SPA. It receives OpenVAS scan results via webhook, tracks vulnerabilities per host, and manages remediation through automated ticketing. No active scanning — purely a results viewer and ticket tracker. Licensed under GPL v3.

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

# Docker
docker compose up -d          # start MariaDB + app
docker compose down -v        # stop and remove volumes
docker compose build app      # rebuild app image

# Database
make migrate-up    # apply migrations (needs DATABASE_URL env)
make migrate-down  # rollback one migration
```

## Architecture

**Layered Go backend:** `cmd/openvas-tracker/main.go` wires everything together.

```
handler (Echo HTTP) → service (business logic) → queries (database/sql) → MariaDB
                                                → scanner (XML parsers only)
                                                → report (html/pdf/excel/md)
```

- **`internal/handler/`** — Echo route handlers. Each has a `RegisterRoutes(*echo.Group)` method mounted in main.go.
  - **`import.go`** — Webhook endpoint `POST /api/import/openvas`. Parses XML, creates scan + vulns, auto-creates/reopens/resolves tickets.
  - **`tickets.go`** — CRUD + status changes + comments + activity log.
- **`internal/service/`** — Business logic. Each takes `*sql.DB` in constructor (e.g., `NewUserService(db)`).
- **`internal/database/queries/`** — Hand-written query stubs. Uses `database/sql` with `go-sql-driver/mysql`.
  - **`tickets.go`** — Ticket queries including `FindTicketByFingerprint`, `AutoResolveStaleTickets`, `LogTicketActivity`.
- **`internal/scanner/`** — `ParseOpenVASXML` XML parser. No CLI wrappers — import only.
- **`internal/report/`** — Report generators: `GenerateHTML`, `GeneratePDF` (maroto v2), `GenerateExcel` (excelize), `GenerateMarkdown`.
- **`internal/websocket/`** — Hub + Client for real-time per-user push via gorilla/websocket.
- **`internal/auth/`** — JWT (golang-jwt) and bcrypt password utilities.
- **`internal/middleware/`** — Echo middleware: JWT auth, API key auth, RBAC, rate limiting, security headers, audit logging.

**Frontend:** React 19 + Vite + Tailwind + shadcn/ui, embedded in the Go binary via `//go:embed all:static` in `cmd/openvas-tracker/frontend.go`. The Makefile copies `frontend/dist/` → `cmd/openvas-tracker/static/` before Go build.

## Configuration

All config via environment variables with `OT_` prefix (Viper, `internal/config/config.go`):

| Variable | Default | Purpose |
|----------|---------|---------|
| `OT_SERVER_PORT` | 8080 | HTTP listen port |
| `OT_DATABASE_DSN` | `...@tcp(localhost:3306)/openvas-tracker?parseTime=true` | MariaDB DSN |
| `OT_JWT_SECRET` | `change-me-in-production` | JWT signing key |
| `OT_IMPORT_APIKEY` | (empty) | API key for import webhook (min 32 chars) |

## Database

- **MariaDB** with `database/sql` + `go-sql-driver/mysql`
- 14 migrations in `sql/migrations/` (golang-migrate format, numbered 001-014)
- `sql/docker-init.sql` sources all migrations for fresh Docker setup
- UUIDs are `CHAR(36)`, generated in Go code (`uuid.New().String()`), not DB-side

## Key Patterns

- **Auth flow:** JWT Bearer tokens. Public routes under `/api/auth/*`, everything else behind `middleware.JWTAuth`. Import endpoint uses `X-API-Key` header via `middleware.APIKeyAuth`. User ID extracted via `middleware.GetUserID(c)` (returns `string`).
- **Import flow:** OpenVAS webhook → parse XML → create scan record + vulnerabilities → for each vuln: find existing ticket by fingerprint (host + CVE/title) → create new / reopen fixed / update last_seen → auto-resolve open tickets not in current scan.
- **Ticket statuses:** `open` → `fixed` | `risk_accepted`. Auto-resolve sets `fixed`. Recurring finding reopens to `open`. All changes logged in `ticket_activity` table.
- **System user:** Import creates vulns/tickets under a dedicated `openvas-import` system user (auto-created on first import).
- **Report generation:** Synchronous — handler calls `ReportService.Generate()` which aggregates vulns from scan IDs and dispatches to the requested format generator.
- **SPA routing:** `cmd/openvas-tracker/frontend.go` serves embedded static files with fallback to `index.html` for client-side routing.
- **Frontend tables:** All list views use `TableFilter` + `SortHeader` components from `components/TableFilter.tsx` for filtering and sortable columns.

## Deployment

Docker Compose (MariaDB + single Go binary). Also supports Debian Trixie as a systemd service. Deploy files in `deploy/`:
- `Dockerfile` — multi-stage build (node → go → debian runtime)
- `docker-compose.yml` — MariaDB + app with health checks
- `openvas-tracker.service` — systemd unit with security hardening
- `install.sh` — creates user, installs binary, copies config, enables service
- `.github/workflows/release-deb.yml` — builds .deb on `v*` tag push (self-hosted runner), uploads to GitHub release

**Production server:** `SCANNER01` (192.168.1.100), Debian Trixie 13, accessible via `ssh scanner01`. Service runs as `openvas-tracker` user, config in `/etc/openvas-tracker/env`.

**Deploy .deb manually:** `scp` package tree to server, `dpkg-deb --build`, `dpkg -i`. Binary must be `chmod 755` (use `install -m 0755`, not `cp`).

## Gotchas

- **No Redis**: Was removed — no async task queue, no caching. Don't re-introduce.
- **No active scanning**: No Nmap, no gvm-cli, no scan launching. Import-only via webhook.
- **docker-init.sql**: Must be updated when adding new migrations (add `SOURCE` line).
- **deploy/openvas-tracker.env.example**: Keep in sync with `internal/config/config.go` — was stale before (had Postgres/Redis refs).
