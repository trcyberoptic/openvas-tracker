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
## 2024-05-24 - [Critical] Fix hardcoded Admin role assignment
**Vulnerability:** A helper function (`ensureUser`) hardcoded the `queries.UserRoleAdmin` role for all created users, including LDAP users which resulted in unintended privilege escalation for non-admin accounts.
**Learning:** Default parameter values (or lack of parameters for role assignments) can easily slip by code reviews but lead to critical authorization bypass. Also, hardcoded `"admin"` or `"user"` strings were overriding the database role during token generation.
**Prevention:** Helper functions for user creation (e.g., `ensureUser`) must explicitly accept roles as parameters instead of hardcoding default privileges to prevent accidental privilege escalation. Also, when generating JWT tokens and authentication responses, ensure the assigned role dynamically matches the user's actual database role (e.g., `string(user.Role)`) rather than using hardcoded default strings.
