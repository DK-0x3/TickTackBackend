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
## Task listing

The legacy `GET /tasks` endpoint was removed on 31 October 2026 after checking
workspace consumers. Use `GET /inbox/views` and
`GET /task-views/:viewID/tasks` for Inbox lists. `POST /tasks` and
`GET /tasks/:id` remain available. Task-view pagination is unchanged.
