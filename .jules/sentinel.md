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

## 2025-06-09 - Missing RBAC in Echo Handler Subgroups
**Vulnerability:** Handlers mapping multiple routes to a single `echo.Group` failed to implement Role-Based Access Control (RBAC) on sensitive endpoints. While the top-level route had authentication applied via JWT middleware, individual routes within the group (such as status updates or setting modifications) did not enforce specific role checks (like "admin" or "analyst"). This allowed users with lower privileges (e.g. "viewer") to perform unauthorized actions.
**Learning:** Registering routes directly on an `echo.Group` applies the same middleware configuration to all routes. If sensitive endpoints require stricter permissions than others in the same group, they need dedicated route definitions or sub-groups.
**Prevention:** Always use sub-groups `g.Group("", middleware.RequireRole(...))` when mixing public or lower-privileged routes with protected endpoints within the same handler group. This allows targeted application of RBAC to specific routes without over-restricting the entire group.
