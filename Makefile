.PHONY: help build run migrate migrate-down migrate-version db-up db-down agent

help:
	@echo "Available targets:"
	@echo "  build           Build all binaries"
	@echo "  run             Run the backend server"
	@echo "  agent           Run the agent in standalone mode"
	@echo "  migrate         Run all pending migrations"
	@echo "  migrate-down    Rollback all migrations"
	@echo "  migrate-version Show current migration version"
	@echo "  db-up           Start database with docker compose"
	@echo "  db-down         Stop database"

build:
	go build -o bin/backend ./cmd/backend
	go build -o bin/migrate ./cmd/migrate
	go build -o bin/agent ./cmd/agent

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

# Run agent in standalone mode (requires OPENAI_API_KEY)
agent: build
	STANDALONE=true ./bin/agent
