# Scan Diff: Split "fixed" into `fixed`, `pending_fix`, `risk_accepted`

## Problem

On the Scan Diff page, every vulnerability present in the old scan but missing from the new scan is labelled `fixed`. That lumps together three very different realities:

1. The ticket is genuinely closed (`status = 'fixed'`).
2. The ticket is still `open` or `pending_resolution` — the finding just didn't appear in the latest scan and hasn't crossed the `OT_AUTORESOLVE_THRESHOLD` yet (flapping / below threshold).
3. The ticket is closed via risk acceptance (`risk_accepted` or `false_positive`) — the risk is consciously accepted, not fixed.

Users reading the green "fixed" column believe those items are remediated when most of them aren't.

## Goals

- Split the existing `fixed` bucket in the scan-diff output into three distinct buckets so the display reflects the actual ticket state.
- Reuse the existing matching logic (host + CVE/title); no changes to how we detect that a vulnerability is "missing from the new scan".
- Keep the change minimal — one SQL join, one new SQL `CASE`, one new badge/filter pair on the frontend.

## Non-goals

- No change to the `new` or `unchanged` bucket semantics.
- No improvement to ZAP diff fingerprinting (pre-existing weakness — ZAP diff falls back to host+title because ZAP findings rarely have CVEs).
- No change to the preferred-but-unreachable `DiffScans` path using `FULL OUTER JOIN` (MariaDB doesn't support it; code always falls through to `diffScansCompat`).
- No backfill / migration — the change is read-only; status values are computed at query time.

## Status Mapping

For every vulnerability in the old scan that is missing from the new scan, we look up the matching ticket (by `affected_host` + CVE-or-title) and emit one of these status values:

| Ticket status                              | Diff status       |
|--------------------------------------------|-------------------|
| `fixed`                                    | `fixed`           |
| `open`, `pending_resolution`               | `pending_fix`     |
| `risk_accepted`, `false_positive`          | `risk_accepted`   |
| no matching ticket found (fallback)        | `fixed`           |

The "no matching ticket" fallback keeps behaviour sane if the ticket was deleted or the vulnerability predates ticket creation (shouldn't happen in normal flow — tickets are created on first import — but it protects us from NULLs).

## Backend Change

File: [internal/database/queries/scans.go](../../../internal/database/queries/scans.go)

### `ScanDiffEntry` struct

No new fields required. The existing `Status string` field accepts the new values `pending_fix` and `risk_accepted`. Update the comment on the struct to list all five possible values.

### `diffScansCompat` — `fixed` branch only

Current (lines 199-210):

```sql
SELECT 'fixed' as status, o.id, o.title, o.affected_host, o.hostname, o.severity, o.cvss_score, o.cve_id
FROM vulnerabilities o
WHERE o.scan_id = ?
AND NOT EXISTS ( ... same-host same-CVE-or-title check against new scan ... )
```

New:

```sql
SELECT
    COALESCE(
        CASE t.status
            WHEN 'fixed'          THEN 'fixed'
            WHEN 'risk_accepted'  THEN 'risk_accepted'
            WHEN 'false_positive' THEN 'risk_accepted'
            WHEN 'open'           THEN 'pending_fix'
            WHEN 'pending_resolution' THEN 'pending_fix'
        END,
        'fixed'
    ) AS status,
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
AND NOT EXISTS ( ... unchanged ... )
```

Notes:

- We match tickets via the ticket's *current* `vulnerability_id` (which points to the most recent vuln record for that finding) rather than via `o.id` directly — because the ticket may have been touched by a later scan after the "old" scan, so `t.vulnerability_id` probably no longer equals `o.id`.
- If multiple tickets match (shouldn't happen — dedup prevents it — but defensively), we accept any one; the join is effectively `LIMIT 1` per `o` row because ticket fingerprints are unique per host.
- If the ticket row points to a vuln that's been deleted (orphan ticket — can happen per CLAUDE.md gotcha about `ON DELETE SET NULL`), the LEFT JOIN on `tv` misses and we fall back to `'fixed'`. Acceptable.

### `ORDER BY`

Extend `FIELD(status, 'new', 'fixed', 'unchanged')` to `FIELD(status, 'new', 'pending_fix', 'fixed', 'risk_accepted', 'unchanged')` — display order mirrors severity of user action required: new issues first, then pending action, then done, then accepted, then boring unchanged rows.

## Frontend Change

File: [frontend/src/pages/ScanDiff.tsx](../../../frontend/src/pages/ScanDiff.tsx)

### Badge map (line 5)

Add:
```ts
pending_fix: 'bg-amber-900 text-amber-300',
risk_accepted: 'bg-sky-900 text-sky-300',
```

### Filter buttons (lines 128-142)

Add two more buttons (between `Fixed` and `Unchanged`):
- `Pending Fix (counts.pending_fix)` — amber when active.
- `Risk Accepted (counts.risk_accepted)` — sky-blue when active.

### `counts` memo (lines 80-87)

Add the two new keys.

### Badge label rendering

`{d.status}` produces the text inside the badge. Replace with a prettified map so `pending_fix` shows as `"pending fix"` and `risk_accepted` shows as `"risk accepted"`. Single `LABEL` lookup object at the top of the file.

## Testing

- Unit test in [internal/database/queries/scans_test.go](../../../internal/database/queries/scans_test.go) — add cases: (a) vuln missing from new scan with ticket status=`fixed` → `fixed`; (b) with status=`open` → `pending_fix`; (c) with status=`pending_resolution` → `pending_fix`; (d) with status=`risk_accepted` → `risk_accepted`; (e) with status=`false_positive` → `risk_accepted`; (f) ticket deleted → `fixed`.
- Smoke-test in the UI against a real diff to confirm counts sum correctly and filters work.

## Risk / Compatibility

- The API response shape is unchanged (same JSON fields). New status string values are backwards-compatible: old frontend ignoring them would render an empty badge class, but we deploy both changes together.
- No schema migration.
- No behaviour change to import, auto-resolve, or anywhere else — this is purely a read-side query change.
