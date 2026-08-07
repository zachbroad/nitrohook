.PHONY: dev dev-setup dev-down run-api run-worker run-mcp build test test-unit test-integration test-all docker-build docker-up docker-down migrate-up migrate-down migrate-create create-db

DATABASE_URL ?= postgres://nitrohook:nitrohook@localhost:5432/nitrohook?sslmode=disable

# dev: one-command dev loop. Starts Postgres + Redis in Docker (waits until
# healthy), applies migrations via the API binary, installs UI deps if missing,
# then runs the API + in-process worker (Air, hot reload) alongside the
# React/Vite dev server (http://localhost:5173). Ctrl-C stops both foreground
# processes; infra keeps running (use `make dev-down` to stop it). Requires: Go,
# Docker, Node/npm, and Air (`make dev-setup`).
dev:
	@command -v air >/dev/null 2>&1 || { \
		echo "air not found. Install it with: make dev-setup"; exit 1; }
	docker compose up -d --wait postgres redis
	go run ./cmd/api --migrate
	@[ -d web/ui/node_modules ] || ( echo "Installing UI deps..."; cd web/ui && npm install )
	@echo "Starting Air (API + worker) and Vite (UI on http://localhost:5173). Ctrl-C stops both."
	@trap 'kill 0' INT TERM EXIT; \
		( cd web/ui && npm run dev ) & \
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

# Integration tests share a single Postgres DB and Redis instance, and each
# test's setup truncates/flushes globally. -p 1 serializes package test binaries
# so packages can't destroy each other's in-flight rows (see TestWorkerRetryFlow).
test-integration:
	go test -tags=integration -p 1 ./...

test-all:
	go test -tags=integration -p 1 ./...

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

ui-dev: ## Run the React admin SPA dev server
	cd web/ui && npm run dev

ui-build: ## Build the React admin SPA
	cd web/ui && npm run build
