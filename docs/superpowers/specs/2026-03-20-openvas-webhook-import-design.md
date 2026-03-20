# OpenVAS Webhook Import Design

## Summary

Replace the OpenVAS scan orchestration (gvm-cli execution, async task queue) with a passive webhook-based import endpoint. OpenVAS pushes completed scan reports to OpenVAS-Tracker via an HTTP Alert. OpenVAS-Tracker parses the XML, creates a scan record, and stores vulnerabilities. Nmap scanning remains unchanged.

## Motivation

OpenVAS-Tracker does not need to control OpenVAS directly. The existing `OpenVASScanner.Scan()` is a simplified stub that only calls `gvm-cli` for report retrieval. Instead, OpenVAS should push results automatically via its built-in Alert mechanism, keeping OpenVAS-Tracker as a results tracker rather than a scan orchestrator.

## Design

### 1. New Import Endpoint

**Route:** `POST /api/import/openvas`

- Registered on the root Echo instance (`e.POST(...)`) — NOT on the JWT-protected `p` group
- Must be registered before `serveFrontend(e)` to avoid being caught by the SPA fallback
- Secured via API-Key middleware (see section 2)
- Accepts `Content-Type: application/xml` with GMP report XML in the body
- **Body size limit:** 10 MB max via `echomw.BodyLimit("10M")` on the import group to prevent memory exhaustion from oversized reports

**Flow:**

1. Validate API-Key from `X-API-Key` header
2. Read and buffer the request body (needed for both parsing and raw storage)
3. Parse XML body using existing `scanner.ParseOpenVASXML()`
4. Resolve or create the system user `openvas-import` (see section 3)
5. Create a scan record via `CreateScan()`, then immediately call `UpdateScanStatus()` to store the raw XML in `raw_output` (see section 4)
6. For each parsed result, split port string (e.g. `"443/tcp"`) into port number and protocol, map severity, and create a vulnerability record (see section 5)
7. Respond `201 Created` with `{ "scan_id": "<id>", "vulnerabilities_imported": <count> }`

**Error responses:**

- `401 Unauthorized` — missing or invalid API-Key
- `400 Bad Request` — malformed XML or empty report
- `413 Request Entity Too Large` — body exceeds 10 MB
- `500 Internal Server Error` — database failure

### 2. API-Key Authentication

**Config:** New field `ImportAPIKey string` in `ScannerConfig`, replacing `OpenVASPath`.

- Config key: `scanner.importapikey` — must be explicitly registered via `v.SetDefault("scanner.importapikey", "")` in `config.go`'s `Load()` function. The old `v.SetDefault("scanner.openvaspath", "gvm-cli")` line is removed at the same time.
- Env variable: `OT_SCANNER_IMPORTAPIKEY`
- No default value — must be explicitly configured
- **Minimum requirement:** key must be at least 32 characters. The middleware rejects startup if configured key is shorter.
- **The key must not be logged at startup or in error messages.**

**Middleware:** New function `APIKeyAuth(key string) echo.MiddlewareFunc` in `internal/middleware/apikey.go`.

- Reads `X-API-Key` header
- Compares against configured key using `subtle.ConstantTimeCompare` (timing-safe)
- Returns `401 Unauthorized` on mismatch or missing header
- If no API key is configured (empty string), the endpoint returns `503 Service Unavailable`

### 3. System User

A dedicated user account `openvas-import` owns all imported scans and vulnerabilities.

- On first import, the handler checks if a user with username `openvas-import` exists
- If not, it creates one with:
  - Username: `openvas-import`
  - Email: `openvas-import@system.local`
  - Password: random 64-character string (the account is never used for login)
- **Concurrency:** Uses `sync.Once` to ensure the system user is resolved/created exactly once, even under concurrent import requests. Inside the `sync.Once` function: first attempt a lookup by username; if not found, create the user; if creation fails with a duplicate key error (e.g. from a previous application run), retry the lookup. After `sync.Once` completes, the cached user ID is guaranteed to be set or the handler returns an error.
- The user ID is cached in the `ImportHandler` struct after first resolution
- All imported vulnerabilities and scan records use this user's ID

### 4. Automatic Scan Record Creation

Each import creates one scan record in the `scans` table. Because `CreateScanParams` does not include a `raw_output` field, this is a two-step process:

1. **Step 1:** `CreateScan()` with the fields below
2. **Step 2:** `UpdateScanStatus()` to set `raw_output` to the original XML string

| Field | Value | Step |
|---|---|---|
| `name` | `OpenVAS Import YYYY-MM-DD HH:MM:SS` | CreateScan |
| `scan_type` | `openvas` | CreateScan |
| `status` | `completed` | CreateScan |
| `user_id` | system user ID | CreateScan |
| `started_at` | now | UpdateScanStatus |
| `completed_at` | now | UpdateScanStatus |
| `raw_output` | the original XML | UpdateScanStatus |

**Note on existing data:** The `ScanTypeOpenvas` constant and DB enum value `"openvas"` remain unchanged. Existing scan rows with `scan_type = 'openvas'` from the old orchestrator are unaffected. No DB migration is needed.

**Note on COALESCE:** The two-step approach works because `CreateScan` leaves `started_at` and `completed_at` as NULL, and `UpdateScanStatus` uses `COALESCE(?, started_at)` which sets the value when it is NULL. If the `CreateScan` query is ever changed to set default timestamps, this flow would need to be revisited.

### 5. Severity Mapping & Field Mapping

**Severity:** Based on `OpenVASResult.Severity` (string, from `Threat` field) and `OpenVASResult.CVSSScore` (float64):

| OpenVASResult.Severity (string) | OpenVASResult.CVSSScore | OpenVAS-Tracker SeverityLevel |
|---|---|---|
| `"High"` | >= 9.0 | `critical` |
| `"High"` | < 9.0 | `high` |
| `"Medium"` | any | `medium` |
| `"Low"` | any | `low` |
| `"Log"` / `"Debug"` / empty | any | `info` |

**Port parsing:** `ParseOpenVASXML()` returns the raw port string (e.g. `"443/tcp"`) in `OpenVASResult.Port`. The import handler splits this via `strings.SplitN(port, "/", 2)` to extract port number and protocol. If the format is unexpected, port is set to `nil` and protocol to `nil`.

**Fields per Vulnerability:**

| OpenVAS-Tracker Field | Source |
|---|---|
| `title` | `OpenVASResult.Title` |
| `description` | `OpenVASResult.Description` |
| `affected_host` | `OpenVASResult.Host` |
| `affected_port` | Port number from splitting `OpenVASResult.Port` (parsed to int32) |
| `protocol` | Protocol from splitting `OpenVASResult.Port` |
| `cvss_score` | `OpenVASResult.CVSSScore` (float64, already resolved from Severity or NVT.CVSSBase by the parser) |
| `cve_id` | `OpenVASResult.CVE` (set to nil if value is `"NOCVE"`) |
| `solution` | `OpenVASResult.Solution` |
| `severity` | Mapped from `OpenVASResult.Severity` + `OpenVASResult.CVSSScore` (see table above) |
| `status` | `open` |
| `scan_id` | ID of the auto-created scan record |
| `user_id` | System user ID |

### 6. Code Removal

The following OpenVAS orchestration code is removed:

| File | What | Stays |
|---|---|---|
| `internal/scanner/openvas.go` | `OpenVASScanner` struct, `NewOpenVASScanner()`, `Scan()` | `ParseOpenVASXML()`, all XML structs, `OpenVASResult` |
| `internal/scanner/openvas_test.go` | Tests for `Scan()` | Tests for `ParseOpenVASXML()` |
| `internal/worker/scan_task.go` | `HandleOpenVASScan()` | `HandleNmapScan()`, `ScanPayload`, `NewScanTask()` |
| `internal/worker/server.go` | `TaskScanOpenVAS` constant, mux registration | `TaskScanNmap`, nmap mux registration |
| `internal/config/config.go` | `ScannerConfig.OpenVASPath`, default `gvm-cli` | `ScannerConfig.NmapPath` |
| `cmd/OpenVAS-Tracker/main.go` | `openvasScanner` instantiation, passing to `NewMux` | Everything else |
| `internal/handler/scans.go` | `oneof=nmap openvas` validation, `TaskScanOpenVAS` branch in `Launch()` | Changed to `oneof=nmap`, OpenVAS task-type selection removed |
| `internal/handler/schedules.go` | `oneof=nmap openvas` validation | Changed to `oneof=nmap` |

### 7. New Files

| File | Purpose |
|---|---|
| `internal/handler/import.go` | `ImportHandler` with `HandleOpenVAS()` method, route registration |
| `internal/middleware/apikey.go` | `APIKeyAuth()` middleware function |

### 8. OpenVAS Configuration (external)

On the OpenVAS/GVM side, the user configures an Alert:

- **Type:** HTTP Get (or a custom script using `curl`)
- **URL:** `http://<OpenVAS-Tracker-host>:8080/api/import/openvas`
- **Header:** `X-API-Key: <configured-secret>`
- **Condition:** Task completed
- **Method:** Send full XML report in POST body

This is documented but not implemented by OpenVAS-Tracker.
