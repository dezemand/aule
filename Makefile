.PHONY: help build run migrate migrate-down migrate-version db-up db-down

help:
	@echo "Available targets:"
	@echo "  build           Build all binaries"
	@echo "  run             Run the backend server"
	@echo "  migrate         Run all pending migrations"
	@echo "  migrate-down    Rollback all migrations"
	@echo "  migrate-version Show current migration version"
	@echo "  db-up           Start database with docker compose"
	@echo "  db-down         Stop database"

build:
	go build -o bin/backend ./cmd/backend
	go build -o bin/migrate ./cmd/migrate

run: build
	./bin/backend

migrate: build
	./bin/migrate up

migrate-down: build
	./bin/migrate down

migrate-version: build
	./bin/migrate version

db-up:
	docker compose up -d postgres

db-down:
	docker compose down
