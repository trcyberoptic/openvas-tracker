# OpenVAS Webhook Import Design

## Summary

Convert OpenVAS-Tracker from a scan orchestrator into a pure vulnerability dashboard. Remove all active scanning (OpenVAS gvm-cli, Nmap CLI), the asynq task queue, and the Redis dependency. Add a webhook-based import endpoint so OpenVAS can push completed scan reports via HTTP Alert. OpenVAS-Tracker parses the XML, creates a scan record, and stores vulnerabilities.

## Motivation

OpenVAS-Tracker does not need to control scanners directly. It should act purely as a results dashboard — receiving, storing, and displaying vulnerability data. The existing scanner integrations (`OpenVASScanner.Scan()`, `NmapScanner.Scan()`) are simplified stubs. Instead, OpenVAS pushes results automatically via its built-in Alert mechanism.

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

**Config:** New field `ImportAPIKey string` in a new `ImportConfig` struct, replacing the entire `ScannerConfig`.

- Config key: `import.apikey` — must be explicitly registered via `v.SetDefault("import.apikey", "")` in `config.go`'s `Load()` function. All `scanner.*` defaults are removed.
- Env variable: `OT_IMPORT_APIKEY`
- No default value — must be explicitly configured
- **Minimum requirement:** key must be at least 32 characters. The app rejects startup if configured key is shorter.
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

**Note on existing data:** The `ScanTypeOpenvas` constant and DB enum value `"openvas"` remain unchanged. Existing scan rows from the old orchestrator are unaffected. No DB migration is needed.

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

All active scanning infrastructure is removed. The app becomes a pure dashboard + import receiver.

| File/Dir | What is removed | What stays |
|---|---|---|
| `internal/scanner/openvas.go` | `OpenVASScanner` struct, `NewOpenVASScanner()`, `Scan()` | `ParseOpenVASXML()`, all XML structs, `OpenVASResult` |
| `internal/scanner/openvas_test.go` | Nothing removed | Tests for `ParseOpenVASXML()` (kept as-is) |
| `internal/scanner/nmap.go` | `NmapScanner` struct, `NewNmapScanner()`, `Scan()` | `ParseNmapXML()`, all XML/result structs (kept for potential future nmap import) |
| `internal/scanner/nmap_test.go` | Nothing removed | Tests for `ParseNmapXML()` (kept as-is) |
| `internal/worker/` | **Entire directory deleted** | Nothing — no more task queue |
| `internal/service/scan.go` | **Entire file deleted** — depends on asynq + worker | Nothing |
| `internal/handler/scans.go` | `Launch()` (POST), asynq dependency, `launchScanRequest` | `List()`, `Get()`, simplified `ScanHandler` using `queries.Queries` only |
| `internal/handler/schedules.go` | `scan_type` `oneof` validation narrowed | Schedules kept but `openvas` removed from validation |
| `internal/config/config.go` | `ScannerConfig` struct, `RedisConfig` struct, all `scanner.*` and `redis.*` defaults | New `ImportConfig` struct with `APIKey` field |
| `cmd/openvas-tracker/main.go` | `asynq` client, `scanner` import, `worker` import, `openvasScanner`/`nmapScanner`, worker startup/shutdown, Redis config usage | Import handler wiring, all other handlers |
| `.env.example` | `OT_SCANNER_*`, `OT_REDIS_*` lines | New `OT_IMPORT_APIKEY=` line |
| `go.mod` / `go.sum` | `github.com/hibiken/asynq` dependency | Everything else |

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
