# OpenVAS Webhook Import Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace OpenVAS scan orchestration with a webhook-based import endpoint that receives GMP XML reports and stores vulnerabilities.

**Architecture:** New `POST /api/import/openvas` endpoint secured by API-Key middleware, using existing `ParseOpenVASXML()` parser. Removes all gvm-cli orchestration code. System user `openvas-import` owns all imported records.

**Tech Stack:** Go, Echo v4, MariaDB (database/sql), asynq (nmap worker only)

**Spec:** `docs/superpowers/specs/2026-03-20-openvas-webhook-import-design.md`

---

## File Map

| File | Action | Responsibility |
|---|---|---|
| `internal/middleware/apikey.go` | **Create** | API-Key authentication middleware |
| `internal/handler/import.go` | **Create** | Import handler: XML parsing, system user, scan+vuln creation |
| `internal/config/config.go` | Modify | Replace `OpenVASPath` with `ImportAPIKey` in `ScannerConfig` |
| `.env.example` | Modify | Replace `OT_SCANNER_OPENVASPATH` with `OT_SCANNER_IMPORTAPIKEY` |
| `internal/scanner/openvas.go` | Modify | Remove `OpenVASScanner` struct, `NewOpenVASScanner()`, `Scan()` — keep parser |
| `internal/scanner/openvas_test.go` | Keep as-is | Only tests `ParseOpenVASXML()` |
| `internal/worker/scan_task.go` | Modify | Remove `HandleOpenVASScan()`, remove `openvas` field from `ScanHandler` |
| `internal/worker/server.go` | Modify | Remove `TaskScanOpenVAS` constant, remove OpenVAS mux registration, remove `openvasScanner` param |
| `internal/handler/scans.go` | Modify | Remove `openvas` from validation, remove OpenVAS task-type branch |
| `internal/handler/schedules.go` | Modify | Remove `openvas` from validation |
| `cmd/openvas-tracker/main.go` | Modify | Remove `openvasScanner`, wire import handler, update `NewMux` call |

---

## Chunk 1: API-Key Middleware + Config

### Task 1: Create API-Key middleware

**Files:**
- Create: `internal/middleware/apikey.go`
- Create: `internal/middleware/apikey_test.go`

- [ ] **Step 1: Write the test file**

```go
// internal/middleware/apikey_test.go
package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
)

func TestAPIKeyAuth_ValidKey(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("X-API-Key", "a]valid-key-that-is-at-least-32-chars!!")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler := APIKeyAuth("a]valid-key-that-is-at-least-32-chars!!")(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})
	if err := handler(c); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestAPIKeyAuth_InvalidKey(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("X-API-Key", "wrong-key-wrong-key-wrong-key-wrong")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler := APIKeyAuth("a]valid-key-that-is-at-least-32-chars!!")(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})
	err := handler(c)
	he, ok := err.(*echo.HTTPError)
	if !ok || he.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %v", err)
	}
}

func TestAPIKeyAuth_MissingHeader(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler := APIKeyAuth("a]valid-key-that-is-at-least-32-chars!!")(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})
	err := handler(c)
	he, ok := err.(*echo.HTTPError)
	if !ok || he.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %v", err)
	}
}

func TestAPIKeyAuth_EmptyConfigKey(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("X-API-Key", "anything")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler := APIKeyAuth("")(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})
	err := handler(c)
	he, ok := err.(*echo.HTTPError)
	if !ok || he.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %v", err)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd e:/Code/openvas-tracker && go test ./internal/middleware/ -run TestAPIKeyAuth -v`
Expected: compilation error — `APIKeyAuth` not defined

- [ ] **Step 3: Implement the middleware**

```go
// internal/middleware/apikey.go
package middleware

import (
	"crypto/subtle"
	"net/http"

	"github.com/labstack/echo/v4"
)

func APIKeyAuth(key string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if key == "" {
				return echo.NewHTTPError(http.StatusServiceUnavailable, "import API key not configured")
			}
			provided := c.Request().Header.Get("X-API-Key")
			if provided == "" {
				return echo.NewHTTPError(http.StatusUnauthorized, "missing API key")
			}
			if subtle.ConstantTimeCompare([]byte(provided), []byte(key)) != 1 {
				return echo.NewHTTPError(http.StatusUnauthorized, "invalid API key")
			}
			return next(c)
		}
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd e:/Code/openvas-tracker && go test ./internal/middleware/ -run TestAPIKeyAuth -v`
Expected: all 4 tests PASS

- [ ] **Step 5: Commit**

```bash
git add internal/middleware/apikey.go internal/middleware/apikey_test.go
git commit -m "feat: add API-Key authentication middleware"
```

### Task 2: Update config — replace OpenVASPath with ImportAPIKey

**Files:**
- Modify: `internal/config/config.go:39-42` (ScannerConfig struct)
- Modify: `internal/config/config.go:58` (SetDefault line)
- Modify: `.env.example:10`

- [ ] **Step 1: Modify `ScannerConfig` struct**

In `internal/config/config.go`, replace:
```go
type ScannerConfig struct {
	NmapPath    string
	OpenVASPath string
}
```
With:
```go
type ScannerConfig struct {
	NmapPath     string
	ImportAPIKey string
}
```

- [ ] **Step 2: Update `SetDefault` in `Load()`**

In `internal/config/config.go`, replace:
```go
v.SetDefault("scanner.openvaspath", "gvm-cli")
```
With:
```go
v.SetDefault("scanner.importapikey", "")
```

- [ ] **Step 3: Update `.env.example`**

Replace:
```
OT_SCANNER_OPENVASPATH=gvm-cli
```
With:
```
OT_SCANNER_IMPORTAPIKEY=
```

- [ ] **Step 4: Run config tests**

Run: `cd e:/Code/openvas-tracker && go test ./internal/config/ -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/config/config.go .env.example
git commit -m "feat: replace OpenVASPath config with ImportAPIKey"
```

---

## Chunk 2: Remove OpenVAS Orchestration

### Task 3: Remove OpenVASScanner from scanner package

**Files:**
- Modify: `internal/scanner/openvas.go` — remove lines 84-103 (`OpenVASScanner`, `NewOpenVASScanner`, `Scan`)
- Verify: `internal/scanner/openvas_test.go` — should still pass (only tests `ParseOpenVASXML`)

- [ ] **Step 1: Remove `OpenVASScanner` struct, constructor, and `Scan` method**

In `internal/scanner/openvas.go`, delete the entire block from `type OpenVASScanner struct` through the end of the `Scan` method (lines 84-103). Also remove the now-unused imports: `"context"`, `"os/exec"`, `"strings"`.

After edit, the file should contain only: `package scanner`, imports for `"encoding/xml"`, `"fmt"`, `"io"`, `"strconv"`, the XML structs, `OpenVASResult`, and `ParseOpenVASXML()`.

- [ ] **Step 2: Verify parser tests still pass**

Run: `cd e:/Code/openvas-tracker && go test ./internal/scanner/ -run TestParseOpenVASXML -v`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add internal/scanner/openvas.go
git commit -m "refactor: remove OpenVASScanner orchestration, keep XML parser"
```

### Task 4: Remove OpenVAS from worker

**Files:**
- Modify: `internal/worker/scan_task.go` — remove `openvas` field, `HandleOpenVASScan`, update `NewScanHandler` signature
- Modify: `internal/worker/server.go` — remove `TaskScanOpenVAS`, update `NewMux` signature

- [ ] **Step 1: Update `ScanHandler` struct and constructor in `scan_task.go`**

Replace:
```go
type ScanHandler struct {
	q       *queries.Queries
	nmap    *scanner.NmapScanner
	openvas *scanner.OpenVASScanner
}

func NewScanHandler(db *sql.DB, nmap *scanner.NmapScanner, openvas *scanner.OpenVASScanner) *ScanHandler {
	return &ScanHandler{
		q:       queries.New(db),
		nmap:    nmap,
		openvas: openvas,
	}
}
```
With:
```go
type ScanHandler struct {
	q    *queries.Queries
	nmap *scanner.NmapScanner
}

func NewScanHandler(db *sql.DB, nmap *scanner.NmapScanner) *ScanHandler {
	return &ScanHandler{
		q:    queries.New(db),
		nmap: nmap,
	}
}
```

- [ ] **Step 2: Delete `HandleOpenVASScan` method entirely (lines 75-110)**

- [ ] **Step 3: Remove unused `scanner` import from `scan_task.go`**

The `scanner` import is still used by `NmapScanner`, so keep it. But remove the `openvas *scanner.OpenVASScanner` references. (Already done in step 1.)

- [ ] **Step 4: Update `server.go`**

Remove `TaskScanOpenVAS` constant and update `NewMux`:

Replace:
```go
const (
	TaskScanNmap    = "scan:nmap"
	TaskScanOpenVAS = "scan:openvas"
	TaskReport      = "report:generate"
	TaskEnrich      = "vuln:enrich"
)
```
With:
```go
const (
	TaskScanNmap = "scan:nmap"
	TaskReport   = "report:generate"
	TaskEnrich   = "vuln:enrich"
)
```

Replace:
```go
func NewMux(db *sql.DB, nmapScanner *scanner.NmapScanner, openvasScanner *scanner.OpenVASScanner) *asynq.ServeMux {
	mux := asynq.NewServeMux()
	scanHandler := NewScanHandler(db, nmapScanner, openvasScanner)
	mux.HandleFunc(TaskScanNmap, scanHandler.HandleNmapScan)
	mux.HandleFunc(TaskScanOpenVAS, scanHandler.HandleOpenVASScan)
	return mux
}
```
With:
```go
func NewMux(db *sql.DB, nmapScanner *scanner.NmapScanner) *asynq.ServeMux {
	mux := asynq.NewServeMux()
	scanHandler := NewScanHandler(db, nmapScanner)
	mux.HandleFunc(TaskScanNmap, scanHandler.HandleNmapScan)
	return mux
}
```

Remove `scanner` import from `server.go` (no longer needed — `NmapScanner` is passed as an interface arg, but actually it's a `*scanner.NmapScanner` pointer so the import is still needed). Keep the import.

- [ ] **Step 5: Update `scans.go` validation tag**

Note: Tasks 4 and 5 are combined into one commit because `handler/scans.go` references `worker.TaskScanOpenVAS` — removing it from worker without updating the handler would break the build.

Replace:
```go
ScanType string   `json:"scan_type" validate:"required,oneof=nmap openvas"`
```
With:
```go
ScanType string   `json:"scan_type" validate:"required,oneof=nmap"`
```

- [ ] **Step 6: Remove OpenVAS task-type selection in `scans.go` `Launch()`**

Replace:
```go
	taskType := worker.TaskScanNmap
	if req.ScanType == "openvas" {
		taskType = worker.TaskScanOpenVAS
	}

	task, err := worker.NewScanTask(taskType, worker.ScanPayload{
```
With:
```go
	task, err := worker.NewScanTask(worker.TaskScanNmap, worker.ScanPayload{
```

Also remove the `worker` import reference to `TaskScanOpenVAS` — it's no longer exported. The `worker` import itself is still needed for `NewScanTask` and `ScanPayload`.

- [ ] **Step 7: Update `schedules.go` validation tag**

Replace:
```go
ScanType string `json:"scan_type" validate:"required,oneof=nmap openvas"`
```
With:
```go
ScanType string `json:"scan_type" validate:"required,oneof=nmap"`
```

- [ ] **Step 8: Verify compilation**

Run: `cd e:/Code/openvas-tracker && go build ./internal/worker/ && go build ./internal/handler/`
Expected: both succeed

- [ ] **Step 9: Commit**

```bash
git add internal/worker/scan_task.go internal/worker/server.go internal/handler/scans.go internal/handler/schedules.go
git commit -m "refactor: remove OpenVAS orchestration from worker and handler"
```

### Task 5: Update main.go — remove OpenVAS scanner, fix NewMux call

**Files:**
- Modify: `cmd/openvas-tracker/main.go:117-120` — remove openvasScanner, update NewMux

- [ ] **Step 1: Remove OpenVAS scanner instantiation and update NewMux**

Replace:
```go
	// Start Asynq worker in background
	nmapScanner := scanner.NewNmapScanner(cfg.Scanner.NmapPath)
	openvasScanner := scanner.NewOpenVASScanner(cfg.Scanner.OpenVASPath)
	workerSrv := worker.NewServer(cfg, db)
	workerMux := worker.NewMux(db, nmapScanner, openvasScanner)
```
With:
```go
	// Start Asynq worker in background
	nmapScanner := scanner.NewNmapScanner(cfg.Scanner.NmapPath)
	workerSrv := worker.NewServer(cfg, db)
	workerMux := worker.NewMux(db, nmapScanner)
```

- [ ] **Step 2: Verify full build**

Run: `cd e:/Code/openvas-tracker && go build ./cmd/openvas-tracker/`
Expected: success

- [ ] **Step 3: Run all existing tests**

Run: `cd e:/Code/openvas-tracker && go test ./... 2>&1 | head -50`
Expected: all tests pass

- [ ] **Step 4: Commit**

```bash
git add cmd/openvas-tracker/main.go
git commit -m "refactor: remove OpenVAS scanner from main, update worker wiring"
```

---

## Chunk 3: Import Handler + Wiring

### Task 6: Create the import handler

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

const testOpenVASXML = `<?xml version="1.0"?>
<report>
  <results>
    <result>
      <name>SSL/TLS Certificate Expired</name>
      <host>192.168.1.10</host>
      <port>443/tcp</port>
      <threat>High</threat>
      <severity>7.5</severity>
      <description>The SSL certificate has expired.</description>
      <nvt oid="1.3.6.1.4.1.25623.1.0.103955">
        <name>SSL/TLS Certificate Expired</name>
        <cvss_base>7.5</cvss_base>
        <cve>CVE-2024-0001</cve>
        <solution type="VendorFix">Renew the certificate.</solution>
      </nvt>
    </result>
    <result>
      <name>Info Disclosure</name>
      <host>192.168.1.10</host>
      <port>80/tcp</port>
      <threat>Log</threat>
      <severity>0.0</severity>
      <description>Server banner disclosed.</description>
      <nvt oid="1.3.6.1.4.1.25623.1.0.999999">
        <name>Info Disclosure</name>
        <cvss_base>0.0</cvss_base>
        <cve>NOCVE</cve>
        <solution type="Mitigation">Disable server banner.</solution>
      </nvt>
    </result>
  </results>
</report>`

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
		input    string
		wantPort *int32
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

func int32Ptr(v int32) *int32    { return &v }
func strPtr(v string) *string    { return &v }
func derefInt32(p *int32) string { if p == nil { return "<nil>" }; return fmt.Sprintf("%d", *p) }
func derefStr(p *string) string  { if p == nil { return "<nil>" }; return *p }
func int32PtrEq(a, b *int32) bool { if a == nil && b == nil { return true }; if a == nil || b == nil { return false }; return *a == *b }
func strPtrEq(a, b *string) bool  { if a == nil && b == nil { return true }; if a == nil || b == nil { return false }; return *a == *b }
```

Note: `"fmt"` is included in the import block above.

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

	// Step 1: Create scan record
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

	// Step 2: Store raw XML and timestamps
	rawXML := string(body)
	h.q.UpdateScanStatus(c.Request().Context(), queries.UpdateScanStatusParams{
		ID:          scan.ID,
		Status:      queries.ScanStatusCompleted,
		StartedAt:   &now,
		CompletedAt: &now,
		RawOutput:   &rawXML,
	})

	// Step 3: Create vulnerabilities
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
			ID:           uuid.New().String(),
			ScanID:       scan.ID,
			UserID:       h.systemUserID,
			Title:        r.Title,
			Description:  desc,
			Severity:     queries.SeverityLevel(severity),
			CvssScore:    cvss,
			CveID:        cveID,
			AffectedHost: host,
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
		// User not found — create
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
			// Duplicate key — another instance created it; retry lookup
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

- [ ] **Step 4: Run unit tests (mapSeverity, parsePort, bad XML)**

Run: `cd e:/Code/openvas-tracker && go test ./internal/handler/ -run "TestMapSeverity|TestParsePort|TestHandleOpenVAS_BadXML" -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/handler/import.go internal/handler/import_test.go
git commit -m "feat: add OpenVAS webhook import handler with severity mapping and port parsing"
```

### Task 7: Wire import handler into main.go

**Files:**
- Modify: `cmd/openvas-tracker/main.go` — add import handler registration before `serveFrontend`

- [ ] **Step 1: Add import handler registration**

After the WebSocket registration block (line ~114) and before `serveFrontend(e)`, add:

```go
	// OpenVAS import webhook (API-Key auth, outside JWT group)
	if cfg.Scanner.ImportAPIKey != "" {
		if len(cfg.Scanner.ImportAPIKey) < 32 {
			log.Fatal("OT_SCANNER_IMPORTAPIKEY must be at least 32 characters")
		}
		importG := e.Group("/api/import", mw.APIKeyAuth(cfg.Scanner.ImportAPIKey), echomw.BodyLimit("10M"))
		handler.NewImportHandler(db).RegisterRoutes(importG)
	}
```

- [ ] **Step 2: Remove unused `scanner` import from main.go**

The `scanner` package import was used for `scanner.NewOpenVASScanner` and `scanner.NewNmapScanner`. Since `NewNmapScanner` is still used, keep the import.

- [ ] **Step 3: Verify full build**

Run: `cd e:/Code/openvas-tracker && go build ./cmd/openvas-tracker/`
Expected: success

- [ ] **Step 4: Run all tests**

Run: `cd e:/Code/openvas-tracker && go test ./...`
Expected: all pass

- [ ] **Step 5: Commit**

```bash
git add cmd/openvas-tracker/main.go
git commit -m "feat: wire OpenVAS import webhook endpoint into main"
```

---

## Chunk 4: Verification

### Task 8: End-to-end verification

- [ ] **Step 1: Run full test suite**

Run: `cd e:/Code/openvas-tracker && go test ./... -v`
Expected: all tests pass

- [ ] **Step 2: Verify build produces binary**

Run: `cd e:/Code/openvas-tracker && go build -o openvas-tracker.exe ./cmd/openvas-tracker/`
Expected: binary created without errors

- [ ] **Step 3: Verify no references to removed code**

Run: `cd e:/Code/openvas-tracker && grep -r "NewOpenVASScanner\|OpenVASScanner\|TaskScanOpenVAS\|HandleOpenVASScan\|OpenVASPath\|openvaspath" --include="*.go" .`
Expected: no matches (except possibly in the spec/plan docs)

- [ ] **Step 4: Final commit if any cleanup needed**
