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

- **Dashboard** — Open ticket counts by priority, severity pie chart, trend graph, quick links to "My Tickets" and "Unassigned"
- **Tickets** — Filterable/sortable list with checkbox bulk actions (status change, assign), CVSS-sorted by default, full-text search across all columns. URL params: `?assigned=me` / `?assigned=unassigned`
- **Ticket Detail** — Status change buttons, assignment, notes, activity log, also-affected hosts, CVE reference links (NVD/MITRE/Google), risk acceptance with expiry date
- **Scans** — List of imports with detail view showing vulnerabilities
- **Scan Diff** — Compare two scans: new / fixed / unchanged findings with filter buttons. Defaults to two most recent scans
- **Hosts** — Aggregated vulnerability counts + ticket status breakdown per host, expandable rows showing tickets (clickable → ticket detail), hostname resolution
- **Settings** — Profile info, OpenVAS setup guide, .env config editor (all variables), LDAP configuration with test button

## Key Components

- `TableFilter` + `SortHeader` — Reusable filter bar and sortable column headers
- `useWebSocket` — Reconnecting WebSocket hook with leak-safe cleanup
- `api/client.ts` — Fetch wrapper with JWT auth, automatic 401 redirect, Content-Type only on body requests

## Auth

Login by username (not email). Supports admin user and LDAP/AD credentials. No registration page.

## Build Output

`npm run build` produces `dist/` which is copied to `cmd/openvas-tracker/static/` and embedded in the Go binary via `//go:embed`.
