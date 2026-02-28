# Architecture Overview

Aule is an agentic platform where humans and AI agents collaborate in shared workspaces. Agents are first-class participants with memory, budgets, tools, and identities. They coordinate through real-time shared state powered by SpacetimeDB, execute work in isolated K8s pods, and reason through a multi-model LLM router.

## Core Principles

- **Agents are processes, not magic.** An agent is a long-running process with an identity, tools, memory, and a reasoning loop.
- **Shell-first tooling.** Agents use CLIs and shell commands. Formal LLM tools are reserved for platform interaction only (~6 tools). Everything else is `shell()`.
- **SpacetimeDB is the nervous system.** All coordination, configuration, observation, memory, and audit state flows through SpacetimeDB.
- **Rust everywhere.** SpacetimeDB modules (WASM), LLM router, agent runtime -- all Rust. Shared types across the stack.

## System Diagram

```
+--------------------------------------------------+
|  Aule Platform                                    |
|                                                   |
|  +----------------------------------------------+ |
|  |  SpacetimeDB Module (Rust -> WASM)           | |
|  |  Coordination / Memory / Budgets / Config    | |
|  |  Provenance / Events / Approvals / Registry  | |
|  +----------------------------------------------+ |
|                                                   |
|  +----------------------------------------------+ |
|  |  LLM Router (native Rust service)            | |
|  |  Multi-model routing / Budget enforcement    | |
|  |  Caching / Observability                     | |
|  |  Config from SpacetimeDB subscriptions       | |
|  +----------------------------------------------+ |
|                                                   |
|  +----------------------------------------------+ |
|  |  Agent Runtime (K8s pods)                    | |
|  |  Isolated workspaces / Shell + CLIs          | |
|  |  Browser access / External system access     | |
|  +----------------------------------------------+ |
|                                                   |
|  +----------------------------------------------+ |
|  |  LLM Providers                               | |
|  |  Claude / Gemini / OpenAI / Local (vLLM)     | |
|  +----------------------------------------------+ |
+--------------------------------------------------+
```

## Components

### SpacetimeDB Module

The coordination layer. A Rust WASM module deployed to SpacetimeDB. All shared, real-time, transactional, and auditable state lives here: identity, memory, budgets, observations, events, approvals, tool registry, and LLM config.

See [spacetimedb-module.md](spacetimedb-module.md).

### LLM Router

A native Rust HTTP service (axum). Routes LLM requests from agents to the best provider based on task type, quality requirements, budget constraints, and provider health. Config is live-synced from SpacetimeDB subscriptions.

See [llm-router.md](llm-router.md).

### Agent Runtime

Each agent is a K8s pod with an isolated workspace, curated CLI set, and connections to SpacetimeDB and the LLM Router. Agents have 6 platform tools and use `shell()` for everything else.

See [agent-runtime.md](agent-runtime.md).

## Project Structure

```
aule/
├── packages/                          Rust workspace crates
│   ├── aule-core/                     common structs, agent protocol, API types
│   ├── aule-spacetimedb/             coordination layer (compiles to WASM)
│   ├── aule-router/                   LLM routing service (native Rust binary)
│   └── aule-runtime/                  agent process (native Rust binary, in pods)
│
├── app/                               front-end application (Bun/TypeScript)
├── docs/                              documentation
└── docker/                            agent container images
```

### Crate Details

| Crate | Type | Purpose |
|-------|------|---------|
| `aule-core` | Library | Shared types: agent identity, routing types, memory, budget, observation, provenance, versioning |
| `aule-spacetimedb` | WASM module | SpacetimeDB tables, reducers, scheduled reducers |
| `aule-router` | Binary | axum HTTP server for LLM routing with SpacetimeDB config sync |
| `aule-runtime` | Binary | Agent process lifecycle, task execution, shell safety, tool implementation |

### Docker Images

| Image | Base + additions | Used by |
|-------|-----------------|---------|
| `Dockerfile.base` | Core utils, shared tools | All agents |
| `Dockerfile.builder` | + cargo, git, spacetime CLI | Builder agents |
| `Dockerfile.research` | + playwright, trafilatura | Research agents |
| `Dockerfile.data` | + duckdb, python, pandas | Data agents |
| `Dockerfile.ops` | + spacetime, kubectl, terraform | Ops agents |
