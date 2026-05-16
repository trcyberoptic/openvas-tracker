## 2025-05-06 - Prevent Timing Attacks in Password Comparison
**Vulnerability:** The application used a simple string comparison (`==`) for checking the admin password. This could allow an attacker to perform a timing attack to guess the password byte-by-byte.
**Learning:** Hardcoded direct string comparisons for sensitive data like passwords or tokens create timing side-channels.
**Prevention:** Always use constant-time comparison functions like `crypto/subtle.ConstantTimeCompare` when comparing sensitive strings or byte slices to prevent timing attacks.

## 2025-05-16 - Prevent Length-Based Timing Leaks in ConstantTimeCompare
**Vulnerability:** The application used `subtle.ConstantTimeCompare` directly on byte slices of potentially differing lengths (e.g., provided API key/password vs. expected one). Because `ConstantTimeCompare` returns immediately if the lengths differ, it could leak the exact length of the expected secret.
**Learning:** `ConstantTimeCompare` is only constant-time for inputs of the *same length*.
**Prevention:** Always hash both the user input and the expected secret (e.g., using `crypto/sha256.Sum256`) before passing them to `subtle.ConstantTimeCompare`. This guarantees both slices have the same length (e.g., 32 bytes for SHA-256) and entirely prevents length-based timing leaks.
