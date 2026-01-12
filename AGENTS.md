# AGENTS.md

This file provides guidance to AI coding agents when working with code in this repository.

## Project Overview

Aule is a project management system that runs specialized agents to complete work tasks autonomously. It consists of a Go backend API, a React/Bun frontend, and an agent execution framework designed to run in Kubernetes.

## Commands

### Backend
```bash
make build          # Build all Go binaries (backend + migrate + agent + llmproxy)
make run            # Build and run the backend server
make agent          # Run agent in standalone mode (requires OPENAI_API_KEY)
make llmproxy       # Run LLM proxy server (requires OPENAI_API_KEY, JWT_SECRET)
make migrate        # Run pending database migrations
make migrate-down   # Rollback all migrations
make db-up          # Start PostgreSQL via docker compose
make db-down        # Stop database
```

### Frontend
```bash
cd frontend
bun install         # Install dependencies
bun run dev         # Development server with HMR (proxies API to localhost:9000)
bun run build       # Production build
bun run generate-routes  # Generate TanStack Router routes
```

### Running Full Stack
1. `make db-up` - Start PostgreSQL
2. `make migrate` - Run migrations
3. `make run` - Start backend on :9000
4. `cd frontend && bun run dev` - Start frontend on :3000

## Architecture

For detailed documentation, see:
- **[API Schemas](docs/architecture/api-schemas.md)** - Schema structure, YAML to code flow
- **[WebSocket Protocol](docs/architecture/websocket.md)** - Message format, connection lifecycle
- **[Subscriptions](docs/architecture/subscriptions.md)** - Real-time updates pattern
- **[Backend Models](docs/architecture/backend-models.md)** - Go type organization

### Backend (Go)
- **Fiber HTTP framework** with WebSocket support via `gofiber/websocket`
- **Entry point**: `cmd/backend/main.go` -> `internal/backend/api/api.go`
- **Configuration**: Environment-based via `internal/backend/config/`

### WebSocket Protocol (`internal/backend/wsproto/`)
Primary communication channel for UI. REST is only used for auth flows.

**Envelope structure** (defined in `internal/model/ws/envelope.go`):
```json
{
  "type": "message.type",
  "id": "uuid",
  "reply_to": "uuid (optional)",
  "idempotency_key": "string (optional)",
  "subscription_id": "uuid (optional)",
  "seq": 123,
  "time": "ISO timestamp",
  "payload": {}
}
```

**Event-based routing**: Incoming messages are published to the event bus. Service handlers subscribe via `WsToEvent()` to transform WS messages into domain events.

**Subscriptions** (`wsproto/subscriptions/`): Server-push pattern for real-time updates. Clients subscribe to topics and receive events when data changes. See [Subscriptions doc](docs/architecture/subscriptions.md) for implementation guide.

### Frontend (React + Bun)
- **Runtime**: Bun with HTML imports (not Vite)
- **Routing**: TanStack Router with file-based routes in `src/routes/`
- **State**: Zustand for auth, TanStack Query for server state
- **WebSocket client**: `src/services/websocket/websocket-client.ts` handles connection, reconnection, and message dispatch

**Frontend proxies** `/api/*` requests to the Go backend at `localhost:9000`.

### Service Layer Pattern
Services are wired in `internal/backend/api/services.go`:
- `auth.AuthService` - OAuth/JWT authentication
- `project.Service` - Project CRUD
- `agentapi.Service` - Agent task execution API
- `wssubscriptions.Service` - WebSocket subscription management

Each service typically has:
- `repository.go` - Repository interface
- `service.go` - Business logic
- `ws.go` - WebSocket handlers (or `http.go` for REST)
- `data.go` - Request/response types

### Agent System (`internal/agent/`)
The agent binary (`cmd/agent/main.go`) executes tasks autonomously:
- **LLM Provider** (`llm/`) - OpenAI-compatible API client or proxy client
- **Tools** (`tool/`) - File operations (read, write, edit, glob, grep) and bash
- **Runner** (`runner/`) - Agent loop that orchestrates LLM calls and tool execution
- **Client** (`client/`) - HTTP client for backend API

### LLM Proxy (`internal/llmproxy/`)
Separate service that manages LLM API credentials (`cmd/llmproxy/main.go`):
- **Provider** (`provider/`) - LLM provider implementations (OpenAI)
- **API** (`api/`) - HTTP handlers with JWT auth and SSE streaming
- Agents receive LLM config from backend and use proxy instead of direct API keys

See `docs/agent.md` for full documentation.

### Database
- PostgreSQL with migrations in `internal/database/migrations/`
- Repositories in `internal/repository/postgres/`
- Run `make migrate` after schema changes

## Key Conventions

### Go
- Message types are constants: `const MsgTypeProjectsList = "projects.list.req"`
- WebSocket handlers decode payload via `ctx.Message().DecodePayload(&request)`
- Reply with typed messages implementing `Type() string` interface

### Frontend
- Use Bun, not Node.js
- Bun auto-loads `.env` files
- WebSocket messages use the same envelope schema as backend
- Routes under `_auth/` require authentication

### API Client Generation
Frontend uses Orval for API client generation from OpenAPI specs:
```bash
cd frontend && bunx orval
```
Generated clients are in `src/services/*/api.gen.ts`.

## Domain Model

Tasks are the core entity with:
- **TaskType**: exploration, research, architecture, development, documentation, integration
- **TaskStage**: Type-specific workflow steps (e.g., plan -> implement -> review -> merge)
- **TaskStatus**: Execution state (ready, running, blocked, done, failed, cancelled)

Agent types are matched to tasks based on TaskType and TaskStage eligibility.
