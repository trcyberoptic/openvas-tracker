## 2025-05-30 - [Missing Authorization on Ticket State Changes]
**Vulnerability:** Found broken access control in `internal/handler/tickets.go` where `UpdateStatus`, `Assign`, `BulkUpdate`, and `CreateRiskRule` routes were missing the `RequireRole` middleware, despite a comment in `main.go` indicating they should require 'admin' or 'analyst' roles.
**Learning:** Comments indicating security controls do not guarantee their implementation. Route registration in `echo` can easily miss applying crucial middleware if not explicitly chained or grouped.
**Prevention:** Always verify that security-related middleware (like RBAC) is actually applied to the route definitions, rather than relying on comments or assuming group-level middleware covers specific handlers.
