# Dual-Token Authentication Design

## Overview

The dual-token mechanism implements a complete user authentication system.

The current implementation runs on Go microservices:

| Component | Responsibility |
|------|------|
| `services/api-gateway` | Exposes HTTP endpoints (Gin), manages Cookies, proxies gRPC |
| `services/user-service` | Auth business logic, token issuance/revocation, user data access |
| `proto/user/user.proto` | gRPC contract between the two |
| PostgreSQL `user_svc` schema | User master table and other persisted data (see `services/user-service/database/user_svc.sql`) |
| Redis | refresh-token allow-list, access-token deny-list, device session list |

### Core Design

- **Dual tokens**: access token (short-lived 1h, stateless JWT, in response body) + refresh token (long-lived 30d, JWT + Cache allow-list, in HttpOnly Cookie)
- **No refresh rotation**: the same refresh token can repeatedly mint new access tokens within its validity
- **refresh allow-list eviction**: revocation deletes the `valid` marker in Cache, saving memory
- **access deny-list**: on logout the access token's `jti` is written to the deny-list with a TTL equal to its **remaining lifetime**, auto-cleaned by Redis. Thus an unexpired access token becomes invalid immediately after logout
- **Device fingerprint**: `sha256(deviceId + User-Agent + IP)` prevents cross-device use of a stolen refresh token
- **Password hashing**: bcrypt (cost configurable, default 10; password length capped at 72 bytes)
- **Login account**: email (`user_svc.users.email` unique and non-null)

> Why refresh uses an allow-list while access uses a deny-list:
> refresh tokens are few and long-lived, so an allow-list eases device management and active revocation;
> access tokens are many and short-lived, and only need to be blocked during the "remaining lifetime after logout" window,
> so deny-list entries vanish with TTL and memory cost stays bounded.

## Token Mechanism

### Access Token

| Attribute | Value |
|------|---|
| Format | JWT (HS256 signature) |
| Lifetime | 1 hour (`ACCESS_TTL_SECONDS=3600`) |
| Storage | response body → frontend memory (XSS-safe) |
| Transport | `Authorization: Bearer <token>` |
| Verification | JWT signature + type check + expiry check + **Redis deny-list check** |
| payload | `{sub, jti, type:"access", iat, exp}` |

`sub` is the user `uuid` (external id, not an auto-increment id); `jti` is the token's unique id, used as the deny-list key on logout.

### Refresh Token

| Attribute | Value |
|------|---|
| Format | JWT (HS256 signature) |
| Lifetime | 30 days (`REFRESH_TTL_SECONDS=2592000`) |
| Storage | HttpOnly Cookie (unreadable by JS, XSS-safe) |
| Transport | Cookie auto-sent |
| Verification | JWT signature + device check + Cache allow-list check |
| payload | `{sub, type:"refresh", tokenId, deviceId, deviceFingerprint, iat, exp}` |

### Cookie Attributes

| Attribute | Value |
|------|---|
| `HttpOnly` | True (refresh token), False (device id) |
| `SameSite` | lax (configurable: lax / strict / none) |
| `Secure` | enabled in prod (`COOKIE_SECURE=true`, HTTPS only) |
| `Path` | / |
| `Max-Age` | 30 days |

> Because Cookies are sent, CORS `Access-Control-Allow-Origin` cannot be `*`;
> it must echo specific origins from the `ALLOW_ORIGINS` allow-list and set `Allow-Credentials: true`.

## Cache Data Structures

### 1. refresh token validity allow-list

```
key: auth:refresh:valid:{tokenId}
value: "active"
TTL: 30 days
```

Present = valid; deleted = invalid (allow-list eviction strategy).

### 2. access token deny-list

```
key: auth:access:blacklist:{jti}
value: "revoked"
TTL: remaining lifetime of that access token (<=1h)
```

Present = revoked. Auto-cleaned on TTL expiry, no extra cleanup job needed.
If the access token is already expired at logout, the write is skipped.

### 3. user token metadata list

```
key: auth:user:tokens:{userId}
value: JSON array
TTL: 30 days
```

```json
[
  {
    "tokenId": "uuid",
    "deviceId": "device-uuid",
    "deviceName": "Chrome on Windows",
    "loginTime": "2026-08-27T12:00:00+08:00",
    "lastRefreshTime": "2026-08-27T12:30:00+08:00"
  }
]
```

Used for device list display and bulk revocation.

## Full Flows

### 1. Register (`POST /api/v1/auth/register`)

```
request: email + password + nickname(optional)
  │
  ├─ param validation (email format, password 8-72 bytes)
  ├─ email normalization (trim + lowercase)
  ├─ uniqueness check (email)
  ├─ nickname fallback (email prefix when empty, since users.nickname is NOT NULL)
  ├─ bcrypt hash password (cost=10)
  └─ insert user_svc.users (uuid generated server-side)

response: {code, message, data: UserInfo}
```

### 2. Login (`POST /api/v1/auth/login`)

```
request: email + password + device_name(optional); Cookie may carry existing device_id
  │
  ├─ query user by email
  ├─ bcrypt password check
  │   same error for unknown user and wrong password, to avoid email enumeration
  ├─ generate tokenId = uuid4(); deviceId generated server-side if missing
  ├─ compute deviceFingerprint = sha256(deviceId + UA + IP)
  ├─ issue access token (JWT, 1h, with jti)
  ├─ issue refresh token (JWT, 30d, with deviceId + deviceFingerprint)
  ├─ Redis write allow-list: auth:refresh:valid:{tokenId} = "active" (TTL=30d)
  ├─ Redis write user token list
  ├─ update users.last_login_at / last_login_ip (failure only warns, does not block login)
  ├─ access token → response body
  ├─ refresh token → HttpOnly Cookie
  └─ device_id → plain Cookie

response: {access_token, token_type, expires_in, user} + Set-Cookie: refresh_token, device_id
```

> Note: the `login_fail_count` / `locked_until` / `status` fields are kept in the data model,
> but the current login flow does not yet do failure counting, locking, or status blocking; reserved for later iterations.

### 3. Refresh (`POST /api/v1/auth/refresh`)

```
request: no body (read from Cookie)
  │
  ├─ read refresh_token + device_id from Cookie
  ├─ missing either → 401
  │
  ├─ step 1: JWT decode + signature verify (HS256 only)
  │   fail → 401 "refresh token invalid"
  │
  ├─ step 2: token type check
  │   payload.type == "refresh" ?
  │   no → 401 (prevent access token being used as refresh)
  │
  ├─ step 3: extract tokenId + sub
  │   missing → 401 "missing tokenId or sub claim"
  │
  ├─ step 4: deviceId compare
  │   JWT deviceId == Cookie device_id ?
  │   no → 403 "device verification failed"
  │
  ├─ step 5: device fingerprint compare
  │   recompute sha256(device_id + User-Agent + IP)
  │   compare with JWT deviceFingerprint
  │   mismatch → 403 "device verification failed"
  │
  ├─ step 6: Redis allow-list check
  │   EXISTS auth:refresh:valid:{tokenId} ?
  │   absent → 401 "token invalid or expired"
  │
  ├─ issue new access token (refresh unchanged, no rotation)
  └─ update lastRefreshTime in Redis token

response: {access_token, token_type, expires_in}
On 401 failure the gateway clears the Cookie to avoid client retries
```

### 4. Logout (`POST /api/v1/auth/logout`)

```
request: requires access token (Authorization header), Cookie carries refresh_token
  │
  ├─ handle refresh token (if present)
  │   ├─ decode to get tokenId
  │   ├─ delete allow-list: DEL auth:refresh:valid:{tokenId}
  │   └─ remove tokenId from user token list
  │
  ├─ handle access token (if present and unexpired)
  │   ├─ decode to get jti and exp
  │   ├─ ttl = exp - now
  │   └─ write deny-list: SET auth:access:blacklist:{jti} "revoked" EX ttl
  │
  └─ clear both Cookies (refresh_token + device_id)

response: {code, message}
```

Each token is processed best-effort: a failure parsing one does not block revoking the other,
so logout works when the client carries only one of them.

## Auth Middleware

`Auth` in `api-gateway/internal/middleware/auth.go`:

1. Extract access token from `Authorization: Bearer <token>`; missing → 401
2. Call user-service's `ValidateToken` (signature, type, expiry, **deny-list** check)
3. On success, write user `uuid` into `gin.Context`; downstream uses `middleware.CurrentUserUUID(c)`

> The gateway does not verify signatures locally but calls back to user-service:
> because the deny-list lives in user-service's Redis,
> local-only verification cannot sense that a token was revoked by logout.

## Configuration

### api-gateway

| Env var | Default | Description |
|------|--------|------|
| `PORT` | 5001 | HTTP listen port |
| `USER_SERVICE_URL` | localhost:5002 | user-service gRPC address |
| `JWT_SECRET` | (required) | JWT signing key, must match user-service |
| `ALLOW_ORIGINS` | http://localhost:5173 | CORS allow-list, comma-separated |
| `COOKIE_SECURE` | false | set true in prod (HTTPS) |
| `COOKIE_SAMESITE` | lax | Cookie SameSite policy |
| `COOKIE_DOMAIN` | (empty) | Cookie scope |

### user-service

| Env var | Default | Description |
|------|--------|------|
| `PORT` | 5002 | gRPC listen port |
| `DB_HOST` / `DB_PORT` / `DB_USER` / `DB_PASSWORD` / `DB_NAME` | see `.env.example` | DB connection (shared config, assembled into DSN) |
| `JWT_SECRET` | (required) | JWT signing key (shared gateway+user-service for verification) |
| `REDIS_ADDR` | localhost:6379 | Redis address |
| `REDIS_PASSWORD` | (empty) | Redis password |
| `REDIS_DB` | 0 | Redis DB number |
| `ACCESS_TTL_SECONDS` | 3600 | access token lifetime |
| `REFRESH_TTL_SECONDS` | 2592000 | refresh token lifetime |
| `BCRYPT_COST` | 10 | bcrypt cost |

## API Endpoint Summary

| Method | Path | Auth | Function |
|------|------|------|------|
| POST | `/api/v1/auth/register` | None | User registration (email) |
| POST | `/api/v1/auth/login` | None | Login (email + password) |
| POST | `/api/v1/auth/refresh` | None (Cookie) | Refresh access token |
| POST | `/api/v1/auth/logout` | access required | Logout (revoke current device) |
| GET | `/api/v1/user/profile` | access required | Get current user info |
| GET | `/health` | None | Health check |
| GET | `/swagger/index.html` | None | Swagger docs |

## Future Iterations

- Device management endpoints (`GET /auth/devices`, `DELETE /auth/devices/{token_id}`): Redis structures and metadata are ready
- Registration/login CAPTCHA
- Login failure counting and account lockout (`login_fail_count` / `locked_until`)
- Account status blocking (reject login when `status != 1`)
- Third-party login (`user_oauths` table) and MFA (`mfa_enabled` / `mfa_secret`)
- Operation audit (`audit_logs` table) and login session persistence (`user_sessions` table)
