## 2025-05-06 - Prevent Timing Attacks in Password Comparison
**Vulnerability:** The application used a simple string comparison (`==`) for checking the admin password. This could allow an attacker to perform a timing attack to guess the password byte-by-byte.
**Learning:** Hardcoded direct string comparisons for sensitive data like passwords or tokens create timing side-channels.
**Prevention:** Always use constant-time comparison functions like `crypto/subtle.ConstantTimeCompare` when comparing sensitive strings or byte slices to prevent timing attacks.

## 2025-05-15 - Prevent Length-Based Timing Leaks in ConstantTimeCompare
**Vulnerability:** The application passed variable-length inputs (e.g. from user input) directly to `crypto/subtle.ConstantTimeCompare`. This leaks the length of the secret because `ConstantTimeCompare` compares lengths first and returns early if they differ, creating a length-based timing leak.
**Learning:** Passing variable length inputs directly to `ConstantTimeCompare` allows attackers to infer the length of a secret via timing, reducing the search space for brute force attacks.
**Prevention:** Always hash both inputs (e.g., using `crypto/sha256.Sum256`) before passing them to `ConstantTimeCompare` to guarantee they are of equal length (e.g., 32 bytes) regardless of the original inputs.
