.PHONY: dev dev-setup dev-down run-api run-worker run-mcp build test test-unit test-integration test-all docker-build docker-up docker-down migrate-up migrate-down migrate-create create-db

DATABASE_URL ?= postgres://nitrohook:nitrohook@localhost:5432/nitrohook?sslmode=disable

# dev: one-command dev loop. Starts Postgres + Redis in Docker (waits until
# healthy), applies migrations via the API binary, then runs the API + in-process
# worker under Air for hot reload. Ctrl-C stops Air; infra keeps running (use
# `make dev-down` to stop it). Requires: Go, Docker, and Air (`make dev-setup`).
dev:
	@command -v air >/dev/null 2>&1 || { \
		echo "air not found. Install it with: make dev-setup"; exit 1; }
	docker compose up -d --wait postgres redis
	go run ./cmd/api --migrate
	air

# dev-setup: install the Air hot-reload tool.
dev-setup:
	go install github.com/air-verse/air@latest

# dev-down: stop the dev infra (Postgres + Redis) started by `make dev`.
dev-down:
	docker compose down

run-api:
	go run ./cmd/api

run-worker:
	go run ./cmd/worker

run-mcp:
	go run ./cmd/mcp

build:
	go build -o bin/api ./cmd/api
	go build -o bin/worker ./cmd/worker
	go build -o bin/mcp ./cmd/mcp

test:
	go test ./...

test-unit:
	go test ./...

test-integration:
	go test -tags=integration ./...

test-all:
	go test -tags=integration ./...

docker-build:
	docker compose build

docker-up:
	docker compose up -d

docker-down:
	docker compose down

docker-up-supporting-svc:
	docker compose up -d postgres redis

migrate-up:
	migrate -database "$(DATABASE_URL)" -path migrations up

migrate-down:
	migrate -database "$(DATABASE_URL)" -path migrations down

migrate-create:
	@read -p "Migration name: " name; \
	migrate create -ext sql -dir migrations -seq $$name

create-db:
	psql -d postgres -c "CREATE ROLE nitrohook WITH LOGIN PASSWORD 'nitrohook';" 2>/dev/null || true
	psql -d postgres -c "CREATE DATABASE nitrohook OWNER nitrohook;" 2>/dev/null || true
	psql -d nitrohook -c "GRANT ALL ON SCHEMA public TO nitrohook;" 2>/dev/null || true
