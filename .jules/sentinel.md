## 2025-05-06 - Prevent Timing Attacks in Password Comparison
**Vulnerability:** The application used a simple string comparison (`==`) for checking the admin password. This could allow an attacker to perform a timing attack to guess the password byte-by-byte.
**Learning:** Hardcoded direct string comparisons for sensitive data like passwords or tokens create timing side-channels.
**Prevention:** Always use constant-time comparison functions like `crypto/subtle.ConstantTimeCompare` when comparing sensitive strings or byte slices to prevent timing attacks.

## 2025-05-17 - Prevent Length-Based Timing Attacks in ConstantTimeCompare
**Vulnerability:** Even when using `crypto/subtle.ConstantTimeCompare`, an early return based on unequal slice lengths can leak information about the expected length of a sensitive string (like a password or API key).
**Learning:** `ConstantTimeCompare` performs a constant-time comparison *only* if the lengths match. If lengths differ, it immediately returns 0. This creates a timing leak that reveals the target string length.
**Prevention:** Hash both strings (e.g. using `sha256.Sum256`) before passing them to `ConstantTimeCompare`. This ensures both inputs always have the same length (e.g. 32 bytes) regardless of the original string lengths.

## 2025-06-05 - Enforce Struct Validation Tags in Echo Handlers
**Vulnerability:** The application used `c.Bind(&req)` to bind incoming JSON requests to structs but failed to subsequently call `c.Validate(&req)`. Because `c.Bind()` alone does not enforce `validate:"required"` tags, attackers could submit incomplete or malformed payloads without being rejected by validation rules, potentially leading to logic errors or unauthorized behaviors.
**Learning:** In the Echo framework, binding and validation are separate steps. `c.Bind()` does not automatically invoke validation logic based on struct tags.
**Prevention:** Always pair `c.Bind(&req)` with an explicit `c.Validate(&req)` (or `c.Validate(req)`) call to ensure that payload constraints and required fields are correctly enforced.
## 2025-05-18 - [Fix CRITICAL Missing RBAC on Settings API allowing Privilege Escalation]
**Vulnerability:** The `/api/settings` API endpoints (such as `PUT /api/settings/env/batch`) were missing Role-Based Access Control (`mw.RequireRole("admin")`), allowing any authenticated user to update critical `.env` configurations including the `OT_ADMIN_PASSWORD`, `OT_JWT_SECRET`, and `OT_LDAP_BIND_PASSWORD`. This creates an easy path to full system compromise through privilege escalation.
**Learning:** Even when top-level route groups (e.g. `p := e.Group("/api", mw.JWTAuth(cfg.JWT.Secret))`) are protected with authentication, individual router groupings (`g.Group("/settings")`) may fail to enforce necessary authorization / role checks for sensitive operations. Always verify that sub-routers explicitly apply RBAC when they manage global application state or secrets.
**Prevention:** Within Echo handler registration routines (e.g. `RegisterRoutes`), explicitly group and protect administrative endpoints using `g.Group("", middleware.RequireRole("admin"))` to assure strict segregation of duties from read-only or low-privileged endpoints.
