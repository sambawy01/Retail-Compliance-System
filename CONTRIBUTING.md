# Contributing to Watch Dog

## Code Review Checklist

Before submitting a PR, ensure:
- [ ] All tests pass (`go test ./...` and `cd web && npm run test`)
- [ ] No secrets/credentials committed
- [ ] Error messages don't leak internal details to clients
- [ ] New endpoints have rate limiting and auth guard
- [ ] New DB tables have RLS policies
- [ ] New DB columns are documented in the canonical migration
- [ ] Frontend changes include EN + AR i18n strings
- [ ] No silent error swallowing (catch blocks must log or report)

## Branch Naming
- `feat/` — new features
- `fix/` — bug fixes
- `chore/` — maintenance, deps, config
- `refactor/` — code restructuring

## Commit Messages
Use conventional commits: `type(scope): description`

Examples:
- `feat(staff): add performance scoring algorithm`
- `fix(auth): sanitize login error messages`
- `chore(ci): add golangci-lint to pipeline`

## Architecture Rules
- Handlers in `internal/api/` — HTTP parsing, validation, response formatting only
- Business logic in service packages (`internal/vision/`, `internal/identity/`, etc.)
- DB queries in service methods, not in handlers
- All DB access through `database.TenantTx` for RLS
- Every table needs `org_id` column + RLS policy + grant to `watchdog_app`

## Security Rules
- Never log passwords, tokens, or biometric data
- Never return `password_hash` in API responses
- Always validate and sanitize user input
- Use parameterized queries (never string concatenation for SQL)
- Rate-limit sensitive endpoints (login, refresh, match)