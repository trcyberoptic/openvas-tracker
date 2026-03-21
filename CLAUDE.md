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
                                                → scanner (XML parser only)
                                                → report (html/pdf/excel/md)
```

- **`internal/handler/`** — Echo route handlers. Each has a `RegisterRoutes(*echo.Group)` method mounted in main.go.
  - **`import.go`** — Thin HTTP adapter for import. Parses XML, delegates to `ImportService`. Also `GET /api/import/openvas` triggers `openvas-tracker-fetch-latest` script to pull latest report from GVM via GMP socket.
  - **`tickets.go`** — CRUD + status changes + comments + activity log. All inputs validated via `c.Validate()`.
  - **`pagination.go`** — Shared `paginate(c)` helper, returns `(limit, offset int32)`. Default 500, max 5000.
  - **`settings.go`** — Setup guide endpoint (API key masked, never exposed in plaintext) + user list for ticket assignment.
- **`internal/service/`** — Business logic. Each takes `*sql.DB` in constructor.
  - **`import.go`** — Core import logic: transaction-wrapped scan+vuln+ticket creation, system user management, auto-resolve, risk expiry. All ticket lifecycle logic lives here.
- **`internal/database/queries/`** — Hand-written query stubs. Uses `database/sql` with `go-sql-driver/mysql`.
  - **`db.go`** — `DBTX` interface accepted by `New()` — supports both `*sql.DB` and `*sql.Tx` for transaction support.
  - **`tickets.go`** — Ticket queries including `FindTicketByFingerprint`, `AutoResolveStaleTickets` (SELECT-then-UPDATE, no time-window race), `LogTicketActivity`.
- **`internal/scanner/`** — `ParseOpenVASXML` XML parser. No CLI wrappers — import only.
- **`internal/report/`** — Report generators: `GenerateHTML`, `GeneratePDF` (maroto v2), `GenerateExcel` (excelize), `GenerateMarkdown`.
- **`internal/websocket/`** — Hub + Client for real-time per-user push via gorilla/websocket. Origin-validated.
- **`internal/auth/`** — JWT (golang-jwt) and bcrypt password utilities.
- **`internal/middleware/`** — Echo middleware: JWT auth, API key auth (timing-safe), rate limiting, security headers (CSP + Permissions-Policy), audit logging.

**Frontend:** React 19 + Vite + Tailwind + shadcn/ui, embedded in the Go binary via `//go:embed all:static` in `cmd/openvas-tracker/frontend.go`. The Makefile copies `frontend/dist/` → `cmd/openvas-tracker/static/` before Go build.

## Configuration

All config via environment variables with `OT_` prefix (`godotenv` + `os.Getenv`, `internal/config/config.go`):

| Variable | Default | Purpose |
|----------|---------|---------|
| `OT_SERVER_PORT` | 8080 | HTTP listen port |
| `OT_DATABASE_DSN` | `...@tcp(localhost:3306)/openvas-tracker?parseTime=true` | MariaDB DSN |
| `OT_JWT_SECRET` | (none — **required**, min 32 chars) | JWT signing key |
| `OT_IMPORT_APIKEY` | (empty) | API key for import webhook (min 32 chars) |

## Database

- **MariaDB** with `database/sql` + `go-sql-driver/mysql`
- 15 migrations in `sql/migrations/` (golang-migrate format, numbered 001-015)
- `sql/docker-init.sql` sources all migrations for fresh Docker setup
- UUIDs are `CHAR(36)`, generated in Go code (`uuid.New().String()`), not DB-side
- Connection pool: `SetMaxOpenConns`, `SetMaxIdleConns`, `SetConnMaxLifetime(5m)`, `SetConnMaxIdleTime(3m)`

## Key Patterns

- **Auth flow:** JWT Bearer tokens. Public routes under `/api/auth/*`, everything else behind `middleware.JWTAuth`. Import endpoint uses `X-API-Key` header via `middleware.APIKeyAuth` (also accepted as `?api_key=` query param for GVM HTTP Get alerts). User ID extracted via `middleware.GetUserID(c)` (returns `string`). JWT secret enforced at startup (min 32 chars, no default). Login rate limited to 10/min/IP.
- **No roles:** All authenticated users have equal access. Role column exists in DB but is not checked anywhere.
- **Import flow:** OpenVAS webhook → parse XML → `ImportService.Import()` → transaction: create scan + vulnerabilities → for each vuln: find existing ticket by fingerprint (host + CVE/title) → create new / reopen fixed / update last_seen → auto-resolve open tickets not in current scan → commit.
- **Ticket statuses:** `open` → `fixed` | `risk_accepted` | `false_positive`. Auto-resolve sets `fixed`. Recurring finding reopens to `open`. False positives never reopened. Risk accepted has optional expiry date (auto-reopens when expired). All changes logged in `ticket_activity` table.
- **System user:** Import creates vulns/tickets under a dedicated `openvas-import` system user (auto-created on first import, mutex-protected with retry on failure).
- **Report generation:** Synchronous — handler calls `ReportService.Generate()` which aggregates vulns from scan IDs and dispatches to the requested format generator.
- **SPA routing:** `cmd/openvas-tracker/frontend.go` serves embedded static files with fallback to `index.html` for client-side routing.
- **Frontend tables:** All list views use `TableFilter` + `SortHeader` components from `components/TableFilter.tsx` for filtering and sortable columns.
- **Security headers:** CSP (no unsafe-inline), Permissions-Policy, X-Frame-Options DENY, global 1M body limit (10M for import).

## Deployment

Docker Compose (MariaDB + single Go binary). Also supports Debian Trixie as a systemd service. Deploy files in `deploy/`:
- `Dockerfile` — multi-stage build (node → go → debian runtime)
- `docker-compose.yml` — MariaDB + app with health checks
- `openvas-tracker.service` — systemd unit with security hardening
- `install.sh` — creates user, installs binary, copies config, enables service
- `.github/workflows/release-deb.yml` — builds .deb on `v*` tag push (self-hosted runner), uploads to GitHub release

**Production server:** `SCANNER01` (192.168.1.100), Debian Trixie 13, accessible via `ssh scanner01`. Service runs as `openvas-tracker` user, config in `/etc/openvas-tracker/env`.

## GVM Integration (scanner01)

- **Greenbone Community Edition** runs as Docker stack (~16 containers) on scanner01.
- **GMP socket:** `/var/lib/docker/volumes/greenbone-community-edition_gvmd_socket_vol/_data/gvmd.sock`
- **GVM admin creds:** `admin` / `admin`
- **Import trigger:** GVM "HTTP Get" alert → `GET /api/import/openvas?api_key=...` → Go handler calls `sudo /usr/local/bin/openvas-tracker-fetch-latest` → script connects GMP socket, fetches report (120s timeout), POSTs to self.
- **NoNewPrivileges=no** in systemd unit — required for sudo to GMP socket.
- **Sudoers:** `/etc/sudoers.d/openvas-tracker-fetch` allows openvas-tracker user to run fetch script as root.

## Quick Deploy to Production

```bash
cd frontend && npm run build && cd ..
rm -rf cmd/openvas-tracker/static && cp -r frontend/dist cmd/openvas-tracker/static
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o bin/openvas-tracker-linux-amd64 ./cmd/openvas-tracker
scp bin/openvas-tracker-linux-amd64 scanner01:/usr/local/bin/openvas-tracker.new
ssh scanner01 "chmod 755 /usr/local/bin/openvas-tracker.new && systemctl stop openvas-tracker && mv /usr/local/bin/openvas-tracker.new /usr/local/bin/openvas-tracker && systemctl start openvas-tracker"
```

## Gotchas

- **No Redis**: Was removed — no async task queue, no caching. Don't re-introduce.
- **No active scanning**: No Nmap, no gvm-cli, no scan launching. Import-only via webhook.
- **No roles/RBAC**: All users have equal access. Role column exists but is never checked.
- **JWT secret required**: App refuses to start with default or short secret.
- **docker-init.sql**: Must be updated when adding new migrations (add `SOURCE` line).
- **DBTX interface**: `queries.New()` accepts both `*sql.DB` and `*sql.Tx` — use `*sql.Tx` in transactional flows (see `ImportService`).
- **Graceful shutdown**: Handles both SIGINT and SIGTERM (important for systemd/Docker).
