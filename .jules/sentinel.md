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
## 2025-02-28 - [Echo Group Sub-Routing Authorization Bypass]
**Vulnerability:** Authorization bypass found in the Tickets API. Methods to update ticket status (`PATCH /:id/status`), assign tickets (`PATCH /:id/assign`), and perform bulk actions (`POST /bulk`) were mistakenly assigned to the general `/api` group, which required JWT authentication but didn't restrict to specific roles (like "admin" or "analyst"), meaning standard users/viewers could modify or assign tickets.
**Learning:** In the Echo framework, public or widely-accessible protected routes (`/api`) and specifically restricted routes (role-specific operations) within the same generic handler group can mistakenly expose administrative functions. The `middleware.RequireRole()` wrapper needed to be applied to a sub-group rather than omitted or applied at a global level for these specific operations.
**Prevention:** Always create a targeted sub-group `g.Group("", middleware.RequireRole(...))` inside the `RegisterRoutes` function to isolate actions requiring specific roles, such as writing/updating records versus standard reading.
