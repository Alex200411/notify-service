# Notification Delivery Service

A reliable API notification delivery service that accepts internal notification requests and delivers them to external vendor HTTP(S) APIs with retry and dead-letter handling.

## Architecture

```
Internal Systems → API Server → PostgreSQL (persist) → Redis (queue) → Worker → External Vendor APIs
```

- **API Server**: receives notification requests, persists to DB, enqueues to Redis
- **Worker**: consumes from Redis, renders vendor-specific HTTP requests, delivers with retry

Key guarantee: **persist before acknowledge** — every notification is written to PostgreSQL before the caller gets a 201. If Redis is down, a recovery sweep picks up orphaned notifications.

## Prerequisites

- Go 1.22+
- Docker and Docker Compose

## Quick Start

```bash
# 1. Start PostgreSQL and Redis
make docker-up

# 2. Run database migrations
make migrate

# 3. Start the API server (terminal 1)
make run-server

# 4. Start the worker (terminal 2)
make run-worker
```

## Demo Walkthrough

### Create a vendor configuration

```bash
curl -s -X POST http://localhost:8080/api/v1/vendors \
  -H "Content-Type: application/json" \
  -d '{
    "name": "test_vendor",
    "url": "https://httpbin.org/post",
    "method": "POST",
    "headers": {"Content-Type": "application/json"},
    "body_template": "{\"user\": \"{{.user_id}}\", \"event\": \"{{.event}}\"}"
  }' | jq .
```

Save the returned `id` — you'll need it for the next step.

### Send a notification

```bash
curl -s -X POST http://localhost:8080/api/v1/notifications \
  -H "Content-Type: application/json" \
  -d '{
    "vendor_id": "<vendor-uuid-from-above>",
    "payload": {"user_id": "u_123", "event": "signup"}
  }' | jq .
```

### Check delivery status

```bash
curl -s http://localhost:8080/api/v1/notifications/<notification-uuid> | jq .
```

The status should transition from `pending` → `processing` → `delivered`.

## API Reference

| Method | Path | Description |
|--------|------|-------------|
| POST | /api/v1/notifications | Submit a notification |
| GET | /api/v1/notifications/:id | Check delivery status |
| POST | /api/v1/vendors | Create vendor config |
| GET | /api/v1/vendors | List all vendors |
| GET | /api/v1/vendors/:id | Get vendor details |
| PUT | /api/v1/vendors/:id | Update vendor config |
| DELETE | /api/v1/vendors/:id | Delete vendor |
| GET | /healthz | Health check |

## Configuration

All configuration is via environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| SERVER_PORT | :8080 | API server listen address |
| DATABASE_URL | postgres://notify:notify@localhost:5432/notify?sslmode=disable | PostgreSQL connection string |
| REDIS_ADDR | localhost:6379 | Redis address |
| REDIS_QUEUE | notifications | Redis queue key name |
| WORKER_COUNT | 5 | Number of delivery goroutines |
| RETRY_INTERVAL | 10s | How often to check for retryable notifications |
| RECOVERY_INTERVAL | 30s | How often to check for stale notifications |

## Running Tests

```bash
make test
```

Tests use mock implementations and `httptest` servers — no external dependencies required.

## Project Structure

```
notify-service/
├── cmd/
│   ├── server/main.go          # API server entrypoint
│   └── worker/main.go          # Queue consumer entrypoint
├── internal/
│   ├── api/                    # HTTP handlers, router, middleware
│   ├── config/                 # Environment-based configuration
│   ├── db/                     # Database connection and migrations
│   ├── model/                  # Domain structs
│   ├── queue/                  # Redis queue abstraction
│   ├── repository/             # Database queries
│   ├── service/                # Business logic (create + delivery)
│   └── template/               # Body template rendering
├── docker-compose.yml
└── Makefile
```

## Design Document

See [design.md](../design.md) for the full design document covering interaction sequences, system boundary, reliability guarantees, failure handling, and trade-offs.

## Branch Model & Contributing

```
main (stable) ← dev (integration) ← feat/xxx or bug/xxx
```

- `main` — production-ready, only accepts merges from `dev`
- `dev` — integration branch, all work merges here first
- Feature branches: `feat/xxx`, branched from `dev`
- Bug fix branches: `bug/xxx`, named after the bug (e.g. `bug/redis-reconnect`), branched from `dev`

### Workflow

```bash
# Start from dev
git checkout dev && git pull

# Create a branch
git checkout -b bug/template-nil-payload   # for bugs
git checkout -b feat/rate-limiting         # for features

# Make changes, then push and create PR to dev
git push -u origin bug/template-nil-payload
gh pr create --base dev

# CI must pass before merge
```

CI (go vet + test + build) runs on every push/PR to `main` and `dev`.
