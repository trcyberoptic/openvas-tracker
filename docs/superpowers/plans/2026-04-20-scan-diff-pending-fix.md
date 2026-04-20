# Scan Diff: Split "fixed" into `fixed` / `pending_fix` / `risk_accepted` — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make the scan-diff page distinguish between actually-fixed tickets, tickets still open/pending auto-resolve, and tickets closed via risk acceptance — instead of lumping them all as "fixed".

**Architecture:** Pure read-side change. The backend query in `diffScansCompat` gets a `LEFT JOIN` onto the matching ticket (via the ticket's current vulnerability record matched by host + CVE/title fingerprint) and a `CASE` that returns one of `fixed`, `pending_fix`, `risk_accepted`. The frontend gets two extra badge styles and two extra filter buttons.

**Tech Stack:** Go 1.22 + `database/sql` + MariaDB 10.11 (backend); React 19 + Vite + Tailwind (frontend). No new deps, no schema migration.

**Test approach:** The repo has no DB-integration-test scaffolding (no sqlmock, no test MariaDB container) and no frontend tests — we follow that convention. Verification is manual smoke-test via `make dev` against the real database, using a known pair of scans that triggered auto-resolve and risk-accept in the past.

---

## File Structure

- **Modify:** [internal/database/queries/scans.go](../../../internal/database/queries/scans.go) — `ScanDiffEntry` doc comment, `diffScansCompat` query, `DiffScans` query's `ORDER BY`.
- **Modify:** [frontend/src/pages/ScanDiff.tsx](../../../frontend/src/pages/ScanDiff.tsx) — `BADGE` map, new `LABEL` map, `counts` memo, filter buttons JSX.
- **No new files.**

---

### Task 1: Backend — extend `ScanDiffEntry` comment

**Files:**
- Modify: `internal/database/queries/scans.go:134`

- [ ] **Step 1: Update the status-value comment so future readers know all five values**

Replace line 134:

```go
	Status       string  `json:"status"` // "new", "fixed", "unchanged"
```

with:

```go
	Status       string  `json:"status"` // "new", "fixed", "pending_fix", "risk_accepted", "unchanged"
```

- [ ] **Step 2: Verify the Go file still compiles**

Run: `go build ./internal/database/queries/`
Expected: no output, exit 0.

- [ ] **Step 3: Commit**

```bash
git add internal/database/queries/scans.go
git commit -m "refactor(scans): document new scan-diff status values"
```

---

### Task 2: Backend — rewrite the `fixed` branch of `diffScansCompat`

**Files:**
- Modify: `internal/database/queries/scans.go:186-230`

- [ ] **Step 1: Replace the entire `diffScansCompat` function body**

The current function at [internal/database/queries/scans.go:186](../../../internal/database/queries/scans.go#L186) has three `SELECT`s joined by `UNION ALL`: `new`, `fixed`, `unchanged`. We replace ONLY the `fixed` block and update the `ORDER BY` to include the new status values. The SQL argument list grows by zero (the ticket join is correlated).

Open `internal/database/queries/scans.go` and locate the `diffScansCompat` function. Replace the entire function (lines 186-231) with:

```go
func (q *Queries) diffScansCompat(ctx context.Context, oldScanID, newScanID string) ([]ScanDiffEntry, error) {
	const query = `
		SELECT 'new' as status, n.id, n.title, n.affected_host, n.hostname, n.severity, n.cvss_score, n.cve_id
		FROM vulnerabilities n
		WHERE n.scan_id = ?
		AND NOT EXISTS (
			SELECT 1 FROM vulnerabilities o WHERE o.scan_id = ?
			AND COALESCE(o.affected_host,'') = COALESCE(n.affected_host,'')
			AND (
				(o.cve_id IS NOT NULL AND o.cve_id != '' AND o.cve_id = n.cve_id)
				OR ((o.cve_id IS NULL OR o.cve_id = '') AND (n.cve_id IS NULL OR n.cve_id = '') AND o.title = n.title)
			)
		)
		UNION ALL
		SELECT
			COALESCE(
				CASE t.status
					WHEN 'fixed'              THEN 'fixed'
					WHEN 'risk_accepted'      THEN 'risk_accepted'
					WHEN 'false_positive'     THEN 'risk_accepted'
					WHEN 'open'               THEN 'pending_fix'
					WHEN 'pending_resolution' THEN 'pending_fix'
				END,
				'fixed'
			) as status,
			o.id, o.title, o.affected_host, o.hostname, o.severity, o.cvss_score, o.cve_id
		FROM vulnerabilities o
		LEFT JOIN vulnerabilities tv
			ON tv.affected_host = o.affected_host
			AND (
				(tv.cve_id IS NOT NULL AND tv.cve_id != '' AND tv.cve_id = o.cve_id)
				OR ((tv.cve_id IS NULL OR tv.cve_id = '') AND (o.cve_id IS NULL OR o.cve_id = '') AND tv.title = o.title)
			)
		LEFT JOIN tickets t ON t.vulnerability_id = tv.id
		WHERE o.scan_id = ?
		AND NOT EXISTS (
			SELECT 1 FROM vulnerabilities n WHERE n.scan_id = ?
			AND COALESCE(n.affected_host,'') = COALESCE(o.affected_host,'')
			AND (
				(n.cve_id IS NOT NULL AND n.cve_id != '' AND n.cve_id = o.cve_id)
				OR ((n.cve_id IS NULL OR n.cve_id = '') AND (o.cve_id IS NULL OR o.cve_id = '') AND n.title = o.title)
			)
		)
		UNION ALL
		SELECT 'unchanged' as status, n.id, n.title, n.affected_host, n.hostname, n.severity, n.cvss_score, n.cve_id
		FROM vulnerabilities n
		WHERE n.scan_id = ?
		AND EXISTS (
			SELECT 1 FROM vulnerabilities o WHERE o.scan_id = ?
			AND COALESCE(o.affected_host,'') = COALESCE(n.affected_host,'')
			AND (
				(o.cve_id IS NOT NULL AND o.cve_id != '' AND o.cve_id = n.cve_id)
				OR ((o.cve_id IS NULL OR o.cve_id = '') AND (n.cve_id IS NULL OR n.cve_id = '') AND o.title = n.title)
			)
		)
		ORDER BY FIELD(status, 'new', 'pending_fix', 'fixed', 'risk_accepted', 'unchanged'), cvss_score DESC`

	rows, err := q.db.QueryContext(ctx, query, newScanID, oldScanID, oldScanID, newScanID, newScanID, oldScanID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanDiffRows(rows)
}
```

Key points:
- **Argument count is unchanged** — six `?` placeholders, same order.
- The `LEFT JOIN tv` finds any vulnerability record with the same host+CVE-or-title as the old-scan vuln `o`. The `LEFT JOIN tickets t ON t.vulnerability_id = tv.id` then grabs the one ticket pointing at any such vuln. If a ticket's `vulnerability_id` is NULL (orphan — see the CLAUDE.md gotcha about `ON DELETE SET NULL`), the join misses and the `COALESCE` fallback returns `'fixed'`.
- The `CASE` covers all five ticket statuses defined in the codebase. If a future status value is added without updating this `CASE`, it falls through to NULL and then `COALESCE` → `'fixed'` (safe default).
- `ORDER BY FIELD(...)` displays rows grouped by severity-of-action: new first, then pending, then done, then risk-accepted, then unchanged.

- [ ] **Step 2: Leave `DiffScans` (the unreachable `FULL OUTER JOIN` variant) untouched**

`DiffScans` at [internal/database/queries/scans.go:145](../../../internal/database/queries/scans.go#L145) is dead on MariaDB — the query errors and code falls through to `diffScansCompat`. Don't rewrite it.

- [ ] **Step 3: Verify the Go package still compiles**

Run: `go build ./...`
Expected: no output, exit 0.

- [ ] **Step 4: Run the existing test suite to catch accidental regressions**

Run: `go test ./... -count=1`
Expected: PASS (no query-layer tests exist, but the rest of the suite should still be green).

- [ ] **Step 5: Commit**

```bash
git add internal/database/queries/scans.go
git commit -m "feat(scans): split diff 'fixed' bucket by ticket status"
```

---

### Task 3: Frontend — badge + label maps for new statuses

**Files:**
- Modify: `frontend/src/pages/ScanDiff.tsx:5-9`

- [ ] **Step 1: Extend the `BADGE` map and add a new `LABEL` map**

Current lines 5-9:

```tsx
const BADGE: Record<string, string> = {
  new: 'bg-red-900 text-red-300',
  fixed: 'bg-green-900 text-green-300',
  unchanged: 'bg-slate-700 text-slate-300',
}
```

Replace with:

```tsx
const BADGE: Record<string, string> = {
  new: 'bg-red-900 text-red-300',
  pending_fix: 'bg-amber-900 text-amber-300',
  fixed: 'bg-green-900 text-green-300',
  risk_accepted: 'bg-sky-900 text-sky-300',
  unchanged: 'bg-slate-700 text-slate-300',
}
const LABEL: Record<string, string> = {
  new: 'new',
  pending_fix: 'pending fix',
  fixed: 'fixed',
  risk_accepted: 'risk accepted',
  unchanged: 'unchanged',
}
```

- [ ] **Step 2: Use `LABEL[d.status]` in the badge rendering at line 157**

Find:

```tsx
<td className="p-3"><span className={`px-2 py-1 rounded text-xs ${BADGE[d.status]}`}>{d.status}</span></td>
```

Replace with:

```tsx
<td className="p-3"><span className={`px-2 py-1 rounded text-xs ${BADGE[d.status] ?? BADGE.unchanged}`}>{LABEL[d.status] ?? d.status}</span></td>
```

(Defensive `?? BADGE.unchanged` / `?? d.status` in case a status value arrives that we haven't mapped — renders as the raw string in neutral slate, rather than an invisible badge.)

- [ ] **Step 3: Verify the dev build still typechecks**

Run: `cd frontend && npm run build`
Expected: exit 0, `dist/` directory produced.

---

### Task 4: Frontend — counts + filter buttons

**Files:**
- Modify: `frontend/src/pages/ScanDiff.tsx:80-87` (counts memo)
- Modify: `frontend/src/pages/ScanDiff.tsx:128-142` (filter buttons)

- [ ] **Step 1: Extend the `counts` memo**

Current lines 80-87:

```tsx
const counts = useMemo(() => {
  if (!diff) return { new: 0, fixed: 0, unchanged: 0 }
  return {
    new: diff.filter(d => d.status === 'new').length,
    fixed: diff.filter(d => d.status === 'fixed').length,
    unchanged: diff.filter(d => d.status === 'unchanged').length,
  }
}, [diff])
```

Replace with:

```tsx
const counts = useMemo(() => {
  if (!diff) return { new: 0, pending_fix: 0, fixed: 0, risk_accepted: 0, unchanged: 0 }
  return {
    new: diff.filter(d => d.status === 'new').length,
    pending_fix: diff.filter(d => d.status === 'pending_fix').length,
    fixed: diff.filter(d => d.status === 'fixed').length,
    risk_accepted: diff.filter(d => d.status === 'risk_accepted').length,
    unchanged: diff.filter(d => d.status === 'unchanged').length,
  }
}, [diff])
```

- [ ] **Step 2: Add two filter buttons (Pending Fix, Risk Accepted)**

Current lines 128-142:

```tsx
<div className="flex gap-3 mb-4">
  <button onClick={() => setFilter('')} className={`px-3 py-1.5 rounded text-sm ${!filter ? 'bg-blue-600 text-white' : 'bg-slate-800 text-slate-400'}`}>
    All ({diff.length})
  </button>
  <button onClick={() => setFilter('new')} className={`px-3 py-1.5 rounded text-sm ${filter === 'new' ? 'bg-red-600 text-white' : 'bg-slate-800 text-slate-400'}`}>
    New ({counts.new})
  </button>
  <button onClick={() => setFilter('fixed')} className={`px-3 py-1.5 rounded text-sm ${filter === 'fixed' ? 'bg-green-600 text-white' : 'bg-slate-800 text-slate-400'}`}>
    Fixed ({counts.fixed})
  </button>
  <button onClick={() => setFilter('unchanged')} className={`px-3 py-1.5 rounded text-sm ${filter === 'unchanged' ? 'bg-slate-600 text-white' : 'bg-slate-800 text-slate-400'}`}>
    Unchanged ({counts.unchanged})
  </button>
</div>
```

Replace with:

```tsx
<div className="flex gap-3 mb-4">
  <button onClick={() => setFilter('')} className={`px-3 py-1.5 rounded text-sm ${!filter ? 'bg-blue-600 text-white' : 'bg-slate-800 text-slate-400'}`}>
    All ({diff.length})
  </button>
  <button onClick={() => setFilter('new')} className={`px-3 py-1.5 rounded text-sm ${filter === 'new' ? 'bg-red-600 text-white' : 'bg-slate-800 text-slate-400'}`}>
    New ({counts.new})
  </button>
  <button onClick={() => setFilter('pending_fix')} className={`px-3 py-1.5 rounded text-sm ${filter === 'pending_fix' ? 'bg-amber-600 text-white' : 'bg-slate-800 text-slate-400'}`}>
    Pending Fix ({counts.pending_fix})
  </button>
  <button onClick={() => setFilter('fixed')} className={`px-3 py-1.5 rounded text-sm ${filter === 'fixed' ? 'bg-green-600 text-white' : 'bg-slate-800 text-slate-400'}`}>
    Fixed ({counts.fixed})
  </button>
  <button onClick={() => setFilter('risk_accepted')} className={`px-3 py-1.5 rounded text-sm ${filter === 'risk_accepted' ? 'bg-sky-600 text-white' : 'bg-slate-800 text-slate-400'}`}>
    Risk Accepted ({counts.risk_accepted})
  </button>
  <button onClick={() => setFilter('unchanged')} className={`px-3 py-1.5 rounded text-sm ${filter === 'unchanged' ? 'bg-slate-600 text-white' : 'bg-slate-800 text-slate-400'}`}>
    Unchanged ({counts.unchanged})
  </button>
</div>
```

- [ ] **Step 3: Verify the frontend build**

Run: `cd frontend && npm run build`
Expected: exit 0, no TypeScript errors.

- [ ] **Step 4: Commit**

```bash
git add frontend/src/pages/ScanDiff.tsx
git commit -m "feat(scan-diff): show pending-fix and risk-accepted status filters"
```

---

### Task 5: Manual verification via `make dev`

**Files:** none — smoke-test against the running app.

- [ ] **Step 1: Start backend + frontend dev servers**

Run: `make dev`
Expected: backend listens on :8080, frontend on :5173 proxying API to :8080. Leave both running in the background.

- [ ] **Step 2: Log in and open the Scan Diff page**

- Visit http://localhost:5173/
- Log in (admin + `OT_ADMIN_PASSWORD`).
- Click "Scan Diff" in the sidebar.

- [ ] **Step 3: Pick two OpenVAS scans with a known diff**

Pick the two most-recent OpenVAS scans (the page auto-selects them). Expected: the row count summary now shows five buckets — `All`, `New`, `Pending Fix`, `Fixed`, `Risk Accepted`, `Unchanged`.

- [ ] **Step 4: Verify each bucket is reachable and non-empty where expected**

Against the current production data: there should be at least some `pending_fix` rows (tickets still `open` but not found in the newest scan) because the auto-resolve threshold is 3. Click each filter button and confirm the table rows match the badge.

- [ ] **Step 5: Spot-check one `pending_fix` row against the tickets page**

Copy a `pending_fix` row's host + title. Navigate to All Tickets, filter by that host, and confirm a matching ticket exists with status `open` or `pending_resolution`. This proves the SQL join is actually picking up ticket status correctly (and not just returning the string literal).

- [ ] **Step 6: Spot-check a `fixed` row**

Find a `fixed` row. Confirm on the tickets page that the corresponding ticket has status `fixed` (may need to toggle "Show fixed" or filter status=fixed).

- [ ] **Step 7: Spot-check a `risk_accepted` row, if any**

If the `Risk Accepted ( > 0 )` button is present, click one, copy a row, and confirm the ticket has status `risk_accepted` or `false_positive`.

- [ ] **Step 8: Stop dev servers**

Ctrl-C the `make dev` process.

---

### Task 6: Production smoke verification (optional — only if deploying)

**Files:** none — run after `/deploy` per the CLAUDE.md guidance.

- [ ] **Step 1: Deploy**

Run `/deploy` skill.

- [ ] **Step 2: Verify on production URL**

Open the Scan Diff page in production and repeat the five-bucket sanity check from Task 5 Step 3.

---

## Out of scope (for this plan)

- Changing ZAP fingerprinting in the diff (currently falls back to host+title; proper fix would use host+CWE+URL+param — separate plan).
- Making `DiffScans` work on MariaDB.
- Adding DB-integration test infrastructure (no sqlmock, no test DB container in this repo).
- Adding frontend component tests (none exist in this project).
