## 2025-05-06 - Prevent Timing Attacks in Password Comparison
**Vulnerability:** The application used a simple string comparison (`==`) for checking the admin password. This could allow an attacker to perform a timing attack to guess the password byte-by-byte.
**Learning:** Hardcoded direct string comparisons for sensitive data like passwords or tokens create timing side-channels.
**Prevention:** Always use constant-time comparison functions like `crypto/subtle.ConstantTimeCompare` when comparing sensitive strings or byte slices to prevent timing attacks.

## 2025-05-17 - Prevent Length-Based Timing Attacks in ConstantTimeCompare
**Vulnerability:** Even when using `crypto/subtle.ConstantTimeCompare`, an early return based on unequal slice lengths can leak information about the expected length of a sensitive string (like a password or API key).
**Learning:** `ConstantTimeCompare` performs a constant-time comparison *only* if the lengths match. If lengths differ, it immediately returns 0. This creates a timing leak that reveals the target string length.
**Prevention:** Hash both strings (e.g. using `sha256.Sum256`) before passing them to `ConstantTimeCompare`. This ensures both inputs always have the same length (e.g. 32 bytes) regardless of the original string lengths.
