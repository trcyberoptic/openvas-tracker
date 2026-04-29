## 2026-04-29 - [Missing RBAC Middleware]
**Vulnerability:** Missing authorization checks on sensitive endpoints (/settings, /audit, and ticket status updates).
**Learning:** The application's routes were correctly separated but missing the RBAC middleware required to enforce role checks.
**Prevention:** Apply middleware.RequireRole middleware to all sensitive routes during route registration.
