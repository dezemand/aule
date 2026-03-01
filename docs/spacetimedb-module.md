# SpacetimeDB Module

The coordination layer. A Rust WASM module deployed to SpacetimeDB. All shared, real-time, transactional, and auditable state lives here. The SpacetimeDB module is the single source of truth for all platform state. The K8s operator, LLM router, agent runtimes, and frontend all connect as SpacetimeDB clients.

## Table Domains

| Domain | Tables | Purpose |
|--------|--------|---------|
| Identity | `users`, `agent_runtimes`, `agents` | Who is who, runtime capabilities, agent instances |
| Agent Types | `agent_types`, `agent_type_versions` | Agent definitions, versioned releases with system prompts |
| Workspaces | `workspaces`, `workspace_pools` | Shared context boundaries, warm pod pool config |
| Tasks | `tasks`, `task_attempts`, `task_plans`, `task_steps`, `task_dependencies`, `task_contexts`, `task_results`, `task_reviews` | Work lifecycle from creation to review |
| Observations | `observations` | Agent findings, progress, errors, questions |
| Memory | `agent_memory` | Working / short-term / long-term layers with decay |
| Budgets | `budget_allocations`, `budget_usage` | Token/cost budgets per workspace, agent type, and task |
| Events | `events` | Insert-only event bus, subscriptions as consumers |
| Approvals | `approvals` | Human-in-the-loop gates for agent actions |
| Messages | `messages` | Direct user-agent communication per task |
| Tools | `tools` | Capability discovery and registration |
| Provenance | `provenance_nodes`, `provenance_edges` | Causal graph of all reasoning and actions |
| LLM Config | `llm_providers`, `routing_rules`, `provider_health`, `rate_limits` | Router configuration and health |
| LLM Requests | `llm_requests` | Request logging, feeds back into health/routing |

## Scheduled Reducers

| Reducer | Frequency | Purpose |
|---------|-----------|---------|
| `decay_memory` | Every 10 min | Erode short-term memory based on decay rates |
| `replenish_budgets` | Daily / per-cycle | Reset or top up budgets |
| `update_provider_health` | Every 30 sec | Aggregate recent LLM requests into health metrics |
| `check_rate_limits` | Every 10 sec | Update current RPM/TPM |
| `detect_anomalies` | Every 1 min | Error rate spikes, budget burn anomalies |
| `expire_approvals` | Every 5 min | Auto-escalate or auto-reject stale approvals |
| `cleanup_events` | Every 1 hour | TTL on old events |
| `check_runtime_health` | Every 1 min | Flag runtimes with stale heartbeats |
| `expire_tasks` | Every 1 min | Cancel tasks past their expiry time |

## Key SpacetimeDB Properties Used

- **Real-time subscriptions** — agents, operator, router, and humans see state changes instantly
- **Transactional reducers** — every state change is atomic, no partial states
- **Scheduled reducers** — built-in heartbeat for maintenance, decay, health checks
- **Transaction log** — complete audit trail, enables replay and analysis
- **Client identity** — agents, operator, router, and humans distinguishable by connection identity
- **Lifecycle reducers** — detect agent connect/disconnect

## Reducer Domains

### Identity Reducers (`reducers/identity.rs`)

User registration. Runtime registration/deregistration and heartbeat. Agent launch and stop.

### Task Reducers (`reducers/task.rs`)

Task CRUD. Task lifecycle (create, queue, assign, complete, fail, cancel). Attempt management. Plan and step management. Context attachment. Result submission. Review and revision.

### Observation Reducers (`reducers/observation.rs`)

Post observations (findings, progress, errors, results, questions). Acknowledge observations.

### Memory Reducers (`reducers/memory.rs`)

Read and write memory entries across layers (working, short-term, long-term). Promote entries between layers. Prune expired or low-relevance entries.

### Budget Reducers (`reducers/budget.rs`)

Allocate budgets to workspaces, agent types, and tasks. Consume budget on LLM usage. Check remaining budget. Block requests when exhausted.

### Provenance Reducers (`reducers/provenance.rs`)

Record provenance nodes (LLM calls, tool executions, observations, memory reads, human inputs). Link edges between nodes to build causal graphs.

### Event Reducers (`reducers/events.rs`)

Publish events to the insert-only event table. Events and state changes combine atomically in single reducers.

### Approval Reducers (`reducers/approvals.rs`)

Request approval for agent-proposed actions. Approve or reject with notes. Approval + action execution in one atomic transaction.

### Tool Reducers (`reducers/tools.rs`)

Register tools with capability tags, endpoints, schemas. Deregister tools. Update health metrics and reliability scores.

### Versioning Reducers (`reducers/versioning.rs`)

CRUD for agent types. Create, promote, deprecate, retire agent type versions. Transition validation (draft → testing → active → deprecated → retired).

### Workspace Reducers (`reducers/workspace.rs`)

Workspace CRUD. Pool configuration. Lifecycle management (creating → active → draining → archived).

### LLM Config Reducers (`reducers/llm_config.rs`)

CRUD for LLM providers and routing rules. Log LLM requests with latency, cost, status. Update rate limit counters.

## Module Source Layout

```text
packages/aule-spacetimedb/
└── src/
    ├── tables.rs              all SpacetimeDB table definitions
    ├── reducers/
    │   ├── identity.rs        user reg, runtime reg, agent launch/stop
    │   ├── task.rs            task lifecycle, attempts, plans, steps, review
    │   ├── observation.rs     post, acknowledge
    │   ├── memory.rs          read, write, promote, prune
    │   ├── budget.rs          allocate, consume, check
    │   ├── provenance.rs      record nodes, link edges
    │   ├── events.rs          publish events
    │   ├── approvals.rs       request, approve, reject
    │   ├── tools.rs           register, deregister, health
    │   ├── versioning.rs      agent type CRUD, version lifecycle
    │   ├── workspace.rs       workspace CRUD, pool config
    │   └── llm_config.rs      provider CRUD, routing rules, log requests
    ├── scheduled.rs           all scheduled reducers
    └── lib.rs
```
