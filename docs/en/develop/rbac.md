# RBAC (Role-Based Access Control) Design

## Overview

user-service uses a lightweight RBAC: three system roles plus permission codes, along two lines:

- **Data**: roles, permission definitions and role-permission mappings all live in the database.
  On startup `internal/bootstrap` seeds whatever is missing, and admins can adjust the mappings later.
- **Enforcement**: the gateway declares the required permission code per route and calls
  `CheckPermission` in user-service; `super_admin` short-circuits via a wildcard.

Permission data is no longer hard-coded, so **an empty database boots straight away** and heals
its own RBAC data.

## Role Model

| Role code | Name | Default permissions | Adjustable |
|---|---|---|---|
| `super_admin` | Super admin | Everything (wildcard, mapping not consulted) | Mapping editable, but not used for checks |
| `admin` | Admin | None by default (matches the SQL seed) | Yes, via the admin UI |
| `user` | Default user (granted on sign-up and social login) | 8 authoring permissions | Yes |

`admin` having nothing by default mirrors the seed data: which operational permissions it needs is
a deployment decision, so the code does not widen it on its own.

## Permission Codes

`user_svc.permission` holds 17 definitions (startup seeding fills in any missing code):

| Resource | Permission codes |
|---|---|
| `user` | `user:read` / `user:write` / `user:delete` / `user:admin` |
| `novel` | `novel:read` / `novel:write` / `novel:delete` / `novel:publish` / `novel:admin` |
| `material` | `material:read` / `material:write` |
| `publish` | `publish:read` / `publish:write` / `publish:admin` |
| `system` | `system:admin` |
| `hotspot` | `hotspot:read` / `hotspot:write` |

The `user` role gets 8 of them: `novel:read`, `novel:write`, `novel:publish`, `material:read`,
`material:write`, `publish:read`, `publish:write`, `hotspot:read`.

## Startup Seeding (internal/bootstrap)

Runs once in `cmd/server` and is fully idempotent:

| Seeded | Rule |
|---|---|
| System roles | Created when missing (`role.id` has no sequence, so `max(id)+1` is assigned) |
| Permission definitions | Missing codes are inserted, existing ones skipped |
| Default role permissions | Written **only when the role currently has none** |
| Admin account | Looked up by `USER_ADMIN_EMAIL`, created and granted `super_admin` when absent |

Never-overwrite rules (so a restart cannot break a live system):

- an existing account's password stays untouched; use `USER_ADMIN_RESET_PASSWORD=true` for one
  restart to reset it;
- role permissions edited in the admin UI stay untouched (defaults are only written when a role has none);
- an account that already holds `super_admin` is not granted it again.

Configuration:

| Variable | Default | Purpose |
|---|---|---|
| `USER_ADMIN_EMAIL` | empty | Admin email; leaving it empty skips admin seeding |
| `USER_ADMIN_PASSWORD` | empty | Initial password, used only when the account is absent (or a reset is requested) |
| `USER_ADMIN_NICKNAME` | `系统管理员` | Nickname |
| `USER_ADMIN_RESET_PASSWORD` | `false` | Escape hatch: one restart resets the password to the value above |

Failure handling: a seeding failure only logs an ERROR and never blocks startup (existing accounts
can still sign in), so a configuration gap cannot cause a crash loop.

Relationship with `database/user_svc.sql`:

- seeding mirrors the SQL export one-to-one (including permission order, so a fresh database gets
  the same ids);
- the code is what lets the service boot in any state, including a wiped database;
- the SQL file is a full export for reference and manual rebuilds — it does **not** need to be run
  on deploy, and re-running it wipes data (`DROP` + `CREATE`).

## super_admin Wildcard

`super_admin` is not granted permission-by-permission:

- when permissions are resolved (`resolvePermissions`), the role makes the cache hold a single
  wildcard marker `*`;
- `CheckPermission` short-circuits on that marker — even for a permission code that does not exist yet;
- the API (`/user/permissions`) expands the wildcard into every code in the permission table, so
  frontend menus and buttons need no special handling.

Reasons:

1. new permission codes never require re-granting anything to the super admin;
2. expanding eager lists would open a window where the cache lacks a newly added permission (the
   cache TTL equals the refresh-token lifetime, 30 days by default); the wildcard has no such window.

The `super_admin` → permission mapping is still written to the database for UI display and SQL-export
parity, but **checks do not depend on it**.

## Permission Check Path

1. The gateway declares the required code per route: everything under `/api/v1/admin/**` requires
   `user:admin` (see `services/api-gateway/internal/router/v1/admin_router.go`).
2. `middleware.RequirePermission` calls `CheckPermission` in user-service with the `user_uuid` from the access token.
3. `rbac.Service.CheckPermission` reads the `perm:user:{uuid}` cache, resolves on a miss (wildcard or
   `role_permission` aggregation) and writes it back, then compares the code.
4. Not held → 403.

Business code additionally compares the caller's `user_id` with the target resource owner: when they
differ, the management permission code is required. The two stages are complementary — one governs
resource ownership, the other capability boundaries.

## Permission Cache

- key: `perm:user:{uuid}`, value is a comma-separated list of codes (`*` for `super_admin`)
- TTL: equal to the refresh-token lifetime (`USER_REFRESH_TTL_SECONDS`, 30 days by default)
- Invalidation: assigning/removing a role, overwriting role permissions or deleting a role clears
  the affected users' caches
- Cache unavailable: falls back to the database and logs a WARN; requests are never blocked

## What Admins Can Adjust

- Role permissions: `PUT /api/v1/admin/roles/{id}/permissions` (overwrite), clearing the caches of
  every user holding that role
- Role name and description: `PUT /api/v1/admin/roles/{id}`; system roles cannot change `code` or be deleted
- Custom roles: roles beyond the three system ones can be created and given permissions

## Troubleshooting

| Symptom | What to do |
|---|---|
| Everything under `/admin` returns 403 | Does the account hold `super_admin` (wildcard) or `user:admin`? Inspect the permission cache and `CheckPermission` result |
| Some permission codes are missing | Check the startup log `已补齐系统权限 count=N`; restarting heals an empty permission table |
| Cleared role permissions come back | Seeding only writes defaults when a role has none — expected (see "Startup Seeding") |
| Sign-up fails with `duplicate key ... "user_pkey"` | Sequence out of sync with `max(id)`; the realign statement is in the "序列对齐" section of `database/user_svc.sql` |
| Admin password forgotten | `USER_ADMIN_RESET_PASSWORD=true` for one restart, then flip it back |

## Audit

All sensitive operations (role changes, failed permission checks, 2FA events, etc.) are written to
`user_svc.audit_log`, recording operator, action, target and IP for security traceability.
