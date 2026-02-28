# SpacetimeDB Module

The coordination layer. A Rust WASM module deployed to SpacetimeDB. All shared, real-time, transactional, and auditable state lives here.

## Table Domains

| Domain | Tables | Purpose |
|--------|--------|---------|
| Identity | `users`, `agent_runtimes`, `agent_tasks` | Who is who, what they're doing, what they're allowed to do |
| Agent Types | `agent_types`, `agent_type_versions` | Agent definitions and versioned releases |
| Memory | `agent_memory` | Working / short-term / long-term layers with decay |
| Budgets | `budget_allocations`, `budget_usage` | Token/cost budgets per agent and project |
| Observations | `observations`, `suggestions` | Agent outputs, proposals for action |
| Provenance | `provenance_nodes`, `provenance_edges` | Causal graph of all reasoning and actions |
| Events | `events` | Insert-only event bus, subscriptions as consumers |
| Approvals | `approval_queue` | Human-in-the-loop gates |
| Tools | `tool_registry`, `project_tools` | Capability discovery and registration |
| LLM Config | `llm_providers`, `routing_rules`, `provider_health`, `rate_limits` | Router configuration and telemetry |
| LLM Requests | `llm_requests` | Request logging, feeds back into health/routing |

## Scheduled Reducers

| Reducer | Frequency | Purpose |
|---------|-----------|---------|
| `decay_memory` | Every 10 min | Erode short-term memory based on decay rates |
| `replenish_budgets` | Daily / per-cycle | Reset or top up agent budgets |
| `update_provider_health` | Every 30 sec | Aggregate recent LLM requests into health metrics |
| `check_rate_limits` | Every 10 sec | Update current RPM/TPM, set throttle flags |
| `detect_anomalies` | Every 1 min | Error rate spikes, budget burn anomalies |
| `expire_approvals` | Every 5 min | Auto-escalate or auto-reject stale approval requests |
| `cleanup_events` | Every 1 hour | TTL on old events |
| `check_runtime_health` | Every 1 min | Flag runtimes with stale heartbeats |
| `expire_tasks` | Every 1 min | Cancel tasks past their expiry time |

## Key SpacetimeDB Properties Used

- **Real-time subscriptions** -- agents and humans see state changes instantly
- **Transactional reducers** -- every state change is atomic
- **Scheduled reducers** -- built-in heartbeat for maintenance, decay, health checks
- **Transaction log** -- complete audit trail, enables replay and analysis
- **Client identity** -- agents and humans distinguishable by connection identity
- **Lifecycle reducers** -- detect agent connect/disconnect

## Reducer Domains

### Identity Reducers (`reducers/identity.rs`)

User registration, runtime registration/deregistration, task lifecycle (assign, start, complete, fail, cancel), heartbeat updates.

### Memory Reducers (`reducers/memory.rs`)

Read and write memory entries across layers (working, short-term, long-term). Promote entries between layers. Prune expired or low-relevance entries.

### Budget Reducers (`reducers/budget.rs`)

Allocate budgets to agents/projects. Consume budget on LLM usage. Check remaining budget. Block requests when exhausted.

### Observation Reducers (`reducers/observations.rs`)

Post observations (agent findings/outputs). Acknowledge observations. Act on suggestions (approve/reject/modify).

### Provenance Reducers (`reducers/provenance.rs`)

Record provenance nodes (inputs, LLM calls, tool executions, outputs). Link edges between nodes to build causal graphs.

### Event Reducers (`reducers/events.rs`)

Publish events to the insert-only event table. Events and state changes combine atomically in single reducers.

### Approval Reducers (`reducers/approvals.rs`)

Request approval for agent-proposed actions. Approve or reject with notes. Approval + action execution in one atomic transaction.

### Tool Reducers (`reducers/tools.rs`)

Register tools with capability tags, endpoints, schemas. Deregister tools. Update health metrics and reliability scores.

### Versioning Reducers (`reducers/versioning.rs`)

CRUD for agent types. Create, promote, deprecate, retire agent type versions. Transition validation (draft->testing->active->deprecated->retired).

### LLM Config Reducers (`reducers/llm_config.rs`)

CRUD for LLM providers and routing rules. Log LLM requests with latency, cost, status. Update rate limit counters.

## Module Source Layout

```
packages/aule-spacetimedb/
└── src/
    ├── tables.rs              all SpacetimeDB table definitions
    ├── reducers/
    │   ├── identity.rs        user reg, runtime reg, task lifecycle
    │   ├── memory.rs          read, write, promote, prune
    │   ├── budget.rs          allocate, consume, check
    │   ├── observations.rs    post, acknowledge, act
    │   ├── provenance.rs      record nodes, link edges
    │   ├── events.rs          publish events
    │   ├── approvals.rs       request, approve, reject
    │   ├── tools.rs           register, deregister, health
    │   ├── versioning.rs      agent type CRUD, version lifecycle
    │   └── llm_config.rs      provider CRUD, routing rules, log requests
    ├── scheduled.rs           all scheduled reducers
    └── lib.rs
```
