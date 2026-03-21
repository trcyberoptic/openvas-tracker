# OpenVAS-Tracker

Vulnerability management dashboard that imports OpenVAS scan results and tracks remediation through automated ticketing.

## Features

- **OpenVAS Import**: Webhook endpoint receives scan results automatically when scans complete
- **Automatic Ticketing**: New findings create tickets, missing findings auto-resolve, recurring findings reopen
- **Ticket Lifecycle**: open → fixed / risk_accepted / false_positive, with full activity audit trail
- **Risk Acceptance with Expiry**: Risk-accepted tickets auto-reopen after expiry date
- **Host-centric View**: Aggregated vulnerability counts per host with expandable details
- **Dashboard**: Open ticket counts by priority, severity trend over time, ticket stats
- **Filterable & Sortable Tables**: All list views with column sorting and multi-filter support
- **Report Generation**: HTML, PDF, Excel, Markdown
- **Real-time Updates**: WebSocket push notifications
- **Embedded React SPA**: Single binary, no separate frontend deploy

## Architecture

```mermaid
sequenceDiagram
    participant OV as OpenVAS (GVM)
    participant TR as OpenVAS-Tracker
    participant DB as MariaDB
    participant UI as React Dashboard

    Note over OV: Scan completes
    OV->>TR: HTTP GET /api/import/openvas?api_key=...
    TR->>OV: GMP Socket: fetch latest report
    OV-->>TR: XML report
    TR->>DB: Begin transaction
    TR->>DB: Create scan record
    loop For each vulnerability
        TR->>DB: Create vulnerability
        alt New finding
            TR->>DB: Create ticket (open)
        else Known finding (ticket open)
            TR->>DB: Update last_seen_at
        else Recurring finding (ticket fixed)
            TR->>DB: Reopen ticket → open
        else False positive
            TR->>DB: Skip (never reopen)
        end
    end
    TR->>DB: Auto-fix tickets for missing findings
    TR->>DB: Reopen expired risk acceptances
    TR->>DB: Commit transaction
    TR->>UI: WebSocket: push update
    UI->>TR: GET /api/dashboard
    TR-->>UI: Priority counts, ticket stats, trend
```

```
┌──────────────┐     HTTP GET (alert)      ┌──────────────────┐
│   OpenVAS    │ ──────────────────────────▶│                  │
│   (GVM)      │◀── GMP socket (fetch) ────│  OpenVAS-Tracker │──▶ MariaDB
│              │         XML report         │   (Go + React)   │
└──────────────┘                            └──────────────────┘
                                                    │
                                              :8080 │
                                                    ▼
                                            ┌──────────────┐
                                            │   Browser     │
                                            │  Dashboard    │
                                            └──────────────┘
```

## Quick Start with Docker

```bash
cp .env.example .env    # edit: set OT_JWT_SECRET and OT_IMPORT_APIKEY
docker compose up -d
```

This starts MariaDB + the app. The UI is at http://localhost:8080.

> **Important:** `OT_JWT_SECRET` must be set to a random string of at least 32 characters. The app will refuse to start with the default value.

### Register a user

```bash
curl -X POST http://localhost:8080/api/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"email":"admin@local.dev","username":"admin","password":"changeme123"}'
```

### Import an OpenVAS report

```bash
curl -X POST http://localhost:8080/api/import/openvas \
  -H 'X-API-Key: <your-api-key-min-32-chars>' \
  -H 'Content-Type: application/xml' \
  --data-binary @scan-report.xml
```

Response:
```json
{
  "scan_id": "...",
  "vulnerabilities_imported": 10,
  "vulnerabilities_skipped": 2,
  "tickets_created": 10,
  "tickets_reopened": 0,
  "tickets_auto_resolved": 0
}
```

## Quick Start without Docker (Bare Metal)

### Prerequisites

- Go 1.24+
- Node.js 22+ and npm
- MariaDB 10.6+ (or MySQL 8+)
- [golang-migrate](https://github.com/golang-migrate/migrate) CLI (for migrations)

### 1. Set up MariaDB

```sql
CREATE DATABASE `openvas-tracker` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE USER 'otracker'@'localhost' IDENTIFIED BY 'your-db-password';
GRANT ALL PRIVILEGES ON `openvas-tracker`.* TO 'otracker'@'localhost';
FLUSH PRIVILEGES;
```

### 2. Run migrations

```bash
export DATABASE_URL="mysql://otracker:your-db-password@tcp(localhost:3306)/openvas-tracker"
make migrate-up
```

### 3. Configure environment

```bash
export OT_DATABASE_DSN="otracker:your-db-password@tcp(localhost:3306)/openvas-tracker?parseTime=true"
export OT_JWT_SECRET="$(openssl rand -hex 32)"
export OT_IMPORT_APIKEY="$(openssl rand -hex 32)"
```

Or create an `.env` file — `godotenv` reads it automatically.

### 4. Build and run

```bash
make build          # builds frontend + Go binary → bin/openvas-tracker
./bin/openvas-tracker
```

The UI is at http://localhost:8080.

### Development mode

```bash
make dev            # runs Go backend + Vite dev server with HMR
```

Frontend dev server on port 5173 proxies API calls to the backend on port 8080.

## Configuration

All config via environment variables with `OT_` prefix:

| Variable | Default | Purpose |
|----------|---------|---------|
| `OT_SERVER_PORT` | 8080 | HTTP listen port |
| `OT_DATABASE_DSN` | `...@tcp(localhost:3306)/openvas-tracker?parseTime=true` | MariaDB DSN |
| `OT_JWT_SECRET` | (none — **required**) | JWT signing key (min 32 chars) |
| `OT_IMPORT_APIKEY` | (empty) | API key for import webhook (min 32 chars) |

## OpenVAS Configuration

OpenVAS-Tracker does not control OpenVAS — it only receives scan results. You configure OpenVAS to push reports to the tracker via its built-in Alert system.

### Prerequisites

- A running OpenVAS/GVM installation (Greenbone Community Edition or Greenbone Enterprise)
- Network access from the OpenVAS host to the tracker's import endpoint

### Step 1: Set the API key

In your `.env` or environment config, set a strong API key (min 32 characters):

```
OT_IMPORT_APIKEY=your-secret-import-key-at-least-32-chars
```

### Step 2: Create an Alert in OpenVAS

1. Log in to the **Greenbone Security Assistant (GSA)** web UI
2. Go to **Configuration → Alerts → New Alert**
3. Configure:

| Field | Value |
|-------|-------|
| Name | `OpenVAS-Tracker Import` |
| Event | Task run status changed → Done |
| Method | HTTP Get |
| HTTP Get URL | `http://<tracker-host>:8080/api/import/openvas?api_key=<your-api-key>` |

When the alert fires, OpenVAS-Tracker receives the GET request, connects to GVM via the GMP Unix socket, fetches the latest completed report, and imports it automatically.

> **Note:** If GVM runs in Docker, use the Docker bridge IP (typically `172.17.0.1`) instead of `localhost` in the alert URL. The API key must be passed as a query parameter because GVM HTTP Get alerts cannot set custom headers.

### Step 3: Attach the Alert to a Scan Task

1. Go to **Scans → Tasks**
2. Edit your scan task (or create a new one)
3. Under **Alerts**, select the `OpenVAS-Tracker Import` alert
4. Save

Now every time this scan completes, the results are automatically imported.

### GMP Fetch Script

The GET endpoint triggers `/usr/local/bin/openvas-tracker-fetch-latest` which:
1. Connects to the GVM Manager via GMP Unix socket
2. Fetches the latest completed report in XML format
3. POSTs it to the tracker's import endpoint
4. Tracks the last imported report ID to avoid duplicates

### Manual Import

You can also import reports manually via curl:

```bash
curl -X POST http://localhost:8080/api/import/openvas \
  -H 'X-API-Key: your-secret-import-key-at-least-32-chars' \
  -H 'Content-Type: application/xml' \
  --data-binary @scan-report.xml
```

### Exporting from OpenVAS manually

1. In GSA, go to **Scans → Reports**
2. Select a completed report
3. Click the download icon → choose **XML** format
4. Import the downloaded file using the curl command above

## Ticket Lifecycle

```
Import finds new vulnerability     →  Ticket created (open)
Import finds same vulnerability    →  Ticket updated (last_seen_at)
Import missing old vulnerability   →  Ticket auto-fixed
Import re-finds fixed vuln        →  Ticket reopened (open)
Import re-finds false_positive     →  Skipped (never reopened)
Risk acceptance expires            →  Ticket auto-reopened
User marks ticket                  →  fixed / risk_accepted / false_positive
```

All status changes are logged in the activity trail with actor (user ID or "Automatic").

## API

All endpoints under `/api/` require `Authorization: Bearer <token>` except auth and health.

Import endpoint uses `X-API-Key` header (or `?api_key=` query param) instead of JWT.

| Method | Path | Description |
|--------|------|-------------|
| POST | /api/auth/register | Register user |
| POST | /api/auth/login | Login, get JWT (rate limited: 10/min/IP) |
| POST | /api/import/openvas | Import OpenVAS XML (API-Key auth) |
| GET | /api/import/openvas | Trigger GMP fetch (API-Key auth) |
| GET | /api/hosts | Aggregated host summaries |
| GET | /api/hosts/:host/vulnerabilities | Vulns for a specific host |
| GET | /api/scans | List scans (imports) |
| GET | /api/scans/:id | Scan detail |
| GET | /api/scans/:id/vulnerabilities | Vulns in a scan |
| GET | /api/vulnerabilities | List all vulnerabilities |
| GET | /api/tickets | List all tickets (shared board) |
| GET | /api/tickets/:id | Ticket detail |
| PATCH | /api/tickets/:id/status | Change status (open/fixed/risk_accepted/false_positive) |
| PATCH | /api/tickets/:id/assign | Assign to user |
| POST/GET | /api/tickets/:id/comments | Add/list notes |
| GET | /api/tickets/:id/activity | Ticket activity log |
| GET | /api/dashboard | Dashboard: priority counts + ticket stats |
| GET | /api/dashboard/trend | Vulnerability trend over time |
| GET | /api/settings/setup | Setup guide (masked API key) |
| GET | /api/settings/users | User list for assignment |
| GET | /api/health | Health check (includes DB ping) |
| WS | /ws?token= | Real-time updates (origin-validated) |

## Tech Stack

- **Backend**: Go 1.26, Echo v4, MariaDB, golang-jwt, bcrypt, godotenv
- **Frontend**: React 19, Vite, Tailwind CSS, TanStack Query, Recharts, Zustand
- **Deploy**: Docker Compose (MariaDB + single Go binary with embedded SPA)

## License

GPL v3
