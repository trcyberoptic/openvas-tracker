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

- **Dashboard** — Open ticket counts by priority, severity pie chart, trend graph
- **Tickets** — Filterable/sortable list, click for detail page with status changes, notes, activity
- **Scans** — List of imports with detail view showing vulnerabilities
- **Hosts** — Aggregated vulnerability counts per host, expandable rows
- **Settings** — OpenVAS setup guide with masked API key

## Key Components

- `TableFilter` + `SortHeader` — Reusable filter bar and sortable column headers
- `useWebSocket` — Reconnecting WebSocket hook with leak-safe cleanup
- `api/client.ts` — Fetch wrapper with JWT auth and automatic 401 redirect to login

## Build Output

`npm run build` produces `dist/` which is copied to `cmd/openvas-tracker/static/` and embedded in the Go binary via `//go:embed`.
