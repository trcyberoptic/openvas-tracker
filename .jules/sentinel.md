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
## 2024-05-31 - [LDAPS Hardcoded InsecureSkipVerify]
**Vulnerability:** The `internal/service/ldap.go` file hardcodes `InsecureSkipVerify: true` for LDAPS connections. This disables TLS certificate verification, leaving connections susceptible to Man-In-The-Middle (MITM) attacks.
**Learning:** This is a classic example of security debt where developer convenience or a lack of proper certificate authority setup in a testing environment gets carried over into production code. The lack of an option to disable this flag or provide a custom CA certificate is a critical security gap.
**Prevention:** Avoid hardcoding `InsecureSkipVerify: true` in production code. Provide an option via configuration to toggle this behavior or, ideally, support loading custom CA certificates for proper validation.
## 2025-06-25 - Prevent Authorization Bypass in Sub-Groups
**Vulnerability:** The application defined custom RBAC middleware (`middleware.RequireRole`) but failed to apply it to sensitive endpoints in `settings.go` and `tickets.go`. This allowed users with lower privileges (like 'viewer') to access admin-only functionality such as modifying `.env` configuration or executing bulk updates.
**Learning:** Simply creating an RBAC middleware is not enough; it must be explicitly applied. In the Echo framework, if a handler serves a mix of public and protected routes, the protected routes must be separated into a subgroup with the required middleware explicitly attached.
**Prevention:** Group sensitive endpoints inside `g.Group("", middleware.RequireRole(...))` and register them on the protected group instead of the top-level group.
