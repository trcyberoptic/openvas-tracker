# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

OpenVAS-Tracker is a vulnerability management dashboard built in Go with an embedded React SPA. It receives OpenVAS and OWASP ZAP scan results via webhook, tracks vulnerabilities per host/URL, and manages remediation through automated ticketing. Multi-scanner architecture with pluggable parsers. No active scanning — purely a results viewer and ticket tracker. Licensed under GPL v3.

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
                                                → scanner (XML/JSON parsers)
                                                → report (html/pdf/excel/md)
```

- **`internal/handler/`** — Echo route handlers. Each has a `RegisterRoutes(*echo.Group)` method.
  - `auth.go` — Login (no registration). Admin + LDAP + DB fallback. By username.
  - `import.go` — Thin adapter → `ImportService`. `POST /openvas` (XML), `POST /zap` (JSON), `GET /openvas` triggers fetch script.
  - `tickets.go` — CRUD, status, comments, activity, bulk ops, risk rule creation. Validated via `c.Validate()`. Detail page shows prominent CVSS score box.
  - `scans.go` — List/Get + diff endpoint.
  - `settings.go` — Setup guide, user list, .env read/write, LDAP test, risk rules.
  - `pagination.go` — `paginate(c)` → `(limit, offset int32)`. Default 500, max 5000.
- **`internal/service/`** — Business logic.
  - `import.go` — Transaction-wrapped import: scan+vulns+tickets, risk rule matching, auto-resolve, PTR hostname backfill.
  - `ldap.go` — AD auth, group membership check, member listing.
  - `envfile.go` — Read/write `.env` for Settings UI.
- **`internal/database/queries/`** — Hand-written SQL. `db.go` defines `DBTX` interface (accepts `*sql.DB` and `*sql.Tx`).
- **`internal/scanner/`** — Multi-scanner parser package.
  - `scanner.go` — `Finding` struct (scanner-agnostic) + `ScanMeta` (optional scan timestamps) + `Parser` interface + `Fingerprint()` method.
  - `openvas.go` — `ParseOpenVASXML`: CVE from `<refs><ref type="cve">` and `<nvt><cve>`, hostname from `<host><hostname>`. Returns `([]Finding, *ScanMeta, error)`. Extracts `scan_start`/`scan_end` from GMP XML for accurate scan timestamps.
  - `zap.go` — `ParseZAPJSON`: ZAP Traditional JSON Report. Each alert instance → one Finding. Uses `@host`/`@port`/`@ssl` keys. Strips HTML from desc/solution. Maps riskcode 3→high, 2→medium, 1→low, 0→info (skipped). Default CVSS: 7.0/4.0/2.0/0.0.
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
| `OT_AUTORESOLVE_THRESHOLD` | `3` | Consecutive scans without finding before auto-resolve |
| `OT_LDAP_URL` | (empty) | e.g. `ldaps://dc01.example.com:636` |
| `OT_LDAP_BASE_DN` | (empty) | LDAP base DN |
| `OT_LDAP_BIND_DN` | (empty) | LDAP service account DN |
| `OT_LDAP_BIND_PASSWORD` | (empty) | LDAP service account password |
| `OT_LDAP_GROUP_DN` | (empty) | Required group DN for access |
| `OT_LDAP_USER_FILTER` | `(sAMAccountName=%s)` | LDAP user search filter |

## Database

- **MariaDB** with `database/sql` + `go-sql-driver/mysql`
- 20 migrations in `sql/migrations/` (001-020). `sql/docker-init.sql` sources all.
- **Auto-migrate on startup** — `AutoMigrate` applies pending migrations automatically when the app starts. Bootstraps `schema_migrations` for existing databases. Bootstrap only marks CREATE TABLE migrations as applied — ALTER TABLE migrations always run.
- UUIDs are `CHAR(36)`, generated in Go (`uuid.New().String()`)
- Pool: `MaxOpenConns`, `MaxIdleConns`, `ConnMaxLifetime(5m)`, `ConnMaxIdleTime(3m)`

## Key Patterns

### Auth
- Login by username, three sources in order: (1) admin + `OT_ADMIN_PASSWORD`, (2) LDAP bind + group check, (3) DB user fallback. No registration. Rate limited 30/min/IP.
- LDAP re-reads `.env` on each login for live config changes. Auto-creates DB user on first LDAP login. LDAP group members are also auto-created (with real UUIDs) when the user list is loaded, so they can be assigned to tickets before first login. Users without email are skipped.
- No roles — all users have equal access. Role column exists but is never checked.

### Import
- OpenVAS/ZAP webhook → `ImportService.Import(ctx, []Finding, scanType, *ScanMeta)` → single transaction: scan + vulns + tickets.
- `Finding` struct is scanner-agnostic. `ParseOpenVASXML` returns `([]Finding, *ScanMeta, error)` — `ScanMeta` carries `scan_start`/`scan_end` from the GMP XML report so the scan record gets the original scan timestamps instead of the import time. `ParseZAPJSON` returns `([]Finding, error)` — ZAP reports don't include scan timestamps, so `nil` meta is passed and `time.Now()` is used.
- **Fingerprinting** (dedup key per finding):
  - Network findings (OpenVAS): CVE or `"title:" + title`. Key: `(host, fingerprint)`.
  - Web findings (ZAP): `"cwe:" + cweid + ":url:" + urlPath + ":param:" + param` — only when parameter is present. Falls back to `"title:" + title + ":url:" + urlPath + ":param:" + param` if no CWE. CVE always takes priority if present.
  - **Server-wide findings** (no parameter, e.g. missing headers): `"cwe:" + cweid` or `"title:" + title` — one ticket per host, not per URL. All affected URLs shown in ticket detail via `AffectedURLsByPeer` query.
- Per vuln: check risk accept rules → find ticket by fingerprint (host + CVE/title/CWE+URL) → create/reopen/touch → auto-resolve stale → commit.
- **Auto-resolve scoped by scan type** — a ZAP scan only auto-resolves ZAP tickets, never OpenVAS tickets and vice versa. Uses `(host, scan_type)` scope via `scan_hosts` table (migration 018).
- PTR hostname backfill runs async after each import. Hostnames normalized: `UPPERCASE.domain.lowercase`.

### Tickets
- Statuses: `open` → `pending_resolution` (after 1+ scan miss) → `fixed` (after N consecutive misses, configurable via `OT_AUTORESOLVE_THRESHOLD`, default 3) | `risk_accepted` | `false_positive`. False positives never reopened.
- Flapping protection: findings not present in a scan increment `consecutive_misses` counter. After threshold consecutive misses, ticket auto-resolves to `fixed`. If finding reappears before threshold, counter resets and ticket returns to `open` with activity log. `pending_resolution` is visible in UI with amber badge.
- Risk accepted supports optional expiry date (auto-reopens when expired).
- All changes logged in `ticket_activity` with actor (user ID or "Automatic").
- Bulk: `POST /api/tickets/bulk` with `ticket_ids` + `status`/`assigned_to`.

### Risk Accept Rules
- `risk_accept_rules` table: fingerprint (CVE or `title:` + vuln title) + host pattern (`*` or IP).
- Created from ticket detail page ("this host" or "all hosts"). Applied to existing open and pending_resolution tickets on creation.
- Checked during import — matching new tickets auto-set to `risk_accepted`.
- "Refresh Tickets" button on Auto-Accept Rules page re-applies all rules to existing open and pending_resolution tickets (`POST /api/settings/risk-rules/apply`).

### Frontend
- `TableFilter` + `SortHeader` components on all list views. Search matches all visible columns. `FilterOption` supports `searchable: true` with `SelectOption[]` (`{value, label}`) for autocomplete combobox instead of plain `<select>`.
- Ticket list: checkbox bulk selection, default filter `status=open`, CVSS-sorted. Source filter (OpenVAS/ZAP). Host filter is a searchable combobox showing `IP (hostname)` — type to filter by IP or hostname.
- Scan list: scan type badges (OpenVAS green, ZAP blue) with type filter.
- Ticket detail: web finding details section (URL, parameter, evidence, confidence badges, CWE links) — only shown for ZAP findings. Server-wide findings show all affected URLs from peer vulnerabilities.
- Dashboard: open tickets by scan source pie chart (OpenVAS vs ZAP).
- Sidebar: Dashboard, My Tickets, All Tickets, Scans, Scan Diff, Auto-Accept Rules, Settings. GitHub repo link at bottom.
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
- **ticketCols / qualifiedTicketCols / scanTicket** — these three must stay in sync when adding columns to `tickets` table. All are in `internal/database/queries/tickets.go`. Ticket queries JOIN vulnerabilities AND scans (for `scan_type` field).
- **ZAP JSON `@`-prefixed keys** — ZAP Traditional JSON Reports use `@host`, `@port`, `@ssl` (not `host`, `port`, `ssl`). The `zapSite` struct uses `json:"@host"` etc.
- **vulnCols / scanVuln** — must stay in sync with vulnerabilities table. Includes `url`, `parameter`, `evidence`, `confidence` (migration 020).
- **Backend-only features** — Teams (`/api/teams`) and Assets (`/api/assets`) have full backend handlers but no frontend UI beyond a read-only teams list.
- **git-filter-repo** — removes `origin` remote (re-add with `git remote add origin <url>`), resets working copy (uncommitted edits lost), and breaks tracking (`git branch --set-upstream-to=origin/master master`).
