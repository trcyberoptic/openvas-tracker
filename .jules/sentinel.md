## 2024-05-01 - Constant Time Compare Timing Leak
**Vulnerability:** Length-based timing leak in `subtle.ConstantTimeCompare` when comparing direct string inputs.
**Learning:** `ConstantTimeCompare` leaks the length of the inputs if they are not the same length, since it returns 0 early if lengths differ. When comparing user-provided data directly to a secret (like an API key or password), an attacker could deduce the length of the secret.
**Prevention:** Always hash both inputs (e.g., with `crypto/sha256.Sum256`) before passing them to `subtle.ConstantTimeCompare` to guarantee they are of equal length.
