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

- **`internal/handler/`** — Echo route handlers. Each has a `RegisterRoutes(*echo.Group)` method.
  - `auth.go` — Login (no registration). Admin + LDAP + DB fallback. By username.
  - `import.go` — Thin adapter → `ImportService`. GET triggers fetch script.
  - `tickets.go` — CRUD, status, comments, activity, bulk ops, risk rule creation. Validated via `c.Validate()`.
  - `scans.go` — List/Get + diff endpoint.
  - `settings.go` — Setup guide, user list, .env read/write, LDAP test, risk rules.
  - `pagination.go` — `paginate(c)` → `(limit, offset int32)`. Default 500, max 5000.
- **`internal/service/`** — Business logic.
  - `import.go` — Transaction-wrapped import: scan+vulns+tickets, risk rule matching, auto-resolve, PTR hostname backfill.
  - `ldap.go` — AD auth, group membership check, member listing.
  - `envfile.go` — Read/write `.env` for Settings UI.
- **`internal/database/queries/`** — Hand-written SQL. `db.go` defines `DBTX` interface (accepts `*sql.DB` and `*sql.Tx`).
- **`internal/scanner/`** — `ParseOpenVASXML`: CVE from `<refs><ref type="cve">` and `<nvt><cve>`, hostname from `<host><hostname>`.
- **`internal/report/`** — HTML, PDF (maroto v2), Excel (excelize), Markdown generators.
- **`internal/middleware/`** — JWT auth, API key auth (timing-safe), rate limiting, security headers.

**Frontend:** React 19 + Vite + Tailwind, embedded via `//go:embed all:static` in `cmd/openvas-tracker/frontend.go`.

## Configuration

All via `.env` file (`godotenv` + `os.Getenv`). Editable via Settings page. Auto-detects `/etc/openvas-tracker/env` on production, override with `OT_ENV_FILE`.

| Variable | Default | Purpose |
|----------|---------|---------|
| `OT_SERVER_PORT` | 8080 | HTTP listen port |
| `OT_DATABASE_DSN` | `...@tcp(localhost:3306)/openvas-tracker?parseTime=true` | MariaDB DSN |
| `OT_JWT_SECRET` | (none — **required**, min 32 chars) | JWT signing key |
| `OT_IMPORT_APIKEY` | (empty) | Import webhook API key (min 32 chars) |
| `OT_ADMIN_PASSWORD` | (empty) | Admin user password (username: `admin`) |
| `OT_LDAP_URL` | (empty) | e.g. `ldaps://dc01.example.com:636` |
| `OT_LDAP_BASE_DN` | (empty) | LDAP base DN |
| `OT_LDAP_BIND_DN` | (empty) | LDAP service account DN |
| `OT_LDAP_BIND_PASSWORD` | (empty) | LDAP service account password |
| `OT_LDAP_GROUP_DN` | (empty) | Required group DN for access |
| `OT_LDAP_USER_FILTER` | `(sAMAccountName=%s)` | LDAP user search filter |

## Database

- **MariaDB** with `database/sql` + `go-sql-driver/mysql`
- 17 migrations in `sql/migrations/` (001-017). `sql/docker-init.sql` sources all.
- UUIDs are `CHAR(36)`, generated in Go (`uuid.New().String()`)
- Pool: `MaxOpenConns`, `MaxIdleConns`, `ConnMaxLifetime(5m)`, `ConnMaxIdleTime(3m)`

## Key Patterns

### Auth
- Login by username, three sources in order: (1) admin + `OT_ADMIN_PASSWORD`, (2) LDAP bind + group check, (3) DB user fallback. No registration. Rate limited 30/min/IP.
- LDAP re-reads `.env` on each login for live config changes. Auto-creates DB user on first LDAP login.
- No roles — all users have equal access. Role column exists but is never checked.

### Import
- OpenVAS webhook → `ImportService.Import()` → single transaction: scan + vulns + tickets.
- Per vuln: check risk accept rules → find ticket by fingerprint (host + CVE/title) → create/reopen/touch → auto-resolve stale → commit.
- PTR hostname backfill runs async after each import. Hostnames normalized: `UPPERCASE.domain.lowercase`.

### Tickets
- Statuses: `open` → `fixed` | `risk_accepted` | `false_positive`. False positives never reopened.
- Risk accepted supports optional expiry date (auto-reopens when expired).
- All changes logged in `ticket_activity` with actor (user ID or "Automatic").
- Bulk: `POST /api/tickets/bulk` with `ticket_ids` + `status`/`assigned_to`.

### Risk Accept Rules
- `risk_accept_rules` table: fingerprint (CVE or `title:` + vuln title) + host pattern (`*` or IP).
- Created from ticket detail page ("this host" or "all hosts"). Applied to existing open tickets on creation.
- Checked during import — matching new tickets auto-set to `risk_accepted`.

### Frontend
- `TableFilter` + `SortHeader` components on all list views. Search matches all visible columns.
- Ticket list: checkbox bulk selection, default filter `status=open`, CVSS-sorted.
- Sidebar: Dashboard, My Tickets, All Tickets, Scans, Scan Diff, Auto-Accept Rules, Settings.
- Trend chart: 30-day daily snapshots of open tickets via recursive CTE.

## Deployment

Docker Compose or Debian Trixie systemd service. Use the `/deploy` skill for automated production deploys.

**Production:** Debian server accessible via SSH. Service runs as `openvas-tracker` user, config in `/etc/openvas-tracker/env`.

**GVM:** Greenbone CE Docker stack on production server. GMP socket at `/var/lib/docker/volumes/greenbone-community-edition_gvmd_socket_vol/_data/gvmd.sock`. Import triggered by GVM "HTTP Get" alert → fetch script → GMP socket → POST to self.

## Gotchas

- **No Redis, no active scanning, no registration** — import-only dashboard.
- **JWT secret required** — app refuses to start with default or short secret.
- **docker-init.sql** — must add `SOURCE` line when adding migrations.
- **DBTX interface** — `queries.New()` accepts `*sql.DB` and `*sql.Tx`. Use `*sql.Tx` in transactional flows.
- **Graceful shutdown** — handles SIGINT + SIGTERM (systemd/Docker).
- **Body limit** — global 5M with `Skipper` for `/api/import` (50M). `BodyLimitWithConfig` required — group-level limits can't override global.
- **GMP XML** — CVEs in `<refs><ref type="cve">`, not `<nvt><cve>`. Hostnames in `<host><hostname>` child element, IP is chardata.
- **Ticket title ≠ vuln title** — tickets formatted `[SEV] Title — Host`. Risk rules and dedup use raw vuln title.
- **MariaDB** — no FULL OUTER JOIN (use UNION ALL + NOT EXISTS), no `generate_series` (use `WITH RECURSIVE dates`).
- **SSH $ escaping** — passwords with `$` get shell-expanded. Use Python `chr(36)` or single-quoted heredoc.
- **Hostname normalization** — `normalizeHostname()`: UPPERCASE host, lowercase domain. Applied to imports + PTR.
