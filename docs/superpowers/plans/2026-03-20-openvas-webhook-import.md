# OpenVAS Webhook Import Implementation Plan (v2 — Pure Dashboard)

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Convert OpenVAS-Tracker from a scan orchestrator into a pure vulnerability dashboard with webhook-based OpenVAS import. Remove all active scanning, asynq task queue, and Redis.

**Architecture:** New `POST /api/import/openvas` endpoint secured by API-Key middleware, using existing `ParseOpenVASXML()` parser. All scan orchestration code, worker infrastructure, and Redis dependency removed.

**Tech Stack:** Go, Echo v4, MariaDB (database/sql)

**Spec:** `docs/superpowers/specs/2026-03-20-openvas-webhook-import-design.md`

**Already completed:**
- Task 1 (API-Key middleware) — committed as `9c62de4`
- Task 2 (config: replaced `OpenVASPath` with `ImportAPIKey`) — committed as `d263aa3`, but needs further update to also remove `RedisConfig`, `NmapPath`, and rename `ScannerConfig` → `ImportConfig`

---

## File Map

| File | Action | Responsibility |
|---|---|---|
| `internal/middleware/apikey.go` | **Already created** | API-Key authentication middleware |
| `internal/handler/import.go` | **Create** | Import handler: XML parsing, system user, scan+vuln creation |
| `internal/config/config.go` | Modify | Remove `ScannerConfig`+`RedisConfig`, add `ImportConfig` with `APIKey` |
| `.env.example` | Modify | Remove `OT_SCANNER_*`, `OT_REDIS_*`, add `OT_IMPORT_APIKEY` |
| `internal/scanner/openvas.go` | Modify | Remove `OpenVASScanner`, keep parser |
| `internal/scanner/nmap.go` | Modify | Remove `NmapScanner`, keep parser + structs |
| `internal/worker/` | **Delete entire dir** | No more task queue |
| `internal/service/scan.go` | **Delete** | Depended on asynq |
| `internal/handler/scans.go` | Modify | Remove `Launch()`, keep `List()`/`Get()`, remove asynq dep |
| `internal/handler/schedules.go` | Modify | Remove `openvas` from validation |
| `cmd/openvas-tracker/main.go` | Modify | Remove asynq/worker/scanner, wire import handler |

---

## Chunk 1: Remove All Scanning Infrastructure

### Task 3: Remove scanner CLI wrappers (keep parsers)

**Files:**
- Modify: `internal/scanner/openvas.go` — delete `OpenVASScanner`, `NewOpenVASScanner()`, `Scan()`; remove unused imports `"context"`, `"os/exec"`, `"strings"`
- Modify: `internal/scanner/nmap.go` — delete `NmapScanner`, `NewNmapScanner()`, `Scan()`; remove unused imports `"context"`, `"os/exec"`, `"strings"`

- [ ] **Step 1: Edit `openvas.go` — remove lines 84-103 and unused imports**

After edit, file keeps: `package scanner`, imports `"encoding/xml"`, `"fmt"`, `"io"`, `"strconv"`, XML structs, `OpenVASResult`, `ParseOpenVASXML()`.

- [ ] **Step 2: Edit `nmap.go` — remove lines 127-143 and unused imports**

After edit, file keeps: `package scanner`, imports `"encoding/xml"`, `"fmt"`, `"io"`, XML structs, result types, `ParseNmapXML()`.

- [ ] **Step 3: Verify parser tests pass**

Run: `cd e:/Code/openvas-tracker && go test ./internal/scanner/ -v`
Expected: `TestParseOpenVASXML` and `TestParseNmapXML` both PASS

- [ ] **Step 4: Commit**

```bash
git add internal/scanner/openvas.go internal/scanner/nmap.go
git commit -m "refactor: remove scanner CLI wrappers, keep XML parsers"
```

### Task 4: Delete worker package and scan service

**Files:**
- Delete: `internal/worker/scan_task.go`
- Delete: `internal/worker/server.go`
- Delete: `internal/service/scan.go`

- [ ] **Step 1: Delete the files**

```bash
rm internal/worker/scan_task.go internal/worker/server.go internal/service/scan.go
rmdir internal/worker
```

- [ ] **Step 2: Commit**

```bash
git add -A internal/worker/ internal/service/scan.go
git commit -m "refactor: remove worker package and scan service (no more active scanning)"
```

### Task 5: Update config — remove Redis + ScannerConfig, add ImportConfig

**Files:**
- Modify: `internal/config/config.go`
- Modify: `internal/config/config_test.go` (if it references Redis or Scanner)
- Modify: `.env.example`

- [ ] **Step 1: Rewrite config structs**

In `internal/config/config.go`, replace the `Config` struct and remove `RedisConfig` and `ScannerConfig`:

Replace:
```go
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Redis    RedisConfig
	JWT      JWTConfig
	Scanner  ScannerConfig
}
```
With:
```go
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	JWT      JWTConfig
	Import   ImportConfig
}
```

Delete `RedisConfig` struct entirely. Replace `ScannerConfig` with:
```go
type ImportConfig struct {
	APIKey string
}
```

- [ ] **Step 2: Update `Load()` defaults**

Remove all `redis.*` and `scanner.*` defaults. Add:
```go
v.SetDefault("import.apikey", "")
```

- [ ] **Step 3: Update `.env.example`**

Remove:
```
OT_REDIS_ADDR=localhost:6379
OT_REDIS_PASSWORD=
OT_SCANNER_IMPORTAPIKEY=
```

Add:
```
OT_IMPORT_APIKEY=
```

- [ ] **Step 4: Update config tests if needed**

Run: `cd e:/Code/openvas-tracker && go test ./internal/config/ -v`
Fix any test referencing Redis or Scanner config.

- [ ] **Step 5: Commit**

```bash
git add internal/config/config.go internal/config/config_test.go .env.example
git commit -m "refactor: remove Redis and Scanner config, add Import config"
```

### Task 6: Simplify scans handler — remove Launch, remove asynq

**Files:**
- Modify: `internal/handler/scans.go` — remove `Launch()`, `launchScanRequest`, asynq imports; keep `List()`, `Get()`
- Modify: `internal/handler/schedules.go` — remove `openvas` from validation

- [ ] **Step 1: Rewrite `scans.go`**

Replace entire file with:
```go
package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/cyberoptic/openvas-tracker/internal/database/queries"
	"github.com/cyberoptic/openvas-tracker/internal/middleware"
)

type ScanHandler struct {
	q *queries.Queries
}

func NewScanHandler(q *queries.Queries) *ScanHandler {
	return &ScanHandler{q: q}
}

func (h *ScanHandler) List(c echo.Context) error {
	userID := middleware.GetUserID(c)
	scans, err := h.q.ListScans(c.Request().Context(), queries.ListScansParams{
		UserID: userID, Limit: 50, Offset: 0,
	})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to list scans")
	}
	return c.JSON(http.StatusOK, scans)
}

func (h *ScanHandler) Get(c echo.Context) error {
	id := c.Param("id")
	scan, err := h.q.GetScan(c.Request().Context(), id)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "scan not found")
	}
	return c.JSON(http.StatusOK, scan)
}

func (h *ScanHandler) RegisterRoutes(g *echo.Group) {
	g.GET("", h.List)
	g.GET("/:id", h.Get)
}
```

- [ ] **Step 2: Update `schedules.go` validation**

Replace `oneof=nmap openvas` with `oneof=nmap` in `createScheduleRequest.ScanType`.

- [ ] **Step 3: Verify handler package compiles**

Run: `cd e:/Code/openvas-tracker && go build ./internal/handler/`
Expected: success

- [ ] **Step 4: Commit**

```bash
git add internal/handler/scans.go internal/handler/schedules.go
git commit -m "refactor: simplify scans handler to read-only, remove asynq dependency"
```

### Task 7: Update main.go — remove asynq, Redis, scanner, worker

**Files:**
- Modify: `cmd/openvas-tracker/main.go`

- [ ] **Step 1: Remove imports**

Remove these imports:
- `"github.com/hibiken/asynq"`
- `"github.com/cyberoptic/openvas-tracker/internal/scanner"`
- `"github.com/cyberoptic/openvas-tracker/internal/worker"`

- [ ] **Step 2: Remove asynq client block (lines ~49-53)**

Delete:
```go
	// Asynq client (enqueue jobs)
	asynqClient := asynq.NewClient(asynq.RedisClientOpt{
		Addr: cfg.Redis.Addr, Password: cfg.Redis.Password, DB: cfg.Redis.DB,
	})
	defer asynqClient.Close()
```

- [ ] **Step 3: Update ScanHandler construction (line ~99)**

Replace:
```go
	handler.NewScanHandler(q, asynqClient).RegisterRoutes(p.Group("/scans"))
```
With:
```go
	handler.NewScanHandler(q).RegisterRoutes(p.Group("/scans"))
```

- [ ] **Step 4: Remove worker startup block (lines ~116-125)**

Delete:
```go
	// Start Asynq worker in background
	nmapScanner := scanner.NewNmapScanner(cfg.Scanner.NmapPath)
	openvasScanner := scanner.NewOpenVASScanner(cfg.Scanner.OpenVASPath)
	workerSrv := worker.NewServer(cfg, db)
	workerMux := worker.NewMux(db, nmapScanner, openvasScanner)
	go func() {
		if err := workerSrv.Run(workerMux); err != nil {
			log.Printf("worker error: %v", err)
		}
	}()
```

- [ ] **Step 5: Remove worker shutdown (line ~144)**

Delete:
```go
	workerSrv.Shutdown()
```

- [ ] **Step 6: Verify build**

Run: `cd e:/Code/openvas-tracker && go build ./cmd/openvas-tracker/`
Expected: success

- [ ] **Step 7: Run all tests**

Run: `cd e:/Code/openvas-tracker && go test ./...`
Expected: all pass

- [ ] **Step 8: Commit**

```bash
git add cmd/openvas-tracker/main.go
git commit -m "refactor: remove asynq, Redis, scanner, and worker from main"
```

---

## Chunk 2: Import Handler + Wiring

### Task 8: Create the import handler

**Files:**
- Create: `internal/handler/import.go`
- Create: `internal/handler/import_test.go`

- [ ] **Step 1: Write the import handler test**

```go
// internal/handler/import_test.go
package handler

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
)

func TestMapSeverity(t *testing.T) {
	tests := []struct {
		threat string
		cvss   float64
		want   string
	}{
		{"High", 9.5, "critical"},
		{"High", 7.5, "high"},
		{"Medium", 5.0, "medium"},
		{"Low", 2.0, "low"},
		{"Log", 0.0, "info"},
		{"Debug", 0.0, "info"},
		{"", 0.0, "info"},
	}
	for _, tt := range tests {
		got := mapSeverity(tt.threat, tt.cvss)
		if got != tt.want {
			t.Errorf("mapSeverity(%q, %.1f) = %q, want %q", tt.threat, tt.cvss, got, tt.want)
		}
	}
}

func TestParsePort(t *testing.T) {
	tests := []struct {
		input     string
		wantPort  *int32
		wantProto *string
	}{
		{"443/tcp", int32Ptr(443), strPtr("tcp")},
		{"80/tcp", int32Ptr(80), strPtr("tcp")},
		{"general/tcp", nil, nil},
		{"", nil, nil},
	}
	for _, tt := range tests {
		port, proto := parsePort(tt.input)
		if !int32PtrEq(port, tt.wantPort) || !strPtrEq(proto, tt.wantProto) {
			t.Errorf("parsePort(%q) = (%v, %v), want (%v, %v)", tt.input, derefInt32(port), derefStr(proto), derefInt32(tt.wantPort), derefStr(tt.wantProto))
		}
	}
}

func TestHandleOpenVAS_BadXML(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/import/openvas", strings.NewReader("not xml"))
	req.Header.Set("Content-Type", "application/xml")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	h := &ImportHandler{}
	err := h.HandleOpenVAS(c)
	he, ok := err.(*echo.HTTPError)
	if !ok || he.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %v", err)
	}
}

func int32Ptr(v int32) *int32     { return &v }
func strPtr(v string) *string     { return &v }
func derefInt32(p *int32) string  { if p == nil { return "<nil>" }; return fmt.Sprintf("%d", *p) }
func derefStr(p *string) string   { if p == nil { return "<nil>" }; return *p }
func int32PtrEq(a, b *int32) bool { if a == nil && b == nil { return true }; if a == nil || b == nil { return false }; return *a == *b }
func strPtrEq(a, b *string) bool  { if a == nil && b == nil { return true }; if a == nil || b == nil { return false }; return *a == *b }
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd e:/Code/openvas-tracker && go test ./internal/handler/ -run "TestMapSeverity|TestParsePort|TestHandleOpenVAS_BadXML" -v`
Expected: compilation error — `mapSeverity`, `parsePort`, `ImportHandler` not defined

- [ ] **Step 3: Implement the import handler**

```go
// internal/handler/import.go
package handler

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/cyberoptic/openvas-tracker/internal/auth"
	"github.com/cyberoptic/openvas-tracker/internal/database/queries"
	"github.com/cyberoptic/openvas-tracker/internal/scanner"
)

type ImportHandler struct {
	q            *queries.Queries
	systemUserID string
	once         sync.Once
	onceErr      error
}

func NewImportHandler(db *sql.DB) *ImportHandler {
	return &ImportHandler{q: queries.New(db)}
}

func (h *ImportHandler) HandleOpenVAS(c echo.Context) error {
	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "failed to read request body")
	}

	results, err := scanner.ParseOpenVASXML(strings.NewReader(string(body)))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "failed to parse OpenVAS XML")
	}
	if len(results) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "empty report — no results found")
	}

	if err := h.resolveSystemUser(c.Request().Context()); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to resolve system user")
	}

	scanID := uuid.New().String()
	now := time.Now()
	scan, err := h.q.CreateScan(c.Request().Context(), queries.CreateScanParams{
		ID:       scanID,
		Name:     fmt.Sprintf("OpenVAS Import %s", now.Format("2006-01-02 15:04:05")),
		ScanType: queries.ScanTypeOpenvas,
		Status:   queries.ScanStatusCompleted,
		UserID:   h.systemUserID,
	})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create scan record")
	}

	rawXML := string(body)
	h.q.UpdateScanStatus(c.Request().Context(), queries.UpdateScanStatusParams{
		ID:          scan.ID,
		Status:      queries.ScanStatusCompleted,
		StartedAt:   &now,
		CompletedAt: &now,
		RawOutput:   &rawXML,
	})

	imported := 0
	for _, r := range results {
		port, proto := parsePort(r.Port)
		severity := mapSeverity(r.Severity, r.CVSSScore)

		var cveID *string
		if r.CVE != "" && r.CVE != "NOCVE" {
			cveID = &r.CVE
		}
		var desc *string
		if r.Description != "" {
			desc = &r.Description
		}
		var sol *string
		if r.Solution != "" {
			sol = &r.Solution
		}
		var host *string
		if r.Host != "" {
			host = &r.Host
		}
		var cvss *float64
		if r.CVSSScore > 0 {
			cvss = &r.CVSSScore
		}

		_, err := h.q.CreateVulnerability(c.Request().Context(), queries.CreateVulnerabilityParams{
			ID:             uuid.New().String(),
			ScanID:         scan.ID,
			UserID:         h.systemUserID,
			Title:          r.Title,
			Description:    desc,
			Severity:       queries.SeverityLevel(severity),
			CvssScore:      cvss,
			CveID:          cveID,
			AffectedHost:   host,
			AffectedPort:   port,
			Protocol:       proto,
			Solution:       sol,
			VulnReferences: []byte("[]"),
		})
		if err != nil {
			continue
		}
		imported++
	}

	return c.JSON(http.StatusCreated, map[string]interface{}{
		"scan_id":                  scan.ID,
		"vulnerabilities_imported": imported,
	})
}

func (h *ImportHandler) resolveSystemUser(ctx context.Context) error {
	h.once.Do(func() {
		user, err := h.q.GetUserByUsername(ctx, "openvas-import")
		if err == nil {
			h.systemUserID = user.ID
			return
		}
		randBytes := make([]byte, 32)
		rand.Read(randBytes)
		password := hex.EncodeToString(randBytes)
		hash, err := auth.HashPassword(password)
		if err != nil {
			h.onceErr = fmt.Errorf("failed to hash password: %w", err)
			return
		}
		user, err = h.q.CreateUser(ctx, queries.CreateUserParams{
			ID:       uuid.New().String(),
			Email:    "openvas-import@system.local",
			Username: "openvas-import",
			Password: hash,
			Role:     queries.UserRoleViewer,
		})
		if err != nil {
			user, err = h.q.GetUserByUsername(ctx, "openvas-import")
			if err != nil {
				h.onceErr = fmt.Errorf("failed to resolve system user: %w", err)
				return
			}
		}
		h.systemUserID = user.ID
	})
	return h.onceErr
}

func (h *ImportHandler) RegisterRoutes(g *echo.Group) {
	g.POST("/openvas", h.HandleOpenVAS)
}

func mapSeverity(threat string, cvss float64) string {
	switch strings.ToLower(threat) {
	case "high":
		if cvss >= 9.0 {
			return "critical"
		}
		return "high"
	case "medium":
		return "medium"
	case "low":
		return "low"
	default:
		return "info"
	}
}

func parsePort(portStr string) (*int32, *string) {
	if portStr == "" {
		return nil, nil
	}
	parts := strings.SplitN(portStr, "/", 2)
	if len(parts) != 2 {
		return nil, nil
	}
	num, err := strconv.ParseInt(parts[0], 10, 32)
	if err != nil {
		return nil, nil
	}
	port := int32(num)
	proto := parts[1]
	return &port, &proto
}
```

- [ ] **Step 4: Run unit tests**

Run: `cd e:/Code/openvas-tracker && go test ./internal/handler/ -run "TestMapSeverity|TestParsePort|TestHandleOpenVAS_BadXML" -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/handler/import.go internal/handler/import_test.go
git commit -m "feat: add OpenVAS webhook import handler"
```

### Task 9: Wire import handler into main.go

**Files:**
- Modify: `cmd/openvas-tracker/main.go`

- [ ] **Step 1: Add import handler registration**

After the WebSocket block and before `serveFrontend(e)`, add:

```go
	// OpenVAS import webhook (API-Key auth, outside JWT group)
	if cfg.Import.APIKey != "" {
		if len(cfg.Import.APIKey) < 32 {
			log.Fatal("OT_IMPORT_APIKEY must be at least 32 characters")
		}
		importG := e.Group("/api/import", mw.APIKeyAuth(cfg.Import.APIKey), echomw.BodyLimit("10M"))
		handler.NewImportHandler(db).RegisterRoutes(importG)
	}
```

- [ ] **Step 2: Verify full build**

Run: `cd e:/Code/openvas-tracker && go build ./cmd/openvas-tracker/`
Expected: success

- [ ] **Step 3: Run all tests**

Run: `cd e:/Code/openvas-tracker && go test ./...`
Expected: all pass

- [ ] **Step 4: Commit**

```bash
git add cmd/openvas-tracker/main.go
git commit -m "feat: wire OpenVAS import webhook endpoint into main"
```

---

## Chunk 3: Cleanup + Verification

### Task 10: Remove asynq dependency from go.mod

- [ ] **Step 1: Run go mod tidy**

```bash
cd e:/Code/openvas-tracker && go mod tidy
```

- [ ] **Step 2: Verify build and tests**

```bash
cd e:/Code/openvas-tracker && go build ./cmd/openvas-tracker/ && go test ./...
```

- [ ] **Step 3: Commit**

```bash
git add go.mod go.sum
git commit -m "chore: remove asynq/redis dependency via go mod tidy"
```

### Task 11: End-to-end verification

- [ ] **Step 1: Run full test suite**

Run: `cd e:/Code/openvas-tracker && go test ./... -v`
Expected: all tests pass

- [ ] **Step 2: Verify build produces binary**

Run: `cd e:/Code/openvas-tracker && go build -o openvas-tracker.exe ./cmd/openvas-tracker/`
Expected: binary created without errors

- [ ] **Step 3: Verify no references to removed code**

Run: `cd e:/Code/openvas-tracker && grep -rn "NewOpenVASScanner\|NmapScanner\|OpenVASScanner\|TaskScan\|HandleOpenVASScan\|HandleNmapScan\|OpenVASPath\|openvaspath\|asynq\|RedisConfig\|ScannerConfig" --include="*.go" .`
Expected: no matches in source files

- [ ] **Step 4: Final commit if any cleanup needed**
