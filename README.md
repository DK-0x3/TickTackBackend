# Tick Tack Backend

Go API for Tick Tack.

## Requirements

- Go 1.27+
- Docker with Docker Compose

## Run

```bash
cp .env.example .env
make db-up
make migrate-up
go run ./cmd/api
```

The example environment listens on `:8887`, matching the Web development
proxy. Without `.env` or `HTTP_ADDRESS`, the API defaults to `:8080`.

```http
GET /health
GET /ready
```

```json
{"status":"ok"}
```

`/health` reports that the process is running. `/ready` reports that the API
can reach PostgreSQL and is ready to serve database-backed requests.

## Tasks API

```http
POST  /tasks
GET   /tasks
GET   /tasks/:id
PATCH /tasks/:id/completion
DELETE /tasks/:id
POST  /tasks/:id/restore
```

`GET /tasks` uses cursor pagination. `limit` defaults to 20 and may be from 1
to 100. Omit `cursor` for the first page; pass the previous response's
`next_cursor` to get the next one. The cursor is an opaque string for clients.
Tasks are ordered by `created_at DESC, id DESC`.

```json
{
  "items": [{ "id": "...", "title": "..." }],
  "next_cursor": "...",
  "total": 42,
  "completed_total": 17
}
```

`next_cursor` is `null` at the end of the list. `total` counts all active tasks
for the current user, and `completed_total` counts completed active tasks for
the same user. Both counts cover all pages, regardless of the requested cursor.
They are recalculated for each request, so they may change while pages are
being loaded.

Authentication is not implemented yet. Task requests temporarily use the
fixed `DEVELOPMENT_USER_ID` from application configuration.

## Configuration

| Variable | Default | Description |
|---|---|---|
| `APP_ENV` | `local` | `local`, `test`, or `production` |
| `HTTP_ADDRESS` | `:8080` | HTTP server listen address |
| `DATABASE_URL` | local PostgreSQL URL | PostgreSQL connection string |
| `DEVELOPMENT_USER_ID` | fixed local UUID | Temporary user until authentication is implemented |

## Database

```bash
make db-up
make db-status
make migrate-up
make migrate-status
make sqlc
make db-down
```

Goose and sqlc versions are pinned in the Makefile and run through the Go
toolchain. They are development tools and do not become application runtime
dependencies.

## Checks

```bash
go test ./...
go vet ./...
```

The PostgreSQL integration test is skipped unless `TEST_DATABASE_URL` is set.
Use PostgreSQL 18 or newer, because the migration uses `uuidv7()`. The test
creates and removes a uniquely named schema in that database.
