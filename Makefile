include .env
export

COMPOSE ?= docker compose

MIGRATION_DATABASE_URL := postgres://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@postgres:5432/$(POSTGRES_DB)?sslmode=disable

.PHONY: up down logs test lint db-up db-down ps run migrate-create migrate-up migrate-down swag

deploy:
	@MSYS_NO_PATHCONV=1 $(COMPOSE) up -d --build

undeploy:
	@$(COMPOSE) down

logs:
	@$(COMPOSE) logs -f --tail=100

test:
	@cd backend && go test ./... -cover

lint:
	@cd backend && golangci-lint run ./... --fix

db-up:
	@MSYS_NO_PATHCONV=1 $(COMPOSE) up -d postgres

db-down:
	@$(COMPOSE) stop postgres

ps:
	@$(COMPOSE) ps

run:
	@cd backend && go run ./cmd/antiscam

migrate-create:
	@if [ -z "$(seq)" ]; then \
		echo "No seq parameter found. Usage: make migrate-create seq=init"; \
		exit 1; \
	fi; \
	MSYS_NO_PATHCONV=1 $(COMPOSE) run --rm postgres-migrate \
		create \
		-ext sql \
		-dir /migrations \
		-seq "$(seq)"

migrate-up:
	@MSYS_NO_PATHCONV=1 $(COMPOSE) run --rm postgres-migrate

migrate-down:
	@MSYS_NO_PATHCONV=1 $(COMPOSE) run --rm postgres-migrate \
		-path /migrations \
		-database $(MIGRATION_DATABASE_URL) \
		down

swag:
	@cd backend && swag init -g ./cmd/antiscam/main.go -o ./docs