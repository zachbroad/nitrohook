.PHONY: run-api run-worker build test test-unit test-integration test-all docker-build docker-up docker-down migrate-up migrate-down migrate-create create-db

DATABASE_URL ?= postgres://nitrohook:nitrohook@localhost:5432/nitrohook?sslmode=disable

run-api:
	go run ./cmd/api

run-worker:
	go run ./cmd/worker

build:
	go build -o bin/api ./cmd/api
	go build -o bin/worker ./cmd/worker

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
