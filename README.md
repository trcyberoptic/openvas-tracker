# VulnTrack Pro

Single-binary vulnerability management platform built in Go.

## Features

- Nmap and OpenVAS scanning with async job queue
- Vulnerability tracking with CVE enrichment and risk scoring
- Ticket system with comments and priority management
- Report generation (HTML, PDF, Excel, Markdown)
- Real-time scan updates via WebSocket
- Team collaboration with invitations and RBAC
- Scheduled scans with cron expressions
- Asset inventory with auto-discovery
- Full audit logging
- Plugin system for custom scanners
- Embedded React dashboard — no separate frontend deploy

## Quick Start

### Prerequisites
- Go 1.23+, Node 22+, PostgreSQL 16+, Redis 7+
- nmap and/or gvm-cli installed

### Development
```bash
cp .env.example .env    # edit with your DB/Redis credentials
make migrate-up
make dev                # starts backend + frontend dev server
```

### Production Build
```bash
make build-linux        # produces bin/vulntrack-linux-amd64
```

### Deploy on Debian Trixie
```bash
sudo bash deploy/install.sh bin/vulntrack-linux-amd64
sudo vim /etc/vulntrack/env   # set secrets
sudo systemctl restart vulntrack
```

## API

All endpoints under `/api/` require `Authorization: Bearer <token>` except `/api/auth/*` and `/api/health`.

| Method | Path | Description |
|--------|------|-------------|
| POST | /api/auth/register | Register user |
| POST | /api/auth/login | Login, get JWT |
| GET/POST | /api/targets | List/create targets |
| GET/POST | /api/scans | List/launch scans |
| GET | /api/vulnerabilities | List vulnerabilities |
| GET/POST | /api/tickets | List/create tickets |
| GET/POST | /api/reports | List/generate reports |
| GET | /api/dashboard | Dashboard metrics |
| GET/POST | /api/teams | List/create teams |
| GET | /api/notifications | List notifications |
| GET/POST | /api/schedules | List/create schedules |
| GET | /api/assets | List discovered assets |
| GET | /api/audit | Audit log |
| GET | /api/search?q= | Global search |
| WS | /ws?token= | Real-time updates |
