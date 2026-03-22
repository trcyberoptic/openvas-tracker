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
  - **`auth.go`** — Login only (no registration). Admin auth via `OT_ADMIN_PASSWORD`, LDAP auth via AD, fallback to DB user. Login by username.
  - **`import.go`** — Thin HTTP adapter for import. Parses XML, delegates to `ImportService`. Also `GET /api/import/openvas` triggers `openvas-tracker-fetch-latest` script.
  - **`tickets.go`** — CRUD + status changes + comments + activity log + bulk operations + risk accept rule creation from ticket. All inputs validated via `c.Validate()`.
  - **`scans.go`** — List/Get scans + scan diff comparison endpoint.
  - **`settings.go`** — Setup guide, user list (local + LDAP), .env file read/write, LDAP test, risk accept rules list/delete.
  - **`pagination.go`** — Shared `paginate(c)` helper, returns `(limit, offset int32)`. Default 500, max 5000.
- **`internal/service/`** — Business logic. Each takes `*sql.DB` in constructor.
  - **`import.go`** — Core import logic: transaction-wrapped scan+vuln+ticket creation, system user management, auto-resolve, risk expiry, hostname PTR resolution + normalization, risk accept rule matching. All ticket lifecycle logic lives here.
  - **`ldap.go`** — LDAP authentication against Active Directory with group membership check, group member listing, connection test.
  - **`envfile.go`** — Read/write `.env` file for config management via Settings UI.
- **`internal/database/queries/`** — Hand-written query stubs. Uses `database/sql` with `go-sql-driver/mysql`.
  - **`db.go`** — `DBTX` interface accepted by `New()` — supports both `*sql.DB` and `*sql.Tx` for transaction support.
  - **`tickets.go`** — Ticket queries including `FindTicketByFingerprint`, `AutoResolveStaleTickets` (SELECT-then-UPDATE), `LogTicketActivity`, `ListTicketsByHost`, `AlsoAffectedHosts`.
  - **`scans.go`** — Scan queries including `DiffScans` for scan comparison.
  - **`risk_rules.go`** — Risk accept rule CRUD, matching, and batch application to existing tickets.
- **`internal/scanner/`** — `ParseOpenVASXML` XML parser. Extracts CVE from `<nvt><cve>` and `<nvt><refs><ref type="cve">`. Parses hostname from `<host><hostname>`.
- **`internal/report/`** — Report generators: `GenerateHTML`, `GeneratePDF` (maroto v2), `GenerateExcel` (excelize), `GenerateMarkdown`.
- **`internal/websocket/`** — Hub + Client for real-time per-user push via gorilla/websocket. Origin-validated.
- **`internal/auth/`** — JWT (golang-jwt) and bcrypt password utilities.
- **`internal/middleware/`** — Echo middleware: JWT auth, API key auth (timing-safe), rate limiting, security headers (CSP + Permissions-Policy), audit logging.

**Frontend:** React 19 + Vite + Tailwind + shadcn/ui, embedded in the Go binary via `//go:embed all:static` in `cmd/openvas-tracker/frontend.go`. The Makefile copies `frontend/dist/` → `cmd/openvas-tracker/static/` before Go build.

## Configuration

All config via `.env` file (`godotenv` + `os.Getenv`, `internal/config/config.go`). Editable via Settings page in the UI.

| Variable | Default | Purpose |
|----------|---------|---------|
| `OT_SERVER_PORT` | 8080 | HTTP listen port |
| `OT_DATABASE_DSN` | `...@tcp(localhost:3306)/openvas-tracker?parseTime=true` | MariaDB DSN |
| `OT_JWT_SECRET` | (none — **required**, min 32 chars) | JWT signing key |
| `OT_IMPORT_APIKEY` | (empty) | API key for import webhook (min 32 chars) |
| `OT_ADMIN_PASSWORD` | (empty) | Admin user password (username: `admin`) |
| `OT_LDAP_URL` | (empty) | LDAP server URL (e.g. `ldaps://dc01.example.com:636`) |
| `OT_LDAP_BASE_DN` | (empty) | LDAP base DN |
| `OT_LDAP_BIND_DN` | (empty) | LDAP service account DN |
| `OT_LDAP_BIND_PASSWORD` | (empty) | LDAP service account password |
| `OT_LDAP_GROUP_DN` | (empty) | Required group DN for access |
| `OT_LDAP_USER_FILTER` | `(sAMAccountName=%s)` | LDAP user search filter |

## Database

- **MariaDB** with `database/sql` + `go-sql-driver/mysql`
- 17 migrations in `sql/migrations/` (golang-migrate format, numbered 001-017)
- `sql/docker-init.sql` sources all migrations for fresh Docker setup
- UUIDs are `CHAR(36)`, generated in Go code (`uuid.New().String()`), not DB-side
- Connection pool: `SetMaxOpenConns`, `SetMaxIdleConns`, `SetConnMaxLifetime(5m)`, `SetConnMaxIdleTime(3m)`

## Key Patterns

- **Auth flow:** Login by username. Three auth sources tried in order: (1) admin user if username is `admin` and `OT_ADMIN_PASSWORD` matches, (2) LDAP bind if configured, (3) DB user fallback. No registration endpoint. JWT Bearer tokens for API auth. Import endpoint uses `X-API-Key` header. Login rate limited to 30/min/IP. LDAP config re-read from `.env` on each login for live updates.
- **LDAP:** Optional Active Directory integration. Configured via `.env` (editable in Settings page). Authenticates via bind, checks group membership. Auto-creates DB user on first LDAP login. Group members listed for ticket assignment.
- **No roles:** All authenticated users have equal access. Role column exists in DB but is not checked.
- **Import flow:** OpenVAS webhook → parse XML → `ImportService.Import()` → transaction: create scan + vulnerabilities → for each vuln: check risk accept rules → find existing ticket by fingerprint (host + CVE/title) → create new / reopen fixed / update last_seen → auto-resolve open tickets not in current scan → commit. PTR hostname backfill runs async after each import.
- **Ticket statuses:** `open` → `fixed` | `risk_accepted` | `false_positive`. Auto-resolve sets `fixed`. Recurring finding reopens to `open`. False positives never reopened. Risk accepted has optional expiry date. All changes logged in `ticket_activity` table.
- **Risk accept rules:** `risk_accept_rules` table with fingerprint (CVE or `title:` + vuln title) + host pattern (`*` or specific IP). Created from ticket detail page. Applied to existing open tickets on creation and checked during import. Managed via Auto-Accept Rules page.
- **Scan diff:** `GET /api/scans/diff?old=X&new=Y` compares two scans by vuln fingerprint, returns new/fixed/unchanged.
- **Bulk actions:** `POST /api/tickets/bulk` accepts array of ticket IDs + status/assigned_to for batch operations.
- **Trend:** Daily snapshot of open ticket count over last 30 days, using `created_at`/`resolved_at` timestamps. Excludes false positives.
- **Hostname normalization:** All hostnames stored as UPPERCASE.domain.lowercase (e.g. `VGITLAB01.example.local`). Applied during import and PTR resolution.
- **System user:** Import creates vulns/tickets under a dedicated `openvas-import` system user (auto-created on first import, mutex-protected with retry on failure).
- **Settings page:** Reads/writes `.env` file directly. Sensitive values masked. LDAP test connection button. Changes require service restart. Auto-detects `/etc/openvas-tracker/env` on production.
- **SPA routing:** `cmd/openvas-tracker/frontend.go` serves embedded static files with fallback to `index.html` for client-side routing.
- **Frontend tables:** All list views use `TableFilter` + `SortHeader` components. Search matches all visible columns. Ticket list has checkbox bulk selection. Default filter: status=open.
- **Security headers:** CSP (no unsafe-inline), Permissions-Policy, X-Frame-Options DENY, global 5M body limit with skipper for import (50M).

## Deployment

Docker Compose (MariaDB + single Go binary). Also supports Debian Trixie as a systemd service. Deploy files in `deploy/`:
- `Dockerfile` — multi-stage build (node → go → debian runtime)
- `docker-compose.yml` — MariaDB + app with health checks
- `openvas-tracker.service` — systemd unit with security hardening, `ReadWritePaths` includes `/etc/openvas-tracker` for .env editing
- `install.sh` — creates user, installs binary, copies config, enables service
- `.github/workflows/release-deb.yml` — builds .deb on `v*` tag push (self-hosted runner), uploads to GitHub release

**Production server:** `SCANNER01` (192.168.1.100), Debian Trixie 13, accessible via `ssh scanner01`. Service runs as `openvas-tracker` user, config in `/etc/openvas-tracker/env`.

## GVM Integration (scanner01)

- **Greenbone Community Edition** runs as Docker stack (~16 containers) on scanner01.
- **GMP socket:** `/var/lib/docker/volumes/greenbone-community-edition_gvmd_socket_vol/_data/gvmd.sock`
- **GVM admin creds:** `admin` / `admin`
- **Import trigger:** GVM "HTTP Get" alert → `GET /api/import/openvas?api_key=...` → Go handler calls `sudo /usr/local/bin/openvas-tracker-fetch-latest` (120s timeout) → script connects GMP socket, fetches report, POSTs to self.
- **LDAP:** AD auth via `ldaps://pdc.example.local`, group `IT-Security`, service account `svc_scanner`.

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
- **No registration**: Users authenticate via admin password or LDAP. No self-registration endpoint.
- **No roles/RBAC**: All users have equal access. Role column exists but is never checked.
- **JWT secret required**: App refuses to start with default or short secret.
- **docker-init.sql**: Must be updated when adding new migrations (add `SOURCE` line).
- **DBTX interface**: `queries.New()` accepts both `*sql.DB` and `*sql.Tx` — use `*sql.Tx` in transactional flows (see `ImportService`).
- **Graceful shutdown**: Handles both SIGINT and SIGTERM (important for systemd/Docker).
- **Body limit**: Global 5M with skipper for `/api/import` (50M). Large OpenVAS reports can be 10MB+.
- **LDAP config**: Stored in `.env`, editable via Settings page. `currentLDAPConfig()` re-reads `.env` on each login.
- **Hostname normalization**: `normalizeHostname()` in import service — UPPERCASE host, lowercase domain. Applied to all imports and PTR lookups.
- **Risk accept fingerprint**: Uses `VulnFingerprint()` — CVE ID if available, otherwise `title:` + raw vulnerability title (not formatted ticket title).
- **Env file path**: Auto-detects `/etc/openvas-tracker/env` if it exists, otherwise `.env`. Override with `OT_ENV_FILE`.
- **SSH $ escaping**: Passwords with `$` get shell-expanded via SSH. Use Python `chr(36)` or heredoc with single-quoted delimiter to write literal `$` to files on remote hosts.
- **Echo BodyLimit stacking**: Global `BodyLimit` cannot be overridden by group-level limits. Use `BodyLimitWithConfig` with a `Skipper` function to exempt specific paths.
- **GMP XML quirks**: CVEs are in `<refs><ref type="cve">`, not `<nvt><cve>`. Hostnames are in `<host><hostname>` as a child element, IP is chardata.
- **Ticket title ≠ vuln title**: Ticket titles are formatted `[SEV] Title — Host`. Risk rule fingerprints and dedup must use raw vulnerability title from `vulnerabilities` table.
- **MariaDB no FULL OUTER JOIN**: Use UNION ALL with NOT EXISTS for each direction instead.
- **Recursive CTE for date series**: MariaDB has no `generate_series`. Use `WITH RECURSIVE dates AS (SELECT CURDATE() - INTERVAL 29 DAY AS d UNION ALL SELECT d + INTERVAL 1 DAY FROM dates WHERE d < CURDATE())`.
