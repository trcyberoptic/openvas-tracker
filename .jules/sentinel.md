## 2025-05-06 - Prevent Timing Attacks in Password Comparison
**Vulnerability:** The application used a simple string comparison (`==`) for checking the admin password. This could allow an attacker to perform a timing attack to guess the password byte-by-byte.
**Learning:** Hardcoded direct string comparisons for sensitive data like passwords or tokens create timing side-channels.
**Prevention:** Always use constant-time comparison functions like `crypto/subtle.ConstantTimeCompare` when comparing sensitive strings or byte slices to prevent timing attacks.

## 2026-05-13 - Prevent Length Leak in Constant Time Compare
**Vulnerability:** The application used `crypto/subtle.ConstantTimeCompare` to compare API keys and passwords directly with user input. Since `ConstantTimeCompare` returns immediately if the lengths differ, it leaks the length of the server's API key and admin password.
**Learning:** `ConstantTimeCompare` protects against byte-by-byte timing attacks but not length-based timing attacks if inputs can be of varying lengths.
**Prevention:** Always hash both the user input and the expected secret (e.g. using `sha256.Sum256`) before passing them to `ConstantTimeCompare` to ensure both inputs have an identical, fixed length.
