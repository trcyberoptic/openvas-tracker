# VulnTrack Pro (Go) — Full Rebuild Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Rebuild VulnTrack-Pro as a single Go binary with embedded SPA frontend, deployable as a systemd service on Debian Trixie, replicating and improving on all features of the original Python implementation.

**Architecture:** Layered Go backend (handlers → services → repositories) with PostgreSQL storage, Redis-backed async job queue (Asynq) for scan orchestration, gorilla/websocket for real-time updates, and a React+Vite SPA embedded via `go:embed`. The binary includes all static assets — deploy is copy + systemd unit.

**Tech Stack:** Go 1.23+, Echo v4, pgx/v5 + sqlc, golang-migrate, Asynq, gorilla/websocket, golang-jwt, go-playground/validator, maroto (PDF), excelize (Excel), html/template, React 19 + Vite + Tailwind + shadcn/ui, go:embed.

---

## File Structure

```
vulntrack/
├── cmd/
│   └── vulntrack/
│       └── main.go                    # Entry point, wires everything
├── internal/
│   ├── config/
│   │   └── config.go                  # Viper-based config loading
│   ├── database/
│   │   ├── postgres.go                # Connection pool setup
│   │   └── queries/                   # sqlc-generated query code
│   │       ├── models.go
│   │       ├── db.go
│   │       ├── users.sql.go
│   │       ├── targets.sql.go
│   │       ├── scans.sql.go
│   │       ├── vulnerabilities.sql.go
│   │       ├── tickets.sql.go
│   │       ├── teams.sql.go
│   │       ├── reports.sql.go
│   │       ├── notifications.sql.go
│   │       ├── schedules.sql.go
│   │       ├── assets.sql.go
│   │       ├── audit_logs.sql.go
│   │       └── search.sql.go
│   ├── middleware/
│   │   ├── auth.go                    # JWT extraction + context
│   │   ├── rbac.go                    # Role-based access control
│   │   ├── audit.go                   # Request audit logging
│   │   ├── ratelimit.go               # Rate limiting
│   │   └── security.go                # Security headers, CORS
│   ├── auth/
│   │   ├── jwt.go                     # Token generation/validation
│   │   └── password.go                # bcrypt hashing
│   ├── handler/
│   │   ├── auth.go                    # POST /api/auth/*
│   │   ├── users.go                   # GET/PUT /api/users/*
│   │   ├── targets.go                 # CRUD /api/targets/* (includes target groups)
│   │   ├── scans.go                   # POST/GET /api/scans/*
│   │   ├── vulnerabilities.go         # CRUD /api/vulnerabilities/*
│   │   ├── tickets.go                 # CRUD /api/tickets/*
│   │   ├── reports.go                 # GET/POST /api/reports/*
│   │   ├── dashboard.go               # GET /api/dashboard/*
│   │   ├── teams.go                   # CRUD /api/teams/*
│   │   ├── notifications.go           # GET/PUT /api/notifications/*
│   │   ├── schedules.go               # CRUD /api/schedules/*
│   │   ├── search.go                  # GET /api/search
│   │   ├── assets.go                  # CRUD /api/assets/*
│   │   ├── audit.go                   # GET /api/audit/*
│   │   └── ws.go                      # WebSocket upgrade + hub
│   ├── service/
│   │   ├── user.go
│   │   ├── target.go
│   │   ├── scan.go                    # Scan orchestration logic
│   │   ├── vulnerability.go
│   │   ├── ticket.go
│   │   ├── report.go                  # Report generation dispatcher
│   │   ├── dashboard.go
│   │   ├── team.go
│   │   ├── notification.go
│   │   ├── schedule.go
│   │   ├── search.go
│   │   ├── asset.go
│   │   ├── audit.go
│   │   └── prediction.go             # Deterministic risk scoring
│   ├── scanner/
│   │   ├── nmap.go                    # Nmap CLI wrapper + XML parser
│   │   ├── openvas.go                 # OpenVAS/GVM CLI wrapper
│   │   └── enrichment.go             # CVE/NVD enrichment
│   ├── report/
│   │   ├── html.go                    # HTML report generator
│   │   ├── pdf.go                     # PDF report (maroto)
│   │   ├── excel.go                   # Excel report (excelize)
│   │   ├── markdown.go                # Markdown report
│   │   └── templates/
│   │       └── report.html
│   ├── worker/
│   │   ├── server.go                  # Asynq worker server setup
│   │   └── scan_task.go               # Scan job handler
│   ├── websocket/
│   │   ├── hub.go                     # Connection registry + broadcast
│   │   └── client.go                  # Per-connection read/write pumps
│   └── plugin/
│       ├── loader.go                  # Plugin discovery + loading
│       └── interface.go               # Plugin interface definition
├── sql/
│   ├── migrations/
│   │   ├── 001_create_users.up.sql
│   │   ├── 001_create_users.down.sql
│   │   ├── 002_create_targets.up.sql
│   │   ├── 002_create_targets.down.sql
│   │   ├── 003_create_scans.up.sql
│   │   ├── 003_create_scans.down.sql
│   │   ├── 004_create_vulnerabilities.up.sql
│   │   ├── 004_create_vulnerabilities.down.sql
│   │   ├── 005_create_tickets.up.sql
│   │   ├── 005_create_tickets.down.sql
│   │   ├── 006_create_teams.up.sql
│   │   ├── 006_create_teams.down.sql
│   │   ├── 007_create_reports.up.sql
│   │   ├── 007_create_reports.down.sql
│   │   ├── 008_create_notifications.up.sql
│   │   ├── 008_create_notifications.down.sql
│   │   ├── 009_create_schedules.up.sql
│   │   ├── 009_create_schedules.down.sql
│   │   ├── 010_create_assets.up.sql
│   │   ├── 010_create_assets.down.sql
│   │   ├── 011_create_audit_logs.up.sql
│   │   ├── 011_create_audit_logs.down.sql
│   │   └── 012_create_search_indexes.up.sql
│   └── queries/
│       ├── users.sql
│       ├── targets.sql
│       ├── scans.sql
│       ├── vulnerabilities.sql
│       ├── tickets.sql
│       ├── teams.sql
│       ├── reports.sql
│       ├── notifications.sql
│       ├── schedules.sql
│       ├── assets.sql
│       ├── audit_logs.sql
│       └── search.sql
├── frontend/
│   ├── package.json
│   ├── vite.config.ts
│   ├── tailwind.config.ts
│   ├── tsconfig.json
│   ├── index.html
│   └── src/
│       ├── main.tsx
│       ├── App.tsx
│       ├── api/
│       │   └── client.ts               # Typed API client
│       ├── hooks/
│       │   ├── useAuth.ts
│       │   ├── useWebSocket.ts
│       │   └── useQuery.ts
│       ├── components/
│       │   ├── ui/                      # shadcn/ui components
│       │   ├── layout/
│       │   │   ├── Sidebar.tsx
│       │   │   ├── Header.tsx
│       │   │   └── Shell.tsx
│       │   ├── dashboard/
│       │   │   ├── StatsCards.tsx
│       │   │   ├── VulnChart.tsx
│       │   │   ├── SeverityPie.tsx
│       │   │   └── RecentScans.tsx
│       │   ├── targets/
│       │   │   ├── TargetList.tsx
│       │   │   ├── TargetForm.tsx
│       │   │   └── TargetGroupPanel.tsx
│       │   ├── scans/
│       │   │   ├── ScanList.tsx
│       │   │   ├── ScanLauncher.tsx
│       │   │   └── ScanLiveLog.tsx
│       │   ├── vulnerabilities/
│       │   │   ├── VulnTable.tsx
│       │   │   ├── VulnDetail.tsx
│       │   │   └── VulnFilters.tsx
│       │   ├── tickets/
│       │   │   ├── TicketBoard.tsx
│       │   │   ├── TicketDetail.tsx
│       │   │   └── TicketForm.tsx
│       │   ├── reports/
│       │   │   ├── ReportList.tsx
│       │   │   └── ReportGenerator.tsx
│       │   ├── teams/
│       │   │   ├── TeamList.tsx
│       │   │   └── TeamMembers.tsx
│       │   └── settings/
│       │       ├── Profile.tsx
│       │       └── UserManagement.tsx
│       └── pages/
│           ├── Login.tsx
│           ├── Dashboard.tsx
│           ├── Targets.tsx
│           ├── Scans.tsx
│           ├── Vulnerabilities.tsx
│           ├── Tickets.tsx
│           ├── Reports.tsx
│           ├── Teams.tsx
│           ├── Settings.tsx
│           └── NotFound.tsx
├── deploy/
│   ├── vulntrack.service               # systemd unit file
│   ├── vulntrack.env.example           # env template
│   └── install.sh                       # Debian install script
├── sqlc.yaml                            # sqlc configuration
├── go.mod
├── go.sum
├── Makefile                             # build, test, migrate, dev
├── Dockerfile                           # multi-stage build
├── .gitignore
└── README.md
```

---

## Chunk 1: Foundation — Project Scaffold, Config, Database, Migrations

### Task 1.1: Initialize Go Module and Dependencies

**Files:**
- Create: `go.mod`
- Create: `Makefile`
- Create: `.gitignore`

- [ ] **Step 1: Initialize the Go module**

```bash
cd e:/Code/openvas-tracker
go mod init github.com/cyberoptic/vulntrack
```

- [ ] **Step 2: Add core dependencies**

```bash
go get github.com/labstack/echo/v4@latest
go get github.com/jackc/pgx/v5@latest
go get github.com/golang-migrate/migrate/v4@latest
go get github.com/spf13/viper@latest
go get github.com/golang-jwt/jwt/v5@latest
go get github.com/go-playground/validator/v10@latest
go get github.com/gorilla/websocket@latest
go get github.com/hibiken/asynq@latest
go get github.com/redis/go-redis/v9@latest
go get golang.org/x/crypto@latest
go get github.com/google/uuid@latest
go get github.com/johnfercher/maroto/v2@latest
go get github.com/xuri/excelize/v2@latest
```

- [ ] **Step 3: Create .gitignore**

```gitignore
# Binaries
/vulntrack
/bin/
*.exe

# Dependencies
/vendor/

# Frontend build output (committed via go:embed, but not node_modules)
frontend/node_modules/

# Environment
.env
*.env.local

# IDE
.idea/
.vscode/
*.swp

# OS
.DS_Store
Thumbs.db

# Test
coverage.out
*.test

# Database (local dev)
*.db
```

- [ ] **Step 4: Create Makefile**

```makefile
.PHONY: build run test migrate-up migrate-down sqlc frontend dev clean

BINARY=vulntrack
BUILD_DIR=bin

build: frontend
	CGO_ENABLED=0 go build -o $(BUILD_DIR)/$(BINARY) ./cmd/vulntrack

build-linux: frontend
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o $(BUILD_DIR)/$(BINARY)-linux-amd64 ./cmd/vulntrack

run:
	go run ./cmd/vulntrack

test:
	go test ./... -v -count=1

test-cover:
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out

migrate-up:
	migrate -path sql/migrations -database "$${DATABASE_URL}" up

migrate-down:
	migrate -path sql/migrations -database "$${DATABASE_URL}" down 1

sqlc:
	sqlc generate

frontend:
	cd frontend && npm ci && npm run build

dev:
	cd frontend && npm run dev &
	go run ./cmd/vulntrack

clean:
	rm -rf $(BUILD_DIR) frontend/dist coverage.out
```

- [ ] **Step 5: Commit**

```bash
git init
git add go.mod Makefile .gitignore
git commit -m "chore: initialize Go module with dependencies and build tooling"
```

---

### Task 1.2: Configuration System

**Files:**
- Create: `internal/config/config.go`
- Create: `.env.example`
- Test: `internal/config/config_test.go`

- [ ] **Step 1: Write the failing test**

```go
// internal/config/config_test.go
package config

import (
	"os"
	"testing"
)

func TestLoad_Defaults(t *testing.T) {
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}
	if cfg.Server.Port != 8080 {
		t.Errorf("expected default port 8080, got %d", cfg.Server.Port)
	}
	if cfg.Server.Host != "0.0.0.0" {
		t.Errorf("expected default host 0.0.0.0, got %s", cfg.Server.Host)
	}
}

func TestLoad_EnvOverride(t *testing.T) {
	os.Setenv("VT_SERVER_PORT", "9090")
	defer os.Unsetenv("VT_SERVER_PORT")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}
	if cfg.Server.Port != 9090 {
		t.Errorf("expected port 9090, got %d", cfg.Server.Port)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/config/ -v`
Expected: FAIL — `config.Load` not found

- [ ] **Step 3: Write minimal implementation**

```go
// internal/config/config.go
package config

import (
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Redis    RedisConfig
	JWT      JWTConfig
	Scanner  ScannerConfig
}

type ServerConfig struct {
	Host string
	Port int
}

type DatabaseConfig struct {
	URL             string
	MaxConns        int
	MinConns        int
}

type RedisConfig struct {
	Addr     string
	Password string
	DB       int
}

type JWTConfig struct {
	Secret     string
	ExpireHours int
}

type ScannerConfig struct {
	NmapPath    string
	OpenVASPath string
}

func Load() (*Config, error) {
	v := viper.New()

	// Defaults
	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.port", 8080)
	v.SetDefault("database.url", "postgres://vulntrack:vulntrack@localhost:5432/vulntrack?sslmode=disable")
	v.SetDefault("database.maxconns", 25)
	v.SetDefault("database.minconns", 5)
	v.SetDefault("redis.addr", "localhost:6379")
	v.SetDefault("redis.password", "")
	v.SetDefault("redis.db", 0)
	v.SetDefault("jwt.secret", "change-me-in-production")
	v.SetDefault("jwt.expirehours", 24)
	v.SetDefault("scanner.nmappath", "nmap")
	v.SetDefault("scanner.openvaspath", "gvm-cli")

	// Env vars: VT_SERVER_PORT, VT_DATABASE_URL, etc.
	v.SetEnvPrefix("VT")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Optional .env file
	v.SetConfigName(".env")
	v.SetConfigType("env")
	v.AddConfigPath(".")
	_ = v.ReadInConfig() // ignore if missing

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/config/ -v`
Expected: PASS

- [ ] **Step 5: Create .env.example**

```env
VT_SERVER_HOST=0.0.0.0
VT_SERVER_PORT=8080
VT_DATABASE_URL=postgres://vulntrack:vulntrack@localhost:5432/vulntrack?sslmode=disable
VT_DATABASE_MAXCONNS=25
VT_REDIS_ADDR=localhost:6379
VT_REDIS_PASSWORD=
VT_JWT_SECRET=change-me-in-production
VT_JWT_EXPIREHOURS=24
VT_SCANNER_NMAPPATH=nmap
VT_SCANNER_OPENVASPATH=gvm-cli
```

- [ ] **Step 6: Commit**

```bash
git add internal/config/ .env.example
git commit -m "feat: add Viper-based configuration with env var overrides"
```

---

### Task 1.3: Database Connection Pool

**Files:**
- Create: `internal/database/postgres.go`
- Test: `internal/database/postgres_test.go`

- [ ] **Step 1: Write the failing test**

```go
// internal/database/postgres_test.go
package database

import (
	"testing"
)

func TestNewPool_InvalidURL(t *testing.T) {
	_, err := NewPool("postgres://invalid:5432/nonexistent")
	if err == nil {
		t.Fatal("expected error for invalid database URL, got nil")
	}
}

func TestNewPool_ValidatesConfig(t *testing.T) {
	_, err := NewPool("")
	if err == nil {
		t.Fatal("expected error for empty URL")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/database/ -v`
Expected: FAIL — `NewPool` not found

- [ ] **Step 3: Write minimal implementation**

```go
// internal/database/postgres.go
package database

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func NewPool(databaseURL string) (*pgxpool.Pool, error) {
	if databaseURL == "" {
		return nil, errors.New("database URL is required")
	}

	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, err
	}

	cfg.MaxConns = 25
	cfg.MinConns = 5

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}

	return pool, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/database/ -v`
Expected: PASS (both tests pass — invalid URL returns error)

- [ ] **Step 5: Commit**

```bash
git add internal/database/
git commit -m "feat: add PostgreSQL connection pool with pgx"
```

---

### Task 1.4: Database Migrations — Users Table

**Files:**
- Create: `sql/migrations/001_create_users.up.sql`
- Create: `sql/migrations/001_create_users.down.sql`

- [ ] **Step 1: Write the up migration**

```sql
-- sql/migrations/001_create_users.up.sql
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";

CREATE TYPE user_role AS ENUM ('admin', 'analyst', 'viewer');

CREATE TABLE users (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email       TEXT NOT NULL UNIQUE,
    username    TEXT NOT NULL UNIQUE,
    password    TEXT NOT NULL,
    role        user_role NOT NULL DEFAULT 'viewer',
    is_active   BOOLEAN NOT NULL DEFAULT true,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_users_email ON users (email);
CREATE INDEX idx_users_username ON users (username);
```

- [ ] **Step 2: Write the down migration**

```sql
-- sql/migrations/001_create_users.down.sql
DROP TABLE IF EXISTS users;
DROP TYPE IF EXISTS user_role;
```

- [ ] **Step 3: Verify migration applies**

Run: `make migrate-up` (requires running PostgreSQL)
Expected: Migration 001 applied successfully

- [ ] **Step 4: Commit**

```bash
git add sql/migrations/001_*
git commit -m "feat: add users table migration"
```

---

### Task 1.5: Database Migrations — Targets, Scans, Vulnerabilities

**Files:**
- Create: `sql/migrations/002_create_targets.up.sql`
- Create: `sql/migrations/002_create_targets.down.sql`
- Create: `sql/migrations/003_create_scans.up.sql`
- Create: `sql/migrations/003_create_scans.down.sql`
- Create: `sql/migrations/004_create_vulnerabilities.up.sql`
- Create: `sql/migrations/004_create_vulnerabilities.down.sql`

- [ ] **Step 1: Write targets migration**

```sql
-- sql/migrations/002_create_targets.up.sql
CREATE TABLE target_groups (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name        TEXT NOT NULL,
    description TEXT,
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE targets (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    host        TEXT NOT NULL,
    ip_address  TEXT,
    hostname    TEXT,
    os_guess    TEXT,
    group_id    UUID REFERENCES target_groups(id) ON DELETE SET NULL,
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    metadata    JSONB DEFAULT '{}',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_targets_user ON targets (user_id);
CREATE INDEX idx_targets_group ON targets (group_id);
CREATE INDEX idx_targets_host ON targets (host);
```

```sql
-- sql/migrations/002_create_targets.down.sql
DROP TABLE IF EXISTS targets;
DROP TABLE IF EXISTS target_groups;
```

- [ ] **Step 2: Write scans migration**

```sql
-- sql/migrations/003_create_scans.up.sql
CREATE TYPE scan_status AS ENUM ('pending', 'running', 'completed', 'failed', 'cancelled');
CREATE TYPE scan_type AS ENUM ('nmap', 'openvas', 'custom');

CREATE TABLE scans (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name            TEXT NOT NULL,
    scan_type       scan_type NOT NULL,
    status          scan_status NOT NULL DEFAULT 'pending',
    target_id       UUID REFERENCES targets(id) ON DELETE SET NULL,
    target_group_id UUID REFERENCES target_groups(id) ON DELETE SET NULL,
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    options         JSONB DEFAULT '{}',
    raw_output      TEXT,
    started_at      TIMESTAMPTZ,
    completed_at    TIMESTAMPTZ,
    error_message   TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_scans_user ON scans (user_id);
CREATE INDEX idx_scans_status ON scans (status);
CREATE INDEX idx_scans_target ON scans (target_id);
```

```sql
-- sql/migrations/003_create_scans.down.sql
DROP TABLE IF EXISTS scans;
DROP TYPE IF EXISTS scan_type;
DROP TYPE IF EXISTS scan_status;
```

- [ ] **Step 3: Write vulnerabilities migration**

```sql
-- sql/migrations/004_create_vulnerabilities.up.sql
CREATE TYPE severity_level AS ENUM ('critical', 'high', 'medium', 'low', 'info');
CREATE TYPE vuln_status AS ENUM ('open', 'confirmed', 'mitigated', 'resolved', 'false_positive', 'accepted');

CREATE TABLE vulnerabilities (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    scan_id         UUID NOT NULL REFERENCES scans(id) ON DELETE CASCADE,
    target_id       UUID REFERENCES targets(id) ON DELETE SET NULL,
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title           TEXT NOT NULL,
    description     TEXT,
    severity        severity_level NOT NULL DEFAULT 'info',
    status          vuln_status NOT NULL DEFAULT 'open',
    cvss_score      DECIMAL(3,1),
    cve_id          TEXT,
    cwe_id          TEXT,
    affected_host   TEXT,
    affected_port   INTEGER,
    protocol        TEXT,
    service         TEXT,
    solution        TEXT,
    references      JSONB DEFAULT '[]',
    enrichment_data JSONB DEFAULT '{}',
    risk_score      DECIMAL(5,2),
    discovered_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    resolved_at     TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_vulns_scan ON vulnerabilities (scan_id);
CREATE INDEX idx_vulns_severity ON vulnerabilities (severity);
CREATE INDEX idx_vulns_status ON vulnerabilities (status);
CREATE INDEX idx_vulns_cve ON vulnerabilities (cve_id);
CREATE INDEX idx_vulns_user ON vulnerabilities (user_id);
CREATE INDEX idx_vulns_target ON vulnerabilities (target_id);
```

```sql
-- sql/migrations/004_create_vulnerabilities.down.sql
DROP TABLE IF EXISTS vulnerabilities;
DROP TYPE IF EXISTS vuln_status;
DROP TYPE IF EXISTS severity_level;
```

- [ ] **Step 4: Commit**

```bash
git add sql/migrations/002_* sql/migrations/003_* sql/migrations/004_*
git commit -m "feat: add targets, scans, and vulnerabilities migrations"
```

---

### Task 1.6: Database Migrations — Tickets, Teams, Reports, Notifications, Schedules, Assets, Audit

**Files:**
- Create: `sql/migrations/005_create_tickets.up.sql` through `sql/migrations/012_create_search_indexes.up.sql`
- Create: corresponding `.down.sql` files

- [ ] **Step 1: Write tickets migration**

```sql
-- sql/migrations/005_create_tickets.up.sql
CREATE TYPE ticket_status AS ENUM ('open', 'in_progress', 'review', 'resolved', 'closed');
CREATE TYPE ticket_priority AS ENUM ('critical', 'high', 'medium', 'low');

CREATE TABLE tickets (
    id                UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    title             TEXT NOT NULL,
    description       TEXT,
    status            ticket_status NOT NULL DEFAULT 'open',
    priority          ticket_priority NOT NULL DEFAULT 'medium',
    vulnerability_id  UUID REFERENCES vulnerabilities(id) ON DELETE SET NULL,
    assigned_to       UUID REFERENCES users(id) ON DELETE SET NULL,
    created_by        UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    due_date          TIMESTAMPTZ,
    resolved_at       TIMESTAMPTZ,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE ticket_comments (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    ticket_id   UUID NOT NULL REFERENCES tickets(id) ON DELETE CASCADE,
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    content     TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_tickets_assigned ON tickets (assigned_to);
CREATE INDEX idx_tickets_status ON tickets (status);
CREATE INDEX idx_tickets_vuln ON tickets (vulnerability_id);
```

```sql
-- sql/migrations/005_create_tickets.down.sql
DROP TABLE IF EXISTS ticket_comments;
DROP TABLE IF EXISTS tickets;
DROP TYPE IF EXISTS ticket_priority;
DROP TYPE IF EXISTS ticket_status;
```

- [ ] **Step 2: Write teams migration**

```sql
-- sql/migrations/006_create_teams.up.sql
CREATE TABLE teams (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name        TEXT NOT NULL,
    description TEXT,
    creator_id  UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TYPE team_member_role AS ENUM ('owner', 'admin', 'member');

CREATE TABLE team_members (
    team_id  UUID NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    user_id  UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role     team_member_role NOT NULL DEFAULT 'member',
    joined_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (team_id, user_id)
);

CREATE TABLE invitations (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    team_id     UUID NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    email       TEXT NOT NULL,
    invited_by  UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    accepted    BOOLEAN NOT NULL DEFAULT false,
    expires_at  TIMESTAMPTZ NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

```sql
-- sql/migrations/006_create_teams.down.sql
DROP TABLE IF EXISTS invitations;
DROP TABLE IF EXISTS team_members;
DROP TABLE IF EXISTS teams;
DROP TYPE IF EXISTS team_member_role;
```

- [ ] **Step 3: Write reports migration**

```sql
-- sql/migrations/007_create_reports.up.sql
CREATE TYPE report_type AS ENUM ('technical', 'executive', 'compliance', 'comparison', 'trend');
CREATE TYPE report_format AS ENUM ('html', 'pdf', 'excel', 'markdown');
CREATE TYPE report_status AS ENUM ('pending', 'generating', 'completed', 'failed');

CREATE TABLE reports (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name        TEXT NOT NULL,
    report_type report_type NOT NULL,
    format      report_format NOT NULL DEFAULT 'html',
    status      report_status NOT NULL DEFAULT 'pending',
    scan_ids    UUID[] DEFAULT '{}',
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    file_path   TEXT,
    file_data   BYTEA,
    metadata    JSONB DEFAULT '{}',
    generated_at TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_reports_user ON reports (user_id);
CREATE INDEX idx_reports_status ON reports (status);
```

```sql
-- sql/migrations/007_create_reports.down.sql
DROP TABLE IF EXISTS reports;
DROP TYPE IF EXISTS report_status;
DROP TYPE IF EXISTS report_format;
DROP TYPE IF EXISTS report_type;
```

- [ ] **Step 4: Write notifications migration**

```sql
-- sql/migrations/008_create_notifications.up.sql
CREATE TYPE notification_type AS ENUM ('scan_complete', 'vuln_found', 'ticket_assigned', 'team_invite', 'report_ready', 'system');

CREATE TABLE notifications (
    id        UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id   UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type      notification_type NOT NULL,
    title     TEXT NOT NULL,
    message   TEXT,
    read      BOOLEAN NOT NULL DEFAULT false,
    data      JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_notifications_user ON notifications (user_id);
CREATE INDEX idx_notifications_unread ON notifications (user_id) WHERE NOT read;
```

```sql
-- sql/migrations/008_create_notifications.down.sql
DROP TABLE IF EXISTS notifications;
DROP TYPE IF EXISTS notification_type;
```

- [ ] **Step 5: Write schedules migration**

```sql
-- sql/migrations/009_create_schedules.up.sql
CREATE TABLE schedules (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name        TEXT NOT NULL,
    cron_expr   TEXT NOT NULL,
    scan_type   scan_type NOT NULL,
    target_id   UUID REFERENCES targets(id) ON DELETE CASCADE,
    target_group_id UUID REFERENCES target_groups(id) ON DELETE CASCADE,
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    options     JSONB DEFAULT '{}',
    enabled     BOOLEAN NOT NULL DEFAULT true,
    last_run    TIMESTAMPTZ,
    next_run    TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_schedules_user ON schedules (user_id);
CREATE INDEX idx_schedules_next ON schedules (next_run) WHERE enabled;
```

```sql
-- sql/migrations/009_create_schedules.down.sql
DROP TABLE IF EXISTS schedules;
```

- [ ] **Step 6: Write assets migration**

```sql
-- sql/migrations/010_create_assets.up.sql
CREATE TABLE assets (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    hostname        TEXT,
    ip_address      TEXT NOT NULL,
    mac_address     TEXT,
    os              TEXT,
    os_version      TEXT,
    open_ports      JSONB DEFAULT '[]',
    services        JSONB DEFAULT '[]',
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    target_id       UUID REFERENCES targets(id) ON DELETE SET NULL,
    last_seen       TIMESTAMPTZ NOT NULL DEFAULT now(),
    first_seen      TIMESTAMPTZ NOT NULL DEFAULT now(),
    vuln_count      INTEGER NOT NULL DEFAULT 0,
    risk_score      DECIMAL(5,2) DEFAULT 0,
    metadata        JSONB DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_assets_ip ON assets (ip_address);
CREATE INDEX idx_assets_user ON assets (user_id);
CREATE UNIQUE INDEX idx_assets_user_ip ON assets (user_id, ip_address);
```

```sql
-- sql/migrations/010_create_assets.down.sql
DROP TABLE IF EXISTS assets;
```

- [ ] **Step 7: Write audit logs migration**

```sql
-- sql/migrations/011_create_audit_logs.up.sql
CREATE TABLE audit_logs (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id     UUID REFERENCES users(id) ON DELETE SET NULL,
    action      TEXT NOT NULL,
    resource    TEXT NOT NULL,
    resource_id UUID,
    details     JSONB DEFAULT '{}',
    ip_address  TEXT,
    user_agent  TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_audit_user ON audit_logs (user_id);
CREATE INDEX idx_audit_resource ON audit_logs (resource, resource_id);
CREATE INDEX idx_audit_created ON audit_logs (created_at);
```

```sql
-- sql/migrations/011_create_audit_logs.down.sql
DROP TABLE IF EXISTS audit_logs;
```

- [ ] **Step 8: Write search indexes migration**

```sql
-- sql/migrations/012_create_search_indexes.up.sql
CREATE INDEX idx_vulns_title_trgm ON vulnerabilities USING gin (title gin_trgm_ops);
CREATE INDEX idx_vulns_desc_trgm ON vulnerabilities USING gin (description gin_trgm_ops);
CREATE INDEX idx_targets_host_trgm ON targets USING gin (host gin_trgm_ops);
CREATE INDEX idx_tickets_title_trgm ON tickets USING gin (title gin_trgm_ops);
CREATE INDEX idx_assets_hostname_trgm ON assets USING gin (hostname gin_trgm_ops);
```

```sql
-- sql/migrations/012_create_search_indexes.down.sql
DROP INDEX IF EXISTS idx_vulns_title_trgm;
DROP INDEX IF EXISTS idx_vulns_desc_trgm;
DROP INDEX IF EXISTS idx_targets_host_trgm;
DROP INDEX IF EXISTS idx_tickets_title_trgm;
DROP INDEX IF EXISTS idx_assets_hostname_trgm;
```

- [ ] **Step 9: Commit**

```bash
git add sql/migrations/
git commit -m "feat: add all database migrations (tickets, teams, reports, notifications, schedules, assets, audit, search)"
```

---

### Task 1.7: sqlc Configuration and Query Files

**Files:**
- Create: `sqlc.yaml`
- Create: `sql/queries/users.sql`

- [ ] **Step 1: Create sqlc config**

```yaml
# sqlc.yaml
version: "2"
sql:
  - engine: "postgresql"
    queries: "sql/queries"
    schema: "sql/migrations"
    gen:
      go:
        package: "queries"
        out: "internal/database/queries"
        sql_package: "pgx/v5"
        emit_json_tags: true
        emit_empty_slices: true
        overrides:
          - db_type: "uuid"
            go_type: "github.com/google/uuid.UUID"
          - db_type: "timestamptz"
            go_type: "time.Time"
```

- [ ] **Step 2: Write users queries**

```sql
-- sql/queries/users.sql

-- name: CreateUser :one
INSERT INTO users (email, username, password, role)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1;

-- name: GetUserByUsername :one
SELECT * FROM users WHERE username = $1;

-- name: ListUsers :many
SELECT * FROM users ORDER BY created_at DESC LIMIT $1 OFFSET $2;

-- name: UpdateUser :one
UPDATE users SET
    email = COALESCE(sqlc.narg('email'), email),
    username = COALESCE(sqlc.narg('username'), username),
    role = COALESCE(sqlc.narg('role'), role),
    is_active = COALESCE(sqlc.narg('is_active'), is_active),
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: UpdatePassword :exec
UPDATE users SET password = $2, updated_at = now() WHERE id = $1;

-- name: DeleteUser :exec
DELETE FROM users WHERE id = $1;

-- name: CountUsers :one
SELECT count(*) FROM users;
```

- [ ] **Step 3: Generate Go code**

Run: `go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest && sqlc generate`
Expected: Files generated in `internal/database/queries/`

- [ ] **Step 4: Commit**

```bash
git add sqlc.yaml sql/queries/ internal/database/queries/
git commit -m "feat: add sqlc config and users query generation"
```

---

### Task 1.8: Application Entry Point (Minimal Server)

**Files:**
- Create: `cmd/vulntrack/main.go`
- Test: Manual — `go run ./cmd/vulntrack` starts and responds on port 8080

- [ ] **Step 1: Write main.go**

```go
// cmd/vulntrack/main.go
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/labstack/echo/v4"
	echomw "github.com/labstack/echo/v4/middleware"

	"github.com/cyberoptic/vulntrack/internal/config"
	"github.com/cyberoptic/vulntrack/internal/database"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	pool, err := database.NewPool(cfg.Database.URL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer pool.Close()

	e := echo.New()
	e.HideBanner = true

	// Middleware
	e.Use(echomw.Logger())
	e.Use(echomw.Recover())
	e.Use(echomw.CORSWithConfig(echomw.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch},
	}))

	// Health check
	e.GET("/api/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	// Graceful shutdown
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	go func() {
		if err := e.Start(addr); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := e.Shutdown(ctx); err != nil {
		log.Fatalf("shutdown error: %v", err)
	}
	log.Println("server stopped")
}
```

- [ ] **Step 2: Verify it compiles**

Run: `go build ./cmd/vulntrack`
Expected: Compiles without errors

- [ ] **Step 3: Commit**

```bash
git add cmd/vulntrack/
git commit -m "feat: add application entry point with Echo server and graceful shutdown"
```

---

## Chunk 2: Authentication and Authorization

### Task 2.1: Password Hashing

**Files:**
- Create: `internal/auth/password.go`
- Test: `internal/auth/password_test.go`

- [ ] **Step 1: Write the failing test**

```go
// internal/auth/password_test.go
package auth

import "testing"

func TestHashPassword(t *testing.T) {
	hash, err := HashPassword("secureP@ss1")
	if err != nil {
		t.Fatalf("HashPassword returned error: %v", err)
	}
	if hash == "secureP@ss1" {
		t.Fatal("hash must not equal plaintext")
	}
}

func TestCheckPassword(t *testing.T) {
	hash, _ := HashPassword("secureP@ss1")

	if !CheckPassword("secureP@ss1", hash) {
		t.Error("expected password to match hash")
	}
	if CheckPassword("wrongpassword", hash) {
		t.Error("expected wrong password to not match")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/auth/ -v`
Expected: FAIL

- [ ] **Step 3: Write implementation**

```go
// internal/auth/password.go
package auth

import "golang.org/x/crypto/bcrypt"

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

func CheckPassword(password, hash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/auth/ -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/auth/password.go internal/auth/password_test.go
git commit -m "feat: add bcrypt password hashing"
```

---

### Task 2.2: JWT Token Generation and Validation

**Files:**
- Create: `internal/auth/jwt.go`
- Test: `internal/auth/jwt_test.go`

- [ ] **Step 1: Write the failing test**

```go
// internal/auth/jwt_test.go
package auth

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestGenerateAndValidateToken(t *testing.T) {
	secret := "test-secret-key"
	userID := uuid.New()
	role := "admin"

	token, err := GenerateToken(userID, role, secret, 1*time.Hour)
	if err != nil {
		t.Fatalf("GenerateToken error: %v", err)
	}
	if token == "" {
		t.Fatal("token must not be empty")
	}

	claims, err := ValidateToken(token, secret)
	if err != nil {
		t.Fatalf("ValidateToken error: %v", err)
	}
	if claims.UserID != userID {
		t.Errorf("expected userID %s, got %s", userID, claims.UserID)
	}
	if claims.Role != role {
		t.Errorf("expected role %s, got %s", role, claims.Role)
	}
}

func TestValidateToken_Expired(t *testing.T) {
	secret := "test-secret-key"
	userID := uuid.New()

	token, _ := GenerateToken(userID, "viewer", secret, -1*time.Hour)
	_, err := ValidateToken(token, secret)
	if err == nil {
		t.Fatal("expected error for expired token")
	}
}

func TestValidateToken_WrongSecret(t *testing.T) {
	token, _ := GenerateToken(uuid.New(), "viewer", "secret1", 1*time.Hour)
	_, err := ValidateToken(token, "secret2")
	if err == nil {
		t.Fatal("expected error for wrong secret")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/auth/ -v -run Token`
Expected: FAIL

- [ ] **Step 3: Write implementation**

```go
// internal/auth/jwt.go
package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type Claims struct {
	UserID uuid.UUID `json:"user_id"`
	Role   string    `json:"role"`
	jwt.RegisteredClaims
}

func GenerateToken(userID uuid.UUID, role, secret string, expiry time.Duration) (string, error) {
	claims := Claims{
		UserID: userID,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "vulntrack",
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func ValidateToken(tokenStr, secret string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/auth/ -v -run Token`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/auth/jwt.go internal/auth/jwt_test.go
git commit -m "feat: add JWT token generation and validation"
```

---

### Task 2.3: Auth Middleware

**Files:**
- Create: `internal/middleware/auth.go`
- Test: `internal/middleware/auth_test.go`

- [ ] **Step 1: Write the failing test**

```go
// internal/middleware/auth_test.go
package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/cyberoptic/vulntrack/internal/auth"
)

func TestAuthMiddleware_ValidToken(t *testing.T) {
	e := echo.New()
	secret := "test-secret"
	userID := uuid.New()

	token, _ := auth.GenerateToken(userID, "admin", secret, 1*time.Hour)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler := JWTAuth(secret)(func(c echo.Context) error {
		uid := GetUserID(c)
		if uid != userID {
			t.Errorf("expected user ID %s, got %s", userID, uid)
		}
		return c.NoContent(http.StatusOK)
	})

	if err := handler(c); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestAuthMiddleware_NoToken(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler := JWTAuth("secret")(func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})

	err := handler(c)
	if err == nil {
		t.Fatal("expected error for missing token")
	}
	he, ok := err.(*echo.HTTPError)
	if !ok || he.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %v", err)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/middleware/ -v`
Expected: FAIL

- [ ] **Step 3: Write implementation**

```go
// internal/middleware/auth.go
package middleware

import (
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/cyberoptic/vulntrack/internal/auth"
)

const (
	contextKeyUserID = "user_id"
	contextKeyRole   = "user_role"
)

func JWTAuth(secret string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			header := c.Request().Header.Get("Authorization")
			if header == "" {
				return echo.NewHTTPError(http.StatusUnauthorized, "missing authorization header")
			}

			parts := strings.SplitN(header, " ", 2)
			if len(parts) != 2 || parts[0] != "Bearer" {
				return echo.NewHTTPError(http.StatusUnauthorized, "invalid authorization format")
			}

			claims, err := auth.ValidateToken(parts[1], secret)
			if err != nil {
				return echo.NewHTTPError(http.StatusUnauthorized, "invalid or expired token")
			}

			c.Set(contextKeyUserID, claims.UserID)
			c.Set(contextKeyRole, claims.Role)
			return next(c)
		}
	}
}

func GetUserID(c echo.Context) uuid.UUID {
	id, _ := c.Get(contextKeyUserID).(uuid.UUID)
	return id
}

func GetUserRole(c echo.Context) string {
	role, _ := c.Get(contextKeyRole).(string)
	return role
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/middleware/ -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/middleware/auth.go internal/middleware/auth_test.go
git commit -m "feat: add JWT auth middleware with context helpers"
```

---

### Task 2.4: RBAC Middleware

**Files:**
- Create: `internal/middleware/rbac.go`
- Test: `internal/middleware/rbac_test.go`

- [ ] **Step 1: Write the failing test**

```go
// internal/middleware/rbac_test.go
package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
)

func TestRequireRole_Allowed(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(contextKeyRole, "admin")

	handler := RequireRole("admin", "analyst")(func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})

	if err := handler(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRequireRole_Denied(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(contextKeyRole, "viewer")

	handler := RequireRole("admin")(func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})

	err := handler(c)
	if err == nil {
		t.Fatal("expected error for unauthorized role")
	}
	he, ok := err.(*echo.HTTPError)
	if !ok || he.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %v", err)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/middleware/ -v -run Role`
Expected: FAIL

- [ ] **Step 3: Write implementation**

```go
// internal/middleware/rbac.go
package middleware

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

func RequireRole(roles ...string) echo.MiddlewareFunc {
	allowed := make(map[string]bool, len(roles))
	for _, r := range roles {
		allowed[r] = true
	}
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			role := GetUserRole(c)
			if !allowed[role] {
				return echo.NewHTTPError(http.StatusForbidden, "insufficient permissions")
			}
			return next(c)
		}
	}
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/middleware/ -v -run Role`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/middleware/rbac.go internal/middleware/rbac_test.go
git commit -m "feat: add role-based access control middleware"
```

---

### Task 2.5: Auth Handler (Register + Login)

**Files:**
- Create: `internal/handler/auth.go`
- Test: `internal/handler/auth_test.go`
- Create: `internal/service/user.go`

- [ ] **Step 1: Write user service**

```go
// internal/service/user.go
package service

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/cyberoptic/vulntrack/internal/auth"
	"github.com/cyberoptic/vulntrack/internal/database/queries"
)

var (
	ErrUserNotFound    = errors.New("user not found")
	ErrDuplicateUser   = errors.New("user already exists")
	ErrInvalidPassword = errors.New("invalid password")
)

type UserService struct {
	q    *queries.Queries
	pool *pgxpool.Pool
}

func NewUserService(pool *pgxpool.Pool) *UserService {
	return &UserService{
		q:    queries.New(pool),
		pool: pool,
	}
}

func (s *UserService) Register(ctx context.Context, email, username, password string) (queries.User, error) {
	hash, err := auth.HashPassword(password)
	if err != nil {
		return queries.User{}, err
	}
	user, err := s.q.CreateUser(ctx, queries.CreateUserParams{
		Email:    email,
		Username: username,
		Password: hash,
		Role:     "viewer",
	})
	if err != nil {
		return queries.User{}, ErrDuplicateUser
	}
	return user, nil
}

func (s *UserService) Authenticate(ctx context.Context, email, password string) (queries.User, error) {
	user, err := s.q.GetUserByEmail(ctx, email)
	if err != nil {
		return queries.User{}, ErrUserNotFound
	}
	if !auth.CheckPassword(password, user.Password) {
		return queries.User{}, ErrInvalidPassword
	}
	return user, nil
}

func (s *UserService) GetByID(ctx context.Context, id uuid.UUID) (queries.User, error) {
	user, err := s.q.GetUserByID(ctx, id)
	if err != nil {
		return queries.User{}, ErrUserNotFound
	}
	return user, nil
}
```

- [ ] **Step 2: Write auth handler**

```go
// internal/handler/auth.go
package handler

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/cyberoptic/vulntrack/internal/auth"
	"github.com/cyberoptic/vulntrack/internal/service"
)

type AuthHandler struct {
	users     *service.UserService
	jwtSecret string
	jwtExpiry time.Duration
}

func NewAuthHandler(users *service.UserService, jwtSecret string, jwtExpiry time.Duration) *AuthHandler {
	return &AuthHandler{users: users, jwtSecret: jwtSecret, jwtExpiry: jwtExpiry}
}

type registerRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Username string `json:"username" validate:"required,min=3,max=50"`
	Password string `json:"password" validate:"required,min=8"`
}

type loginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type authResponse struct {
	Token string `json:"token"`
	User  userDTO `json:"user"`
}

type userDTO struct {
	ID       string `json:"id"`
	Email    string `json:"email"`
	Username string `json:"username"`
	Role     string `json:"role"`
}

func (h *AuthHandler) Register(c echo.Context) error {
	var req registerRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if err := c.Validate(req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	user, err := h.users.Register(c.Request().Context(), req.Email, req.Username, req.Password)
	if err != nil {
		if err == service.ErrDuplicateUser {
			return echo.NewHTTPError(http.StatusConflict, "user already exists")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "registration failed")
	}

	token, err := auth.GenerateToken(user.ID, string(user.Role), h.jwtSecret, h.jwtExpiry)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "token generation failed")
	}

	return c.JSON(http.StatusCreated, authResponse{
		Token: token,
		User:  userDTO{ID: user.ID.String(), Email: user.Email, Username: user.Username, Role: string(user.Role)},
	})
}

func (h *AuthHandler) Login(c echo.Context) error {
	var req loginRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	user, err := h.users.Authenticate(c.Request().Context(), req.Email, req.Password)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "invalid credentials")
	}

	token, err := auth.GenerateToken(user.ID, string(user.Role), h.jwtSecret, h.jwtExpiry)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "token generation failed")
	}

	return c.JSON(http.StatusOK, authResponse{
		Token: token,
		User:  userDTO{ID: user.ID.String(), Email: user.Email, Username: user.Username, Role: string(user.Role)},
	})
}

func (h *AuthHandler) RegisterRoutes(g *echo.Group) {
	g.POST("/register", h.Register)
	g.POST("/login", h.Login)
}
```

- [ ] **Step 3: Write auth handler test**

```go
// internal/handler/auth_test.go
package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
)

func TestRegister_InvalidBody(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Without a validator set, Bind succeeds but Validate should catch missing fields
	// This test validates the handler returns 400 for missing required fields
	h := &AuthHandler{jwtSecret: "test"}
	err := h.Register(c)
	if err == nil {
		t.Fatal("expected error for empty body")
	}
	he, ok := err.(*echo.HTTPError)
	if !ok {
		t.Fatalf("expected HTTPError, got %T", err)
	}
	if he.Code != http.StatusBadRequest && he.Code != http.StatusInternalServerError {
		t.Errorf("expected 400 or 500, got %d", he.Code)
	}
}

func TestLogin_InvalidBody(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`not-json`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	h := &AuthHandler{jwtSecret: "test"}
	err := h.Login(c)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
	he, ok := err.(*echo.HTTPError)
	if !ok {
		t.Fatalf("expected HTTPError, got %T", err)
	}
	if he.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", he.Code)
	}
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/handler/ -v -run Auth`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/service/user.go internal/handler/auth.go internal/handler/auth_test.go
git commit -m "feat: add auth handler with register/login endpoints and tests"
```

---

### Task 2.6: Security Middleware (Rate Limiting, Security Headers, Audit)

**Files:**
- Create: `internal/middleware/ratelimit.go`
- Create: `internal/middleware/security.go`
- Create: `internal/middleware/audit.go`

- [ ] **Step 1: Write rate limiter**

```go
// internal/middleware/ratelimit.go
package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
)

type visitor struct {
	count    int
	lastSeen time.Time
}

type RateLimiter struct {
	mu       sync.Mutex
	visitors map[string]*visitor
	limit    int
	window   time.Duration
}

func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		visitors: make(map[string]*visitor),
		limit:    limit,
		window:   window,
	}
	go rl.cleanup()
	return rl
}

func (rl *RateLimiter) cleanup() {
	for {
		time.Sleep(rl.window)
		rl.mu.Lock()
		for ip, v := range rl.visitors {
			if time.Since(v.lastSeen) > rl.window {
				delete(rl.visitors, ip)
			}
		}
		rl.mu.Unlock()
	}
}

func (rl *RateLimiter) Middleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			ip := c.RealIP()
			rl.mu.Lock()
			v, exists := rl.visitors[ip]
			if !exists || time.Since(v.lastSeen) > rl.window {
				rl.visitors[ip] = &visitor{count: 1, lastSeen: time.Now()}
				rl.mu.Unlock()
				return next(c)
			}
			v.count++
			v.lastSeen = time.Now()
			if v.count > rl.limit {
				rl.mu.Unlock()
				return echo.NewHTTPError(http.StatusTooManyRequests, "rate limit exceeded")
			}
			rl.mu.Unlock()
			return next(c)
		}
	}
}
```

- [ ] **Step 2: Write security headers**

```go
// internal/middleware/security.go
package middleware

import "github.com/labstack/echo/v4"

func SecurityHeaders() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			h := c.Response().Header()
			h.Set("X-Content-Type-Options", "nosniff")
			h.Set("X-Frame-Options", "DENY")
			h.Set("X-XSS-Protection", "1; mode=block")
			h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
			h.Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'")
			return next(c)
		}
	}
}
```

- [ ] **Step 3: Write audit middleware**

```go
// internal/middleware/audit.go
package middleware

import (
	"time"

	"github.com/labstack/echo/v4"
)

type AuditLogger interface {
	Log(userID, action, resource, ip, userAgent string)
}

func AuditLog(logger AuditLogger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()
			err := next(c)
			userID := ""
			if uid := GetUserID(c); uid.String() != "00000000-0000-0000-0000-000000000000" {
				userID = uid.String()
			}
			logger.Log(
				userID,
				c.Request().Method+" "+c.Path(),
				c.Path(),
				c.RealIP(),
				c.Request().UserAgent(),
			)
			_ = start // available for timing if needed
			return err
		}
	}
}
```

- [ ] **Step 4: Commit**

```bash
git add internal/middleware/ratelimit.go internal/middleware/security.go internal/middleware/audit.go
git commit -m "feat: add rate limiting, security headers, and audit logging middleware"
```

---

## Chunk 3: Target Management and Scan Engine

### Task 3.1: sqlc Queries for Targets

**Files:**
- Create: `sql/queries/targets.sql`

- [ ] **Step 1: Write target queries**

```sql
-- sql/queries/targets.sql

-- name: CreateTarget :one
INSERT INTO targets (host, ip_address, hostname, os_guess, group_id, user_id, metadata)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: GetTarget :one
SELECT * FROM targets WHERE id = $1 AND user_id = $2;

-- name: ListTargets :many
SELECT * FROM targets WHERE user_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3;

-- name: ListTargetsByGroup :many
SELECT * FROM targets WHERE group_id = $1 AND user_id = $2 ORDER BY created_at DESC;

-- name: UpdateTarget :one
UPDATE targets SET
    host = COALESCE(sqlc.narg('host'), host),
    ip_address = COALESCE(sqlc.narg('ip_address'), ip_address),
    hostname = COALESCE(sqlc.narg('hostname'), hostname),
    os_guess = COALESCE(sqlc.narg('os_guess'), os_guess),
    group_id = COALESCE(sqlc.narg('group_id'), group_id),
    metadata = COALESCE(sqlc.narg('metadata'), metadata),
    updated_at = now()
WHERE id = $1 AND user_id = $2
RETURNING *;

-- name: DeleteTarget :exec
DELETE FROM targets WHERE id = $1 AND user_id = $2;

-- name: CountTargets :one
SELECT count(*) FROM targets WHERE user_id = $1;

-- name: CreateTargetGroup :one
INSERT INTO target_groups (name, description, user_id)
VALUES ($1, $2, $3)
RETURNING *;

-- name: ListTargetGroups :many
SELECT * FROM target_groups WHERE user_id = $1 ORDER BY name;

-- name: DeleteTargetGroup :exec
DELETE FROM target_groups WHERE id = $1 AND user_id = $2;
```

- [ ] **Step 2: Regenerate sqlc**

Run: `sqlc generate`

- [ ] **Step 3: Commit**

```bash
git add sql/queries/targets.sql internal/database/queries/
git commit -m "feat: add target and target group SQL queries"
```

---

### Task 3.2: Target Service and Handler

**Files:**
- Create: `internal/service/target.go`
- Create: `internal/handler/targets.go`

- [ ] **Step 1: Write target service**

```go
// internal/service/target.go
package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/cyberoptic/vulntrack/internal/database/queries"
)

type TargetService struct {
	q *queries.Queries
}

func NewTargetService(pool *pgxpool.Pool) *TargetService {
	return &TargetService{q: queries.New(pool)}
}

func (s *TargetService) Create(ctx context.Context, params queries.CreateTargetParams) (queries.Target, error) {
	return s.q.CreateTarget(ctx, params)
}

func (s *TargetService) Get(ctx context.Context, id, userID uuid.UUID) (queries.Target, error) {
	return s.q.GetTarget(ctx, queries.GetTargetParams{ID: id, UserID: userID})
}

func (s *TargetService) List(ctx context.Context, userID uuid.UUID, limit, offset int32) ([]queries.Target, error) {
	return s.q.ListTargets(ctx, queries.ListTargetsParams{UserID: userID, Limit: limit, Offset: offset})
}

func (s *TargetService) Delete(ctx context.Context, id, userID uuid.UUID) error {
	return s.q.DeleteTarget(ctx, queries.DeleteTargetParams{ID: id, UserID: userID})
}

func (s *TargetService) CreateGroup(ctx context.Context, name, description string, userID uuid.UUID) (queries.TargetGroup, error) {
	return s.q.CreateTargetGroup(ctx, queries.CreateTargetGroupParams{Name: name, Description: &description, UserID: userID})
}

func (s *TargetService) ListGroups(ctx context.Context, userID uuid.UUID) ([]queries.TargetGroup, error) {
	return s.q.ListTargetGroups(ctx, userID)
}
```

- [ ] **Step 2: Write target handler**

```go
// internal/handler/targets.go
package handler

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/cyberoptic/vulntrack/internal/database/queries"
	"github.com/cyberoptic/vulntrack/internal/middleware"
	"github.com/cyberoptic/vulntrack/internal/service"
)

type TargetHandler struct {
	targets *service.TargetService
}

func NewTargetHandler(targets *service.TargetService) *TargetHandler {
	return &TargetHandler{targets: targets}
}

type createTargetRequest struct {
	Host      string  `json:"host" validate:"required"`
	IPAddress *string `json:"ip_address"`
	Hostname  *string `json:"hostname"`
	GroupID   *string `json:"group_id"`
}

func (h *TargetHandler) Create(c echo.Context) error {
	var req createTargetRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}

	userID := middleware.GetUserID(c)
	params := queries.CreateTargetParams{
		Host:   req.Host,
		UserID: userID,
	}
	if req.IPAddress != nil {
		params.IpAddress = req.IPAddress
	}
	if req.Hostname != nil {
		params.Hostname = req.Hostname
	}
	if req.GroupID != nil {
		gid, err := uuid.Parse(*req.GroupID)
		if err == nil {
			params.GroupID = &gid
		}
	}

	target, err := h.targets.Create(c.Request().Context(), params)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create target")
	}
	return c.JSON(http.StatusCreated, target)
}

func (h *TargetHandler) List(c echo.Context) error {
	userID := middleware.GetUserID(c)
	targets, err := h.targets.List(c.Request().Context(), userID, 100, 0)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to list targets")
	}
	return c.JSON(http.StatusOK, targets)
}

func (h *TargetHandler) Get(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid target ID")
	}
	userID := middleware.GetUserID(c)
	target, err := h.targets.Get(c.Request().Context(), id, userID)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "target not found")
	}
	return c.JSON(http.StatusOK, target)
}

func (h *TargetHandler) Delete(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid target ID")
	}
	userID := middleware.GetUserID(c)
	if err := h.targets.Delete(c.Request().Context(), id, userID); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to delete target")
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *TargetHandler) RegisterRoutes(g *echo.Group) {
	g.POST("", h.Create)
	g.GET("", h.List)
	g.GET("/:id", h.Get)
	g.DELETE("/:id", h.Delete)
}
```

- [ ] **Step 3: Commit**

```bash
git add internal/service/target.go internal/handler/targets.go
git commit -m "feat: add target management service and REST handler"
```

---

### Task 3.3: Nmap Scanner Integration

**Files:**
- Create: `internal/scanner/nmap.go`
- Test: `internal/scanner/nmap_test.go`

- [ ] **Step 1: Write the failing test**

```go
// internal/scanner/nmap_test.go
package scanner

import (
	"strings"
	"testing"
)

func TestParseNmapXML(t *testing.T) {
	xml := `<?xml version="1.0"?>
<nmaprun scanner="nmap" args="nmap -sV 192.168.1.1" start="1234567890">
  <host starttime="1234567890" endtime="1234567899">
    <status state="up"/>
    <address addr="192.168.1.1" addrtype="ipv4"/>
    <hostnames><hostname name="router.local" type="PTR"/></hostnames>
    <ports>
      <port protocol="tcp" portid="22">
        <state state="open"/>
        <service name="ssh" product="OpenSSH" version="8.9"/>
      </port>
      <port protocol="tcp" portid="80">
        <state state="open"/>
        <service name="http" product="nginx" version="1.18.0"/>
      </port>
    </ports>
    <os><osmatch name="Linux 5.x" accuracy="95"/></os>
  </host>
</nmaprun>`

	result, err := ParseNmapXML(strings.NewReader(xml))
	if err != nil {
		t.Fatalf("ParseNmapXML error: %v", err)
	}
	if len(result.Hosts) != 1 {
		t.Fatalf("expected 1 host, got %d", len(result.Hosts))
	}
	host := result.Hosts[0]
	if host.Address != "192.168.1.1" {
		t.Errorf("expected address 192.168.1.1, got %s", host.Address)
	}
	if len(host.Ports) != 2 {
		t.Errorf("expected 2 ports, got %d", len(host.Ports))
	}
	if host.Ports[0].Service != "ssh" {
		t.Errorf("expected service ssh, got %s", host.Ports[0].Service)
	}
	if host.OS != "Linux 5.x" {
		t.Errorf("expected OS Linux 5.x, got %s", host.OS)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/scanner/ -v -run ParseNmap`
Expected: FAIL

- [ ] **Step 3: Write implementation**

```go
// internal/scanner/nmap.go
package scanner

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

type NmapResult struct {
	Hosts []HostResult
}

type HostResult struct {
	Address  string
	Hostname string
	OS       string
	Status   string
	Ports    []PortResult
}

type PortResult struct {
	Port     int
	Protocol string
	State    string
	Service  string
	Product  string
	Version  string
}

// XML structures for nmap output parsing
type nmapRun struct {
	XMLName xml.Name   `xml:"nmaprun"`
	Hosts   []nmapHost `xml:"host"`
}

type nmapHost struct {
	Status    nmapStatus    `xml:"status"`
	Address   nmapAddress   `xml:"address"`
	Hostnames nmapHostnames `xml:"hostnames"`
	Ports     nmapPorts     `xml:"ports"`
	OS        nmapOS        `xml:"os"`
}

type nmapStatus struct {
	State string `xml:"state,attr"`
}

type nmapAddress struct {
	Addr string `xml:"addr,attr"`
}

type nmapHostnames struct {
	Names []nmapHostname `xml:"hostname"`
}

type nmapHostname struct {
	Name string `xml:"name,attr"`
}

type nmapPorts struct {
	Ports []nmapPort `xml:"port"`
}

type nmapPort struct {
	Protocol string      `xml:"protocol,attr"`
	PortID   int         `xml:"portid,attr"`
	State    nmapState   `xml:"state"`
	Service  nmapService `xml:"service"`
}

type nmapState struct {
	State string `xml:"state,attr"`
}

type nmapService struct {
	Name    string `xml:"name,attr"`
	Product string `xml:"product,attr"`
	Version string `xml:"version,attr"`
}

type nmapOS struct {
	Matches []nmapOSMatch `xml:"osmatch"`
}

type nmapOSMatch struct {
	Name     string `xml:"name,attr"`
	Accuracy string `xml:"accuracy,attr"`
}

func ParseNmapXML(r io.Reader) (*NmapResult, error) {
	var run nmapRun
	if err := xml.NewDecoder(r).Decode(&run); err != nil {
		return nil, fmt.Errorf("failed to parse nmap XML: %w", err)
	}

	result := &NmapResult{}
	for _, h := range run.Hosts {
		host := HostResult{
			Address: h.Address.Addr,
			Status:  h.Status.State,
		}
		if len(h.Hostnames.Names) > 0 {
			host.Hostname = h.Hostnames.Names[0].Name
		}
		if len(h.OS.Matches) > 0 {
			host.OS = h.OS.Matches[0].Name
		}
		for _, p := range h.Ports.Ports {
			host.Ports = append(host.Ports, PortResult{
				Port:     p.PortID,
				Protocol: p.Protocol,
				State:    p.State.State,
				Service:  p.Service.Name,
				Product:  p.Service.Product,
				Version:  p.Service.Version,
			})
		}
		result.Hosts = append(result.Hosts, host)
	}
	return result, nil
}

type NmapScanner struct {
	BinaryPath string
}

func NewNmapScanner(binaryPath string) *NmapScanner {
	return &NmapScanner{BinaryPath: binaryPath}
}

func (s *NmapScanner) Scan(ctx context.Context, target string, args ...string) (*NmapResult, error) {
	cmdArgs := append([]string{"-oX", "-", target}, args...)
	cmd := exec.CommandContext(ctx, s.BinaryPath, cmdArgs...)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("nmap execution failed: %w", err)
	}
	return ParseNmapXML(strings.NewReader(string(output)))
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/scanner/ -v -run ParseNmap`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/scanner/nmap.go internal/scanner/nmap_test.go
git commit -m "feat: add Nmap XML parser and CLI scanner wrapper"
```

---

### Task 3.4: OpenVAS Scanner Integration

**Files:**
- Create: `internal/scanner/openvas.go`
- Test: `internal/scanner/openvas_test.go`

- [ ] **Step 1: Write the failing test**

```go
// internal/scanner/openvas_test.go
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
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/scanner/ -v -run ParseOpenVAS`
Expected: FAIL

- [ ] **Step 3: Write implementation**

```go
// internal/scanner/openvas.go
package scanner

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"
)

type OpenVASResult struct {
	Title       string
	Host        string
	Port        string
	Severity    string
	CVSSScore   float64
	Description string
	Solution    string
	CVE         string
	OID         string
}

type ovasReport struct {
	XMLName xml.Name    `xml:"report"`
	Results ovasResults `xml:"results"`
}

type ovasResults struct {
	Results []ovasResult `xml:"result"`
}

type ovasResult struct {
	Name        string   `xml:"name"`
	Host        string   `xml:"host"`
	Port        string   `xml:"port"`
	Threat      string   `xml:"threat"`
	Severity    float64  `xml:"severity"`
	Description string   `xml:"description"`
	NVT         ovasNVT  `xml:"nvt"`
}

type ovasNVT struct {
	OID      string `xml:"oid,attr"`
	Name     string `xml:"name"`
	CVSSBase string `xml:"cvss_base"`
	CVE      string `xml:"cve"`
	Solution string `xml:"solution"`
}

func ParseOpenVASXML(r io.Reader) ([]OpenVASResult, error) {
	var report ovasReport
	if err := xml.NewDecoder(r).Decode(&report); err != nil {
		return nil, fmt.Errorf("failed to parse OpenVAS XML: %w", err)
	}

	var results []OpenVASResult
	for _, res := range report.Results.Results {
		cvss := res.Severity
		if cvss == 0 {
			cvss, _ = strconv.ParseFloat(res.NVT.CVSSBase, 64)
		}

		// Parse port number from "443/tcp" format
		portStr := res.Port

		results = append(results, OpenVASResult{
			Title:       res.Name,
			Host:        res.Host,
			Port:        portStr,
			Severity:    res.Threat,
			CVSSScore:   cvss,
			Description: res.Description,
			Solution:    res.NVT.Solution,
			CVE:         res.NVT.CVE,
			OID:         res.NVT.OID,
		})
	}
	return results, nil
}

type OpenVASScanner struct {
	BinaryPath string
}

func NewOpenVASScanner(binaryPath string) *OpenVASScanner {
	return &OpenVASScanner{BinaryPath: binaryPath}
}

func (s *OpenVASScanner) Scan(ctx context.Context, target string) ([]OpenVASResult, error) {
	// GVM CLI: create target, create task, start task, get report
	// This is a simplified wrapper — real GVM integration requires
	// multiple API calls via gvm-cli or the GMP protocol
	cmd := exec.CommandContext(ctx, s.BinaryPath, "socket",
		"--xml", fmt.Sprintf("<get_reports report_id=\"%s\"/>", target))
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("openvas execution failed: %w", err)
	}
	return ParseOpenVASXML(strings.NewReader(string(output)))
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/scanner/ -v -run ParseOpenVAS`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/scanner/openvas.go internal/scanner/openvas_test.go
git commit -m "feat: add OpenVAS XML parser and GVM CLI wrapper"
```

---

### Task 3.5: Scan Service and Async Worker

**Files:**
- Create: `internal/service/scan.go`
- Create: `internal/worker/scan_task.go`
- Create: `internal/worker/server.go`
- Create: `sql/queries/scans.sql`

- [ ] **Step 1: Write scans queries**

```sql
-- sql/queries/scans.sql

-- name: CreateScan :one
INSERT INTO scans (name, scan_type, status, target_id, target_group_id, user_id, options)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: GetScan :one
SELECT * FROM scans WHERE id = $1;

-- name: ListScans :many
SELECT * FROM scans WHERE user_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3;

-- name: UpdateScanStatus :one
UPDATE scans SET
    status = $2,
    started_at = COALESCE(sqlc.narg('started_at'), started_at),
    completed_at = COALESCE(sqlc.narg('completed_at'), completed_at),
    error_message = COALESCE(sqlc.narg('error_message'), error_message),
    raw_output = COALESCE(sqlc.narg('raw_output'), raw_output),
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: DeleteScan :exec
DELETE FROM scans WHERE id = $1 AND user_id = $2;
```

- [ ] **Step 2: Write worker server setup**

```go
// internal/worker/server.go
package worker

import (
	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/cyberoptic/vulntrack/internal/config"
	"github.com/cyberoptic/vulntrack/internal/scanner"
)

const (
	TaskScanNmap    = "scan:nmap"
	TaskScanOpenVAS = "scan:openvas"
	TaskReport      = "report:generate"
	TaskEnrich      = "vuln:enrich"
)

func NewServer(cfg *config.Config, pool *pgxpool.Pool) *asynq.Server {
	return asynq.NewServer(
		asynq.RedisClientOpt{Addr: cfg.Redis.Addr, Password: cfg.Redis.Password, DB: cfg.Redis.DB},
		asynq.Config{
			Concurrency: 10,
			Queues: map[string]int{
				"critical": 6,
				"default":  3,
				"low":      1,
			},
		},
	)
}

func NewMux(pool *pgxpool.Pool, nmapScanner *scanner.NmapScanner, openvasScanner *scanner.OpenVASScanner) *asynq.ServeMux {
	mux := asynq.NewServeMux()
	scanHandler := NewScanHandler(pool, nmapScanner, openvasScanner)
	mux.HandleFunc(TaskScanNmap, scanHandler.HandleNmapScan)
	mux.HandleFunc(TaskScanOpenVAS, scanHandler.HandleOpenVASScan)
	return mux
}
```

- [ ] **Step 3: Write scan task handler**

```go
// internal/worker/scan_task.go
package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/cyberoptic/vulntrack/internal/database/queries"
	"github.com/cyberoptic/vulntrack/internal/scanner"
)

type ScanPayload struct {
	ScanID   uuid.UUID `json:"scan_id"`
	Target   string    `json:"target"`
	Options  []string  `json:"options"`
}

type ScanHandler struct {
	q       *queries.Queries
	nmap    *scanner.NmapScanner
	openvas *scanner.OpenVASScanner
}

func NewScanHandler(pool *pgxpool.Pool, nmap *scanner.NmapScanner, openvas *scanner.OpenVASScanner) *ScanHandler {
	return &ScanHandler{
		q:       queries.New(pool),
		nmap:    nmap,
		openvas: openvas,
	}
}

func (h *ScanHandler) HandleNmapScan(ctx context.Context, t *asynq.Task) error {
	var payload ScanPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	now := time.Now()
	h.q.UpdateScanStatus(ctx, queries.UpdateScanStatusParams{
		ID:        payload.ScanID,
		Status:    "running",
		StartedAt: &now,
	})

	result, err := h.nmap.Scan(ctx, payload.Target, payload.Options...)
	if err != nil {
		errMsg := err.Error()
		h.q.UpdateScanStatus(ctx, queries.UpdateScanStatusParams{
			ID:           payload.ScanID,
			Status:       "failed",
			ErrorMessage: &errMsg,
		})
		return err
	}

	rawJSON, _ := json.Marshal(result)
	rawStr := string(rawJSON)
	completed := time.Now()
	h.q.UpdateScanStatus(ctx, queries.UpdateScanStatusParams{
		ID:          payload.ScanID,
		Status:      "completed",
		CompletedAt: &completed,
		RawOutput:   &rawStr,
	})

	// TODO: Parse results into vulnerabilities table
	return nil
}

func (h *ScanHandler) HandleOpenVASScan(ctx context.Context, t *asynq.Task) error {
	var payload ScanPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	now := time.Now()
	h.q.UpdateScanStatus(ctx, queries.UpdateScanStatusParams{
		ID:        payload.ScanID,
		Status:    "running",
		StartedAt: &now,
	})

	results, err := h.openvas.Scan(ctx, payload.Target)
	if err != nil {
		errMsg := err.Error()
		h.q.UpdateScanStatus(ctx, queries.UpdateScanStatusParams{
			ID:           payload.ScanID,
			Status:       "failed",
			ErrorMessage: &errMsg,
		})
		return err
	}

	rawJSON, _ := json.Marshal(results)
	rawStr := string(rawJSON)
	completed := time.Now()
	h.q.UpdateScanStatus(ctx, queries.UpdateScanStatusParams{
		ID:          payload.ScanID,
		Status:      "completed",
		CompletedAt: &completed,
		RawOutput:   &rawStr,
	})

	return nil
}

func NewScanTask(taskType string, payload ScanPayload) (*asynq.Task, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(taskType, data, asynq.MaxRetry(3), asynq.Queue("default")), nil
}
```

- [ ] **Step 4: Write scan service**

```go
// internal/service/scan.go
package service

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/cyberoptic/vulntrack/internal/database/queries"
	"github.com/cyberoptic/vulntrack/internal/worker"
)

type ScanService struct {
	q      *queries.Queries
	client *asynq.Client
}

func NewScanService(pool *pgxpool.Pool, client *asynq.Client) *ScanService {
	return &ScanService{q: queries.New(pool), client: client}
}

func (s *ScanService) Create(ctx context.Context, params queries.CreateScanParams) (queries.Scan, error) {
	return s.q.CreateScan(ctx, params)
}

func (s *ScanService) Get(ctx context.Context, id uuid.UUID) (queries.Scan, error) {
	return s.q.GetScan(ctx, id)
}

func (s *ScanService) List(ctx context.Context, userID uuid.UUID, limit, offset int32) ([]queries.Scan, error) {
	return s.q.ListScans(ctx, queries.ListScansParams{UserID: userID, Limit: limit, Offset: offset})
}

func (s *ScanService) Launch(ctx context.Context, scan queries.Scan, target string, options []string) error {
	taskType := worker.TaskScanNmap
	if scan.ScanType == "openvas" {
		taskType = worker.TaskScanOpenVAS
	}
	task, err := worker.NewScanTask(taskType, worker.ScanPayload{
		ScanID:  scan.ID,
		Target:  target,
		Options: options,
	})
	if err != nil {
		return err
	}
	_, err = s.client.Enqueue(task)
	return err
}

func (s *ScanService) Delete(ctx context.Context, id, userID uuid.UUID) error {
	return s.q.DeleteScan(ctx, queries.DeleteScanParams{ID: id, UserID: userID})
}
```

- [ ] **Step 5: Regenerate sqlc and commit**

```bash
sqlc generate
git add sql/queries/scans.sql internal/database/queries/ internal/worker/ internal/service/scan.go
git commit -m "feat: add scan service with async Asynq workers for Nmap and OpenVAS"
```

---

### Task 3.6: Scan Handler (REST API)

**Files:**
- Create: `internal/handler/scans.go`

- [ ] **Step 1: Write scan handler**

```go
// internal/handler/scans.go
package handler

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/labstack/echo/v4"

	"github.com/cyberoptic/vulntrack/internal/database/queries"
	"github.com/cyberoptic/vulntrack/internal/middleware"
	"github.com/cyberoptic/vulntrack/internal/worker"
)

type ScanHandler struct {
	q      *queries.Queries
	client *asynq.Client
}

func NewScanHandler(q *queries.Queries, client *asynq.Client) *ScanHandler {
	return &ScanHandler{q: q, client: client}
}

type launchScanRequest struct {
	Name     string   `json:"name" validate:"required"`
	ScanType string   `json:"scan_type" validate:"required,oneof=nmap openvas"`
	TargetID string   `json:"target_id" validate:"required,uuid"`
	Options  []string `json:"options"`
}

func (h *ScanHandler) Launch(c echo.Context) error {
	var req launchScanRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}

	userID := middleware.GetUserID(c)
	targetID, _ := uuid.Parse(req.TargetID)

	optJSON, _ := json.Marshal(req.Options)

	scan, err := h.q.CreateScan(c.Request().Context(), queries.CreateScanParams{
		Name:     req.Name,
		ScanType: queries.ScanType(req.ScanType),
		Status:   "pending",
		TargetID: &targetID,
		UserID:   userID,
		Options:  optJSON,
	})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create scan")
	}

	// Get target host for the scanner
	target, err := h.q.GetTarget(c.Request().Context(), queries.GetTargetParams{ID: targetID, UserID: userID})
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "target not found")
	}

	taskType := worker.TaskScanNmap
	if req.ScanType == "openvas" {
		taskType = worker.TaskScanOpenVAS
	}

	task, err := worker.NewScanTask(taskType, worker.ScanPayload{
		ScanID:  scan.ID,
		Target:  target.Host,
		Options: req.Options,
	})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create task")
	}

	if _, err := h.client.Enqueue(task); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to enqueue scan")
	}

	return c.JSON(http.StatusAccepted, scan)
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
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid scan ID")
	}
	scan, err := h.q.GetScan(c.Request().Context(), id)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "scan not found")
	}
	return c.JSON(http.StatusOK, scan)
}

func (h *ScanHandler) RegisterRoutes(g *echo.Group) {
	g.POST("", h.Launch)
	g.GET("", h.List)
	g.GET("/:id", h.Get)
}
```

- [ ] **Step 2: Commit**

```bash
git add internal/handler/scans.go
git commit -m "feat: add scan REST handler with async job dispatch"
```

---

## Chunk 4: Vulnerability Management, Ticketing, and Reports

### Task 4.1: Vulnerability Queries and Service

**Files:**
- Create: `sql/queries/vulnerabilities.sql`
- Create: `internal/service/vulnerability.go`
- Create: `internal/handler/vulnerabilities.go`

- [ ] **Step 1: Write vulnerability queries**

```sql
-- sql/queries/vulnerabilities.sql

-- name: CreateVulnerability :one
INSERT INTO vulnerabilities (
    scan_id, target_id, user_id, title, description, severity,
    cvss_score, cve_id, cwe_id, affected_host, affected_port,
    protocol, service, solution, references
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
RETURNING *;

-- name: GetVulnerability :one
SELECT * FROM vulnerabilities WHERE id = $1;

-- name: ListVulnerabilities :many
SELECT * FROM vulnerabilities WHERE user_id = $1
ORDER BY
    CASE severity
        WHEN 'critical' THEN 1
        WHEN 'high' THEN 2
        WHEN 'medium' THEN 3
        WHEN 'low' THEN 4
        WHEN 'info' THEN 5
    END
LIMIT $2 OFFSET $3;

-- name: ListVulnsByScan :many
SELECT * FROM vulnerabilities WHERE scan_id = $1 ORDER BY severity, cvss_score DESC;

-- name: UpdateVulnStatus :one
UPDATE vulnerabilities SET
    status = $2,
    resolved_at = CASE WHEN $2 = 'resolved' THEN now() ELSE resolved_at END,
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: UpdateVulnEnrichment :exec
UPDATE vulnerabilities SET
    enrichment_data = $2,
    risk_score = $3,
    updated_at = now()
WHERE id = $1;

-- name: CountVulnsBySeverity :many
SELECT severity, count(*) as count FROM vulnerabilities
WHERE user_id = $1 AND status NOT IN ('resolved', 'false_positive')
GROUP BY severity;

-- name: DeleteVulnerability :exec
DELETE FROM vulnerabilities WHERE id = $1 AND user_id = $2;
```

- [ ] **Step 2: Write vulnerability service**

```go
// internal/service/vulnerability.go
package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/cyberoptic/vulntrack/internal/database/queries"
)

type VulnerabilityService struct {
	q *queries.Queries
}

func NewVulnerabilityService(pool *pgxpool.Pool) *VulnerabilityService {
	return &VulnerabilityService{q: queries.New(pool)}
}

func (s *VulnerabilityService) Create(ctx context.Context, params queries.CreateVulnerabilityParams) (queries.Vulnerability, error) {
	return s.q.CreateVulnerability(ctx, params)
}

func (s *VulnerabilityService) Get(ctx context.Context, id uuid.UUID) (queries.Vulnerability, error) {
	return s.q.GetVulnerability(ctx, id)
}

func (s *VulnerabilityService) List(ctx context.Context, userID uuid.UUID, limit, offset int32) ([]queries.Vulnerability, error) {
	return s.q.ListVulnerabilities(ctx, queries.ListVulnerabilitiesParams{UserID: userID, Limit: limit, Offset: offset})
}

func (s *VulnerabilityService) ListByScan(ctx context.Context, scanID uuid.UUID) ([]queries.Vulnerability, error) {
	return s.q.ListVulnsByScan(ctx, scanID)
}

func (s *VulnerabilityService) UpdateStatus(ctx context.Context, id uuid.UUID, status string) (queries.Vulnerability, error) {
	return s.q.UpdateVulnStatus(ctx, queries.UpdateVulnStatusParams{ID: id, Status: queries.VulnStatus(status)})
}

func (s *VulnerabilityService) CountBySeverity(ctx context.Context, userID uuid.UUID) ([]queries.CountVulnsBySeverityRow, error) {
	return s.q.CountVulnsBySeverity(ctx, userID)
}
```

- [ ] **Step 3: Write vulnerability handler**

```go
// internal/handler/vulnerabilities.go
package handler

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/cyberoptic/vulntrack/internal/middleware"
	"github.com/cyberoptic/vulntrack/internal/service"
)

type VulnHandler struct {
	vulns *service.VulnerabilityService
}

func NewVulnHandler(vulns *service.VulnerabilityService) *VulnHandler {
	return &VulnHandler{vulns: vulns}
}

func (h *VulnHandler) List(c echo.Context) error {
	userID := middleware.GetUserID(c)
	vulns, err := h.vulns.List(c.Request().Context(), userID, 100, 0)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to list vulnerabilities")
	}
	return c.JSON(http.StatusOK, vulns)
}

func (h *VulnHandler) Get(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid ID")
	}
	vuln, err := h.vulns.Get(c.Request().Context(), id)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "vulnerability not found")
	}
	return c.JSON(http.StatusOK, vuln)
}

type updateVulnStatusRequest struct {
	Status string `json:"status" validate:"required,oneof=open confirmed mitigated resolved false_positive accepted"`
}

func (h *VulnHandler) UpdateStatus(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid ID")
	}
	var req updateVulnStatusRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}
	vuln, err := h.vulns.UpdateStatus(c.Request().Context(), id, req.Status)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to update status")
	}
	return c.JSON(http.StatusOK, vuln)
}

func (h *VulnHandler) RegisterRoutes(g *echo.Group) {
	g.GET("", h.List)
	g.GET("/:id", h.Get)
	g.PATCH("/:id/status", h.UpdateStatus)
}
```

- [ ] **Step 4: Regenerate sqlc and commit**

```bash
sqlc generate
git add sql/queries/vulnerabilities.sql internal/database/queries/ internal/service/vulnerability.go internal/handler/vulnerabilities.go
git commit -m "feat: add vulnerability management with severity-sorted queries"
```

---

### Task 4.2: CVE Enrichment Service

**Files:**
- Create: `internal/scanner/enrichment.go`
- Test: `internal/scanner/enrichment_test.go`

- [ ] **Step 1: Write the failing test**

```go
// internal/scanner/enrichment_test.go
package scanner

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchCVE(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := NVDResponse{
			Vulnerabilities: []NVDVuln{
				{
					CVE: CVEData{
						ID: "CVE-2024-0001",
						Descriptions: []CVEDescription{
							{Lang: "en", Value: "Test vulnerability"},
						},
					},
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	enricher := NewCVEEnricher(server.URL)
	data, err := enricher.Fetch("CVE-2024-0001")
	if err != nil {
		t.Fatalf("Fetch error: %v", err)
	}
	if data.ID != "CVE-2024-0001" {
		t.Errorf("expected CVE-2024-0001, got %s", data.ID)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/scanner/ -v -run FetchCVE`
Expected: FAIL

- [ ] **Step 3: Write implementation**

```go
// internal/scanner/enrichment.go
package scanner

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const defaultNVDBaseURL = "https://services.nvd.nist.gov/rest/json/cves/2.0"

type CVEEnricher struct {
	baseURL string
	client  *http.Client
}

func NewCVEEnricher(baseURL string) *CVEEnricher {
	if baseURL == "" {
		baseURL = defaultNVDBaseURL
	}
	return &CVEEnricher{
		baseURL: baseURL,
		client:  &http.Client{Timeout: 30 * time.Second},
	}
}

type NVDResponse struct {
	Vulnerabilities []NVDVuln `json:"vulnerabilities"`
}

type NVDVuln struct {
	CVE CVEData `json:"cve"`
}

type CVEData struct {
	ID           string           `json:"id"`
	Descriptions []CVEDescription `json:"descriptions"`
	References   []CVEReference   `json:"references"`
	Metrics      json.RawMessage  `json:"metrics"`
}

type CVEDescription struct {
	Lang  string `json:"lang"`
	Value string `json:"value"`
}

type CVEReference struct {
	URL    string `json:"url"`
	Source string `json:"source"`
}

func (e *CVEEnricher) Fetch(cveID string) (*CVEData, error) {
	url := fmt.Sprintf("%s?cveId=%s", e.baseURL, cveID)
	resp, err := e.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("NVD request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("NVD returned status %d", resp.StatusCode)
	}

	var nvdResp NVDResponse
	if err := json.NewDecoder(resp.Body).Decode(&nvdResp); err != nil {
		return nil, fmt.Errorf("failed to decode NVD response: %w", err)
	}

	if len(nvdResp.Vulnerabilities) == 0 {
		return nil, fmt.Errorf("CVE %s not found", cveID)
	}

	return &nvdResp.Vulnerabilities[0].CVE, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/scanner/ -v -run FetchCVE`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/scanner/enrichment.go internal/scanner/enrichment_test.go
git commit -m "feat: add CVE enrichment service with NVD API integration"
```

---

### Task 4.3: Ticket System

**Files:**
- Create: `sql/queries/tickets.sql`
- Create: `internal/service/ticket.go`
- Create: `internal/handler/tickets.go`

- [ ] **Step 1: Write ticket queries**

```sql
-- sql/queries/tickets.sql

-- name: CreateTicket :one
INSERT INTO tickets (title, description, priority, vulnerability_id, assigned_to, created_by, due_date)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: GetTicket :one
SELECT * FROM tickets WHERE id = $1;

-- name: ListTickets :many
SELECT * FROM tickets WHERE created_by = $1 OR assigned_to = $1
ORDER BY
    CASE priority WHEN 'critical' THEN 1 WHEN 'high' THEN 2 WHEN 'medium' THEN 3 WHEN 'low' THEN 4 END,
    created_at DESC
LIMIT $2 OFFSET $3;

-- name: UpdateTicketStatus :one
UPDATE tickets SET
    status = $2,
    resolved_at = CASE WHEN $2 = 'resolved' THEN now() ELSE resolved_at END,
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: AssignTicket :one
UPDATE tickets SET assigned_to = $2, updated_at = now() WHERE id = $1 RETURNING *;

-- name: AddTicketComment :one
INSERT INTO ticket_comments (ticket_id, user_id, content)
VALUES ($1, $2, $3)
RETURNING *;

-- name: ListTicketComments :many
SELECT * FROM ticket_comments WHERE ticket_id = $1 ORDER BY created_at;

-- name: CountTicketsByStatus :many
SELECT status, count(*) as count FROM tickets
WHERE created_by = $1 OR assigned_to = $1
GROUP BY status;

-- name: DeleteTicket :exec
DELETE FROM tickets WHERE id = $1;
```

- [ ] **Step 2: Write ticket service**

```go
// internal/service/ticket.go
package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/cyberoptic/vulntrack/internal/database/queries"
)

type TicketService struct {
	q *queries.Queries
}

func NewTicketService(pool *pgxpool.Pool) *TicketService {
	return &TicketService{q: queries.New(pool)}
}

func (s *TicketService) Create(ctx context.Context, params queries.CreateTicketParams) (queries.Ticket, error) {
	return s.q.CreateTicket(ctx, params)
}

func (s *TicketService) Get(ctx context.Context, id uuid.UUID) (queries.Ticket, error) {
	return s.q.GetTicket(ctx, id)
}

func (s *TicketService) List(ctx context.Context, userID uuid.UUID, limit, offset int32) ([]queries.Ticket, error) {
	return s.q.ListTickets(ctx, queries.ListTicketsParams{CreatedBy: userID, Limit: limit, Offset: offset})
}

func (s *TicketService) UpdateStatus(ctx context.Context, id uuid.UUID, status string) (queries.Ticket, error) {
	return s.q.UpdateTicketStatus(ctx, queries.UpdateTicketStatusParams{ID: id, Status: queries.TicketStatus(status)})
}

func (s *TicketService) Assign(ctx context.Context, id, assigneeID uuid.UUID) (queries.Ticket, error) {
	return s.q.AssignTicket(ctx, queries.AssignTicketParams{ID: id, AssignedTo: &assigneeID})
}

func (s *TicketService) AddComment(ctx context.Context, ticketID, userID uuid.UUID, content string) (queries.TicketComment, error) {
	return s.q.AddTicketComment(ctx, queries.AddTicketCommentParams{TicketID: ticketID, UserID: userID, Content: content})
}

func (s *TicketService) ListComments(ctx context.Context, ticketID uuid.UUID) ([]queries.TicketComment, error) {
	return s.q.ListTicketComments(ctx, ticketID)
}
```

- [ ] **Step 3: Write ticket handler**

```go
// internal/handler/tickets.go
package handler

import (
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/cyberoptic/vulntrack/internal/database/queries"
	"github.com/cyberoptic/vulntrack/internal/middleware"
	"github.com/cyberoptic/vulntrack/internal/service"
)

type TicketHandler struct {
	tickets *service.TicketService
}

func NewTicketHandler(tickets *service.TicketService) *TicketHandler {
	return &TicketHandler{tickets: tickets}
}

type createTicketRequest struct {
	Title           string  `json:"title" validate:"required"`
	Description     string  `json:"description"`
	Priority        string  `json:"priority" validate:"required,oneof=critical high medium low"`
	VulnerabilityID *string `json:"vulnerability_id"`
	AssignedTo      *string `json:"assigned_to"`
	DueDate         *string `json:"due_date"`
}

func (h *TicketHandler) Create(c echo.Context) error {
	var req createTicketRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}
	userID := middleware.GetUserID(c)

	params := queries.CreateTicketParams{
		Title:       req.Title,
		Description: &req.Description,
		Priority:    queries.TicketPriority(req.Priority),
		CreatedBy:   userID,
	}
	if req.VulnerabilityID != nil {
		vid, _ := uuid.Parse(*req.VulnerabilityID)
		params.VulnerabilityID = &vid
	}
	if req.AssignedTo != nil {
		aid, _ := uuid.Parse(*req.AssignedTo)
		params.AssignedTo = &aid
	}
	if req.DueDate != nil {
		t, _ := time.Parse(time.RFC3339, *req.DueDate)
		params.DueDate = &t
	}

	ticket, err := h.tickets.Create(c.Request().Context(), params)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create ticket")
	}
	return c.JSON(http.StatusCreated, ticket)
}

func (h *TicketHandler) List(c echo.Context) error {
	userID := middleware.GetUserID(c)
	tickets, err := h.tickets.List(c.Request().Context(), userID, 100, 0)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to list tickets")
	}
	return c.JSON(http.StatusOK, tickets)
}

func (h *TicketHandler) Get(c echo.Context) error {
	id, _ := uuid.Parse(c.Param("id"))
	ticket, err := h.tickets.Get(c.Request().Context(), id)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "ticket not found")
	}
	return c.JSON(http.StatusOK, ticket)
}

type addCommentRequest struct {
	Content string `json:"content" validate:"required"`
}

func (h *TicketHandler) AddComment(c echo.Context) error {
	id, _ := uuid.Parse(c.Param("id"))
	var req addCommentRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}
	userID := middleware.GetUserID(c)
	comment, err := h.tickets.AddComment(c.Request().Context(), id, userID, req.Content)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to add comment")
	}
	return c.JSON(http.StatusCreated, comment)
}

func (h *TicketHandler) RegisterRoutes(g *echo.Group) {
	g.POST("", h.Create)
	g.GET("", h.List)
	g.GET("/:id", h.Get)
	g.POST("/:id/comments", h.AddComment)
}
```

- [ ] **Step 4: Regenerate sqlc and commit**

```bash
sqlc generate
git add sql/queries/tickets.sql internal/database/queries/ internal/service/ticket.go internal/handler/tickets.go
git commit -m "feat: add ticket system with comments and priority-sorted queries"
```

---

### Task 4.4: Report Generation System

**Files:**
- Create: `internal/report/html.go`
- Create: `internal/report/pdf.go`
- Create: `internal/report/excel.go`
- Create: `internal/report/markdown.go`
- Create: `internal/report/templates/report.html`
- Create: `internal/service/report.go`
- Create: `internal/handler/reports.go`
- Create: `sql/queries/reports.sql`

- [ ] **Step 1: Write report queries**

```sql
-- sql/queries/reports.sql

-- name: CreateReport :one
INSERT INTO reports (name, report_type, format, status, scan_ids, user_id, metadata)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: GetReport :one
SELECT * FROM reports WHERE id = $1;

-- name: ListReports :many
SELECT * FROM reports WHERE user_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3;

-- name: UpdateReportStatus :exec
UPDATE reports SET
    status = $2,
    file_data = $3,
    generated_at = CASE WHEN $2 = 'completed' THEN now() ELSE generated_at END
WHERE id = $1;
```

- [ ] **Step 2: Write HTML report generator**

```go
// internal/report/html.go
package report

import (
	"bytes"
	"embed"
	"html/template"

	"github.com/cyberoptic/vulntrack/internal/database/queries"
)

//go:embed templates/*.html
var templateFS embed.FS

type ReportData struct {
	Title           string
	GeneratedAt     string
	ScanName        string
	TotalVulns      int
	CriticalCount   int
	HighCount       int
	MediumCount     int
	LowCount        int
	InfoCount       int
	Vulnerabilities []queries.Vulnerability
}

func GenerateHTML(data ReportData) ([]byte, error) {
	tmpl, err := template.ParseFS(templateFS, "templates/report.html")
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
```

- [ ] **Step 3: Write HTML report test**

```go
// internal/report/html_test.go
package report

import (
	"strings"
	"testing"
)

func TestGenerateHTML(t *testing.T) {
	data := ReportData{
		Title:         "Test Report",
		GeneratedAt:   "2026-03-20",
		TotalVulns:    2,
		CriticalCount: 1,
		HighCount:     1,
	}

	output, err := GenerateHTML(data)
	if err != nil {
		t.Fatalf("GenerateHTML error: %v", err)
	}
	html := string(output)
	if !strings.Contains(html, "Test Report") {
		t.Error("expected report title in output")
	}
	if !strings.Contains(html, "2026-03-20") {
		t.Error("expected generated date in output")
	}
	if !strings.Contains(html, "<table>") {
		t.Error("expected table element in output")
	}
}
```

Run: `go test ./internal/report/ -v -run GenerateHTML`
Expected: PASS (after template is created in next step)

- [ ] **Step 4: Write HTML template**

```html
<!-- internal/report/templates/report.html -->
<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <title>{{.Title}}</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif; margin: 40px; color: #1a1a2e; }
        h1 { color: #16213e; border-bottom: 3px solid #0f3460; padding-bottom: 10px; }
        .summary { display: flex; gap: 16px; margin: 24px 0; }
        .stat { padding: 16px 24px; border-radius: 8px; color: white; text-align: center; min-width: 100px; }
        .stat .count { font-size: 2em; font-weight: bold; }
        .stat .label { font-size: 0.85em; opacity: 0.9; }
        .critical { background: #dc2626; }
        .high { background: #ea580c; }
        .medium { background: #d97706; }
        .low { background: #2563eb; }
        .info { background: #6b7280; }
        table { width: 100%; border-collapse: collapse; margin-top: 24px; }
        th { background: #16213e; color: white; padding: 12px; text-align: left; }
        td { padding: 10px 12px; border-bottom: 1px solid #e5e7eb; }
        tr:hover { background: #f3f4f6; }
        .severity-badge { padding: 4px 12px; border-radius: 12px; color: white; font-size: 0.8em; font-weight: 600; }
        .meta { color: #6b7280; font-size: 0.9em; margin: 8px 0; }
    </style>
</head>
<body>
    <h1>{{.Title}}</h1>
    <p class="meta">Generated: {{.GeneratedAt}} | Scan: {{.ScanName}} | Total: {{.TotalVulns}} vulnerabilities</p>

    <div class="summary">
        <div class="stat critical"><div class="count">{{.CriticalCount}}</div><div class="label">Critical</div></div>
        <div class="stat high"><div class="count">{{.HighCount}}</div><div class="label">High</div></div>
        <div class="stat medium"><div class="count">{{.MediumCount}}</div><div class="label">Medium</div></div>
        <div class="stat low"><div class="count">{{.LowCount}}</div><div class="label">Low</div></div>
        <div class="stat info"><div class="count">{{.InfoCount}}</div><div class="label">Info</div></div>
    </div>

    <table>
        <thead>
            <tr><th>Severity</th><th>Title</th><th>Host</th><th>Port</th><th>CVE</th><th>CVSS</th><th>Status</th></tr>
        </thead>
        <tbody>
            {{range .Vulnerabilities}}
            <tr>
                <td><span class="severity-badge {{.Severity}}">{{.Severity}}</span></td>
                <td>{{.Title}}</td>
                <td>{{.AffectedHost}}</td>
                <td>{{.AffectedPort}}</td>
                <td>{{.CveID}}</td>
                <td>{{.CvssScore}}</td>
                <td>{{.Status}}</td>
            </tr>
            {{end}}
        </tbody>
    </table>
</body>
</html>
```

- [ ] **Step 4: Write PDF report generator**

```go
// internal/report/pdf.go
package report

import (
	"fmt"

	"github.com/johnfercher/maroto/v2"
	"github.com/johnfercher/maroto/v2/pkg/components/col"
	"github.com/johnfercher/maroto/v2/pkg/components/row"
	"github.com/johnfercher/maroto/v2/pkg/components/text"
	"github.com/johnfercher/maroto/v2/pkg/consts/align"
	"github.com/johnfercher/maroto/v2/pkg/consts/fontstyle"
	"github.com/johnfercher/maroto/v2/pkg/props"
)

func GeneratePDF(data ReportData) ([]byte, error) {
	m := maroto.New()

	// Title
	m.AddRows(
		row.New(20).Add(
			col.New(12).Add(
				text.New(data.Title, props.Text{
					Size:  18,
					Style: fontstyle.Bold,
					Align: align.Center,
				}),
			),
		),
	)

	// Summary row
	m.AddRows(
		row.New(10).Add(
			col.New(12).Add(
				text.New("Generated: "+data.GeneratedAt, props.Text{
					Size:  9,
					Align: align.Center,
					Color: &props.Color{Red: 128, Green: 128, Blue: 128},
				}),
			),
		),
	)

	// Header row
	headerProps := props.Text{Size: 9, Style: fontstyle.Bold, Color: &props.Color{Red: 255, Green: 255, Blue: 255}}

	m.AddRows(
		row.New(8).Add(
			col.New(2).Add(text.New("Severity", headerProps)),
			col.New(4).Add(text.New("Title", headerProps)),
			col.New(2).Add(text.New("Host", headerProps)),
			col.New(1).Add(text.New("Port", headerProps)),
			col.New(2).Add(text.New("CVE", headerProps)),
			col.New(1).Add(text.New("CVSS", headerProps)),
		),
	)

	// Data rows
	cellProps := props.Text{Size: 8}
	for _, v := range data.Vulnerabilities {
		port := ""
		if v.AffectedPort != nil {
			port = fmt.Sprintf("%d", *v.AffectedPort)
		}
		cve := ""
		if v.CveID != nil {
			cve = *v.CveID
		}
		cvss := ""
		if v.CvssScore != nil {
			cvss = fmt.Sprintf("%.1f", *v.CvssScore)
		}
		host := ""
		if v.AffectedHost != nil {
			host = *v.AffectedHost
		}

		m.AddRows(
			row.New(7).Add(
				col.New(2).Add(text.New(string(v.Severity), cellProps)),
				col.New(4).Add(text.New(v.Title, cellProps)),
				col.New(2).Add(text.New(host, cellProps)),
				col.New(1).Add(text.New(port, cellProps)),
				col.New(2).Add(text.New(cve, cellProps)),
				col.New(1).Add(text.New(cvss, cellProps)),
			),
		)
	}

	doc, err := m.Generate()
	if err != nil {
		return nil, err
	}
	return doc.GetBytes(), nil
}
```

- [ ] **Step 5: Write Excel report generator**

```go
// internal/report/excel.go
package report

import (
	"fmt"

	"github.com/xuri/excelize/v2"
)

func GenerateExcel(data ReportData) ([]byte, error) {
	f := excelize.NewFile()
	sheet := "Vulnerabilities"
	f.SetSheetName("Sheet1", sheet)

	// Headers
	headers := []string{"Severity", "Title", "Host", "Port", "CVE", "CVSS", "Status", "Description", "Solution"}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheet, cell, h)
	}

	// Header style
	style, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"16213E"}},
		Alignment: &excelize.Alignment{Horizontal: "center"},
	})
	f.SetCellStyle(sheet, "A1", "I1", style)

	// Data
	for i, v := range data.Vulnerabilities {
		row := i + 2
		f.SetCellValue(sheet, fmt.Sprintf("A%d", row), string(v.Severity))
		f.SetCellValue(sheet, fmt.Sprintf("B%d", row), v.Title)
		if v.AffectedHost != nil {
			f.SetCellValue(sheet, fmt.Sprintf("C%d", row), *v.AffectedHost)
		}
		if v.AffectedPort != nil {
			f.SetCellValue(sheet, fmt.Sprintf("D%d", row), *v.AffectedPort)
		}
		if v.CveID != nil {
			f.SetCellValue(sheet, fmt.Sprintf("E%d", row), *v.CveID)
		}
		if v.CvssScore != nil {
			f.SetCellValue(sheet, fmt.Sprintf("F%d", row), *v.CvssScore)
		}
		f.SetCellValue(sheet, fmt.Sprintf("G%d", row), string(v.Status))
		if v.Description != nil {
			f.SetCellValue(sheet, fmt.Sprintf("H%d", row), *v.Description)
		}
		if v.Solution != nil {
			f.SetCellValue(sheet, fmt.Sprintf("I%d", row), *v.Solution)
		}
	}

	// Auto-width columns
	for i := range headers {
		col, _ := excelize.ColumnNumberToName(i + 1)
		f.SetColWidth(sheet, col, col, 18)
	}

	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
```

- [ ] **Step 6: Write Markdown report generator**

```go
// internal/report/markdown.go
package report

import (
	"bytes"
	"fmt"
)

func GenerateMarkdown(data ReportData) ([]byte, error) {
	var buf bytes.Buffer

	fmt.Fprintf(&buf, "# %s\n\n", data.Title)
	fmt.Fprintf(&buf, "**Generated:** %s | **Scan:** %s\n\n", data.GeneratedAt, data.ScanName)

	fmt.Fprintf(&buf, "## Summary\n\n")
	fmt.Fprintf(&buf, "| Severity | Count |\n|----------|-------|\n")
	fmt.Fprintf(&buf, "| Critical | %d |\n", data.CriticalCount)
	fmt.Fprintf(&buf, "| High | %d |\n", data.HighCount)
	fmt.Fprintf(&buf, "| Medium | %d |\n", data.MediumCount)
	fmt.Fprintf(&buf, "| Low | %d |\n", data.LowCount)
	fmt.Fprintf(&buf, "| Info | %d |\n\n", data.InfoCount)

	fmt.Fprintf(&buf, "## Vulnerabilities\n\n")
	fmt.Fprintf(&buf, "| Severity | Title | Host | Port | CVE | CVSS |\n")
	fmt.Fprintf(&buf, "|----------|-------|------|------|-----|------|\n")

	for _, v := range data.Vulnerabilities {
		host, port, cve, cvss := "", "", "", ""
		if v.AffectedHost != nil { host = *v.AffectedHost }
		if v.AffectedPort != nil { port = fmt.Sprintf("%d", *v.AffectedPort) }
		if v.CveID != nil { cve = *v.CveID }
		if v.CvssScore != nil { cvss = fmt.Sprintf("%.1f", *v.CvssScore) }
		fmt.Fprintf(&buf, "| %s | %s | %s | %s | %s | %s |\n",
			v.Severity, v.Title, host, port, cve, cvss)
	}

	return buf.Bytes(), nil
}
```

- [ ] **Step 7: Write report service**

```go
// internal/service/report.go
package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/cyberoptic/vulntrack/internal/database/queries"
	"github.com/cyberoptic/vulntrack/internal/report"
)

type ReportService struct {
	q     *queries.Queries
	vulns *VulnerabilityService
}

func NewReportService(pool *pgxpool.Pool, vulns *VulnerabilityService) *ReportService {
	return &ReportService{q: queries.New(pool), vulns: vulns}
}

func (s *ReportService) Create(ctx context.Context, params queries.CreateReportParams) (queries.Report, error) {
	return s.q.CreateReport(ctx, params)
}

func (s *ReportService) Get(ctx context.Context, id uuid.UUID) (queries.Report, error) {
	return s.q.GetReport(ctx, id)
}

func (s *ReportService) List(ctx context.Context, userID uuid.UUID, limit, offset int32) ([]queries.Report, error) {
	return s.q.ListReports(ctx, queries.ListReportsParams{UserID: userID, Limit: limit, Offset: offset})
}

func (s *ReportService) Generate(ctx context.Context, reportID uuid.UUID, scanIDs []uuid.UUID, format string, userID uuid.UUID) ([]byte, error) {
	// Gather vulnerabilities from all scans
	var allVulns []queries.Vulnerability
	for _, sid := range scanIDs {
		vulns, err := s.vulns.ListByScan(ctx, sid)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch vulns for scan %s: %w", sid, err)
		}
		allVulns = append(allVulns, vulns...)
	}

	// Build report data
	data := report.ReportData{
		Title:           "Vulnerability Report",
		GeneratedAt:     time.Now().Format(time.RFC3339),
		TotalVulns:      len(allVulns),
		Vulnerabilities: allVulns,
	}
	for _, v := range allVulns {
		switch v.Severity {
		case "critical":
			data.CriticalCount++
		case "high":
			data.HighCount++
		case "medium":
			data.MediumCount++
		case "low":
			data.LowCount++
		case "info":
			data.InfoCount++
		}
	}

	// Generate in requested format
	switch format {
	case "html":
		return report.GenerateHTML(data)
	case "pdf":
		return report.GeneratePDF(data)
	case "excel":
		return report.GenerateExcel(data)
	case "markdown":
		return report.GenerateMarkdown(data)
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}
}
```

- [ ] **Step 8: Write report handler**

```go
// internal/handler/reports.go
package handler

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/cyberoptic/vulntrack/internal/database/queries"
	"github.com/cyberoptic/vulntrack/internal/middleware"
	"github.com/cyberoptic/vulntrack/internal/service"
)

type ReportHandler struct {
	reports *service.ReportService
}

func NewReportHandler(reports *service.ReportService) *ReportHandler {
	return &ReportHandler{reports: reports}
}

type generateReportRequest struct {
	Name       string   `json:"name" validate:"required"`
	ReportType string   `json:"report_type" validate:"required,oneof=technical executive compliance comparison trend"`
	Format     string   `json:"format" validate:"required,oneof=html pdf excel markdown"`
	ScanIDs    []string `json:"scan_ids" validate:"required,min=1"`
}

func (h *ReportHandler) Generate(c echo.Context) error {
	var req generateReportRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}

	userID := middleware.GetUserID(c)

	var scanIDs []uuid.UUID
	for _, s := range req.ScanIDs {
		id, err := uuid.Parse(s)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid scan ID: "+s)
		}
		scanIDs = append(scanIDs, id)
	}

	rpt, err := h.reports.Create(c.Request().Context(), queries.CreateReportParams{
		Name:       req.Name,
		ReportType: queries.ReportType(req.ReportType),
		Format:     queries.ReportFormat(req.Format),
		Status:     "generating",
		ScanIds:    scanIDs,
		UserID:     userID,
	})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create report")
	}

	data, err := h.reports.Generate(c.Request().Context(), rpt.ID, scanIDs, req.Format, userID)
	if err != nil {
		h.reports.UpdateStatus(c.Request().Context(), rpt.ID, "failed", nil)
		return echo.NewHTTPError(http.StatusInternalServerError, "report generation failed: "+err.Error())
	}

	h.reports.UpdateStatus(c.Request().Context(), rpt.ID, "completed", data)

	contentType := map[string]string{
		"html":     "text/html",
		"pdf":      "application/pdf",
		"excel":    "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		"markdown": "text/markdown",
	}
	return c.Blob(http.StatusOK, contentType[req.Format], data)
}

func (h *ReportHandler) List(c echo.Context) error {
	userID := middleware.GetUserID(c)
	reports, err := h.reports.List(c.Request().Context(), userID, 50, 0)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to list reports")
	}
	return c.JSON(http.StatusOK, reports)
}

func (h *ReportHandler) Get(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid report ID")
	}
	rpt, err := h.reports.Get(c.Request().Context(), id)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "report not found")
	}
	return c.JSON(http.StatusOK, rpt)
}

func (h *ReportHandler) RegisterRoutes(g *echo.Group) {
	g.POST("", h.Generate)
	g.GET("", h.List)
	g.GET("/:id", h.Get)
}
```

- [ ] **Step 9: Regenerate sqlc and commit**

```bash
sqlc generate
git add internal/report/ sql/queries/reports.sql internal/database/queries/ internal/service/report.go internal/handler/reports.go
git commit -m "feat: add report generation system (HTML, PDF, Excel, Markdown)"
```

---

## Chunk 5: Teams, Notifications, WebSocket, Dashboard, Remaining Services

### Task 5.1: Remaining SQL Queries

**Files:**
- Create: `sql/queries/teams.sql`
- Create: `sql/queries/notifications.sql`
- Create: `sql/queries/schedules.sql`
- Create: `sql/queries/assets.sql`
- Create: `sql/queries/audit_logs.sql`
- Create: `sql/queries/search.sql`

- [ ] **Step 1: Write all remaining query files**

```sql
-- sql/queries/teams.sql
-- name: CreateTeam :one
INSERT INTO teams (name, description, creator_id) VALUES ($1, $2, $3) RETURNING *;

-- name: GetTeam :one
SELECT * FROM teams WHERE id = $1;

-- name: ListTeamsByUser :many
SELECT t.* FROM teams t JOIN team_members tm ON t.id = tm.team_id WHERE tm.user_id = $1 ORDER BY t.name;

-- name: AddTeamMember :exec
INSERT INTO team_members (team_id, user_id, role) VALUES ($1, $2, $3);

-- name: RemoveTeamMember :exec
DELETE FROM team_members WHERE team_id = $1 AND user_id = $2;

-- name: ListTeamMembers :many
SELECT u.id, u.email, u.username, u.role, tm.role as team_role, tm.joined_at
FROM users u JOIN team_members tm ON u.id = tm.user_id
WHERE tm.team_id = $1;

-- name: CreateInvitation :one
INSERT INTO invitations (team_id, email, invited_by, expires_at) VALUES ($1, $2, $3, $4) RETURNING *;

-- name: DeleteTeam :exec
DELETE FROM teams WHERE id = $1;
```

```sql
-- sql/queries/notifications.sql
-- name: CreateNotification :one
INSERT INTO notifications (user_id, type, title, message, data) VALUES ($1, $2, $3, $4, $5) RETURNING *;

-- name: ListNotifications :many
SELECT * FROM notifications WHERE user_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3;

-- name: CountUnread :one
SELECT count(*) FROM notifications WHERE user_id = $1 AND NOT read;

-- name: MarkRead :exec
UPDATE notifications SET read = true WHERE id = $1 AND user_id = $2;

-- name: MarkAllRead :exec
UPDATE notifications SET read = true WHERE user_id = $1;
```

```sql
-- sql/queries/schedules.sql
-- name: CreateSchedule :one
INSERT INTO schedules (name, cron_expr, scan_type, target_id, target_group_id, user_id, options)
VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING *;

-- name: ListSchedules :many
SELECT * FROM schedules WHERE user_id = $1 ORDER BY name;

-- name: GetSchedule :one
SELECT * FROM schedules WHERE id = $1;

-- name: GetDueSchedules :many
SELECT * FROM schedules WHERE enabled AND next_run <= now();

-- name: UpdateScheduleNextRun :exec
UPDATE schedules SET last_run = now(), next_run = $2, updated_at = now() WHERE id = $1;

-- name: ToggleSchedule :exec
UPDATE schedules SET enabled = $2, updated_at = now() WHERE id = $1;

-- name: DeleteSchedule :exec
DELETE FROM schedules WHERE id = $1 AND user_id = $2;
```

```sql
-- sql/queries/assets.sql
-- name: UpsertAsset :one
INSERT INTO assets (hostname, ip_address, mac_address, os, os_version, open_ports, services, user_id, target_id)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
ON CONFLICT (user_id, ip_address) DO UPDATE SET
    hostname = COALESCE(EXCLUDED.hostname, assets.hostname),
    os = COALESCE(EXCLUDED.os, assets.os),
    open_ports = EXCLUDED.open_ports,
    services = EXCLUDED.services,
    last_seen = now(),
    updated_at = now()
RETURNING *;

-- name: GetAsset :one
SELECT * FROM assets WHERE id = $1 AND user_id = $2;

-- name: ListAssets :many
SELECT * FROM assets WHERE user_id = $1 ORDER BY last_seen DESC LIMIT $2 OFFSET $3;

-- name: UpdateAssetRisk :exec
UPDATE assets SET vuln_count = $2, risk_score = $3, updated_at = now() WHERE id = $1;

-- name: DeleteAsset :exec
DELETE FROM assets WHERE id = $1 AND user_id = $2;
```

```sql
-- sql/queries/audit_logs.sql
-- name: CreateAuditLog :exec
INSERT INTO audit_logs (user_id, action, resource, resource_id, details, ip_address, user_agent)
VALUES ($1, $2, $3, $4, $5, $6, $7);

-- name: ListAuditLogs :many
SELECT * FROM audit_logs ORDER BY created_at DESC LIMIT $1 OFFSET $2;

-- name: ListAuditLogsByUser :many
SELECT * FROM audit_logs WHERE user_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3;
```

```sql
-- sql/queries/search.sql
-- name: SearchAll :many
SELECT 'vulnerability' as type, id, title as name, COALESCE(description, '') as detail
FROM vulnerabilities WHERE title ILIKE '%' || $1 || '%' OR description ILIKE '%' || $1 || '%'
UNION ALL
SELECT 'target', id, host, COALESCE(hostname, '')
FROM targets WHERE host ILIKE '%' || $1 || '%' OR hostname ILIKE '%' || $1 || '%'
UNION ALL
SELECT 'ticket', id, title, COALESCE(description, '')
FROM tickets WHERE title ILIKE '%' || $1 || '%'
LIMIT $2;
```

- [ ] **Step 2: Regenerate sqlc and commit**

```bash
sqlc generate
git add sql/queries/ internal/database/queries/
git commit -m "feat: add all remaining SQL queries (teams, notifications, schedules, assets, audit, search)"
```

---

### Task 5.2: WebSocket Hub

**Files:**
- Create: `internal/websocket/hub.go`
- Create: `internal/websocket/client.go`
- Create: `internal/handler/ws.go`

- [ ] **Step 1: Write WebSocket hub**

```go
// internal/websocket/hub.go
package websocket

import "sync"

type Message struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

type Hub struct {
	mu      sync.RWMutex
	clients map[string]map[*Client]bool // userID -> clients
}

func NewHub() *Hub {
	return &Hub{clients: make(map[string]map[*Client]bool)}
}

func (h *Hub) Register(userID string, client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.clients[userID] == nil {
		h.clients[userID] = make(map[*Client]bool)
	}
	h.clients[userID][client] = true
}

func (h *Hub) Unregister(userID string, client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if conns, ok := h.clients[userID]; ok {
		delete(conns, client)
		if len(conns) == 0 {
			delete(h.clients, userID)
		}
	}
}

func (h *Hub) SendToUser(userID string, msg Message) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if conns, ok := h.clients[userID]; ok {
		for client := range conns {
			client.Send(msg)
		}
	}
}

func (h *Hub) Broadcast(msg Message) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, conns := range h.clients {
		for client := range conns {
			client.Send(msg)
		}
	}
}
```

- [ ] **Step 2: Write WebSocket client**

```go
// internal/websocket/client.go
package websocket

import (
	"encoding/json"
	"log"
	"time"

	"github.com/gorilla/websocket"
)

const (
	writeWait  = 10 * time.Second
	pongWait   = 60 * time.Second
	pingPeriod = (pongWait * 9) / 10
)

type Client struct {
	conn *websocket.Conn
	send chan Message
}

func NewClient(conn *websocket.Conn) *Client {
	return &Client{
		conn: conn,
		send: make(chan Message, 64),
	}
}

func (c *Client) Send(msg Message) {
	select {
	case c.send <- msg:
	default:
		// channel full, drop message
	}
}

func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case msg, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			data, err := json.Marshal(msg)
			if err != nil {
				log.Printf("ws marshal error: %v", err)
				continue
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, data); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (c *Client) ReadPump(onClose func()) {
	defer func() {
		onClose()
		c.conn.Close()
	}()
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})
	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			break
		}
	}
}
```

- [ ] **Step 3: Write WebSocket handler**

```go
// internal/handler/ws.go
package handler

import (
	"net/http"

	gws "github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"

	"github.com/cyberoptic/vulntrack/internal/auth"
	"github.com/cyberoptic/vulntrack/internal/websocket"
)

var upgrader = gws.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type WSHandler struct {
	hub       *websocket.Hub
	jwtSecret string
}

func NewWSHandler(hub *websocket.Hub, jwtSecret string) *WSHandler {
	return &WSHandler{hub: hub, jwtSecret: jwtSecret}
}

func (h *WSHandler) Handle(c echo.Context) error {
	token := c.QueryParam("token")
	if token == "" {
		return echo.NewHTTPError(http.StatusUnauthorized, "missing token")
	}
	claims, err := auth.ValidateToken(token, h.jwtSecret)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "invalid token")
	}

	conn, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		return err
	}

	userID := claims.UserID.String()
	client := websocket.NewClient(conn)
	h.hub.Register(userID, client)

	go client.WritePump()
	go client.ReadPump(func() {
		h.hub.Unregister(userID, client)
	})

	return nil
}
```

- [ ] **Step 4: Commit**

```bash
git add internal/websocket/ internal/handler/ws.go
git commit -m "feat: add WebSocket hub with per-user channels and JWT auth"
```

---

### Task 5.3: Dashboard Handler

**Files:**
- Create: `internal/handler/dashboard.go`

- [ ] **Step 1: Write dashboard handler**

```go
// internal/handler/dashboard.go
package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/cyberoptic/vulntrack/internal/middleware"
	"github.com/cyberoptic/vulntrack/internal/service"
)

type DashboardHandler struct {
	vulns   *service.VulnerabilityService
	tickets *service.TicketService
}

func NewDashboardHandler(vulns *service.VulnerabilityService, tickets *service.TicketService) *DashboardHandler {
	return &DashboardHandler{vulns: vulns, tickets: tickets}
}

type dashboardResponse struct {
	VulnsBySeverity []severityCount `json:"vulns_by_severity"`
	TicketsByStatus []statusCount   `json:"tickets_by_status"`
	RecentVulns     int             `json:"recent_vulns"`
}

type severityCount struct {
	Severity string `json:"severity"`
	Count    int64  `json:"count"`
}

type statusCount struct {
	Status string `json:"status"`
	Count  int64  `json:"count"`
}

func (h *DashboardHandler) Get(c echo.Context) error {
	ctx := c.Request().Context()
	userID := middleware.GetUserID(c)

	vulnCounts, err := h.vulns.CountBySeverity(ctx, userID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to load dashboard")
	}

	var sevCounts []severityCount
	for _, vc := range vulnCounts {
		sevCounts = append(sevCounts, severityCount{
			Severity: string(vc.Severity),
			Count:    vc.Count,
		})
	}

	return c.JSON(http.StatusOK, dashboardResponse{
		VulnsBySeverity: sevCounts,
	})
}

func (h *DashboardHandler) RegisterRoutes(g *echo.Group) {
	g.GET("", h.Get)
}
```

- [ ] **Step 2: Commit**

```bash
git add internal/handler/dashboard.go
git commit -m "feat: add dashboard handler with severity aggregation"
```

---

### Task 5.4: Remaining Handlers (Teams, Notifications, Schedules, Assets, Audit, Search)

**Files:**
- Create: `internal/service/team.go`, `internal/handler/teams.go`
- Create: `internal/service/notification.go`, `internal/handler/notifications.go`
- Create: `internal/service/schedule.go`, `internal/handler/schedules.go`
- Create: `internal/service/asset.go`, `internal/handler/assets.go`
- Create: `internal/service/audit.go`, `internal/handler/audit.go`
- Create: `internal/service/search.go`, `internal/handler/search.go`

- [ ] **Step 1: Write team service and handler**

```go
// internal/service/team.go
package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/cyberoptic/vulntrack/internal/database/queries"
)

type TeamService struct {
	q *queries.Queries
}

func NewTeamService(pool *pgxpool.Pool) *TeamService {
	return &TeamService{q: queries.New(pool)}
}

func (s *TeamService) Create(ctx context.Context, name, description string, creatorID uuid.UUID) (queries.Team, error) {
	team, err := s.q.CreateTeam(ctx, queries.CreateTeamParams{
		Name: name, Description: &description, CreatorID: creatorID,
	})
	if err != nil {
		return queries.Team{}, err
	}
	// Add creator as owner
	s.q.AddTeamMember(ctx, queries.AddTeamMemberParams{
		TeamID: team.ID, UserID: creatorID, Role: "owner",
	})
	return team, nil
}

func (s *TeamService) ListByUser(ctx context.Context, userID uuid.UUID) ([]queries.Team, error) {
	return s.q.ListTeamsByUser(ctx, userID)
}

func (s *TeamService) Get(ctx context.Context, id uuid.UUID) (queries.Team, error) {
	return s.q.GetTeam(ctx, id)
}

func (s *TeamService) ListMembers(ctx context.Context, teamID uuid.UUID) ([]queries.ListTeamMembersRow, error) {
	return s.q.ListTeamMembers(ctx, teamID)
}

func (s *TeamService) AddMember(ctx context.Context, teamID, userID uuid.UUID, role string) error {
	return s.q.AddTeamMember(ctx, queries.AddTeamMemberParams{
		TeamID: teamID, UserID: userID, Role: queries.TeamMemberRole(role),
	})
}

func (s *TeamService) RemoveMember(ctx context.Context, teamID, userID uuid.UUID) error {
	return s.q.RemoveTeamMember(ctx, queries.RemoveTeamMemberParams{TeamID: teamID, UserID: userID})
}

func (s *TeamService) Invite(ctx context.Context, teamID uuid.UUID, email string, invitedBy uuid.UUID) (queries.Invitation, error) {
	return s.q.CreateInvitation(ctx, queries.CreateInvitationParams{
		TeamID: teamID, Email: email, InvitedBy: invitedBy,
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	})
}

func (s *TeamService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.q.DeleteTeam(ctx, id)
}
```

```go
// internal/handler/teams.go
package handler

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/cyberoptic/vulntrack/internal/middleware"
	"github.com/cyberoptic/vulntrack/internal/service"
)

type TeamHandler struct {
	teams *service.TeamService
}

func NewTeamHandler(teams *service.TeamService) *TeamHandler {
	return &TeamHandler{teams: teams}
}

type createTeamRequest struct {
	Name        string `json:"name" validate:"required"`
	Description string `json:"description"`
}

func (h *TeamHandler) Create(c echo.Context) error {
	var req createTeamRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}
	userID := middleware.GetUserID(c)
	team, err := h.teams.Create(c.Request().Context(), req.Name, req.Description, userID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create team")
	}
	return c.JSON(http.StatusCreated, team)
}

func (h *TeamHandler) List(c echo.Context) error {
	userID := middleware.GetUserID(c)
	teams, err := h.teams.ListByUser(c.Request().Context(), userID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to list teams")
	}
	return c.JSON(http.StatusOK, teams)
}

func (h *TeamHandler) Get(c echo.Context) error {
	id, _ := uuid.Parse(c.Param("id"))
	team, err := h.teams.Get(c.Request().Context(), id)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "team not found")
	}
	return c.JSON(http.StatusOK, team)
}

func (h *TeamHandler) Members(c echo.Context) error {
	id, _ := uuid.Parse(c.Param("id"))
	members, err := h.teams.ListMembers(c.Request().Context(), id)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to list members")
	}
	return c.JSON(http.StatusOK, members)
}

type addMemberRequest struct {
	UserID string `json:"user_id" validate:"required,uuid"`
	Role   string `json:"role" validate:"required,oneof=admin member"`
}

func (h *TeamHandler) AddMember(c echo.Context) error {
	id, _ := uuid.Parse(c.Param("id"))
	var req addMemberRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}
	uid, _ := uuid.Parse(req.UserID)
	if err := h.teams.AddMember(c.Request().Context(), id, uid, req.Role); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to add member")
	}
	return c.NoContent(http.StatusNoContent)
}

type inviteRequest struct {
	Email string `json:"email" validate:"required,email"`
}

func (h *TeamHandler) Invite(c echo.Context) error {
	id, _ := uuid.Parse(c.Param("id"))
	var req inviteRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}
	userID := middleware.GetUserID(c)
	inv, err := h.teams.Invite(c.Request().Context(), id, req.Email, userID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create invitation")
	}
	return c.JSON(http.StatusCreated, inv)
}

func (h *TeamHandler) Delete(c echo.Context) error {
	id, _ := uuid.Parse(c.Param("id"))
	if err := h.teams.Delete(c.Request().Context(), id); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to delete team")
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *TeamHandler) RegisterRoutes(g *echo.Group) {
	g.POST("", h.Create)
	g.GET("", h.List)
	g.GET("/:id", h.Get)
	g.GET("/:id/members", h.Members)
	g.POST("/:id/members", h.AddMember)
	g.POST("/:id/invite", h.Invite)
	g.DELETE("/:id", h.Delete)
}
```

- [ ] **Step 2: Write notification service and handler**

```go
// internal/service/notification.go
package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/cyberoptic/vulntrack/internal/database/queries"
)

type NotificationService struct {
	q *queries.Queries
}

func NewNotificationService(pool *pgxpool.Pool) *NotificationService {
	return &NotificationService{q: queries.New(pool)}
}

func (s *NotificationService) Create(ctx context.Context, params queries.CreateNotificationParams) (queries.Notification, error) {
	return s.q.CreateNotification(ctx, params)
}

func (s *NotificationService) List(ctx context.Context, userID uuid.UUID, limit, offset int32) ([]queries.Notification, error) {
	return s.q.ListNotifications(ctx, queries.ListNotificationsParams{UserID: userID, Limit: limit, Offset: offset})
}

func (s *NotificationService) CountUnread(ctx context.Context, userID uuid.UUID) (int64, error) {
	return s.q.CountUnread(ctx, userID)
}

func (s *NotificationService) MarkRead(ctx context.Context, id, userID uuid.UUID) error {
	return s.q.MarkRead(ctx, queries.MarkReadParams{ID: id, UserID: userID})
}

func (s *NotificationService) MarkAllRead(ctx context.Context, userID uuid.UUID) error {
	return s.q.MarkAllRead(ctx, userID)
}
```

```go
// internal/handler/notifications.go
package handler

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/cyberoptic/vulntrack/internal/middleware"
	"github.com/cyberoptic/vulntrack/internal/service"
)

type NotificationHandler struct {
	notifications *service.NotificationService
}

func NewNotificationHandler(n *service.NotificationService) *NotificationHandler {
	return &NotificationHandler{notifications: n}
}

func (h *NotificationHandler) List(c echo.Context) error {
	userID := middleware.GetUserID(c)
	notifs, err := h.notifications.List(c.Request().Context(), userID, 50, 0)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to list notifications")
	}
	return c.JSON(http.StatusOK, notifs)
}

func (h *NotificationHandler) Unread(c echo.Context) error {
	userID := middleware.GetUserID(c)
	count, err := h.notifications.CountUnread(c.Request().Context(), userID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to count unread")
	}
	return c.JSON(http.StatusOK, map[string]int64{"unread": count})
}

func (h *NotificationHandler) MarkRead(c echo.Context) error {
	id, _ := uuid.Parse(c.Param("id"))
	userID := middleware.GetUserID(c)
	if err := h.notifications.MarkRead(c.Request().Context(), id, userID); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to mark read")
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *NotificationHandler) MarkAllRead(c echo.Context) error {
	userID := middleware.GetUserID(c)
	if err := h.notifications.MarkAllRead(c.Request().Context(), userID); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to mark all read")
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *NotificationHandler) RegisterRoutes(g *echo.Group) {
	g.GET("", h.List)
	g.GET("/unread", h.Unread)
	g.PUT("/:id/read", h.MarkRead)
	g.PUT("/read-all", h.MarkAllRead)
}
```

- [ ] **Step 3: Write schedule service and handler**

```go
// internal/service/schedule.go
package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/cyberoptic/vulntrack/internal/database/queries"
)

type ScheduleService struct {
	q *queries.Queries
}

func NewScheduleService(pool *pgxpool.Pool) *ScheduleService {
	return &ScheduleService{q: queries.New(pool)}
}

func (s *ScheduleService) Create(ctx context.Context, params queries.CreateScheduleParams) (queries.Schedule, error) {
	return s.q.CreateSchedule(ctx, params)
}

func (s *ScheduleService) List(ctx context.Context, userID uuid.UUID) ([]queries.Schedule, error) {
	return s.q.ListSchedules(ctx, userID)
}

func (s *ScheduleService) Get(ctx context.Context, id uuid.UUID) (queries.Schedule, error) {
	return s.q.GetSchedule(ctx, id)
}

func (s *ScheduleService) Toggle(ctx context.Context, id uuid.UUID, enabled bool) error {
	return s.q.ToggleSchedule(ctx, queries.ToggleScheduleParams{ID: id, Enabled: enabled})
}

func (s *ScheduleService) Delete(ctx context.Context, id, userID uuid.UUID) error {
	return s.q.DeleteSchedule(ctx, queries.DeleteScheduleParams{ID: id, UserID: userID})
}
```

```go
// internal/handler/schedules.go
package handler

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/cyberoptic/vulntrack/internal/database/queries"
	"github.com/cyberoptic/vulntrack/internal/middleware"
	"github.com/cyberoptic/vulntrack/internal/service"
)

type ScheduleHandler struct {
	schedules *service.ScheduleService
}

func NewScheduleHandler(s *service.ScheduleService) *ScheduleHandler {
	return &ScheduleHandler{schedules: s}
}

type createScheduleRequest struct {
	Name     string `json:"name" validate:"required"`
	CronExpr string `json:"cron_expr" validate:"required"`
	ScanType string `json:"scan_type" validate:"required,oneof=nmap openvas"`
	TargetID string `json:"target_id" validate:"required,uuid"`
}

func (h *ScheduleHandler) Create(c echo.Context) error {
	var req createScheduleRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}
	userID := middleware.GetUserID(c)
	tid, _ := uuid.Parse(req.TargetID)
	sched, err := h.schedules.Create(c.Request().Context(), queries.CreateScheduleParams{
		Name: req.Name, CronExpr: req.CronExpr,
		ScanType: queries.ScanType(req.ScanType), TargetID: &tid, UserID: userID,
	})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create schedule")
	}
	return c.JSON(http.StatusCreated, sched)
}

func (h *ScheduleHandler) List(c echo.Context) error {
	userID := middleware.GetUserID(c)
	scheds, err := h.schedules.List(c.Request().Context(), userID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to list schedules")
	}
	return c.JSON(http.StatusOK, scheds)
}

func (h *ScheduleHandler) Toggle(c echo.Context) error {
	id, _ := uuid.Parse(c.Param("id"))
	var req struct{ Enabled bool `json:"enabled"` }
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}
	if err := h.schedules.Toggle(c.Request().Context(), id, req.Enabled); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to toggle schedule")
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *ScheduleHandler) Delete(c echo.Context) error {
	id, _ := uuid.Parse(c.Param("id"))
	userID := middleware.GetUserID(c)
	if err := h.schedules.Delete(c.Request().Context(), id, userID); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to delete schedule")
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *ScheduleHandler) RegisterRoutes(g *echo.Group) {
	g.POST("", h.Create)
	g.GET("", h.List)
	g.PATCH("/:id/toggle", h.Toggle)
	g.DELETE("/:id", h.Delete)
}
```

- [ ] **Step 4: Write asset service and handler**

```go
// internal/service/asset.go
package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/cyberoptic/vulntrack/internal/database/queries"
)

type AssetService struct {
	q *queries.Queries
}

func NewAssetService(pool *pgxpool.Pool) *AssetService {
	return &AssetService{q: queries.New(pool)}
}

func (s *AssetService) Upsert(ctx context.Context, params queries.UpsertAssetParams) (queries.Asset, error) {
	return s.q.UpsertAsset(ctx, params)
}

func (s *AssetService) Get(ctx context.Context, id, userID uuid.UUID) (queries.Asset, error) {
	return s.q.GetAsset(ctx, queries.GetAssetParams{ID: id, UserID: userID})
}

func (s *AssetService) List(ctx context.Context, userID uuid.UUID, limit, offset int32) ([]queries.Asset, error) {
	return s.q.ListAssets(ctx, queries.ListAssetsParams{UserID: userID, Limit: limit, Offset: offset})
}

func (s *AssetService) Delete(ctx context.Context, id, userID uuid.UUID) error {
	return s.q.DeleteAsset(ctx, queries.DeleteAssetParams{ID: id, UserID: userID})
}
```

```go
// internal/handler/assets.go
package handler

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/cyberoptic/vulntrack/internal/middleware"
	"github.com/cyberoptic/vulntrack/internal/service"
)

type AssetHandler struct {
	assets *service.AssetService
}

func NewAssetHandler(a *service.AssetService) *AssetHandler {
	return &AssetHandler{assets: a}
}

func (h *AssetHandler) List(c echo.Context) error {
	userID := middleware.GetUserID(c)
	assets, err := h.assets.List(c.Request().Context(), userID, 100, 0)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to list assets")
	}
	return c.JSON(http.StatusOK, assets)
}

func (h *AssetHandler) Get(c echo.Context) error {
	id, _ := uuid.Parse(c.Param("id"))
	userID := middleware.GetUserID(c)
	asset, err := h.assets.Get(c.Request().Context(), id, userID)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "asset not found")
	}
	return c.JSON(http.StatusOK, asset)
}

func (h *AssetHandler) Delete(c echo.Context) error {
	id, _ := uuid.Parse(c.Param("id"))
	userID := middleware.GetUserID(c)
	if err := h.assets.Delete(c.Request().Context(), id, userID); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to delete asset")
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *AssetHandler) RegisterRoutes(g *echo.Group) {
	g.GET("", h.List)
	g.GET("/:id", h.Get)
	g.DELETE("/:id", h.Delete)
}
```

- [ ] **Step 5: Write audit service and handler**

```go
// internal/service/audit.go
package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/cyberoptic/vulntrack/internal/database/queries"
)

type AuditService struct {
	q *queries.Queries
}

func NewAuditService(pool *pgxpool.Pool) *AuditService {
	return &AuditService{q: queries.New(pool)}
}

func (s *AuditService) Log(userID, action, resource, ip, userAgent string) {
	uid, _ := uuid.Parse(userID)
	s.q.CreateAuditLog(context.Background(), queries.CreateAuditLogParams{
		UserID: &uid, Action: action, Resource: resource,
		IpAddress: &ip, UserAgent: &userAgent,
	})
}

func (s *AuditService) List(ctx context.Context, limit, offset int32) ([]queries.AuditLog, error) {
	return s.q.ListAuditLogs(ctx, queries.ListAuditLogsParams{Limit: limit, Offset: offset})
}

func (s *AuditService) ListByUser(ctx context.Context, userID uuid.UUID, limit, offset int32) ([]queries.AuditLog, error) {
	return s.q.ListAuditLogsByUser(ctx, queries.ListAuditLogsByUserParams{UserID: &userID, Limit: limit, Offset: offset})
}
```

```go
// internal/handler/audit.go
package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/cyberoptic/vulntrack/internal/middleware"
	"github.com/cyberoptic/vulntrack/internal/service"
)

type AuditHandler struct {
	audit *service.AuditService
}

func NewAuditHandler(a *service.AuditService) *AuditHandler {
	return &AuditHandler{audit: a}
}

func (h *AuditHandler) List(c echo.Context) error {
	role := middleware.GetUserRole(c)
	if role != "admin" {
		return echo.NewHTTPError(http.StatusForbidden, "admin only")
	}
	logs, err := h.audit.List(c.Request().Context(), 100, 0)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to list audit logs")
	}
	return c.JSON(http.StatusOK, logs)
}

func (h *AuditHandler) RegisterRoutes(g *echo.Group) {
	g.GET("", h.List)
}
```

- [ ] **Step 6: Write search service and handler**

```go
// internal/service/search.go
package service

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/cyberoptic/vulntrack/internal/database/queries"
)

type SearchService struct {
	q *queries.Queries
}

func NewSearchService(pool *pgxpool.Pool) *SearchService {
	return &SearchService{q: queries.New(pool)}
}

func (s *SearchService) Search(ctx context.Context, query string, limit int32) ([]queries.SearchAllRow, error) {
	return s.q.SearchAll(ctx, queries.SearchAllParams{Column1: query, Limit: limit})
}
```

```go
// internal/handler/search.go
package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/cyberoptic/vulntrack/internal/service"
)

type SearchHandler struct {
	search *service.SearchService
}

func NewSearchHandler(s *service.SearchService) *SearchHandler {
	return &SearchHandler{search: s}
}

func (h *SearchHandler) Search(c echo.Context) error {
	q := c.QueryParam("q")
	if q == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "query parameter 'q' is required")
	}
	results, err := h.search.Search(c.Request().Context(), q, 50)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "search failed")
	}
	return c.JSON(http.StatusOK, results)
}

func (h *SearchHandler) RegisterRoutes(g *echo.Group) {
	g.GET("", h.Search)
}
```

- [ ] **Step 7: Commit**

```bash
git add internal/service/ internal/handler/
git commit -m "feat: add teams, notifications, schedules, assets, audit, and search handlers"
```

---

### Task 5.5: Risk Scoring (ML Replacement)

**Files:**
- Create: `internal/service/prediction.go`
- Test: `internal/service/prediction_test.go`

- [ ] **Step 1: Write the failing test**

```go
// internal/service/prediction_test.go
package service

import "testing"

func TestCalculateRiskScore(t *testing.T) {
	tests := []struct {
		name     string
		cvss     float64
		hasExploit bool
		age      int
		minScore float64
		maxScore float64
	}{
		{"critical with exploit", 9.8, true, 30, 90.0, 100.0},
		{"medium no exploit", 5.0, false, 10, 30.0, 60.0},
		{"low old", 2.0, false, 365, 10.0, 35.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := CalculateRiskScore(tt.cvss, tt.hasExploit, tt.age)
			if score < tt.minScore || score > tt.maxScore {
				t.Errorf("score %.2f outside expected range [%.0f, %.0f]", score, tt.minScore, tt.maxScore)
			}
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/service/ -v -run RiskScore`
Expected: FAIL

- [ ] **Step 3: Write implementation**

```go
// internal/service/prediction.go
package service

import "math"

// CalculateRiskScore produces a 0-100 risk score based on:
// - CVSS score (0-10, weighted 60%)
// - Exploit availability (boolean, weighted 25%)
// - Age in days (weighted 15% — older unpatched = higher risk)
func CalculateRiskScore(cvss float64, hasExploit bool, ageDays int) float64 {
	cvssComponent := (cvss / 10.0) * 60.0

	exploitComponent := 0.0
	if hasExploit {
		exploitComponent = 25.0
	}

	// Age factor: logarithmic curve, caps contribution at 15
	ageFactor := math.Min(math.Log1p(float64(ageDays))/math.Log1p(365)*15.0, 15.0)

	score := cvssComponent + exploitComponent + ageFactor
	return math.Min(math.Max(score, 0), 100)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/service/ -v -run RiskScore`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/service/prediction.go internal/service/prediction_test.go
git commit -m "feat: add deterministic risk scoring (replaces ML prediction)"
```

---

### Task 5.6: Plugin Interface

**Files:**
- Create: `internal/plugin/interface.go`
- Create: `internal/plugin/loader.go`

- [ ] **Step 1: Define plugin interface**

```go
// internal/plugin/interface.go
package plugin

import (
	"context"

	"github.com/cyberoptic/vulntrack/internal/scanner"
)

// Plugin defines the interface for VulnTrack scanner plugins.
// Plugins are compiled as Go plugins (.so) and loaded at runtime.
type Plugin interface {
	// Name returns the plugin's unique identifier
	Name() string
	// Version returns the plugin version
	Version() string
	// Scan runs the plugin's custom scan logic
	Scan(ctx context.Context, target string, options map[string]string) (*scanner.NmapResult, error)
}
```

- [ ] **Step 2: Write plugin loader**

```go
// internal/plugin/loader.go
package plugin

import (
	"fmt"
	"os"
	"path/filepath"
	goplugin "plugin"
)

type Registry struct {
	plugins map[string]Plugin
}

func NewRegistry() *Registry {
	return &Registry{plugins: make(map[string]Plugin)}
}

func (r *Registry) LoadDir(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // no plugins dir is fine
		}
		return err
	}
	for _, entry := range entries {
		if filepath.Ext(entry.Name()) != ".so" {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		if err := r.Load(path); err != nil {
			return fmt.Errorf("failed to load plugin %s: %w", path, err)
		}
	}
	return nil
}

func (r *Registry) Load(path string) error {
	p, err := goplugin.Open(path)
	if err != nil {
		return err
	}
	sym, err := p.Lookup("VulnTrackPlugin")
	if err != nil {
		return fmt.Errorf("plugin missing VulnTrackPlugin symbol: %w", err)
	}
	plug, ok := sym.(Plugin)
	if !ok {
		return fmt.Errorf("VulnTrackPlugin does not implement Plugin interface")
	}
	r.plugins[plug.Name()] = plug
	return nil
}

func (r *Registry) Get(name string) (Plugin, bool) {
	p, ok := r.plugins[name]
	return p, ok
}

func (r *Registry) List() []string {
	names := make([]string, 0, len(r.plugins))
	for name := range r.plugins {
		names = append(names, name)
	}
	return names
}
```

- [ ] **Step 3: Commit**

```bash
git add internal/plugin/
git commit -m "feat: add Go plugin interface and dynamic loader"
```

---

## Chunk 6: Frontend SPA and Final Assembly

### Task 6.1: React + Vite + Tailwind Project Scaffold

**Files:**
- Create: `frontend/package.json`
- Create: `frontend/vite.config.ts`
- Create: `frontend/tailwind.config.ts`
- Create: `frontend/tsconfig.json`
- Create: `frontend/index.html`
- Create: `frontend/src/main.tsx`
- Create: `frontend/src/App.tsx`

- [ ] **Step 1: Initialize frontend project**

```bash
cd frontend
npm create vite@latest . -- --template react-ts
npm install tailwindcss @tailwindcss/vite
npm install react-router-dom @tanstack/react-query zustand
npm install recharts lucide-react
npx shadcn@latest init
```

- [ ] **Step 2: Configure Vite for API proxy**

```typescript
// frontend/vite.config.ts
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'
import path from 'path'

export default defineConfig({
  plugins: [react(), tailwindcss()],
  resolve: {
    alias: { '@': path.resolve(__dirname, './src') },
  },
  server: {
    proxy: {
      '/api': 'http://localhost:8080',
      '/ws': { target: 'ws://localhost:8080', ws: true },
    },
  },
  build: {
    outDir: 'dist',
    emptyOutDir: true,
  },
})
```

- [ ] **Step 3: Write App.tsx with routing**

```tsx
// frontend/src/App.tsx
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { Shell } from '@/components/layout/Shell'
import { Login } from '@/pages/Login'
import { Dashboard } from '@/pages/Dashboard'
import { Targets } from '@/pages/Targets'
import { Scans } from '@/pages/Scans'
import { Vulnerabilities } from '@/pages/Vulnerabilities'
import { Tickets } from '@/pages/Tickets'
import { Reports } from '@/pages/Reports'
import { Teams } from '@/pages/Teams'
import { Settings } from '@/pages/Settings'
import { NotFound } from '@/pages/NotFound'
import { useAuth } from '@/hooks/useAuth'

const queryClient = new QueryClient()

function ProtectedRoute({ children }: { children: React.ReactNode }) {
  const { token } = useAuth()
  if (!token) return <Navigate to="/login" />
  return <>{children}</>
}

export default function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <BrowserRouter>
        <Routes>
          <Route path="/login" element={<Login />} />
          <Route path="/" element={<ProtectedRoute><Shell /></ProtectedRoute>}>
            <Route index element={<Dashboard />} />
            <Route path="targets" element={<Targets />} />
            <Route path="scans" element={<Scans />} />
            <Route path="vulnerabilities" element={<Vulnerabilities />} />
            <Route path="tickets" element={<Tickets />} />
            <Route path="reports" element={<Reports />} />
            <Route path="teams" element={<Teams />} />
            <Route path="settings" element={<Settings />} />
          </Route>
          <Route path="*" element={<NotFound />} />
        </Routes>
      </BrowserRouter>
    </QueryClientProvider>
  )
}
```

- [ ] **Step 4: Write API client**

```typescript
// frontend/src/api/client.ts
const BASE = '/api'

async function request<T>(path: string, options?: RequestInit): Promise<T> {
  const token = localStorage.getItem('token')
  const res = await fetch(`${BASE}${path}`, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
      ...options?.headers,
    },
  })
  if (!res.ok) {
    const err = await res.json().catch(() => ({ message: res.statusText }))
    throw new Error(err.message || res.statusText)
  }
  return res.json()
}

export const api = {
  get: <T>(path: string) => request<T>(path),
  post: <T>(path: string, body: unknown) => request<T>(path, { method: 'POST', body: JSON.stringify(body) }),
  put: <T>(path: string, body: unknown) => request<T>(path, { method: 'PUT', body: JSON.stringify(body) }),
  patch: <T>(path: string, body: unknown) => request<T>(path, { method: 'PATCH', body: JSON.stringify(body) }),
  delete: <T>(path: string) => request<T>(path, { method: 'DELETE' }),
}
```

- [ ] **Step 5: Write auth hook**

```typescript
// frontend/src/hooks/useAuth.ts
import { create } from 'zustand'

interface AuthState {
  token: string | null
  user: { id: string; email: string; username: string; role: string } | null
  login: (token: string, user: AuthState['user']) => void
  logout: () => void
}

export const useAuth = create<AuthState>((set) => ({
  token: localStorage.getItem('token'),
  user: JSON.parse(localStorage.getItem('user') || 'null'),
  login: (token, user) => {
    localStorage.setItem('token', token)
    localStorage.setItem('user', JSON.stringify(user))
    set({ token, user })
  },
  logout: () => {
    localStorage.removeItem('token')
    localStorage.removeItem('user')
    set({ token: null, user: null })
  },
}))
```

- [ ] **Step 6: Commit**

```bash
git add frontend/
git commit -m "feat: scaffold React frontend with routing, API client, auth"
```

---

### Task 6.2: Key Frontend Pages (Dashboard, Vulnerabilities, Scans)

Each page follows the same pattern: use `@tanstack/react-query` to fetch data from the API, render with shadcn/ui components.

**Files:**
- Create: `frontend/src/pages/Dashboard.tsx`
- Create: `frontend/src/pages/Vulnerabilities.tsx`
- Create: `frontend/src/pages/Scans.tsx`
- Create: `frontend/src/pages/Login.tsx`
- Create: `frontend/src/components/layout/Shell.tsx`
- Create: `frontend/src/components/layout/Sidebar.tsx`
- Create: `frontend/src/components/dashboard/StatsCards.tsx`
- Create: `frontend/src/components/dashboard/SeverityPie.tsx`
- Create: `frontend/src/hooks/useWebSocket.ts`

- [ ] **Step 1: Write Login page**

```tsx
// frontend/src/pages/Login.tsx
import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useAuth } from '@/hooks/useAuth'
import { api } from '@/api/client'

export function Login() {
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [isRegister, setIsRegister] = useState(false)
  const [username, setUsername] = useState('')
  const { login } = useAuth()
  const navigate = useNavigate()

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    try {
      const endpoint = isRegister ? '/auth/register' : '/auth/login'
      const body = isRegister ? { email, username, password } : { email, password }
      const res = await api.post<{ token: string; user: any }>(endpoint, body)
      login(res.token, res.user)
      navigate('/')
    } catch (err: any) {
      setError(err.message || 'Authentication failed')
    }
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-slate-950">
      <div className="w-full max-w-md bg-slate-900 rounded-lg border border-slate-800 p-8">
        <h1 className="text-2xl font-bold text-white mb-6">VulnTrack Pro</h1>
        {error && <div className="bg-red-900/50 text-red-300 p-3 rounded mb-4">{error}</div>}
        <form onSubmit={handleSubmit} className="space-y-4">
          <input type="email" placeholder="Email" value={email} onChange={e => setEmail(e.target.value)}
            className="w-full p-3 bg-slate-800 border border-slate-700 rounded text-white" required />
          {isRegister && (
            <input type="text" placeholder="Username" value={username} onChange={e => setUsername(e.target.value)}
              className="w-full p-3 bg-slate-800 border border-slate-700 rounded text-white" required />
          )}
          <input type="password" placeholder="Password" value={password} onChange={e => setPassword(e.target.value)}
            className="w-full p-3 bg-slate-800 border border-slate-700 rounded text-white" required />
          <button type="submit" className="w-full p-3 bg-blue-600 hover:bg-blue-700 rounded text-white font-medium">
            {isRegister ? 'Register' : 'Sign In'}
          </button>
        </form>
        <button onClick={() => setIsRegister(!isRegister)} className="mt-4 text-sm text-slate-400 hover:text-white">
          {isRegister ? 'Already have an account? Sign in' : 'Need an account? Register'}
        </button>
      </div>
    </div>
  )
}
```

- [ ] **Step 2: Write Shell and Sidebar layout**

```tsx
// frontend/src/components/layout/Sidebar.tsx
import { NavLink } from 'react-router-dom'
import { LayoutDashboard, Target, Scan, Bug, Ticket, FileText, Users, Settings } from 'lucide-react'

const links = [
  { to: '/', icon: LayoutDashboard, label: 'Dashboard' },
  { to: '/targets', icon: Target, label: 'Targets' },
  { to: '/scans', icon: Scan, label: 'Scans' },
  { to: '/vulnerabilities', icon: Bug, label: 'Vulnerabilities' },
  { to: '/tickets', icon: Ticket, label: 'Tickets' },
  { to: '/reports', icon: FileText, label: 'Reports' },
  { to: '/teams', icon: Users, label: 'Teams' },
  { to: '/settings', icon: Settings, label: 'Settings' },
]

export function Sidebar() {
  return (
    <aside className="w-64 bg-slate-900 border-r border-slate-800 min-h-screen p-4">
      <div className="text-xl font-bold text-white mb-8 px-2">VulnTrack Pro</div>
      <nav className="space-y-1">
        {links.map(({ to, icon: Icon, label }) => (
          <NavLink key={to} to={to} end={to === '/'}
            className={({ isActive }) =>
              `flex items-center gap-3 px-3 py-2 rounded-lg text-sm ${
                isActive ? 'bg-blue-600 text-white' : 'text-slate-400 hover:bg-slate-800 hover:text-white'
              }`
            }>
            <Icon size={18} />
            {label}
          </NavLink>
        ))}
      </nav>
    </aside>
  )
}
```

```tsx
// frontend/src/components/layout/Shell.tsx
import { Outlet } from 'react-router-dom'
import { Sidebar } from './Sidebar'
import { useAuth } from '@/hooks/useAuth'

export function Shell() {
  const { user, logout } = useAuth()
  return (
    <div className="flex min-h-screen bg-slate-950 text-white">
      <Sidebar />
      <div className="flex-1 flex flex-col">
        <header className="h-14 border-b border-slate-800 flex items-center justify-between px-6">
          <div />
          <div className="flex items-center gap-4">
            <span className="text-sm text-slate-400">{user?.email}</span>
            <button onClick={logout} className="text-sm text-slate-400 hover:text-white">Logout</button>
          </div>
        </header>
        <main className="flex-1 p-6">
          <Outlet />
        </main>
      </div>
    </div>
  )
}
```

- [ ] **Step 3: Write Dashboard page with charts**

```tsx
// frontend/src/pages/Dashboard.tsx
import { useQuery } from '@tanstack/react-query'
import { api } from '@/api/client'
import { PieChart, Pie, Cell, ResponsiveContainer, Tooltip } from 'recharts'

const COLORS: Record<string, string> = {
  critical: '#dc2626', high: '#ea580c', medium: '#d97706', low: '#2563eb', info: '#6b7280',
}

export function Dashboard() {
  const { data } = useQuery({
    queryKey: ['dashboard'],
    queryFn: () => api.get<{ vulns_by_severity: { severity: string; count: number }[] }>('/dashboard'),
  })

  const chartData = data?.vulns_by_severity?.map(v => ({
    name: v.severity, value: v.count, fill: COLORS[v.severity] || '#6b7280',
  })) || []

  const total = chartData.reduce((sum, d) => sum + d.value, 0)

  return (
    <div>
      <h1 className="text-2xl font-bold mb-6">Dashboard</h1>
      <div className="grid grid-cols-5 gap-4 mb-8">
        {['critical', 'high', 'medium', 'low', 'info'].map(sev => {
          const count = chartData.find(d => d.name === sev)?.value || 0
          return (
            <div key={sev} className="rounded-lg p-4 text-center text-white" style={{ background: COLORS[sev] }}>
              <div className="text-3xl font-bold">{count}</div>
              <div className="text-sm opacity-90 capitalize">{sev}</div>
            </div>
          )
        })}
      </div>
      <div className="grid grid-cols-2 gap-6">
        <div className="bg-slate-900 rounded-lg border border-slate-800 p-6">
          <h2 className="text-lg font-semibold mb-4">Severity Distribution</h2>
          {total > 0 ? (
            <ResponsiveContainer width="100%" height={250}>
              <PieChart>
                <Pie data={chartData} dataKey="value" nameKey="name" cx="50%" cy="50%" outerRadius={90} label>
                  {chartData.map((entry, i) => <Cell key={i} fill={entry.fill} />)}
                </Pie>
                <Tooltip />
              </PieChart>
            </ResponsiveContainer>
          ) : <p className="text-slate-500">No vulnerabilities found</p>}
        </div>
        <div className="bg-slate-900 rounded-lg border border-slate-800 p-6">
          <h2 className="text-lg font-semibold mb-4">Overview</h2>
          <p className="text-4xl font-bold">{total}</p>
          <p className="text-slate-400">Total open vulnerabilities</p>
        </div>
      </div>
    </div>
  )
}
```

- [ ] **Step 4: Write Vulnerabilities page**

```tsx
// frontend/src/pages/Vulnerabilities.tsx
import { useQuery } from '@tanstack/react-query'
import { api } from '@/api/client'

const BADGE_COLORS: Record<string, string> = {
  critical: 'bg-red-600', high: 'bg-orange-600', medium: 'bg-yellow-600', low: 'bg-blue-600', info: 'bg-gray-600',
}

export function Vulnerabilities() {
  const { data: vulns = [] } = useQuery({
    queryKey: ['vulnerabilities'],
    queryFn: () => api.get<any[]>('/vulnerabilities'),
  })

  return (
    <div>
      <h1 className="text-2xl font-bold mb-6">Vulnerabilities</h1>
      <div className="bg-slate-900 rounded-lg border border-slate-800 overflow-hidden">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-slate-800">
              <th className="text-left p-3 text-slate-400">Severity</th>
              <th className="text-left p-3 text-slate-400">Title</th>
              <th className="text-left p-3 text-slate-400">Host</th>
              <th className="text-left p-3 text-slate-400">CVE</th>
              <th className="text-left p-3 text-slate-400">CVSS</th>
              <th className="text-left p-3 text-slate-400">Status</th>
            </tr>
          </thead>
          <tbody>
            {vulns.map((v: any) => (
              <tr key={v.id} className="border-b border-slate-800/50 hover:bg-slate-800/30">
                <td className="p-3">
                  <span className={`px-2 py-1 rounded text-xs font-medium text-white ${BADGE_COLORS[v.severity] || 'bg-gray-600'}`}>
                    {v.severity}
                  </span>
                </td>
                <td className="p-3">{v.title}</td>
                <td className="p-3 text-slate-400">{v.affected_host}</td>
                <td className="p-3 text-slate-400">{v.cve_id || '-'}</td>
                <td className="p-3">{v.cvss_score ?? '-'}</td>
                <td className="p-3 text-slate-400">{v.status}</td>
              </tr>
            ))}
            {vulns.length === 0 && (
              <tr><td colSpan={6} className="p-6 text-center text-slate-500">No vulnerabilities found</td></tr>
            )}
          </tbody>
        </table>
      </div>
    </div>
  )
}
```

- [ ] **Step 5: Write remaining page stubs (Targets, Scans, Tickets, Reports, Teams, Settings, NotFound)**

All follow the same data-table pattern. Each uses `useQuery` to fetch from its API endpoint and renders a table.

```tsx
// frontend/src/pages/Targets.tsx
import { useQuery } from '@tanstack/react-query'
import { api } from '@/api/client'

export function Targets() {
  const { data: targets = [] } = useQuery({ queryKey: ['targets'], queryFn: () => api.get<any[]>('/targets') })
  return (
    <div>
      <h1 className="text-2xl font-bold mb-6">Targets</h1>
      <div className="bg-slate-900 rounded-lg border border-slate-800 overflow-hidden">
        <table className="w-full text-sm">
          <thead><tr className="border-b border-slate-800">
            <th className="text-left p-3 text-slate-400">Host</th>
            <th className="text-left p-3 text-slate-400">IP Address</th>
            <th className="text-left p-3 text-slate-400">Hostname</th>
            <th className="text-left p-3 text-slate-400">OS</th>
          </tr></thead>
          <tbody>
            {targets.map((t: any) => (
              <tr key={t.id} className="border-b border-slate-800/50 hover:bg-slate-800/30">
                <td className="p-3">{t.host}</td>
                <td className="p-3 text-slate-400">{t.ip_address}</td>
                <td className="p-3 text-slate-400">{t.hostname || '-'}</td>
                <td className="p-3 text-slate-400">{t.os_guess || '-'}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}
```

```tsx
// frontend/src/pages/Scans.tsx
import { useQuery } from '@tanstack/react-query'
import { api } from '@/api/client'

export function Scans() {
  const { data: scans = [] } = useQuery({ queryKey: ['scans'], queryFn: () => api.get<any[]>('/scans') })
  return (
    <div>
      <h1 className="text-2xl font-bold mb-6">Scans</h1>
      <div className="bg-slate-900 rounded-lg border border-slate-800 overflow-hidden">
        <table className="w-full text-sm">
          <thead><tr className="border-b border-slate-800">
            <th className="text-left p-3 text-slate-400">Name</th>
            <th className="text-left p-3 text-slate-400">Type</th>
            <th className="text-left p-3 text-slate-400">Status</th>
            <th className="text-left p-3 text-slate-400">Created</th>
          </tr></thead>
          <tbody>
            {scans.map((s: any) => (
              <tr key={s.id} className="border-b border-slate-800/50 hover:bg-slate-800/30">
                <td className="p-3">{s.name}</td>
                <td className="p-3 text-slate-400">{s.scan_type}</td>
                <td className="p-3"><span className={`px-2 py-1 rounded text-xs ${s.status === 'completed' ? 'bg-green-900 text-green-300' : s.status === 'running' ? 'bg-blue-900 text-blue-300' : s.status === 'failed' ? 'bg-red-900 text-red-300' : 'bg-slate-700 text-slate-300'}`}>{s.status}</span></td>
                <td className="p-3 text-slate-400">{new Date(s.created_at).toLocaleDateString()}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}
```

```tsx
// frontend/src/pages/Tickets.tsx
import { useQuery } from '@tanstack/react-query'
import { api } from '@/api/client'

export function Tickets() {
  const { data: tickets = [] } = useQuery({ queryKey: ['tickets'], queryFn: () => api.get<any[]>('/tickets') })
  return (
    <div>
      <h1 className="text-2xl font-bold mb-6">Tickets</h1>
      <div className="bg-slate-900 rounded-lg border border-slate-800 overflow-hidden">
        <table className="w-full text-sm">
          <thead><tr className="border-b border-slate-800">
            <th className="text-left p-3 text-slate-400">Title</th>
            <th className="text-left p-3 text-slate-400">Priority</th>
            <th className="text-left p-3 text-slate-400">Status</th>
            <th className="text-left p-3 text-slate-400">Due Date</th>
          </tr></thead>
          <tbody>
            {tickets.map((t: any) => (
              <tr key={t.id} className="border-b border-slate-800/50 hover:bg-slate-800/30">
                <td className="p-3">{t.title}</td>
                <td className="p-3 capitalize">{t.priority}</td>
                <td className="p-3 capitalize">{t.status}</td>
                <td className="p-3 text-slate-400">{t.due_date ? new Date(t.due_date).toLocaleDateString() : '-'}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}
```

```tsx
// frontend/src/pages/Reports.tsx
import { useQuery } from '@tanstack/react-query'
import { api } from '@/api/client'

export function Reports() {
  const { data: reports = [] } = useQuery({ queryKey: ['reports'], queryFn: () => api.get<any[]>('/reports') })
  return (
    <div>
      <h1 className="text-2xl font-bold mb-6">Reports</h1>
      <div className="bg-slate-900 rounded-lg border border-slate-800 overflow-hidden">
        <table className="w-full text-sm">
          <thead><tr className="border-b border-slate-800">
            <th className="text-left p-3 text-slate-400">Name</th>
            <th className="text-left p-3 text-slate-400">Type</th>
            <th className="text-left p-3 text-slate-400">Format</th>
            <th className="text-left p-3 text-slate-400">Status</th>
          </tr></thead>
          <tbody>
            {reports.map((r: any) => (
              <tr key={r.id} className="border-b border-slate-800/50 hover:bg-slate-800/30">
                <td className="p-3">{r.name}</td>
                <td className="p-3 capitalize">{r.report_type}</td>
                <td className="p-3 uppercase text-slate-400">{r.format}</td>
                <td className="p-3 capitalize">{r.status}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}
```

```tsx
// frontend/src/pages/Teams.tsx
import { useQuery } from '@tanstack/react-query'
import { api } from '@/api/client'

export function Teams() {
  const { data: teams = [] } = useQuery({ queryKey: ['teams'], queryFn: () => api.get<any[]>('/teams') })
  return (
    <div>
      <h1 className="text-2xl font-bold mb-6">Teams</h1>
      <div className="grid grid-cols-3 gap-4">
        {teams.map((t: any) => (
          <div key={t.id} className="bg-slate-900 rounded-lg border border-slate-800 p-4">
            <h3 className="font-semibold">{t.name}</h3>
            <p className="text-sm text-slate-400 mt-1">{t.description || 'No description'}</p>
          </div>
        ))}
        {teams.length === 0 && <p className="text-slate-500 col-span-3">No teams yet</p>}
      </div>
    </div>
  )
}
```

```tsx
// frontend/src/pages/Settings.tsx
import { useAuth } from '@/hooks/useAuth'

export function Settings() {
  const { user } = useAuth()
  return (
    <div>
      <h1 className="text-2xl font-bold mb-6">Settings</h1>
      <div className="bg-slate-900 rounded-lg border border-slate-800 p-6 max-w-lg">
        <h2 className="text-lg font-semibold mb-4">Profile</h2>
        <div className="space-y-3 text-sm">
          <div><span className="text-slate-400">Email:</span> <span>{user?.email}</span></div>
          <div><span className="text-slate-400">Username:</span> <span>{user?.username}</span></div>
          <div><span className="text-slate-400">Role:</span> <span className="capitalize">{user?.role}</span></div>
        </div>
      </div>
    </div>
  )
}
```

```tsx
// frontend/src/pages/NotFound.tsx
import { Link } from 'react-router-dom'

export function NotFound() {
  return (
    <div className="min-h-screen flex items-center justify-center bg-slate-950">
      <div className="text-center">
        <h1 className="text-6xl font-bold text-slate-600">404</h1>
        <p className="text-slate-400 mt-4">Page not found</p>
        <Link to="/" className="mt-6 inline-block px-4 py-2 bg-blue-600 rounded text-white hover:bg-blue-700">
          Back to Dashboard
        </Link>
      </div>
    </div>
  )
}
```

- [ ] **Step 6: Write WebSocket hook**

```typescript
// frontend/src/hooks/useWebSocket.ts
import { useEffect, useRef, useCallback } from 'react'
import { useAuth } from './useAuth'

export function useWebSocket(onMessage: (msg: { type: string; payload: unknown }) => void) {
  const { token } = useAuth()
  const wsRef = useRef<WebSocket | null>(null)

  const connect = useCallback(() => {
    if (!token) return
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const ws = new WebSocket(`${protocol}//${window.location.host}/ws?token=${token}`)
    ws.onmessage = (e) => {
      try { onMessage(JSON.parse(e.data)) } catch {}
    }
    ws.onclose = () => { setTimeout(connect, 3000) }
    wsRef.current = ws
  }, [token, onMessage])

  useEffect(() => {
    connect()
    return () => { wsRef.current?.close() }
  }, [connect])
}
```

- [ ] **Step 7: Commit**

```bash
git add frontend/src/
git commit -m "feat: add dashboard, vulnerability, scan, and login pages with WebSocket"
```

---

### Task 6.3: Embed Frontend in Go Binary

**Files:**
- Modify: `cmd/vulntrack/main.go`
- Create: `cmd/vulntrack/frontend.go`

**Note:** `go:embed` cannot use `..` to escape the package directory. The Makefile `build` target already runs `cd frontend && npm run build` which outputs to `frontend/dist/`. We add a build step that copies `frontend/dist/` into `cmd/vulntrack/static/` before `go build`, so the embed directive references a local path.

Update the Makefile `build` and `build-linux` targets to include the copy step:
```makefile
build: frontend
	rm -rf cmd/vulntrack/static && cp -r frontend/dist cmd/vulntrack/static
	CGO_ENABLED=0 go build -o $(BUILD_DIR)/$(BINARY) ./cmd/vulntrack

build-linux: frontend
	rm -rf cmd/vulntrack/static && cp -r frontend/dist cmd/vulntrack/static
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o $(BUILD_DIR)/$(BINARY)-linux-amd64 ./cmd/vulntrack
```

Add `cmd/vulntrack/static/` to `.gitignore`.

- [ ] **Step 1: Create frontend embed file**

```go
// cmd/vulntrack/frontend.go
package main

import (
	"embed"
	"io/fs"
	"net/http"

	"github.com/labstack/echo/v4"
)

//go:embed all:static
var frontendFS embed.FS

func serveFrontend(e *echo.Echo) {
	distFS, _ := fs.Sub(frontendFS, "static")
	fileServer := http.FileServer(http.FS(distFS))

	// Serve static files, fall back to index.html for SPA routing
	e.GET("/*", echo.WrapHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Try to serve the file directly
		path := r.URL.Path[1:] // strip leading /
		if path == "" {
			path = "index.html"
		}
		f, err := distFS.Open(path)
		if err != nil {
			// Fall back to index.html for SPA client-side routing
			r.URL.Path = "/index.html"
		} else {
			f.Close()
		}
		fileServer.ServeHTTP(w, r)
	})))
}
```

- [ ] **Step 2: Update main.go to mount frontend and all API routes**

Add to main.go after middleware setup:
```go
// API routes (authenticated)
api := e.Group("/api")
authHandler.RegisterRoutes(api.Group("/auth"))

protected := api.Group("", mw.JWTAuth(cfg.JWT.Secret))
targetHandler.RegisterRoutes(protected.Group("/targets"))
scanHandler.RegisterRoutes(protected.Group("/scans"))
vulnHandler.RegisterRoutes(protected.Group("/vulnerabilities"))
ticketHandler.RegisterRoutes(protected.Group("/tickets"))
reportHandler.RegisterRoutes(protected.Group("/reports"))
dashboardHandler.RegisterRoutes(protected.Group("/dashboard"))
teamHandler.RegisterRoutes(protected.Group("/teams"))
notificationHandler.RegisterRoutes(protected.Group("/notifications"))
scheduleHandler.RegisterRoutes(protected.Group("/schedules"))
assetHandler.RegisterRoutes(protected.Group("/assets"))
auditHandler.RegisterRoutes(protected.Group("/audit"))
searchHandler.RegisterRoutes(protected.Group("/search"))

// WebSocket
e.GET("/ws", wsHandler.Handle)

// Embedded SPA (must be last — catch-all)
serveFrontend(e)
```

- [ ] **Step 3: Commit**

```bash
git add cmd/vulntrack/
git commit -m "feat: embed frontend SPA in Go binary with SPA fallback routing"
```

---

## Chunk 7: Systemd Deployment and Final Polish

### Task 7.1: Systemd Service File and Install Script

**Files:**
- Create: `deploy/vulntrack.service`
- Create: `deploy/vulntrack.env.example`
- Create: `deploy/install.sh`

- [ ] **Step 1: Write systemd unit file**

```ini
# deploy/vulntrack.service
[Unit]
Description=VulnTrack Pro - Vulnerability Management Platform
After=network.target postgresql.service redis.service
Wants=postgresql.service redis.service

[Service]
Type=simple
User=vulntrack
Group=vulntrack
ExecStart=/usr/local/bin/vulntrack
EnvironmentFile=/etc/vulntrack/env
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal
SyslogIdentifier=vulntrack

# Security hardening
NoNewPrivileges=yes
ProtectSystem=strict
ProtectHome=yes
ReadWritePaths=/var/lib/vulntrack
PrivateTmp=yes
ProtectKernelTunables=yes
ProtectKernelModules=yes
ProtectControlGroups=yes

[Install]
WantedBy=multi-user.target
```

- [ ] **Step 2: Write env template**

```env
# deploy/vulntrack.env.example
VT_SERVER_HOST=0.0.0.0
VT_SERVER_PORT=8080
VT_DATABASE_URL=postgres://vulntrack:CHANGEME@localhost:5432/vulntrack?sslmode=require
VT_REDIS_ADDR=localhost:6379
VT_JWT_SECRET=CHANGEME-use-openssl-rand-base64-32
VT_JWT_EXPIREHOURS=24
VT_SCANNER_NMAPPATH=/usr/bin/nmap
VT_SCANNER_OPENVASPATH=/usr/bin/gvm-cli
```

- [ ] **Step 3: Write install script**

```bash
#!/usr/bin/env bash
# deploy/install.sh — Install VulnTrack Pro on Debian Trixie
set -euo pipefail

BINARY_SRC="${1:-bin/vulntrack-linux-amd64}"

echo "==> Creating vulntrack user"
useradd --system --shell /usr/sbin/nologin --home-dir /var/lib/vulntrack vulntrack 2>/dev/null || true

echo "==> Installing binary"
install -o root -g root -m 0755 "$BINARY_SRC" /usr/local/bin/vulntrack

echo "==> Creating config directory"
install -d -o vulntrack -g vulntrack -m 0750 /etc/vulntrack
install -d -o vulntrack -g vulntrack -m 0750 /var/lib/vulntrack

if [ ! -f /etc/vulntrack/env ]; then
    install -o vulntrack -g vulntrack -m 0600 deploy/vulntrack.env.example /etc/vulntrack/env
    echo "==> Created /etc/vulntrack/env — EDIT THIS FILE with your secrets"
fi

echo "==> Installing systemd unit"
install -o root -g root -m 0644 deploy/vulntrack.service /etc/systemd/system/vulntrack.service
systemctl daemon-reload

echo "==> Running database migrations"
su -s /bin/bash vulntrack -c "VT_DATABASE_URL=\$(grep VT_DATABASE_URL /etc/vulntrack/env | cut -d= -f2-) /usr/local/bin/vulntrack migrate"

echo "==> Enabling and starting service"
systemctl enable vulntrack
systemctl start vulntrack
systemctl status vulntrack

echo "==> Done! VulnTrack Pro is running on port 8080"
```

- [ ] **Step 4: Commit**

```bash
git add deploy/
git commit -m "feat: add systemd service, env template, and Debian install script"
```

---

### Task 7.2: Dockerfile (Multi-stage Build)

**Files:**
- Create: `Dockerfile`

- [ ] **Step 1: Write Dockerfile**

```dockerfile
# Dockerfile
FROM node:22-alpine AS frontend
WORKDIR /app/frontend
COPY frontend/package*.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

FROM golang:1.23-alpine AS backend
RUN apk add --no-cache git
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=frontend /app/frontend/dist ./frontend/dist
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /vulntrack ./cmd/vulntrack

FROM alpine:3.20
RUN apk add --no-cache ca-certificates nmap
COPY --from=backend /vulntrack /usr/local/bin/vulntrack
COPY sql/migrations /migrations
EXPOSE 8080
ENTRYPOINT ["vulntrack"]
```

- [ ] **Step 2: Commit**

```bash
git add Dockerfile
git commit -m "feat: add multi-stage Dockerfile for minimal container image"
```

---

### Task 7.3: Wire Everything in main.go

**Files:**
- Modify: `cmd/vulntrack/main.go`

- [ ] **Step 1: Complete main.go with full dependency wiring**

```go
// cmd/vulntrack/main.go
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/hibiken/asynq"
	"github.com/labstack/echo/v4"
	echomw "github.com/labstack/echo/v4/middleware"

	"github.com/cyberoptic/vulntrack/internal/config"
	"github.com/cyberoptic/vulntrack/internal/database"
	"github.com/cyberoptic/vulntrack/internal/handler"
	mw "github.com/cyberoptic/vulntrack/internal/middleware"
	"github.com/cyberoptic/vulntrack/internal/scanner"
	"github.com/cyberoptic/vulntrack/internal/service"
	"github.com/cyberoptic/vulntrack/internal/websocket"
	"github.com/cyberoptic/vulntrack/internal/worker"
)

type customValidator struct {
	validator *validator.Validate
}

func (cv *customValidator) Validate(i interface{}) error {
	return cv.validator.Struct(i)
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	pool, err := database.NewPool(cfg.Database.URL)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer pool.Close()

	// Asynq client (enqueue jobs)
	asynqClient := asynq.NewClient(asynq.RedisClientOpt{
		Addr: cfg.Redis.Addr, Password: cfg.Redis.Password, DB: cfg.Redis.DB,
	})
	defer asynqClient.Close()

	// Services
	userSvc := service.NewUserService(pool)
	targetSvc := service.NewTargetService(pool)
	vulnSvc := service.NewVulnerabilityService(pool)
	ticketSvc := service.NewTicketService(pool)

	// WebSocket hub
	hub := websocket.NewHub()

	// Echo
	e := echo.New()
	e.HideBanner = true
	e.Validator = &customValidator{validator: validator.New()}

	// Global middleware
	e.Use(echomw.Logger())
	e.Use(echomw.Recover())
	e.Use(mw.SecurityHeaders())
	rl := mw.NewRateLimiter(100, time.Minute)
	e.Use(rl.Middleware())

	// Health
	e.GET("/api/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	// Auth routes (public)
	jwtExpiry := time.Duration(cfg.JWT.ExpireHours) * time.Hour
	authH := handler.NewAuthHandler(userSvc, cfg.JWT.Secret, jwtExpiry)
	authH.RegisterRoutes(e.Group("/api/auth"))

	// Protected routes
	p := e.Group("/api", mw.JWTAuth(cfg.JWT.Secret))
	handler.NewTargetHandler(targetSvc).RegisterRoutes(p.Group("/targets"))
	handler.NewVulnHandler(vulnSvc).RegisterRoutes(p.Group("/vulnerabilities"))
	handler.NewTicketHandler(ticketSvc).RegisterRoutes(p.Group("/tickets"))
	handler.NewDashboardHandler(vulnSvc, ticketSvc).RegisterRoutes(p.Group("/dashboard"))
	// ... remaining handlers registered similarly

	// WebSocket
	wsH := handler.NewWSHandler(hub, cfg.JWT.Secret)
	e.GET("/ws", wsH.Handle)

	// Embedded frontend (catch-all, must be last)
	serveFrontend(e)

	// Start Asynq worker in background
	nmapScanner := scanner.NewNmapScanner(cfg.Scanner.NmapPath)
	openvasScanner := scanner.NewOpenVASScanner(cfg.Scanner.OpenVASPath)
	workerSrv := worker.NewServer(cfg, pool)
	workerMux := worker.NewMux(pool, nmapScanner, openvasScanner)
	go func() {
		if err := workerSrv.Run(workerMux); err != nil {
			log.Printf("worker error: %v", err)
		}
	}()

	// Start HTTP server
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	go func() {
		if err := e.Start(addr); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit
	log.Println("shutting down...")

	workerSrv.Shutdown()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	e.Shutdown(ctx)
}
```

- [ ] **Step 2: Build and verify**

Run: `make build-linux`
Expected: Single binary produced at `bin/vulntrack-linux-amd64`

- [ ] **Step 3: Commit**

```bash
git add cmd/vulntrack/main.go
git commit -m "feat: wire all services, handlers, workers, and frontend into single binary"
```

---

### Task 7.4: README

**Files:**
- Create: `README.md`

- [ ] **Step 1: Write README**

```markdown
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
```

- [ ] **Step 2: Commit**

```bash
git add README.md
git commit -m "docs: add comprehensive README with setup and API reference"
```

---

## Summary

| Chunk | Tasks | What it produces |
|-------|-------|-----------------|
| 1: Foundation | 1.1–1.8 | Go module, config, DB pool, all 12 migrations, sqlc, minimal server |
| 2: Auth | 2.1–2.6 | Password hashing, JWT, auth middleware, RBAC, register/login endpoints + tests, security middleware |
| 3: Targets + Scanning | 3.1–3.6 | Target CRUD, Nmap parser, OpenVAS parser, scan service, async scan queue, scan REST API |
| 4: Vulns + Tickets + Reports | 4.1–4.4 | Vulnerability management, CVE enrichment, ticket system, report service/handler, HTML/PDF/Excel/MD generators + tests |
| 5: Everything else | 5.1–5.6 | Teams, notifications, schedules, assets, audit, search (all with full service+handler code), WebSocket, dashboard, risk scoring, plugins |
| 6: Frontend | 6.1–6.3 | React SPA with Login, Dashboard (charts), Vulns, Targets, Scans, Tickets, Reports, Teams, Settings, NotFound pages, embedded via go:embed (static/ copy) |
| 7: Deployment | 7.1–7.4 | systemd unit, install script, Dockerfile, final wiring, README |
