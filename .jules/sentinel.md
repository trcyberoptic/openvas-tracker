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

## 2025-05-18 - Fix hardcoded InsecureSkipVerify for LDAPS connections
**Vulnerability:** The application hardcoded `InsecureSkipVerify: true` when establishing LDAPS connections in `internal/service/ldap.go`. This completely disabled TLS certificate verification, making the LDAP authentication process vulnerable to Man-in-the-Middle (MITM) attacks where an attacker could intercept sensitive credentials.
**Learning:** Forcing insecure TLS configurations by default for integrations (like LDAP) removes the ability for secure deployments to enforce strict certificate verification, introducing unnecessary baseline risk.
**Prevention:** Make TLS verification settings configurable (e.g., via `OT_LDAP_SKIP_VERIFY`) so that administrators can explicitly opt-in to insecure behaviors only when absolutely necessary (like using self-signed certs in development), while defaulting to secure verification (`false`).
