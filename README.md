# Task Management API — Backend Developer Test

A fully working Go REST API with JWT authentication, task management, idempotency, assignment workflows, structured logging, unit tests, and Hurl integration tests — all wrapped in Docker with a Makefile.

## Tech Stack

| Component     | Version | Description                          |
|---------------|---------|--------------------------------------|
| Go            | 1.26.4  | Language & runtime                   |
| PostgreSQL    | 18.4    | Database                             |
| Echo          | v5      | HTTP framework                       |
| pgx           | v5      | PostgreSQL driver                    |
| Echo-JWT      | v5      | JWT authentication middleware        |
| golang-jwt    | v5      | JWT signing & validation             |
| squirrel      | v5      | SQL query builder                    |
| Hurl          | 5.x     | HTTP integration testing             |
| Docker        | latest  | Container runtime                    |
| otel-lgtm     | latest  | Observability (logs, traces, metrics)|

## Prerequisites

- [Go 1.26.4+](https://go.dev/dl/)
- [Docker](https://docs.docker.com/get-docker/)
- [Hurl](https://hurl.dev/docs/installation.html)
- [k6](https://grafana.com/docs/k6/latest/get-started/installation/) (for load testing)

## Quick Start

### Run with Docker Compose

```bash
# Start all services (PostgreSQL + API + otel-lgtm)
docker compose up -d

# API is available at http://localhost:8080
# Grafana UI at http://localhost:3000
```

### Run with Go (requires Postgres running)

```bash
# Start Postgres only
docker compose up -d postgres

# Run migrations
make db-migrate

# Start the API
make run
```

## Testing

### Unit Tests

```bash
make test
# or
make test-unit
```

Runs all unit tests with the `-race` flag to detect race conditions (critical for idempotency tests).

### Integration Tests

```bash
make test-integration
```

Starts Docker services, runs Hurl tests against the live API, generates an HTML report, then cleans up.

### Lint

```bash
make lint
```

## Makefile Commands

| Command            | Description                              |
|--------------------|------------------------------------------|
| `make run`       | Build Docker image + start all services in background |
| `make build`     | Compile the Go binary                   |
| `make test`      | Run unit tests with race detection       |
| `make test-integration` | Run Hurl integration tests         |
| `make load-test` | Run k6 load test against running API     |
| `make db-migrate`| Run database migrations                 |
| `make lint`        | Run `go vet`                             |
| `make clean`       | Remove artifacts and Docker volumes      |

## API Endpoints

### Authentication (Public)

| Method | Path        | Description                  |
|--------|-------------|------------------------------|
| POST   | `/register` | Create a new user account    |
| POST   | `/login`    | Authenticate and get JWT     |
| GET    | `/health`   | Health check                 |

### Register

```bash
curl -X POST http://localhost:8080/register \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"secure123","name":"Test User"}'
```

Response: `201` with `{"user": {...}, "token": "..."}`

### Login

```bash
curl -X POST http://localhost:8080/login \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"secure123"}'
```

Response: `200` with `{"token": "..."}`

### Tasks (Protected — requires `Authorization: Bearer <token>`)

| Method | Path                  | Description                          |
|--------|-----------------------|--------------------------------------|
| POST   | `/tasks`              | Create a task (idempotent)           |
| GET    | `/tasks`              | List own tasks (paginated, filterable)|
| GET    | `/tasks/:id`          | Get task detail                      |
| PUT    | `/tasks/:id`          | Update task (owner only)             |
| DELETE | `/tasks/:id`          | Delete task (owner only)             |
| POST   | `/tasks/:id/assign`   | Assign task (same team, transactional)|

### Idempotent Task Creation

```bash
curl -X POST http://localhost:8080/tasks \
  -H "Authorization: Bearer <token>" \
  -H "Idempotency-Key: $(uuidgen)" \
  -H "Content-Type: application/json" \
  -d '{"title":"My Task","description":"Something to do"}'
```

### List Tasks with Filters

```bash
curl "http://localhost:8080/tasks?status=pending&search=urgent&page=1&limit=10" \
  -H "Authorization: Bearer <token>"
```

### Assign Task

```bash
curl -X POST http://localhost:8080/tasks/<task-id>/assign \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"user_id":"<assignee-uuid>"}'
```

## Structured Error Format

All errors follow this JSON format:

```json
{
  "status": 400,
  "code": "ERR_VALIDATION",
  "message": "Title is required",
  "timestamp": "2026-06-14T10:30:00Z"
}
```

| Code               | HTTP Status | Description              |
|--------------------|-------------|--------------------------|
| `ERR_VALIDATION`   | 400         | Bad input                |
| `ERR_UNAUTHORIZED` | 401         | Missing/invalid token    |
| `ERR_FORBIDDEN`    | 403         | Not owner/not in team    |
| `ERR_NOT_FOUND`    | 404         | Resource not found       |
| `ERR_CONFLICT`     | 409         | Duplicate email, etc.    |
| `ERR_UNPROCESSABLE`| 422         | Business rule violation  |
| `ERR_INTERNAL`     | 500         | Internal server error    |

## Project Structure

```
gdc-backend-test/
├── cmd/server/main.go          # Entry point
├── internal/
│   ├── config/config.go        # Environment configuration
│   ├── db/
│   │   ├── postgres.go         # pgx pool & migrations
│   │   └── migrations/001_init.sql
│   ├── middleware/              # Request ID, logger, auth, idempotency, error handler
│   ├── model/                   # Domain structs, DTOs, error types
│   ├── repository/              # Data access layer (pgx)
│   ├── service/                 # Business logic layer
│   └── handler/                 # HTTP handlers (Echo)
├── tests/
│   ├── unit/                    # Go unit tests (with -race)
│   └── integration/             # Hurl integration tests
├── Dockerfile                   # Multi-stage Go build
├── docker-compose.yml           # postgres 18.4 + api + otel-lgtm
├── Makefile                     # All commands
└── README.md
```

## Architecture

Clean Architecture: **Handler → Service → Repository → DB**

- **Handler**: HTTP concerns only — request parsing, response serialization
- **Service**: Business logic, validation, authorization, transaction orchestration
- **Repository**: Data access with pgx + squirrel query builder
- **Middleware**: Cross-cutting concerns — auth, logging, idempotency, error handling

## Environment Variables

| Variable      | Default                                    | Description           |
|---------------|--------------------------------------------|-----------------------|
| `DATABASE_URL`| `postgres://postgres:***@localhost:5432/taskdb?sslmode=disable` | PostgreSQL connection |
| `JWT_SECRET`  | `dev-secret-change-in-production`          | JWT signing secret    |
| `PORT`        | `8080`                                     | HTTP listen port      |

---

## Development Environment

This project was developed and tested on the following machine:

| Component   | Spec                                              |
|-------------|---------------------------------------------------|
| OS          | Ubuntu 26.04 LTS (Resolute Raccoon) x86_64        |
| Host        | IdeaPad Slim 5 14AKP10 (83HX)                     |
| Kernel      | Linux 7.0.0-22-generic                            |
| CPU         | AMD Ryzen AI 7 350 (16) @ 5.09 GHz                |
| GPU         | AMD Radeon 860M Graphics [Integrated]              |
| Memory      | 18.81 GiB (5.05 GiB used)                         |
| Disk        | 84.73 GiB (39% used)                              |
| Go          | 1.26.4                                            |
| Docker      | 29.5.3                                            |
| Hurl        | 7.1.0                                             |
| k6          | 2.0.0                                             |

## Idempotency

- POST `/tasks` supports `Idempotency-Key` (UUID) header
- Duplicate requests with the same key within 24 hours return the same response
- Race-condition safe via `INSERT ... ON CONFLICT DO NOTHING`
- Unit tests verify concurrent goroutines with the same key produce exactly one task

## Assignment Transaction

Assigning a task to another user:

1. Validates task ownership
2. Verifies both users are in the same team
3. Begins a PostgreSQL transaction
4. Updates `tasks.assignee_id`
5. Inserts audit log in `task_logs`
6. Logs a mock notification via `slog`
7. Commits (or rolls back entirely on failure)
