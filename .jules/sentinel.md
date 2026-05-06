## 2025-05-30 - [Hardcoded JWT logic in LDAP integration limits RBAC usefulness]
**Vulnerability:** The application lacks a robust way to implement RBAC for external users (LDAP). It hardcodes the 'user' role for all LDAP users and doesn't rely on it.
**Learning:** Adding the RequireRole("admin", "analyst") breaks core workflows for users authenticated via LDAP because their role is hardcoded to "user". The app actually relies on a "all users have equal access" model, and the `RequireRole` middleware shouldn't be blindly added.
**Prevention:** Thoroughly verify role issuance before adding `RequireRole` middleware, and ensure the business logic doesn't expect all authenticated users to have access to endpoints.
