# 2FA (Two-Factor Authentication) Design

## Overview

On top of dual-token auth, add TOTP-based (RFC 6238) two-factor authentication to improve account security.
Once 2FA is enabled, login splits into two steps: step 1 verifies the password, step 2 verifies the TOTP code.

Implementation carriers:

| Component | Responsibility |
|------|------|
| `services/api-gateway` | Exposes `/auth/mfa/*` and `/auth/mfa/verify-login` HTTP endpoints, proxies gRPC |
| `services/user-service` | `mfa` subpackage generates keys / verifies codes; `auth` subpackage orchestrates login 2nd-factor; repository persists keys |
| `proto/user/user.proto` | Six RPCs: `SetupMFA` / `VerifyMFA` / `DisableMFA` / `RegenerateRecoveryCodes` / `GetMFAStatus` / `VerifyMFALogin` |
| PostgreSQL `user_svc.users` | Stores `mfa_secret`, `mfa_enabled`, `mfa_recovery_codes` |

## Core Flow

### Login two-step verification

```
1. POST /auth/login {email, password}
   ├─ 2FA disabled → return access/refresh dual tokens directly
   └─ 2FA enabled  → return mfa_ticket (no tokens), requires_mfa=true

2. POST /auth/mfa/verify-login {mfa_ticket, code}
   ├─ code verify fail → 401, and audit logged
   └─ code verify ok   → issue access/refresh dual tokens, write device session
```

`mfa_ticket` is a short-lived JWT (type `mfa_ticket`, default 5-minute expiry), used only to chain the two login steps;
it cannot exchange for any business resource. The ticket carries `user_uuid`; the server issues the real tokens after verifying it.

### Enable 2FA

```
1. POST /auth/mfa/setup      → return secret + otpauth_url (not yet enabled)
2. User scans the QR code in an authenticator app to bind
3. POST /auth/mfa/verify {code} → after verification, enable and return 8 recovery codes
```

A TOTP verification must succeed before enabling, to prevent permanent lockout from a mistyped key.

### Recovery codes

On enabling 2FA, 8 single-use recovery codes (default length 10) are generated. Any recovery code can substitute a TOTP code in
`/auth/mfa/verify-login`; once used it is invalidated and a new one is issued, always keeping 8 available.

## Configuration

| Key | Default | Description |
|--------|--------|------|
| `MFA_TICKET_TTL_SECONDS` | 300 | Login intermediate ticket lifetime (seconds) |
| `MFA_ISSUER` | MuseFlow | Issuer shown in the authenticator app |
| `MFA_SKEW` | 1 | Allowed clock-skew steps (each step 30 seconds) |
| `MFA_RECOVERY_CODE_COUNT` | 8 | Number of recovery codes |
| `MFA_RECOVERY_CODE_LENGTH` | 10 | Length of a single recovery code |

## Security Notes

- `mfa_secret` is stored in plaintext in the database (encryption can be added later); server-side only, never returned to the frontend.
- `otpauth_url` is returned only once during `setup` for QR scanning; not persisted.
- Disabling 2FA also requires the current TOTP code, to prevent an attacker from turning off protection while the session is still valid.
- Verification failures are audited (`mfa_verify_fail`) for security traceability.
