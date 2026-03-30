# ZAP DAST Integration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add OWASP ZAP as a second scanner type with import webhook, URL-granular ticketing, and scanner-type-scoped auto-resolve.

**Architecture:** Introduce a `Finding` struct and `Parser` interface in `internal/scanner/` to abstract over scanner-specific formats. Refactor `ImportService.Import()` to accept `[]Finding` instead of `[]OpenVASResult`. Add ZAP JSON parser, new DB migration, new API endpoint, and minimal frontend changes.

**Tech Stack:** Go (backend), MariaDB (migration), React 19 + Tailwind (frontend), Echo (HTTP framework)

---

### Task 1: Database Migration — Add ZAP fields

**Files:**
- Create: `sql/migrations/020_add_zap_fields.up.sql`
- Create: `sql/migrations/020_add_zap_fields.down.sql`
- Modify: `sql/docker-init.sql`

- [ ] **Step 1: Create up migration**

```sql
-- sql/migrations/020_add_zap_fields.up.sql
ALTER TABLE vulnerabilities
  ADD COLUMN url VARCHAR(2048) DEFAULT '' AFTER hostname,
  ADD COLUMN parameter VARCHAR(255) DEFAULT '' AFTER url,
  ADD COLUMN evidence TEXT AFTER solution,
  ADD COLUMN confidence VARCHAR(20) DEFAULT '' AFTER evidence;

ALTER TABLE scans MODIFY COLUMN scan_type ENUM('nmap', 'openvas', 'zap', 'custom') NOT NULL;
```

Note: `cwe_id` column already exists in the vulnerabilities table (added in an earlier migration, present in `vulnCols` and `CreateVulnerabilityParams`).

- [ ] **Step 2: Create down migration**

```sql
-- sql/migrations/020_add_zap_fields.down.sql
ALTER TABLE vulnerabilities
  DROP COLUMN url,
  DROP COLUMN parameter,
  DROP COLUMN evidence,
  DROP COLUMN confidence;

ALTER TABLE scans MODIFY COLUMN scan_type ENUM('nmap', 'openvas', 'custom') NOT NULL;
```

- [ ] **Step 3: Add SOURCE line to docker-init.sql**

Add at the end of `sql/docker-init.sql`:
```sql
SOURCE /docker-entrypoint-initdb.d/migrations/020_add_zap_fields.up.sql;
```

- [ ] **Step 4: Commit**

```bash
git add sql/migrations/020_add_zap_fields.up.sql sql/migrations/020_add_zap_fields.down.sql sql/docker-init.sql
git commit -m "feat: add migration 020 for ZAP DAST fields"
```

---

### Task 2: Update Vulnerability queries for new columns

**Files:**
- Modify: `internal/database/queries/vulnerabilities.go`
- Modify: `internal/database/queries/scans.go`

- [ ] **Step 1: Add new fields to Vulnerability struct**

In `internal/database/queries/vulnerabilities.go`, add to the `Vulnerability` struct after `Hostname`:

```go
URL         *string  `json:"url"`
Parameter   *string  `json:"parameter"`
```

And after `Solution`:
```go
Evidence    *string  `json:"evidence"`
Confidence  *string  `json:"confidence"`
```

- [ ] **Step 2: Update vulnCols**

Replace the existing `vulnCols` constant with:

```go
const vulnCols = `id, scan_id, target_id, user_id, title, description, severity, status, cvss_score, cve_id, cwe_id, affected_host, hostname, url, parameter, affected_port, protocol, service, solution, evidence, confidence, vuln_references, enrichment_data, risk_score, discovered_at, resolved_at, created_at, updated_at`
```

- [ ] **Step 3: Update scanVuln to scan new fields**

Update the `scanVuln` function to include the new fields in the correct order:

```go
func scanVuln(row interface{ Scan(...any) error }, i *Vulnerability) error {
	return row.Scan(&i.ID, &i.ScanID, &i.TargetID, &i.UserID, &i.Title, &i.Description, &i.Severity, &i.Status, &i.CvssScore, &i.CveID, &i.CweID, &i.AffectedHost, &i.Hostname, &i.URL, &i.Parameter, &i.AffectedPort, &i.Protocol, &i.Service, &i.Solution, &i.Evidence, &i.Confidence, &i.VulnReferences, &i.EnrichmentData, &i.RiskScore, &i.DiscoveredAt, &i.ResolvedAt, &i.CreatedAt, &i.UpdatedAt)
}
```

- [ ] **Step 4: Update CreateVulnerabilityParams and CreateVulnerability**

Add to `CreateVulnerabilityParams` after `Hostname`:
```go
URL         *string  `json:"url"`
Parameter   *string  `json:"parameter"`
```

And after `Solution`:
```go
Evidence    *string  `json:"evidence"`
Confidence  *string  `json:"confidence"`
```

Update the `CreateVulnerability` SQL and args:

```go
func (q *Queries) CreateVulnerability(ctx context.Context, arg CreateVulnerabilityParams) (Vulnerability, error) {
	const createVulnerability = `INSERT INTO vulnerabilities (id, scan_id, target_id, user_id, title, description, severity, cvss_score, cve_id, cwe_id, affected_host, hostname, url, parameter, affected_port, protocol, service, solution, evidence, confidence, vuln_references) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	_, err := q.db.ExecContext(ctx, createVulnerability, arg.ID, arg.ScanID, arg.TargetID, arg.UserID, arg.Title, arg.Description, arg.Severity, arg.CvssScore, arg.CveID, arg.CweID, arg.AffectedHost, arg.Hostname, arg.URL, arg.Parameter, arg.AffectedPort, arg.Protocol, arg.Service, arg.Solution, arg.Evidence, arg.Confidence, arg.VulnReferences)
	if err != nil {
		return Vulnerability{}, err
	}
	return q.GetVulnerability(ctx, arg.ID)
}
```

- [ ] **Step 5: Add ScanTypeZap constant to scans.go**

In `internal/database/queries/scans.go`, add to the ScanType constants:

```go
ScanTypeZap ScanType = "zap"
```

- [ ] **Step 6: Run tests to verify nothing breaks**

Run: `go test ./internal/database/queries/ -v -count=1`
Expected: PASS (or no tests in package — that's fine, the compiler check matters)

Run: `go build ./...`
Expected: Build succeeds

- [ ] **Step 7: Commit**

```bash
git add internal/database/queries/vulnerabilities.go internal/database/queries/scans.go
git commit -m "feat: add ZAP fields to vulnerability queries and ZAP scan type"
```

---

### Task 3: Scanner abstraction — Finding struct and Parser interface

**Files:**
- Create: `internal/scanner/scanner.go`
- Create: `internal/scanner/scanner_test.go`

- [ ] **Step 1: Write the test for Finding fingerprinting**

```go
// internal/scanner/scanner_test.go
package scanner

import "testing"

func TestFindingFingerprint(t *testing.T) {
	tests := []struct {
		name     string
		finding  Finding
		expected string
	}{
		{
			name:     "network finding with CVE",
			finding:  Finding{CVEID: "CVE-2024-0001", Title: "SSL Expired", ScanType: "openvas"},
			expected: "CVE-2024-0001",
		},
		{
			name:     "network finding without CVE",
			finding:  Finding{Title: "SSL Expired", ScanType: "openvas"},
			expected: "title:SSL Expired",
		},
		{
			name:     "web finding with CWE and URL",
			finding:  Finding{CWEID: "79", URL: "/app/search", Parameter: "q", ScanType: "zap"},
			expected: "cwe:79:url:/app/search:param:q",
		},
		{
			name:     "web finding with CWE, URL, no param",
			finding:  Finding{CWEID: "352", URL: "/app/transfer", ScanType: "zap"},
			expected: "cwe:352:url:/app/transfer:param:",
		},
		{
			name:     "web finding without CWE",
			finding:  Finding{Title: "Information Disclosure", URL: "/api/debug", ScanType: "zap"},
			expected: "title:Information Disclosure:url:/api/debug",
		},
		{
			name:     "web finding with CVE takes CVE path",
			finding:  Finding{CVEID: "CVE-2024-9999", CWEID: "79", URL: "/app", ScanType: "zap"},
			expected: "CVE-2024-9999",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.finding.Fingerprint()
			if got != tt.expected {
				t.Errorf("Fingerprint() = %q, want %q", got, tt.expected)
			}
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/scanner/ -v -run TestFindingFingerprint -count=1`
Expected: FAIL — `Finding` type and `Fingerprint()` method not defined

- [ ] **Step 3: Implement Finding struct and Parser interface**

```go
// internal/scanner/scanner.go
package scanner

import "io"

// Finding is a scanner-agnostic vulnerability finding.
type Finding struct {
	Host        string
	Hostname    string
	Port        string
	Protocol    string
	URL         string // Full URL path (empty for network scans)
	Parameter   string // Affected parameter (empty for network scans)
	Title       string
	Description string
	Severity    string // critical/high/medium/low/info
	CVSSScore   float64
	CVEID       string
	CWEID       string
	OID         string
	Solution    string
	Evidence    string // Proof snippet from response
	Confidence  string // confirmed/high/medium/low
	ScanType    string // "openvas" or "zap"
}

// Fingerprint returns the canonical dedup key for this finding.
// Network findings: CVE or "title:" + title
// Web findings: "cwe:" + CWE + ":url:" + URL + ":param:" + param, or title-based fallback
func (f Finding) Fingerprint() string {
	// CVE always takes priority regardless of scan type
	if f.CVEID != "" && f.CVEID != "NOCVE" {
		return f.CVEID
	}
	// Web findings with CWE + URL
	if f.URL != "" && f.CWEID != "" {
		return "cwe:" + f.CWEID + ":url:" + f.URL + ":param:" + f.Parameter
	}
	// Web findings without CWE but with URL
	if f.URL != "" {
		return "title:" + f.Title + ":url:" + f.URL
	}
	// Network findings without CVE
	return "title:" + f.Title
}

// Parser parses scanner-specific report formats into generic Findings.
type Parser interface {
	Parse(r io.Reader) ([]Finding, error)
	ScanType() string
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/scanner/ -v -run TestFindingFingerprint -count=1`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/scanner/scanner.go internal/scanner/scanner_test.go
git commit -m "feat: add Finding struct and Parser interface for scanner abstraction"
```

---

### Task 4: Refactor OpenVAS parser to return []Finding

**Files:**
- Modify: `internal/scanner/openvas.go`
- Modify: `internal/scanner/openvas_test.go`

- [ ] **Step 1: Update the existing test to expect []Finding**

Replace the test in `internal/scanner/openvas_test.go`:

```go
package scanner

import (
	"strings"
	"testing"
)

func TestParseOpenVASXML(t *testing.T) {
	xml := `<?xml version="1.0"?>
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
  </results>
</report>`

	results, err := ParseOpenVASXML(strings.NewReader(xml))
	if err != nil {
		t.Fatalf("ParseOpenVASXML error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	r := results[0]
	if r.Title != "SSL/TLS Certificate Expired" {
		t.Errorf("expected title SSL/TLS Certificate Expired, got %s", r.Title)
	}
	if r.Severity != "High" {
		t.Errorf("expected severity High, got %s", r.Severity)
	}
	if r.CVSSScore != 7.5 {
		t.Errorf("expected CVSS 7.5, got %f", r.CVSSScore)
	}
	if r.Host != "192.168.1.10" {
		t.Errorf("expected host 192.168.1.10, got %s", r.Host)
	}
	if r.CVEID != "CVE-2024-0001" {
		t.Errorf("expected CVE CVE-2024-0001, got %s", r.CVEID)
	}
	if r.ScanType != "openvas" {
		t.Errorf("expected ScanType openvas, got %s", r.ScanType)
	}
	if r.URL != "" {
		t.Errorf("expected empty URL for network scan, got %s", r.URL)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/scanner/ -v -run TestParseOpenVASXML -count=1`
Expected: FAIL — `ParseOpenVASXML` returns `[]OpenVASResult`, not `[]Finding`

- [ ] **Step 3: Update ParseOpenVASXML to return []Finding**

In `internal/scanner/openvas.go`, change the function signature and return type. Keep `OpenVASResult` as a type alias for backward compatibility if needed, but change `ParseOpenVASXML` to return `[]Finding`:

Change the function signature from:
```go
func ParseOpenVASXML(r io.Reader) ([]OpenVASResult, error) {
```
to:
```go
func ParseOpenVASXML(r io.Reader) ([]Finding, error) {
```

Change the results variable from:
```go
var results []OpenVASResult
```
to:
```go
var results []Finding
```

Change the append at the end of the loop from:
```go
results = append(results, OpenVASResult{
    Title:       res.Name,
    Host:        host,
    Hostname:    hostname,
    Port:        res.Port,
    Severity:    threat,
    CVSSScore:   cvss,
    Description: description,
    Solution:    solution,
    CVE:         cve,
    OID:         res.NVT.OID,
})
```
to:
```go
results = append(results, Finding{
    Title:       res.Name,
    Host:        host,
    Hostname:    hostname,
    Port:        res.Port,
    Severity:    threat,
    CVSSScore:   cvss,
    Description: description,
    Solution:    solution,
    CVEID:       cve,
    OID:         res.NVT.OID,
    ScanType:    "openvas",
})
```

Remove the `OpenVASResult` struct (it's no longer needed).

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/scanner/ -v -count=1`
Expected: PASS

- [ ] **Step 5: Verify the build still compiles**

Run: `go build ./...`
Expected: FAIL — `internal/service/import.go` still references `scanner.OpenVASResult` and `.CVE` field. This is expected and will be fixed in Task 5.

- [ ] **Step 6: Commit**

```bash
git add internal/scanner/openvas.go internal/scanner/openvas_test.go
git commit -m "refactor: change ParseOpenVASXML to return []Finding"
```

---

### Task 5: Refactor ImportService to use []Finding

**Files:**
- Modify: `internal/service/import.go`

This task updates the import service to work with `Finding` instead of `OpenVASResult`. The key changes:
- `Import()` accepts `[]Finding` and a `scanType` string
- `processTicket()` and `createTicket()` accept `Finding` instead of `OpenVASResult`
- Fingerprinting uses `Finding.Fingerprint()` method
- `CreateVulnerability` passes new fields (URL, Parameter, Evidence, Confidence, CweID)
- Activity log notes use `Finding.CVEID` instead of `.CVE`
- Scan name includes scan type

- [ ] **Step 1: Update Import() signature and scan creation**

Change signature from:
```go
func (s *ImportService) Import(ctx context.Context, results []scanner.OpenVASResult) (*ImportResult, error) {
```
to:
```go
func (s *ImportService) Import(ctx context.Context, results []scanner.Finding, scanType string) (*ImportResult, error) {
```

Update scan creation to use `scanType`:
```go
scan, err := tq.CreateScan(ctx, queries.CreateScanParams{
    ID:       scanID,
    Name:     fmt.Sprintf("%s Import %s", strings.ToUpper(scanType), now.Format("2006-01-02 15:04:05")),
    ScanType: queries.ScanType(scanType),
    Status:   queries.ScanStatusCompleted,
    UserID:   s.systemUserID,
})
```

- [ ] **Step 2: Update processTicket() and createTicket() signatures**

Change `processTicket` from:
```go
func (s *ImportService) processTicket(ctx context.Context, q *queries.Queries, r scanner.OpenVASResult, vulnID, severity string, now time.Time) (created, reopened bool) {
```
to:
```go
func (s *ImportService) processTicket(ctx context.Context, q *queries.Queries, r scanner.Finding, vulnID, severity string, now time.Time) (created, reopened bool) {
```

Change `createTicket` from:
```go
func (s *ImportService) createTicket(ctx context.Context, q *queries.Queries, r scanner.OpenVASResult, vulnID, severity string) bool {
```
to:
```go
func (s *ImportService) createTicket(ctx context.Context, q *queries.Queries, r scanner.Finding, vulnID, severity string) bool {
```

- [ ] **Step 3: Update fingerprint calls and field references**

In `processTicket`, change all references from `r.CVE` to `r.CVEID` and replace `vulnFingerprint(r.CVE, r.Title)` with `r.Fingerprint()`.

In the `FindTicketByFingerprint` call, change from:
```go
existing, err := q.FindTicketByFingerprint(ctx, r.Host, r.CVE, r.Title)
```
to:
```go
existing, err := q.FindTicketByFingerprint(ctx, r.Host, r.CVEID, r.Title)
```

In all activity log `note` strings in `processTicket`, change `r.CVE` to `r.CVEID`.

In `createTicket`, same changes:
- `vulnFingerprint(r.CVE, r.Title)` → `r.Fingerprint()`
- `r.CVE` → `r.CVEID` in note strings
- `r.CVSSScore` stays the same

- [ ] **Step 4: Update CreateVulnerability call to pass new fields**

In the `Import()` function, update the `CreateVulnerability` call:

```go
_, err := tq.CreateVulnerability(ctx, queries.CreateVulnerabilityParams{
    ID:             vulnID,
    ScanID:         scan.ID,
    UserID:         s.systemUserID,
    Title:          r.Title,
    Description:    strPtr(r.Description),
    Severity:       queries.SeverityLevel(severity),
    CvssScore:      f64Ptr(r.CVSSScore),
    CveID:          cvePtr(r.CVEID),
    CweID:          strPtr(r.CWEID),
    AffectedHost:   strPtr(r.Host),
    Hostname:       strPtr(resolveHostname(r.Host, r.Hostname)),
    URL:            strPtr(r.URL),
    Parameter:      strPtr(r.Parameter),
    AffectedPort:   port,
    Protocol:       proto,
    Solution:       strPtr(r.Solution),
    Evidence:       strPtr(r.Evidence),
    Confidence:     strPtr(r.Confidence),
    VulnReferences: []byte("[]"),
})
```

- [ ] **Step 5: Update autoResolveStale to scope by scan type**

In `IncrementMissesForStaleTickets`, the stale ticket query currently checks if a ticket's host is in `scan_hosts` for the given scan. We need to also scope by scan type so ZAP scans don't auto-resolve OpenVAS tickets.

Update `autoResolveStale` to accept `scanType`:
```go
func (s *ImportService) autoResolveStale(ctx context.Context, q *queries.Queries, scanID string, scanType string) int {
```

And update the call in `Import()`:
```go
res.TicketsAutoResolved = s.autoResolveStale(ctx, tq, scan.ID, scanType)
```

Then update `IncrementMissesForStaleTickets` in `internal/database/queries/tickets.go` to accept and filter by scan type. Change the signature:
```go
func (q *Queries) IncrementMissesForStaleTickets(ctx context.Context, scanID string, scanType string) ([]Ticket, error) {
```

Update the ID query to also filter by scan type — a ticket's scan type is determined by the scan that created its linked vulnerability:
```go
idRows, err := q.db.QueryContext(ctx, `
    SELECT t.id FROM tickets t
    JOIN vulnerabilities v ON t.vulnerability_id = v.id
    JOIN scans s ON v.scan_id = s.id
    WHERE t.status IN ('open', 'pending_resolution')
    AND t.vulnerability_id IS NOT NULL
    AND v.affected_host IN (SELECT host FROM scan_hosts WHERE scan_id = ?)
    AND s.scan_type = ?
    AND t.vulnerability_id NOT IN (SELECT id FROM vulnerabilities WHERE scan_id = ?)`,
    scanID, scanType, scanID)
```

- [ ] **Step 6: Verify the build compiles**

Run: `go build ./...`
Expected: FAIL — `internal/handler/import.go` still calls `Import()` with old signature. Fixed in Task 7.

- [ ] **Step 7: Commit**

```bash
git add internal/service/import.go internal/database/queries/tickets.go
git commit -m "refactor: update ImportService to use Finding and scan-type-scoped auto-resolve"
```

---

### Task 6: ZAP JSON Parser

**Files:**
- Create: `internal/scanner/zap.go`
- Create: `internal/scanner/zap_test.go`

- [ ] **Step 1: Write the test**

```go
// internal/scanner/zap_test.go
package scanner

import (
	"strings"
	"testing"
)

func TestParseZAPJSON(t *testing.T) {
	json := `{
  "@version": "2.16.0",
  "site": [{
    "host": "example.com",
    "port": "443",
    "ssl": "true",
    "alerts": [{
      "pluginid": "40012",
      "alertRef": "40012",
      "alert": "Cross Site Scripting (Reflected)",
      "name": "Cross Site Scripting (Reflected)",
      "riskcode": "3",
      "confidence": "2",
      "riskdesc": "High (Medium)",
      "cweid": "79",
      "wascid": "8",
      "desc": "<p>Cross-site Scripting (XSS) is a vulnerability.</p>",
      "solution": "<p>Validate all input.</p>",
      "instances": [
        {
          "uri": "https://example.com/app/search",
          "method": "GET",
          "param": "q",
          "attack": "<script>alert(1)</script>",
          "evidence": "<script>alert(1)</script>"
        },
        {
          "uri": "https://example.com/app/comment",
          "method": "POST",
          "param": "text",
          "attack": "<img src=x onerror=alert(1)>",
          "evidence": "<img src=x onerror=alert(1)>"
        }
      ]
    }, {
      "pluginid": "10202",
      "alert": "Absence of Anti-CSRF Tokens",
      "riskcode": "2",
      "confidence": "3",
      "cweid": "352",
      "desc": "<p>No Anti-CSRF tokens were found.</p>",
      "solution": "<p>Use anti-CSRF tokens.</p>",
      "instances": [
        {
          "uri": "https://example.com/app/transfer",
          "method": "POST",
          "param": "",
          "evidence": ""
        }
      ]
    }, {
      "pluginid": "10096",
      "alert": "Timestamp Disclosure",
      "riskcode": "0",
      "confidence": "1",
      "cweid": "200",
      "desc": "<p>A timestamp was disclosed.</p>",
      "solution": "<p>Remove timestamps.</p>",
      "instances": [
        {
          "uri": "https://example.com/api/health",
          "method": "GET",
          "param": "",
          "evidence": "1609459200"
        }
      ]
    }]
  }]
}`

	results, err := ParseZAPJSON(strings.NewReader(json))
	if err != nil {
		t.Fatalf("ParseZAPJSON error: %v", err)
	}

	// Info-level findings (riskcode=0) should be skipped
	if len(results) != 3 {
		t.Fatalf("expected 3 results (2 XSS + 1 CSRF, info skipped), got %d", len(results))
	}

	// First XSS instance
	r := results[0]
	if r.Title != "Cross Site Scripting (Reflected)" {
		t.Errorf("title = %q, want %q", r.Title, "Cross Site Scripting (Reflected)")
	}
	if r.Host != "example.com" {
		t.Errorf("host = %q, want %q", r.Host, "example.com")
	}
	if r.Port != "443" {
		t.Errorf("port = %q, want %q", r.Port, "443")
	}
	if r.Severity != "high" {
		t.Errorf("severity = %q, want %q", r.Severity, "high")
	}
	if r.CVSSScore != 7.0 {
		t.Errorf("cvss = %f, want 7.0", r.CVSSScore)
	}
	if r.CWEID != "79" {
		t.Errorf("cweid = %q, want %q", r.CWEID, "79")
	}
	if r.URL != "/app/search" {
		t.Errorf("url = %q, want %q", r.URL, "/app/search")
	}
	if r.Parameter != "q" {
		t.Errorf("parameter = %q, want %q", r.Parameter, "q")
	}
	if r.Evidence != "<script>alert(1)</script>" {
		t.Errorf("evidence = %q, want %q", r.Evidence, "<script>alert(1)</script>")
	}
	if r.Confidence != "medium" {
		t.Errorf("confidence = %q, want %q", r.Confidence, "medium")
	}
	if r.ScanType != "zap" {
		t.Errorf("scanType = %q, want %q", r.ScanType, "zap")
	}
	if r.Description != "Cross-site Scripting (XSS) is a vulnerability." {
		t.Errorf("description HTML not stripped: %q", r.Description)
	}

	// Second XSS instance (different URL)
	r2 := results[1]
	if r2.URL != "/app/comment" {
		t.Errorf("url = %q, want %q", r2.URL, "/app/comment")
	}
	if r2.Parameter != "text" {
		t.Errorf("parameter = %q, want %q", r2.Parameter, "text")
	}

	// CSRF finding
	r3 := results[2]
	if r3.Title != "Absence of Anti-CSRF Tokens" {
		t.Errorf("title = %q, want %q", r3.Title, "Absence of Anti-CSRF Tokens")
	}
	if r3.Severity != "medium" {
		t.Errorf("severity = %q, want %q", r3.Severity, "medium")
	}
	if r3.CVSSScore != 4.0 {
		t.Errorf("cvss = %f, want 4.0", r3.CVSSScore)
	}
	if r3.Confidence != "high" {
		t.Errorf("confidence = %q, want %q", r3.Confidence, "high")
	}
}

func TestParseZAPJSON_Empty(t *testing.T) {
	json := `{"site": []}`
	results, err := ParseZAPJSON(strings.NewReader(json))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestParseZAPJSON_URLPathExtraction(t *testing.T) {
	json := `{
  "site": [{
    "host": "10.0.0.1",
    "port": "8080",
    "ssl": "false",
    "alerts": [{
      "alert": "SQL Injection",
      "riskcode": "3",
      "confidence": "4",
      "cweid": "89",
      "desc": "SQL injection found",
      "solution": "Use parameterized queries",
      "instances": [{
        "uri": "http://10.0.0.1:8080/api/users?id=1",
        "method": "GET",
        "param": "id",
        "evidence": "error in SQL syntax"
      }]
    }]
  }]
}`

	results, err := ParseZAPJSON(strings.NewReader(json))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].URL != "/api/users" {
		t.Errorf("url = %q, want %q (query string stripped)", results[0].URL, "/api/users")
	}
	if results[0].Host != "10.0.0.1" {
		t.Errorf("host = %q, want %q", results[0].Host, "10.0.0.1")
	}
	if results[0].Confidence != "confirmed" {
		t.Errorf("confidence = %q, want %q", results[0].Confidence, "confirmed")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/scanner/ -v -run TestParseZAPJSON -count=1`
Expected: FAIL — `ParseZAPJSON` not defined

- [ ] **Step 3: Implement ZAP parser**

```go
// internal/scanner/zap.go
package scanner

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"regexp"
	"strings"
)

type zapReport struct {
	Site []zapSite `json:"site"`
}

type zapSite struct {
	Host   string     `json:"host"`
	Port   string     `json:"port"`
	SSL    string     `json:"ssl"`
	Alerts []zapAlert `json:"alerts"`
}

type zapAlert struct {
	PluginID   string        `json:"pluginid"`
	Alert      string        `json:"alert"`
	RiskCode   string        `json:"riskcode"`
	Confidence string        `json:"confidence"`
	CWEID      string        `json:"cweid"`
	Desc       string        `json:"desc"`
	Solution   string        `json:"solution"`
	Instances  []zapInstance `json:"instances"`
}

type zapInstance struct {
	URI      string `json:"uri"`
	Method   string `json:"method"`
	Param    string `json:"param"`
	Attack   string `json:"attack"`
	Evidence string `json:"evidence"`
}

// ParseZAPJSON parses a ZAP Traditional JSON Report into []Finding.
// Info-level findings (riskcode=0) are skipped.
func ParseZAPJSON(r io.Reader) ([]Finding, error) {
	var report zapReport
	if err := json.NewDecoder(r).Decode(&report); err != nil {
		return nil, fmt.Errorf("failed to parse ZAP JSON: %w", err)
	}

	var results []Finding
	for _, site := range report.Site {
		for _, alert := range site.Alerts {
			severity, cvss := mapZAPRisk(alert.RiskCode)
			if severity == "info" {
				continue
			}

			for _, inst := range alert.Instances {
				urlPath := extractURLPath(inst.URI)

				results = append(results, Finding{
					Host:        site.Host,
					Port:        site.Port,
					Protocol:    zapProtocol(site.SSL),
					URL:         urlPath,
					Parameter:   inst.Param,
					Title:       alert.Alert,
					Description: stripHTML(alert.Desc),
					Severity:    severity,
					CVSSScore:   cvss,
					CWEID:       alert.CWEID,
					Solution:    stripHTML(alert.Solution),
					Evidence:    inst.Evidence,
					Confidence:  mapZAPConfidence(alert.Confidence),
					ScanType:    "zap",
				})
			}
		}
	}
	return results, nil
}

func mapZAPRisk(code string) (string, float64) {
	switch code {
	case "3":
		return "high", 7.0
	case "2":
		return "medium", 4.0
	case "1":
		return "low", 2.0
	default:
		return "info", 0.0
	}
}

func mapZAPConfidence(code string) string {
	switch code {
	case "4":
		return "confirmed"
	case "3":
		return "high"
	case "2":
		return "medium"
	default:
		return "low"
	}
}

func zapProtocol(ssl string) string {
	if ssl == "true" {
		return "tcp"
	}
	return "tcp"
}

// extractURLPath extracts just the path from a full URI, stripping scheme, host, and query.
func extractURLPath(uri string) string {
	if uri == "" {
		return ""
	}
	u, err := url.Parse(uri)
	if err != nil {
		return uri
	}
	return u.Path
}

var htmlTagRe = regexp.MustCompile(`<[^>]*>`)

// stripHTML removes HTML tags and trims whitespace.
func stripHTML(s string) string {
	s = htmlTagRe.ReplaceAllString(s, "")
	s = strings.TrimSpace(s)
	return s
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/scanner/ -v -count=1`
Expected: ALL PASS

- [ ] **Step 5: Commit**

```bash
git add internal/scanner/zap.go internal/scanner/zap_test.go
git commit -m "feat: add ZAP JSON Traditional Report parser"
```

---

### Task 7: Update Import Handler and wiring

**Files:**
- Modify: `internal/handler/import.go`
- Modify: `cmd/openvas-tracker/main.go`

- [ ] **Step 1: Update HandleOpenVAS to pass scanType**

In `internal/handler/import.go`, update `HandleOpenVAS`:

```go
func (h *ImportHandler) HandleOpenVAS(c echo.Context) error {
	results, err := scanner.ParseOpenVASXML(c.Request().Body)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "failed to parse OpenVAS XML")
	}
	if len(results) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "empty report — no results found")
	}

	res, err := h.importSvc.Import(c.Request().Context(), results, "openvas")
	if err != nil {
		log.Printf("import error: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "import failed")
	}

	go func() {
		if n, err := h.importSvc.BackfillHostnames(context.Background()); err != nil {
			log.Printf("hostname backfill error: %v", err)
		} else if n > 0 {
			log.Printf("hostname backfill: resolved %d hosts", n)
		}
	}()

	return c.JSON(http.StatusCreated, res)
}
```

- [ ] **Step 2: Add HandleZAP handler**

Add the new handler method in `internal/handler/import.go`:

```go
func (h *ImportHandler) HandleZAP(c echo.Context) error {
	results, err := scanner.ParseZAPJSON(c.Request().Body)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "failed to parse ZAP JSON")
	}
	if len(results) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "empty report — no results found")
	}

	res, err := h.importSvc.Import(c.Request().Context(), results, "zap")
	if err != nil {
		log.Printf("import error: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "import failed")
	}

	go func() {
		if n, err := h.importSvc.BackfillHostnames(context.Background()); err != nil {
			log.Printf("hostname backfill error: %v", err)
		} else if n > 0 {
			log.Printf("hostname backfill: resolved %d hosts", n)
		}
	}()

	return c.JSON(http.StatusCreated, res)
}
```

- [ ] **Step 3: Register ZAP route**

Update `RegisterRoutes` in `internal/handler/import.go`:

```go
func (h *ImportHandler) RegisterRoutes(g *echo.Group) {
	g.POST("/openvas", h.HandleOpenVAS)
	g.GET("/openvas", h.TriggerFetch)
	g.POST("/zap", h.HandleZAP)
}
```

- [ ] **Step 4: Verify the build compiles**

Run: `go build ./...`
Expected: PASS

- [ ] **Step 5: Run all tests**

Run: `go test ./... -v -count=1`
Expected: ALL PASS

- [ ] **Step 6: Commit**

```bash
git add internal/handler/import.go cmd/openvas-tracker/main.go
git commit -m "feat: add POST /api/import/zap endpoint and update OpenVAS handler"
```

---

### Task 8: Frontend — Scan type badges on Scans page

**Files:**
- Modify: `frontend/src/pages/Scans.tsx`

- [ ] **Step 1: Add scan type badge colors**

At the top of the file (after imports), add:

```tsx
const SCAN_TYPE_COLORS: Record<string, string> = {
  openvas: 'bg-green-900 text-green-300',
  zap: 'bg-blue-900 text-blue-300',
  nmap: 'bg-purple-900 text-purple-300',
  custom: 'bg-slate-700 text-slate-300',
}
```

- [ ] **Step 2: Replace the plain text scan_type cell with a badge**

In the table body, replace:
```tsx
<td className="p-3 text-slate-400">{s.scan_type}</td>
```
with:
```tsx
<td className="p-3"><span className={`px-2 py-1 rounded text-xs font-medium ${SCAN_TYPE_COLORS[s.scan_type] || 'bg-slate-700 text-slate-300'}`}>{s.scan_type.toUpperCase()}</span></td>
```

- [ ] **Step 3: Add scan type filter**

In the `TableFilter` filters array, add a scan_type filter:

```tsx
<TableFilter filters={[
  { key: 'search', label: 'Search scans...' },
  { key: 'status', label: 'Status', options: ['completed', 'running', 'failed', 'pending'] },
  { key: 'scan_type', label: 'Type', options: ['openvas', 'zap'] },
]} values={values} onChange={setValues} />
```

Update `useTableFilter` to include `scan_type`:
```tsx
const { values, setValues } = useTableFilter(['search', 'status', 'scan_type'])
```

Add the filter logic in the `filtered` useMemo:
```tsx
if (values.scan_type) result = result.filter(s => s.scan_type === values.scan_type)
```

- [ ] **Step 4: Build frontend**

Run: `cd frontend && npm run build`
Expected: Build succeeds

- [ ] **Step 5: Commit**

```bash
git add frontend/src/pages/Scans.tsx
git commit -m "feat: add scan type badges and filter on Scans page"
```

---

### Task 9: Frontend — ZAP fields on Ticket Detail page

**Files:**
- Modify: `frontend/src/pages/TicketDetail.tsx`

- [ ] **Step 1: Fetch vulnerability data for the ticket**

Add a query to fetch the vulnerability linked to the ticket, which contains the new ZAP fields. Add an interface and query after the existing queries:

```tsx
interface VulnDetail {
  id: string; url?: string; parameter?: string; evidence?: string; confidence?: string; cwe_id?: string
}
```

Add the query:
```tsx
const { data: vuln } = useQuery({
  queryKey: ['vulnerability', ticket?.vulnerability_id],
  queryFn: () => api.get<VulnDetail>(`/vulnerabilities/${ticket!.vulnerability_id}`),
  enabled: !!ticket?.vulnerability_id,
})
```

- [ ] **Step 2: Add CWE reference in the References section**

After the CVE references block (inside the References section), add CWE:

```tsx
{vuln?.cwe_id && (
  <div className="flex items-center gap-2 mt-1.5">
    <span className="text-xs text-slate-500 w-12">CWE</span>
    <a href={`https://cwe.mitre.org/data/definitions/${vuln.cwe_id}.html`} target="_blank" rel="noopener noreferrer" className="text-blue-400 hover:underline text-sm">CWE-{vuln.cwe_id}</a>
  </div>
)}
```

- [ ] **Step 3: Add Web Finding Details section**

After the "Affected Host" section and before "References", add a conditional section for web findings:

```tsx
{vuln?.url && (
  <div className="bg-slate-900 rounded-lg border border-slate-800 p-4 mb-6">
    <h3 className="text-sm font-medium text-slate-400 mb-2">Web Finding Details</h3>
    <div className="space-y-2 text-sm">
      <div className="flex items-start gap-2">
        <span className="text-slate-500 w-20 shrink-0">URL</span>
        <span className="text-slate-300 font-mono">{vuln.url}</span>
      </div>
      {vuln.parameter && (
        <div className="flex items-start gap-2">
          <span className="text-slate-500 w-20 shrink-0">Parameter</span>
          <span className="text-slate-300 font-mono">{vuln.parameter}</span>
        </div>
      )}
      {vuln.confidence && (
        <div className="flex items-start gap-2">
          <span className="text-slate-500 w-20 shrink-0">Confidence</span>
          <span className={`px-2 py-0.5 rounded text-xs font-medium ${
            vuln.confidence === 'confirmed' ? 'bg-green-900 text-green-300' :
            vuln.confidence === 'high' ? 'bg-blue-900 text-blue-300' :
            vuln.confidence === 'medium' ? 'bg-yellow-900 text-yellow-300' :
            'bg-slate-700 text-slate-300'
          }`}>{vuln.confidence}</span>
        </div>
      )}
      {vuln.evidence && (
        <div className="flex items-start gap-2">
          <span className="text-slate-500 w-20 shrink-0">Evidence</span>
          <code className="text-slate-300 bg-slate-800 rounded px-2 py-1 text-xs block max-w-full overflow-x-auto">
            {vuln.evidence.length > 500 ? vuln.evidence.slice(0, 500) + '...' : vuln.evidence}
          </code>
        </div>
      )}
    </div>
  </div>
)}
```

- [ ] **Step 4: Build frontend**

Run: `cd frontend && npm run build`
Expected: Build succeeds

- [ ] **Step 5: Commit**

```bash
git add frontend/src/pages/TicketDetail.tsx
git commit -m "feat: show ZAP web finding details on ticket detail page"
```

---

### Task 10: Full integration test

**Files:**
- No new files — manual verification

- [ ] **Step 1: Verify full build**

Run: `make build`
Expected: Builds frontend, copies to static, compiles Go binary — all succeed

- [ ] **Step 2: Run all Go tests**

Run: `go test ./... -v -count=1`
Expected: ALL PASS

- [ ] **Step 3: Verify migration applies**

If you have a local MariaDB available, verify the migration:
```bash
make migrate-up
```
Otherwise, verify the SQL syntax is correct by visual inspection of `020_add_zap_fields.up.sql`.

- [ ] **Step 4: Commit any remaining fixes**

If anything needed fixing, commit those changes.

- [ ] **Step 5: Final commit — update CLAUDE.md with ZAP info**

No CLAUDE.md changes needed — the spec and migration are self-documenting.
