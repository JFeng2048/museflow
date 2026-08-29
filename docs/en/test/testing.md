# Testing Guide

## Test Strategy

- Backend uses the standard `testing` package; test files sit next to sources (`*_test.go`).
- External dependencies are isolated via in-memory fakes (`repository.UserRepository` / `TokenStore`, etc.);
  no real PostgreSQL / Redis required.
- Frontend uses Vitest, with `*.spec.ts` next to sources.

## user-service Coverage

`internal/service/auth` is the primary coverage package, including:

| Test file | Coverage |
|----------|--------|
| `auth_service_test.go` | register, login (incl. 2FA two-step), logout, change password, lockout, session list |
| `reset_password_test.go` | email code send/verify/reset, resend cooldown |
| `oauth_test.go` | OAuth bind/login/unbind |
| `mfa.go` cases | Setup/Verify/Disable/Regenerate/GetStatus/VerifyMFALogin |

`internal/service/mfa` covers TOTP key generation, code verification (incl. clock skew), recovery-code generation and single-use checks.

## Run Tests

```bash
# All
cd services/user-service && go test ./...

# Single package
go test ./internal/service/auth/ -v

# Single case
go test ./internal/service/auth/ -run TestLoginIssuesUsableTokenPair -v

# Coverage
go test ./... -cover
```

## Test Conventions

- Fake repositories must be extended alongside the real interface (e.g. when adding `SaveMFASecret` / `EnableMFA`, the test fake must implement them too, or it fails to compile).
- 2FA tests build `MFAConfig` via `testMFAConfig()` to avoid scattered magic values.
- `NewTokenManager` signature is `(secret, accessTTL, refreshTTL, mfaTicketTTL)` — tests pass 4 args.
- `AuthService.Login` returns `(*LoginResult, error)` (two values); destructure carefully in assertions.

## Pre-Commit Checks

```bash
go vet ./... && go test ./... && cd ../api-gateway && go vet ./... && go build ./...
```
