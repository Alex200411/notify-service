# CLAUDE.md

## Project Overview

Notification Delivery Service — a Go middleware that reliably delivers HTTP notifications to external vendor APIs. Uses PostgreSQL for persistence and Redis for async queuing.

## Tech Stack

- Go 1.22+
- PostgreSQL 16 (pgx/v5 driver)
- Redis 7+ (go-redis/v9, LPUSH/BRPOP list queue)
- No ORM, no HTTP framework — stdlib `net/http` + `database/sql` style with pgx

## Project Structure

- `cmd/server/` — API server binary
- `cmd/worker/` — queue consumer binary
- `internal/` — all shared code, layered: api → service → repository → db/queue

## Development

```bash
# Start infra
make docker-up          # or run postgres + redis locally

# Run migrations
make migrate

# Run server and worker
make run-server         # terminal 1
make run-worker         # terminal 2

# Run tests
make test
```

## Code Conventions

- No ORM. Write raw SQL with pgx. Keep queries in `internal/repository/`.
- Interfaces live in the same file as their primary implementation.
- Use `log/slog` for structured logging. No `fmt.Println` or `log.Println`.
- Error wrapping: always use `fmt.Errorf("context: %w", err)`.
- No comments unless explaining a non-obvious WHY.
- Keep handlers thin — parse request, call service, write response. Business logic lives in `internal/service/`.

## Git & Merge Rules

- All work on feature branches, never commit directly to `main`.
- Branch naming: `feat/xxx`, `fix/xxx`, `refactor/xxx`.
- Every PR must pass CI (lint + test + build) before merge.
- Squash merge to main — keep history clean.
- Commit messages: `feat:`, `fix:`, `refactor:`, `test:`, `docs:` prefixes.

## CI

GitHub Actions pipeline runs on every push and PR:
1. `golangci-lint` — static analysis
2. `go test ./...` — all tests
3. `go build ./...` — compile check

CI must pass before merge. Do not skip or force-merge failing CI.

## Testing

- Unit tests use mocks (mock repo, mock queue, httptest servers).
- No external dependencies needed to run tests.
- Test file lives next to the code it tests: `delivery.go` → `delivery_test.go`.
- Test names: `TestFunctionName_Scenario` (e.g., `TestDeliverOne_VendorFailureThenRetry`).

## What Not To Do

- Don't add an ORM.
- Don't use Redis Streams — we use simple list queue by design (see design.md for rationale).
- Don't combine server and worker into one binary.
- Don't add vendor-specific logic in code — use vendor_configs table and body_template.
