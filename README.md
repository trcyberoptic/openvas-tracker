# OpenVAS-Tracker

Vulnerability management dashboard that imports OpenVAS scan results and tracks remediation through automated ticketing.

## Screenshots

| Dashboard | Tickets | Ticket Detail |
|-----------|---------|---------------|
| ![Dashboard](docs/screenshots/dashboard.png) | ![Tickets](docs/screenshots/tickets.png) | ![Ticket Detail](docs/screenshots/ticket-detail.png) |

## Features

- **OpenVAS Import**: Webhook endpoint receives scan results automatically when scans complete
- **Automatic Ticketing**: New findings create tickets, missing findings auto-resolve, recurring findings reopen
- **Scope-aware Auto-resolve**: Importing a scan only auto-resolves tickets for hosts that were in that scan's scope — other subnets are unaffected
- **Ticket Lifecycle**: open → fixed / risk_accepted / false_positive, with full activity audit trail
- **Risk Acceptance with Expiry**: Risk-accepted tickets auto-reopen after expiry date
- **Auto-Accept Rules**: Define rules (by CVE or title, per host or globally) to automatically accept known risks on future imports — configurable from any ticket
- **Scan Comparison**: Side-by-side diff of two scans — new, fixed, unchanged findings
- **Bulk Actions**: Select multiple tickets for batch status change or assignment
- **Host-centric View**: Aggregated vulnerability counts and ticket status per host with expandable ticket list
- **Dashboard**: Open ticket counts by priority, 30-day trend chart, "My Tickets" and "Unassigned" quick filters
- **CVE References**: NVD, MITRE, and Google links on tickets with CVE; title-based search for tickets without
- **Also Affected**: See which other hosts have the same vulnerability
- **DNS Hostname Resolution**: Automatic PTR lookup, normalized (UPPERCASE.domain.lowercase), shown everywhere
- **LDAP / Active Directory**: Optional AD authentication with group-based access control
- **Admin + LDAP Auth**: Built-in admin user plus optional LDAP for team access, login by username
- **Settings UI**: Edit all configuration (.env file) from the browser, test LDAP connection
- **Filterable & Sortable Tables**: Column sorting, multi-filter, full-text search across all columns, default filter on open tickets
- **Report Generation**: HTML, PDF, Excel, Markdown
- **Real-time Updates**: WebSocket push notifications
- **Embedded React SPA**: Single binary, no separate frontend deploy

## Architecture

```mermaid
sequenceDiagram
    participant OV as OpenVAS (GVM)
    participant TR as OpenVAS-Tracker
    participant AD as Active Directory
    participant DB as MariaDB
    participant UI as React Dashboard

    Note over OV: Scan completes
    OV->>TR: HTTP GET /api/import/openvas?api_key=...
    TR->>OV: GMP Socket: fetch latest report
    OV-->>TR: XML report
    TR->>DB: Begin transaction
    TR->>DB: Create scan + vulnerabilities + tickets
    TR->>DB: Check risk accept rules → auto-accept matches
    TR->>DB: Auto-fix/reopen tickets
    TR->>DB: Commit
    TR->>UI: WebSocket push

    Note over UI: User logs in
    UI->>TR: POST /api/auth/login (username + password)
    alt Admin user
        TR->>TR: Check OT_ADMIN_PASSWORD
    else LDAP user
        TR->>AD: Bind + group check
    end
    TR-->>UI: JWT token
```

## Quick Start with Docker

```bash
cp .env.example .env    # edit: set OT_JWT_SECRET, OT_ADMIN_PASSWORD, OT_IMPORT_APIKEY
docker compose up -d
```

The UI is at http://localhost:8080. Login: username `admin`, password from `OT_ADMIN_PASSWORD`.

## Quick Start without Docker

```bash
# 1. Create database
mysql -e "CREATE DATABASE \`openvas-tracker\` CHARACTER SET utf8mb4;"

# 2. Run migrations
make migrate-up

# 3. Configure
cat > .env << EOF
OT_DATABASE_DSN=root@tcp(localhost:3306)/openvas-tracker?parseTime=true
OT_JWT_SECRET=$(openssl rand -hex 32)
OT_IMPORT_APIKEY=$(openssl rand -hex 32)
OT_ADMIN_PASSWORD=your-admin-password
EOF

# 4. Build and run
make build && ./bin/openvas-tracker
```

## Configuration

All config via `.env` file. Editable from the Settings page in the UI.

| Variable | Default | Purpose |
|----------|---------|---------|
| `OT_SERVER_PORT` | 8080 | HTTP listen port |
| `OT_DATABASE_DSN` | `...@tcp(localhost:3306)/openvas-tracker?parseTime=true` | MariaDB DSN |
| `OT_JWT_SECRET` | (none — **required**) | JWT signing key (min 32 chars) |
| `OT_IMPORT_APIKEY` | (empty) | API key for import webhook (min 32 chars) |
| `OT_ADMIN_PASSWORD` | (empty) | Admin user password |
| `OT_LDAP_URL` | (empty) | LDAP server URL |
| `OT_LDAP_BASE_DN` | (empty) | LDAP search base DN |
| `OT_LDAP_BIND_DN` | (empty) | LDAP service account DN |
| `OT_LDAP_BIND_PASSWORD` | (empty) | LDAP service account password |
| `OT_LDAP_GROUP_DN` | (empty) | Required AD group for access |
| `OT_LDAP_USER_FILTER` | `(sAMAccountName=%s)` | LDAP user search filter |

## Authentication

1. **Admin**: Username `admin` + `OT_ADMIN_PASSWORD` → always available
2. **LDAP**: Bind against Active Directory, verify group membership → if configured
3. **DB fallback**: Existing database users → for backwards compatibility

No self-registration. LDAP users auto-created in DB on first login and also when the user list is loaded (Settings → Users), so they can be assigned to tickets before their first login.

## OpenVAS Setup

1. Set `OT_IMPORT_APIKEY` in `.env`
2. In GSA: **Configuration → Alerts → New Alert** → HTTP Get → `http://<host>:8080/api/import/openvas?api_key=<key>`
3. Attach alert to scan task

## Ticket Lifecycle

```
Import finds new vulnerability     →  Ticket created (open)
Import matches risk accept rule    →  Ticket created (risk_accepted)
Import finds same vulnerability    →  Ticket updated (last_seen_at)
Import missing old vulnerability   →  Ticket auto-fixed
Import re-finds fixed vuln        →  Ticket reopened (open)
Import re-finds false_positive     →  Skipped (never reopened)
Risk acceptance expires            →  Ticket auto-reopened
```

## Auto-Accept Rules

Rules automatically set matching tickets to `risk_accepted` during import. Created from any ticket's detail page with scope "this host only" or "all hosts". Managed via the Auto-Accept Rules page.

Matching by: CVE ID (if available) or vulnerability title. Optional expiry date.

## API

| Method | Path | Description |
|--------|------|-------------|
| POST | /api/auth/login | Login (username + password) |
| POST | /api/import/openvas | Import OpenVAS XML (API-Key) |
| GET | /api/import/openvas | Trigger GMP fetch (API-Key) |
| GET | /api/hosts | Host summaries with ticket status |
| GET | /api/hosts/:host/tickets | Tickets for a host |
| GET | /api/scans | List scans |
| GET | /api/scans/diff?old=X&new=Y | Compare two scans |
| GET | /api/scans/:id | Scan detail |
| GET | /api/tickets | List all tickets |
| POST | /api/tickets/bulk | Bulk status/assign |
| GET | /api/tickets/:id | Ticket detail |
| PATCH | /api/tickets/:id/status | Change status |
| PATCH | /api/tickets/:id/assign | Assign to user |
| POST | /api/tickets/:id/risk-rule | Create auto-accept rule from ticket |
| POST/GET | /api/tickets/:id/comments | Notes |
| GET | /api/tickets/:id/activity | Activity log |
| GET | /api/tickets/:id/also-affected | Other affected hosts |
| GET | /api/dashboard | Priority counts + ticket stats |
| GET | /api/dashboard/trend | 30-day open ticket trend |
| GET | /api/settings/setup | Setup guide |
| GET | /api/settings/users | User list (local + LDAP) |
| GET/PUT | /api/settings/env | Read/write .env config |
| POST | /api/settings/ldap/test | Test LDAP connection |
| GET | /api/settings/risk-rules | List auto-accept rules |
| POST | /api/settings/risk-rules/apply | Re-apply rules to existing open tickets |
| DELETE | /api/settings/risk-rules/:id | Delete rule |
| GET | /api/health | Health check |

## Tech Stack

- **Backend**: Go 1.26, Echo v4, MariaDB, golang-jwt, bcrypt, godotenv, go-ldap
- **Frontend**: React 19, Vite, Tailwind CSS, TanStack Query, Recharts, Zustand
- **Deploy**: Docker Compose or systemd (Debian). Database migrations auto-applied on startup

## Donate

If you find this project useful, consider supporting development:

**XMR (Monero):**
```
89fMD41wm8n88tgVj836qf3m16odqRjBhLti8dmVbvgsYAuEpTGfHBL7zNW8hingxQJNLWXfP3c2tgyyUMxYBiqHVYWR2rU
```

## License

GPL v3
