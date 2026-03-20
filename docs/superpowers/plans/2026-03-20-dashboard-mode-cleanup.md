# Dashboard-Mode Cleanup — Remove Active Scanning, Keep Import-Only

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Strip all active scan orchestration (nmap CLI, OpenVAS gvm-cli, plugin system, schedules) so the app is a pure vulnerability dashboard fed by the existing OpenVAS webhook import endpoint.

**Architecture:** The webhook import (`POST /api/import/openvas` with API-Key auth) and system-user logic are already implemented. This plan removes dead code: nmap parser/scanner, CVE enrichment, plugin system, and schedule feature. DB migrations and the `schedules` table stay untouched.

**Tech Stack:** Go 1.26, Echo v4, MariaDB, Docker Compose

---

## Chunk 1: Remove Dead Scanner & Plugin Code

### Task 1: Delete nmap scanner files

**Files:**
- Delete: `internal/scanner/nmap.go`
- Delete: `internal/scanner/nmap_test.go`

- [ ] **Step 1: Delete nmap.go and nmap_test.go**

```bash
rm internal/scanner/nmap.go internal/scanner/nmap_test.go
```

- [ ] **Step 2: Verify scanner package still compiles**

Run: `go build ./internal/scanner/`
Expected: SUCCESS (only openvas.go remains with ParseOpenVASXML)

- [ ] **Step 3: Commit**

```bash
git add -A internal/scanner/nmap.go internal/scanner/nmap_test.go
git commit -m "refactor: remove nmap scanner — app is import-only dashboard"
```

---

### Task 2: Delete CVE enrichment files

**Files:**
- Delete: `internal/scanner/enrichment.go`
- Delete: `internal/scanner/enrichment_test.go`

- [ ] **Step 1: Delete enrichment files**

```bash
rm internal/scanner/enrichment.go internal/scanner/enrichment_test.go
```

- [ ] **Step 2: Verify scanner package still compiles**

Run: `go build ./internal/scanner/`
Expected: SUCCESS

- [ ] **Step 3: Commit**

```bash
git add -A internal/scanner/enrichment.go internal/scanner/enrichment_test.go
git commit -m "refactor: remove CVE enrichment — not used by import workflow"
```

---

### Task 3: Delete plugin system

**Files:**
- Delete: `internal/plugin/interface.go`
- Delete: `internal/plugin/loader.go`

- [ ] **Step 1: Delete entire plugin directory**

```bash
rm -rf internal/plugin/
```

- [ ] **Step 2: Verify no references remain**

Run: `grep -r "internal/plugin" --include="*.go" .`
Expected: No output (plugin package was never imported in main.go or handlers)

- [ ] **Step 3: Verify full build**

Run: `go build ./...`
Expected: SUCCESS

- [ ] **Step 4: Commit**

```bash
git add -A internal/plugin/
git commit -m "refactor: remove plugin system — no active scanning"
```

---

## Chunk 2: Remove Schedule Feature

### Task 4: Remove schedule handler, service, and queries

**Files:**
- Delete: `internal/handler/schedules.go`
- Delete: `internal/service/schedule.go`
- Delete: `internal/database/queries/schedules.go`
- Modify: `cmd/openvas-tracker/main.go` (remove schedule wiring)

Note: Migration files `sql/migrations/009_create_schedules.{up,down}.sql` stay untouched to preserve migration history.

- [ ] **Step 1: Delete schedule handler**

```bash
rm internal/handler/schedules.go
```

- [ ] **Step 2: Delete schedule service**

```bash
rm internal/service/schedule.go
```

- [ ] **Step 3: Delete schedule queries**

```bash
rm internal/database/queries/schedules.go
```

- [ ] **Step 4: Remove schedule wiring from main.go**

In `cmd/openvas-tracker/main.go`, remove:
- Line with `scheduleSvc := service.NewScheduleService(db)`
- Line with `handler.NewScheduleHandler(scheduleSvc).RegisterRoutes(p.Group("/schedules"))`

After edit, `main.go` should no longer reference `scheduleSvc`, `ScheduleHandler`, or `ScheduleService`.

- [ ] **Step 5: Verify full build**

Run: `go build ./...`
Expected: SUCCESS

- [ ] **Step 6: Run all tests**

Run: `go test ./... -count=1`
Expected: All PASS

- [ ] **Step 7: Commit**

```bash
git add -A internal/handler/schedules.go internal/service/schedule.go internal/database/queries/schedules.go cmd/openvas-tracker/main.go
git commit -m "refactor: remove schedule feature — no active scan launching"
```

---

## Chunk 3: Cleanup & Docker Deploy

### Task 5: go mod tidy

- [ ] **Step 1: Run go mod tidy**

```bash
go mod tidy
```

- [ ] **Step 2: Verify build + tests still pass**

Run: `go build ./... && go test ./... -count=1`
Expected: SUCCESS

- [ ] **Step 3: Commit if go.mod/go.sum changed**

```bash
git add go.mod go.sum
git commit -m "chore: go mod tidy after removing scanner/plugin/schedule code"
```

---

### Task 6: Docker build & local deploy test

- [ ] **Step 1: Build Docker image**

Run: `docker compose build`
Expected: SUCCESS — multi-stage build completes

- [ ] **Step 2: Start services**

Run: `docker compose up -d`
Expected: Both `db` and `app` containers running. MariaDB healthy, app listening on :8080.

- [ ] **Step 3: Verify health endpoint**

Run: `curl http://localhost:8080/api/health`
Expected: `{"status":"ok"}`

- [ ] **Step 4: Test import endpoint rejects without API key**

Run: `curl -s -o /dev/null -w "%{http_code}" -X POST http://localhost:8080/api/import/openvas`
Expected: `401`

- [ ] **Step 5: Test import endpoint with valid API key + sample XML**

```bash
curl -X POST http://localhost:8080/api/import/openvas \
  -H "X-API-Key: local-dev-import-key-min-32-chars!!" \
  -H "Content-Type: application/xml" \
  -d '<?xml version="1.0"?><report><results><result><name>Test Vuln</name><host>10.0.0.1</host><port>443/tcp</port><threat>High</threat><severity>7.5</severity><description>Test</description><nvt oid="1.2.3"><name>Test</name><cvss_base>7.5</cvss_base><cve>CVE-2024-0001</cve><solution>Fix it</solution></nvt></result></results></report>'
```
Expected: `201` with JSON `{"scan_id":"...","vulnerabilities_imported":1}`

- [ ] **Step 6: Verify frontend loads**

Run: `curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/`
Expected: `200` (index.html served)

- [ ] **Step 7: Stop containers**

Run: `docker compose down`
