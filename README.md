# Aule

A project management system that runs specialized AI agents to complete work tasks autonomously. Aule enables autonomous, safe, and auditable task execution across various work types including research, design, architecture, implementation, documentation, and integration.

## Tech Stack

| Layer | Technology |
|-------|------------|
| Backend | Go 1.24, Fiber HTTP framework |
| Frontend | React 19, Bun, TailwindCSS 4, TanStack Router & Query |
| Database | PostgreSQL 18 |
| State | Zustand (auth), TanStack Query (server) |
| Communication | WebSocket (primary), REST (auth only) |
| LLM | OpenAI-compatible API via proxy service |
| Infrastructure | Docker Compose (dev), Kubernetes (prod) |

## Quick Start

```bash
# 1. Start PostgreSQL
make db-up

# 2. Run database migrations
make migrate

# 3. Start backend server (port 9000)
make run

# 4. Start frontend (port 3000) - in a separate terminal
cd frontend && bun install && bun run dev
```

The application will be available at `http://localhost:3000`.

## Project Structure

```
aule/
├── api/                 # OpenAPI specs (agent, auth, WebSocket schemas)
├── cmd/                 # Go entry points
│   ├── agent/           # Agent binary
│   ├── backend/         # Main API server
│   ├── llmproxy/        # LLM proxy service
│   └── migrate/         # Database migration tool
├── docs/                # Documentation
├── frontend/            # React SPA
│   ├── src/
│   │   ├── components/  # UI components (shadcn/ui patterns)
│   │   ├── routes/      # File-based routing
│   │   ├── services/    # API and WebSocket clients
│   │   └── model/       # TypeScript types
│   └── ...
└── internal/            # Go packages
    ├── agent/           # Agent execution framework
    ├── backend/         # API server implementation
    ├── database/        # Migrations
    ├── domain/          # Domain types
    ├── llmproxy/        # LLM proxy implementation
    ├── model/           # Shared models
    ├── repository/      # Data access layer
    └── service/         # Business logic
```

## Commands

### Backend

| Command | Description |
|---------|-------------|
| `make build` | Build all Go binaries |
| `make run` | Build and run the backend server |
| `make agent` | Run agent in standalone mode |
| `make llmproxy` | Run LLM proxy server |
| `make migrate` | Run pending database migrations |
| `make migrate-down` | Rollback all migrations |
| `make db-up` | Start PostgreSQL via Docker Compose |
| `make db-down` | Stop database |

### Frontend

Run these commands from the `frontend/` directory:

| Command | Description |
|---------|-------------|
| `bun install` | Install dependencies |
| `bun run dev` | Development server with HMR |
| `bun run build` | Production build |
| `bun run generate-routes` | Generate TanStack Router routes |

## Environment Variables

Create a `.env` file in the project root:

```bash
# Database (defaults work with docker-compose)
DATABASE_URL=postgres://aule:aule@localhost:5432/aule?sslmode=disable

# Required for agent execution
OPENAI_API_KEY=your-openai-api-key

# Required for LLM proxy
JWT_SECRET=your-jwt-secret
```

## Architecture

### WebSocket-First Communication

The primary UI communication happens over WebSocket with a structured envelope format supporting request/response correlation, idempotency, and message ordering. REST is only used for authentication flows.

### Agent System

Autonomous task execution using LLM-powered agents with:
- **Tool Framework**: File operations (read, write, edit, glob, grep) and bash execution
- **Backend Client**: Task lifecycle management via API
- **LLM Proxy**: Centralized API key management and streaming

### Domain Model

- **TaskType**: exploration, research, architecture, development, documentation, integration
- **TaskStage**: Type-specific workflow steps (e.g., plan -> implement -> review -> merge)
- **TaskStatus**: Execution state (ready, running, blocked, done, failed, cancelled)

## Documentation

| Document | Description |
|----------|-------------|
| [Concept](docs/concept.md) | Core concept, domain model, execution approach |
| [Agent](docs/agent.md) | Agent implementation details and API |
| [Tools](docs/tools.md) | Agent tool catalogue |
| [API Schemas](docs/architecture/api-schemas.md) | Schema structure, YAML to code flow |
| [WebSocket](docs/architecture/websocket.md) | Message format, connection lifecycle |
| [Subscriptions](docs/architecture/subscriptions.md) | Real-time updates pattern |
| [Backend Models](docs/architecture/backend-models.md) | Go type organization |

## License

Proprietary - All rights reserved.
