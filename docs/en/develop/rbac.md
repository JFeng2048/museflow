# RBAC (Role-Based Access Control) Design

## Overview

user-service uses a lightweight RBAC: three built-in roles, with permissions centrally checked in business code; no standalone permission management UI.

## Role Model

Only three roles are built in; they cannot be deleted or renamed by business code:

| Role key | Description |
|----------|------|
| `super_admin` | Super admin, all permissions |
| `admin` | Admin, operational permissions except role management |
| `user` | Default user, only permissions on own resources |

The role↔permission mapping is defined as code constants in `internal/service/rbac`,
not seeded into the database (to avoid coupling with runtime data).

## Permission Check

Permission checking does not rely on runtime "role → permission" lookups, but is done directly by business code:

1. The interface layer declares the required permission code (e.g. `user:read`, `role:manage`).
2. The gateway parses `user_id` (i.e. `user_uuid`) from the access token.
3. The caller passes the `user_id` of the target resource; the server compares it with the token's `user_id`:
   - match → self operation, allow;
   - mismatch → check whether the caller holds the corresponding management permission code (e.g. `user:manage`);
     allow if held, otherwise return 403.

This two-stage "self compare + permission code" check covers both the ordinary user self-service scenario
and the admin delegated-operation scenario, without querying the role tree every time.

## Permission Cache

Permission check results (whether a user holds a permission code) are cached in Redis:

- key: `perm:user:{uuid}`
- TTL: 7 days
- The key is actively cleared on role/permission change, guaranteeing eventual consistency.

## Audit

All sensitive operations (role changes, failed permission checks, 2FA events, etc.) are written to `user_svc.audit_log`,
recording operator, action, target, and IP for security traceability.
