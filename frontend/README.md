# OpenVAS-Tracker Frontend

React 19 + Vite + Tailwind CSS embedded SPA for the OpenVAS-Tracker dashboard.

## Development

```bash
npm ci          # install dependencies
npm run dev     # start dev server with HMR on port 5173
npm run build   # production build → dist/
npm run lint    # ESLint check
```

The dev server proxies `/api/*` and `/ws` requests to the Go backend on port 8080.

## Pages

- **Dashboard** — Open ticket counts by priority, severity pie chart, 30-day trend of open tickets, quick links to "My Tickets" and "Unassigned"
- **My Tickets** — Filtered view of tickets assigned to current user (default: open)
- **All Tickets** — Full ticket list with checkbox bulk actions (status change, assign), CVSS-sorted by default, full-text search across all columns, default filter on open
- **Ticket Detail** — Affected host + also-affected hosts, change status/assignment, prominent CVSS score box, risk accept rule creation (this host / all hosts), CVE reference links (NVD/MITRE/Google), description, notes, activity log
- **Scans** — List of imports with detail view
- **Scan Diff** — Compare two scans: new / fixed / unchanged. Defaults to two most recent
- **Auto-Accept Rules** — List and delete risk accept rules, shows finding, scope, reason, expiry. "Refresh Tickets" button re-applies all rules to existing open tickets
- **Settings** — Profile, OpenVAS setup guide, .env config editor, LDAP config with test button
- **Sidebar** — Navigation with GitHub repo link at the bottom

## Key Components

- `TableFilter` + `SortHeader` — Reusable filter bar with defaults and sortable column headers
- `useWebSocket` — Reconnecting WebSocket hook with leak-safe cleanup
- `api/client.ts` — Fetch wrapper with JWT auth, automatic 401 redirect

## Auth

Login by username. Supports admin user and LDAP/AD credentials. No registration.

## Build Output

`npm run build` produces `dist/` which is copied to `cmd/openvas-tracker/static/` and embedded in the Go binary via `//go:embed`.
