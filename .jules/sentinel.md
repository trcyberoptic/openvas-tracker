## 2025-04-26 - [Missing RBAC Checks]
**Vulnerability:** Missing authorization checks on sensitive endpoints (`/audit` and `/settings`).
**Learning:** Even with JWT authentication, specific routes were missing Role-Based Access Control (RBAC) middleware (`RequireRole`), allowing lower-privileged users to access admin-only data.
**Prevention:** Always verify that every route registration, especially those grouped under sensitive paths, explicitly applies the appropriate RBAC middleware.
