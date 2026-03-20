# OpenVAS-Tracker

Vulnerability management dashboard that imports OpenVAS scan results and tracks remediation through automated ticketing.

## Features

- **OpenVAS Import**: Webhook endpoint receives scan results via `POST /api/import/openvas`
- **Automatic Ticketing**: New findings create tickets, fixed findings auto-resolve, recurring findings reopen tickets
- **Ticket Lifecycle**: open → fixed / risk_accepted, with full activity audit trail
- **Host-centric View**: Aggregated vulnerability counts per host with expandable details
- **Vulnerability Dashboard**: Severity distribution, filterable/sortable tables, expandable descriptions
- **Report Generation**: HTML, PDF, Excel, Markdown
- **Real-time Updates**: WebSocket push notifications
- **Team Collaboration**: RBAC with admin/analyst/viewer roles
- **Embedded React SPA**: Single binary, no separate frontend deploy

## Quick Start with Docker

```bash
cp .env.example .env
docker compose up -d
```

This starts MariaDB + the app. The UI is at http://localhost:8080.

### Register a user

```bash
curl -X POST http://localhost:8080/api/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"email":"admin@local.dev","username":"admin","password":"changeme123"}'
```

### Import an OpenVAS report

```bash
curl -X POST http://localhost:8080/api/import/openvas \
  -H 'X-API-Key: local-dev-import-key-min-32-characters' \
  -H 'Content-Type: application/xml' \
  --data-binary @testdata/openvas-sample-report.xml
```

Response includes ticket statistics:
```json
{
  "scan_id": "...",
  "vulnerabilities_imported": 10,
  "tickets_created": 10,
  "tickets_reopened": 0,
  "tickets_auto_resolved": 0
}
```

## Configuration

All config via environment variables with `OT_` prefix:

| Variable | Default | Purpose |
|----------|---------|---------|
| `OT_SERVER_PORT` | 8080 | HTTP listen port |
| `OT_DATABASE_DSN` | `...@tcp(localhost:3306)/openvas-tracker?parseTime=true` | MariaDB DSN |
| `OT_JWT_SECRET` | `change-me-in-production` | JWT signing key |
| `OT_IMPORT_APIKEY` | (empty) | API key for import webhook (min 32 chars) |

## Ticket Lifecycle

```
Import finds new vulnerability  →  Ticket created (open)
Import finds same vulnerability →  Ticket updated (last_seen_at)
Import missing old vulnerability → Ticket auto-fixed
Import re-finds fixed vuln     →  Ticket reopened (open)
User marks ticket               →  fixed / risk_accepted
```

All status changes are logged in the activity trail with actor (user ID or "Automatic").

## API

All endpoints under `/api/` require `Authorization: Bearer <token>` except auth and health.

Import endpoint uses `X-API-Key` header instead of JWT.

| Method | Path | Description |
|--------|------|-------------|
| POST | /api/auth/register | Register user |
| POST | /api/auth/login | Login, get JWT |
| POST | /api/import/openvas | Import OpenVAS XML (API-Key auth) |
| GET | /api/hosts | Aggregated host summaries |
| GET | /api/hosts/:host/vulnerabilities | Vulns for a specific host |
| GET | /api/scans | List scans (imports) |
| GET | /api/scans/:id | Scan detail |
| GET | /api/scans/:id/vulnerabilities | Vulns in a scan |
| GET | /api/vulnerabilities | List all vulnerabilities |
| GET | /api/tickets | List tickets |
| GET | /api/tickets/:id | Ticket detail |
| PATCH | /api/tickets/:id/status | Change ticket status |
| POST/GET | /api/tickets/:id/comments | Add/list notes |
| GET | /api/tickets/:id/activity | Ticket activity log |
| GET | /api/dashboard | Dashboard metrics + ticket stats |
| GET | /api/reports | List/generate reports |
| GET | /api/teams | List teams |
| GET | /api/health | Health check |
| WS | /ws?token= | Real-time updates |

## Tech Stack

- **Backend**: Go 1.26, Echo v4, MariaDB, golang-jwt, bcrypt
- **Frontend**: React 19, Vite, Tailwind CSS, TanStack Query, Recharts
- **Deploy**: Docker Compose (MariaDB + single Go binary with embedded SPA)

## License

GPL v3
