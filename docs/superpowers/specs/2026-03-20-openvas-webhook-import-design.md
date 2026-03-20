# OpenVAS Webhook Import Design

## Summary

Replace the OpenVAS scan orchestration (gvm-cli execution, async task queue) with a passive webhook-based import endpoint. OpenVAS pushes completed scan reports to VulnTrack via an HTTP Alert. VulnTrack parses the XML, creates a scan record, and stores vulnerabilities. Nmap scanning remains unchanged.

## Motivation

VulnTrack does not need to control OpenVAS directly. The existing `OpenVASScanner.Scan()` is a simplified stub that only calls `gvm-cli` for report retrieval. Instead, OpenVAS should push results automatically via its built-in Alert mechanism, keeping VulnTrack as a results tracker rather than a scan orchestrator.

## Design

### 1. New Import Endpoint

**Route:** `POST /api/import/openvas`

- Outside the JWT-protected `/api` group
- Secured via API-Key middleware (see section 2)
- Accepts `Content-Type: application/xml` with GMP report XML in the body

**Flow:**

1. Validate API-Key from `X-API-Key` header
2. Parse XML body using existing `scanner.ParseOpenVASXML()`
3. Resolve or create the system user `openvas-import` (see section 3)
4. Create a scan record: type `openvas`, status `completed`, timestamp = now
5. For each parsed result, create a vulnerability record (see section 5 for mapping)
6. Respond `201 Created` with `{ "scan_id": "<uuid>", "vulnerabilities_imported": <count> }`

**Error responses:**

- `401 Unauthorized` — missing or invalid API-Key
- `400 Bad Request` — malformed XML or empty report
- `500 Internal Server Error` — database failure

### 2. API-Key Authentication

**Config:** New field `ImportAPIKey string` in `ScannerConfig`, replacing `OpenVASPath`.

- Env variable: `VT_SCANNER_IMPORTAPIKEY`
- No default value — must be explicitly configured

**Middleware:** New function `APIKeyAuth(key string) echo.MiddlewareFunc` in `internal/middleware/`.

- Reads `X-API-Key` header
- Compares against configured key using `subtle.ConstantTimeCompare` (timing-safe)
- Returns `401 Unauthorized` on mismatch or missing header
- If no API key is configured (empty string), the endpoint returns `503 Service Unavailable`

### 3. System User

A dedicated user account `openvas-import` owns all imported scans and vulnerabilities.

- On first import, the handler checks if a user with username `openvas-import` exists
- If not, it creates one with a random password (the account is never used for login)
- The user ID is cached in-memory after first lookup to avoid repeated DB queries
- All imported vulnerabilities and scan records use this user's ID

### 4. Automatic Scan Record Creation

Each import creates one scan record in the `scans` table:

| Field | Value |
|---|---|
| `name` | `OpenVAS Import YYYY-MM-DD HH:MM:SS` |
| `scan_type` | `openvas` |
| `status` | `completed` |
| `started_at` | now |
| `completed_at` | now |
| `user_id` | system user ID |
| `raw_output` | the original XML (stored for traceability) |

### 5. Severity Mapping & Field Mapping

**Severity:**

| OpenVAS Threat | CVSS Range | VulnTrack SeverityLevel |
|---|---|---|
| `High` | >= 9.0 | `critical` |
| `High` | < 9.0 | `high` |
| `Medium` | any | `medium` |
| `Low` | any | `low` |
| `Log` / `Debug` / empty | any | `info` |

**Fields per Vulnerability:**

| VulnTrack Field | Source |
|---|---|
| `title` | `ovasResult.Name` |
| `description` | `ovasResult.Description` |
| `affected_host` | `ovasResult.Host` |
| `affected_port` | Port number parsed from `ovasResult.Port` (e.g. `443` from `443/tcp`) |
| `protocol` | Protocol parsed from `ovasResult.Port` (e.g. `tcp` from `443/tcp`) |
| `cvss_score` | `ovasResult.Severity` or `NVT.CVSSBase` |
| `cve_id` | `NVT.CVE` (null if `NOCVE`) |
| `solution` | `NVT.Solution` |
| `severity` | Mapped from Threat + CVSS (see table above) |
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
| `cmd/vulntrack/main.go` | `openvasScanner` instantiation, passing to `NewMux` | Everything else |
| `internal/handler/scans.go` | `oneof=nmap openvas` validation | Changed to `oneof=nmap` |
| `internal/handler/schedules.go` | `oneof=nmap openvas` validation | Changed to `oneof=nmap` |

### 7. New Files

| File | Purpose |
|---|---|
| `internal/handler/import.go` | `ImportHandler` with `HandleOpenVAS()` method, route registration |
| `internal/middleware/apikey.go` | `APIKeyAuth()` middleware function |

### 8. OpenVAS Configuration (external)

On the OpenVAS/GVM side, the user configures an Alert:

- **Type:** HTTP Get (or a custom script using `curl`)
- **URL:** `http://<vulntrack-host>:8080/api/import/openvas`
- **Header:** `X-API-Key: <configured-secret>`
- **Condition:** Task completed
- **Method:** Send full XML report in POST body

This is documented but not implemented by VulnTrack.
