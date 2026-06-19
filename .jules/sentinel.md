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

## 2025-06-05 - Privilege Escalation in LDAP Auto-Creation
**Vulnerability:** The application was hardcoding `queries.UserRoleAdmin` inside the `ensureUser` helper function. Because `ensureUser` was used to auto-create LDAP users upon their first login, any successfully authenticated LDAP user was automatically granted full administrator privileges in the application.
**Learning:** Hardcoding privileged roles in shared user-creation utility functions creates a high risk of unintended privilege escalation when those utilities are repurposed or called from new code paths (like SSO/LDAP integration).
**Prevention:** Helper functions for user creation must accept the intended role as an explicit parameter, forcing the caller to consciously decide the privilege level. Default fallback roles should always follow the principle of least privilege (e.g., `viewer` instead of `admin`).
