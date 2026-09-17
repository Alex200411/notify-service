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

### Branch Model

- `main` — production-ready code, only accepts merges from `dev`
- `dev` — integration branch, all feature/bug branches merge here first
- Feature branches: `feat/xxx` — branch off `dev`, merge back to `dev`
- Bug fix branches: `bug/xxx` — named after the bug, branch off `dev`, merge back to `dev`
- Refactor branches: `refactor/xxx`

### Workflow

1. 日常开发在 `dev` 分支上切子分支
2. Bug 修复必须切一个以 bug 命名的分支（如 `bug/redis-reconnect`、`bug/template-nil-payload`）
3. PR 合入 `dev` 时必须通过 CI
4. `dev` 合入 `main` 时必须通过 CI
5. 不允许直接 push 到 `main` 或 `dev`
6. Squash merge — keep history clean
7. Commit messages: `feat:`, `fix:`, `refactor:`, `test:`, `docs:` prefixes

## CI

GitHub Actions pipeline runs on every push and PR to `main` and `dev`:
1. `go vet ./...` — static analysis
2. `go test ./... -race` — all tests with race detector
3. `go build` — compile server + worker

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
