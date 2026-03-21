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

## Quick Start without Docker (Bare Metal)

### Prerequisites

- Go 1.23+
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

### Cross-compile for Linux

```bash
make build-linux    # produces bin/openvas-tracker-linux-amd64
```

## Configuration

All config via environment variables with `OT_` prefix:

| Variable | Default | Purpose |
|----------|---------|---------|
| `OT_SERVER_PORT` | 8080 | HTTP listen port |
| `OT_DATABASE_DSN` | `...@tcp(localhost:3306)/openvas-tracker?parseTime=true` | MariaDB DSN |
| `OT_JWT_SECRET` | `change-me-in-production` | JWT signing key |
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

If you prefer manual exports:

1. In GSA, go to **Scans → Reports**
2. Select a completed report
3. Click the download icon → choose **XML** format
4. Import the downloaded file using the curl command above

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
