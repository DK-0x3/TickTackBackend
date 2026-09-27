DATABASE_URL ?= postgres://tick_tack:tick_tack@localhost:5435/tick_tack?sslmode=disable
GOOSE = go run github.com/pressly/goose/v3/cmd/goose@v3.28.0
SQLC = go run github.com/sqlc-dev/sqlc/cmd/sqlc@v1.31.1

.PHONY: db-up db-down db-status migrate-up migrate-down migrate-status sqlc

db-up:
	docker compose up -d postgres

db-down:
	docker compose down

db-status:
	docker compose ps

migrate-up:
	$(GOOSE) -dir migrations postgres "$(DATABASE_URL)" up

migrate-down:
	$(GOOSE) -dir migrations postgres "$(DATABASE_URL)" down

migrate-status:
	$(GOOSE) -dir migrations postgres "$(DATABASE_URL)" status

sqlc:
	$(SQLC) generate
