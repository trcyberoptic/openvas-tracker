# Threat model: openvas-tracker

## What this project does and where untrusted input enters

openvas-tracker is a self-hosted vulnerability-management dashboard: a Go (Echo) backend with an
embedded React SPA and a MariaDB database. It does no scanning itself. OpenVAS/Greenbone and OWASP ZAP
results arrive via an import webhook, are stored per host/URL, and turn into remediation tickets.
It is typically deployed on an internal network behind a reverse proxy, run by a security/IT team.

Trust boundaries, from least to most trusted:

1. **Unauthenticated network clients**: `POST /api/auth/login` (env admin, then LDAP bind, then
   local DB users), `GET /api/health`, the static SPA (`/*`), and `GET /ws?token=` (JWT in the query).
   Rate limits: 60/min/IP on `/api/auth`, 500/min/IP globally; `X-Forwarded-For` is honoured only from
   a loopback peer.
2. **Content of scan reports** — the most interesting boundary. Reports come in through the API key,
   but their *content* is attacker-influenced: anyone who controls a scanned host or web application
   controls banners, hostnames (PTR records), HTTP responses, ZAP `evidence`/`param`/`uri`, OpenVAS
   result descriptions, etc. That text is parsed (`internal/scanner/`), stored, and rendered in the
   SPA, in HTML/PDF/Excel/Markdown reports (`internal/report/`), and in notification mails
   (`internal/service/mailnotify.go`). Treat every string field of a report as hostile.
3. **Holders of the import API key** (`X-API-Key` header or `?api_key=`): `POST /api/import/openvas`
   (GMP XML), `POST /api/import/zap` (ZAP JSON), `POST /api/import/feeds` (GMP get_feeds XML), and
   `GET /api/import/openvas`, which runs `sudo /usr/local/bin/openvas-tracker-fetch-latest` (source:
   `deploy/openvas-tracker-fetch-latest`, runs as **root**). Bodies up to 50 MB. The key is a machine
   credential, not an admin credential: a key holder must not get code execution, file access, or
   admin rights in the app.
4. **Authenticated users, role `viewer`** (every auto-created LDAP user). By design they may read and
   change all tickets, scans, reports, risk-accept rules and the user list — there is no per-ticket
   ownership. They must NOT reach admin-only routes (guarded by `RequireRole("admin")`):
   `/api/settings/{env,env/batch,ldap/test,mail,mail/test}` and `POST`/`DELETE` on `/api/teams/:id*`,
   and must not see the unmasked import key or other secrets.
5. **Admins**. Can edit a fixed allowlist of `.env` keys via the UI (`internal/service/envfile.go`,
   `editableEnvKeys`), configure LDAP and SMTP. Admin must still not be able to reach code execution on
   the host, write env keys outside the allowlist (`OT_JWT_SECRET`, `OT_IMPORT_APIKEY`,
   `OT_ADMIN_PASSWORD`, `OT_DATABASE_DSN`), inject extra lines into the env file (CR/LF), or escalate to
   root through the sudo fetch script (which reads `OT_GMP_USER`/`OT_GMP_PASSWORD` from that same env
   file).

Out of the trust model: the host OS, MariaDB itself, the Greenbone/GVM stack and its socket, the LDAP
server, and the SMTP relay are trusted infrastructure.

## Components that matter most / least

Most important:
- `internal/handler/auth.go`, `internal/service/ldap.go`, `internal/auth/`, `internal/middleware/`
  (`auth.go` JWT, `apikey.go`, `rbac.go`, `ratelimit.go`, `security.go` CSP): authentication bypass,
  JWT confusion, LDAP filter injection, role escalation, rate-limit bypass.
- `internal/scanner/` (`openvas.go`, `zap.go`, `feeds.go`) and `internal/service/import.go`: parser
  robustness on hostile input, transaction integrity, SQL built from report content.
- `internal/database/queries/`: hand-maintained SQL with string-concatenated column lists and dynamic
  filters/sorting — look for SQL injection via query parameters (filters, sort keys, pagination).
- Rendering of stored scan data: React SPA (`frontend/src/`, including the ticket detail page that shows
  URLs/evidence/CWE links), `internal/report/` (HTML via `html/template`, Excel — formula injection,
  Markdown, PDF), notification mails.
- `internal/handler/settings.go` + `internal/service/envfile.go`: env-file write path, masking of
  secrets in `GET /api/settings/env`, SMTP host/password handling (`passwordRequiredOnHostChange`).
- `deploy/openvas-tracker-fetch-latest` (Python, runs as root via sudo) and `deploy/openvas-tracker-sudoers`.
- `cmd/openvas-tracker/main.go`: route wiring, which routes sit inside the JWT group, body limits.

Less important / dormant (still in scope if actually reachable):
- Teams, Assets, Audit, Notifications, Search handlers: backend-only, mostly no producer of data.
- `internal/websocket/`: the hub never broadcasts anything; only the `/ws` auth handshake matters.
- `UserService.Register` has no route (dead code). `middleware/audit.go` is wired to no route.

Out of scope:
- `docker-compose.yml` hardcodes dev credentials (`admin`/`admin`, fixed secrets) on purpose; it is a
  local demo, not a deployment path.
- `deploy/install.sh` (one-shot installer run by an administrator as root).
- `frontend/node_modules`, vendored or third-party code.

## How to exercise it

The image is built at `/src`; the binary is `/src/bin/openvas-tracker` (debug build, frontend embedded).

- Unit tests: `cd /src && go test ./... -count=1` (no database needed). Single package:
  `go test ./internal/scanner/ -v`. Frontend: `cd /src/frontend && npm run lint && npx tsc -b`.
- Running instance: `start-tracker` (from `.oss-scanner/start-tracker.sh`) starts MariaDB and the app
  on `http://127.0.0.1:8080` with throwaway credentials, and creates two logins:
  `admin` / `scanner-admin-password-0123456789` (role admin) and
  `viewer` / `scanner-viewer-password-012345678` (role viewer). Import key:
  `scanner-import-apikey-0123456789abcdef`. App log: `/tmp/openvas-tracker.log`; DB shell: `mariadb -uroot openvas-tracker`.
- Sample data: `/src/testdata/openvas-sample-report.xml` (GMP report). ZAP fixtures live in
  `internal/scanner/zap_test.go`. Import with
  `curl -X POST -H 'X-API-Key: <key>' --data-binary @file http://127.0.0.1:8080/api/import/openvas`
  (or `/api/import/zap`).
- API calls need `Authorization: Bearer <token>` from `POST /api/auth/login`
  (`{"username":..,"password":..}`).
- `GET /api/import/openvas` (sudo fetch script) cannot work in this image: there is no GVM socket and no
  sudoers rule. Review `deploy/openvas-tracker-fetch-latest` statically, or run it directly with
  `OT_ENV_FILE`/`OT_GMP_SOCKET` pointing at test fixtures.
- LDAP and SMTP have no server in the image; review those paths statically or with a stub listener.

## How you rate severity

- **Critical**: unauthenticated remote code execution; unauthenticated authentication bypass (login as
  any user, forged JWT); unauthenticated SQL injection; root on the host via the sudo fetch script from
  anything less than root.
- **High**: code execution or arbitrary file read/write from the import API key or from an admin
  account; authenticated SQL injection; viewer → admin privilege escalation or reaching any
  `RequireRole("admin")` route as a viewer; stored XSS triggered by **scan report content** (a scanned
  host owner can plant it without any account) in the SPA or HTML reports; leaking secrets
  (JWT secret, import key, admin/LDAP/SMTP passwords, DSN) to non-admins or unauthenticated clients;
  writing non-allowlisted keys or extra lines into the env file.
- **Medium**: stored XSS that requires an authenticated account to plant (e.g. ticket comments, risk
  rule fields); CSV/Excel formula injection from report content; SSRF via LDAP/SMTP settings beyond what
  admin is meant to configure; rate-limit or lockout bypass on login; LDAP filter injection that changes
  who can log in; crashes or unbounded memory/CPU from a single import request (the import key holder is
  semi-trusted); data integrity bugs where a crafted report silently closes, merges, or risk-accepts
  unrelated tickets.
- **Low**: DoS needing many requests, information leaks of non-secret metadata (versions, hostnames
  already visible to all users), missing hardening headers, timing differences without a practical attack.
- Anything only reachable from `docker-compose.yml`'s hardcoded dev credentials is informational.

Please include a reproducer (curl sequence against `start-tracker`, or a Go test) and a minimal patch
in the style of the surrounding code.

## Anything to leave alone

- **By design, not bugs**: every authenticated user (including `viewer`) can read and modify all tickets,
  scans, reports, risk-accept rules and see the user list; there is no per-object authorization.
- The admin login is env-only (`OT_ADMIN_PASSWORD`) and never falls through to LDAP/DB — intended.
- `smtp_password` is stored in plaintext in the `app_settings` table, and the setup endpoint returns the
  masked import key to admins — documented trade-offs.
- `X-Forwarded-For` is trusted only from loopback (reverse proxy on the same host) — intended.
- The import API key may be passed as `?api_key=` because the GVM "HTTP Get" alert cannot set headers;
  the access log deliberately omits query strings.
- The only outbound call is the GitHub releases check in `settings.go`; reporting that it exists is not useful.
- Dormant features listed above with no reachable entry point.
