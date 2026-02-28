# Roadmap

## Guiding Idea

Get a working loop as fast as possible: an agent that reasons, acts, and reports -- observable by a human through a frontend -- coordinated through SpacetimeDB. Everything runs locally. No K8s, no multi-model routing, no container images. Just the core mechanics proving themselves out.

Then layer in the production concerns: isolation, routing, budgets, provenance, multi-agent, deployment.

## POC Phases

### Phase 0 -- Foundation

**Goal:** Learn SpacetimeDB, set up the monorepo, prove the basics work.

- Install SpacetimeDB, complete the Rust quickstart
- Build a trivial module: tables, reducers, scheduled reducers, subscriptions, lifecycle hooks
- Document SpacetimeDB patterns, WASM constraints, and gotchas
- Set up the `aule/` monorepo with workspace and `aule-core` crate
- Verify the SpacetimeDB TypeScript client SDK works with Bun

**Deliverable:** Working SpacetimeDB module with a Rust test client and a Bun client both connecting and subscribing. Documented learnings.

### Phase 1 -- Minimal Coordination Layer

**Goal:** Build the SpacetimeDB tables and reducers needed for a single agent to register, receive a task, and report back.

- Identity tables: `users`, `agent_runtimes`, `agent_tasks`
- Agent type tables: `agent_types`, `agent_type_versions` (system prompt lives here)
- Core reducers: register runtime, assign task, start task, complete/fail task, heartbeat
- Observation table + reducer: agent posts observations, humans can read them
- Basic permission checking on reducers

Skip for now: memory layers, budgets, events, approvals, provenance, tool registry, LLM config tables, all scheduled reducers.

**Deliverable:** Deployed module where a Rust client can register as a runtime, get assigned a task, post observations, and complete the task.

### Phase 2 -- Agent Runtime (local process)

**Goal:** Build the agent as a local Rust process that connects to SpacetimeDB and calls a single hardcoded LLM API.

- Runtime lifecycle: connect to SpacetimeDB, register, subscribe to task assignments, idle loop with heartbeat
- On task pickup: fetch system prompt from `agent_type_versions`, construct prompt with task description
- Call Anthropic API directly (hardcoded provider, no router) for reasoning
- Implement `shell` tool: execute commands locally with basic safety (timeout, output truncation)
- Implement `aule_observe`: post observations to SpacetimeDB
- Implement `aule_status`: report progress back
- Agent loop: reason -> act (shell) -> observe -> repeat until done

Skip for now: `aule_suggest`, `aule_memory`, `aule_request` (uses router), Docker images, K8s, browser access.

**Deliverable:** A local Rust process that registers with SpacetimeDB, picks up a task, reasons about it using Claude, executes shell commands, posts observations, and completes the task.

### Phase 3 -- Frontend

**Goal:** Build a minimal web frontend that shows what's happening in real-time.

- Connect to SpacetimeDB using the TypeScript client SDK
- Real-time view of agent runtimes: status, current task, last heartbeat
- Real-time view of tasks: assigned, running, completed, failed
- Real-time observation feed: see agent outputs as they arrive
- Manual task creation: create a task and assign it to an idle runtime
- Basic agent type/version management: create types, create versions with system prompts

Skip for now: approval queue, intervention controls, memory/budget inspection, provenance visualization, version comparison.

**Deliverable:** Web dashboard where you can create a task, watch an agent pick it up, see its observations stream in, and see it complete.

### Phase 4 -- Close the Loop

**Goal:** Polish the POC into a coherent local development experience. Everything works together smoothly.

- Agent memory: add `agent_memory` table with simple read/write (single layer, no decay yet). Implement `aule_memory` tool so the agent can persist notes across tasks.
- Observation improvements: structured observation types (finding, progress, error, result), acknowledge/dismiss from frontend
- Task improvements: task expiry, cancellation from frontend, richer status reporting
- Error handling: agent reconnection, graceful failure on LLM errors, task timeout
- Developer experience: single command to start SpacetimeDB + agent + frontend locally

**Deliverable:** A reliable local setup where you can repeatedly assign tasks to an agent, watch it work, inspect its memory, and manage its lifecycle from a dashboard.

## Production Phases

### Phase 5 -- LLM Router

**Goal:** Replace the hardcoded Anthropic calls with a proper routing service.

- Implement standard completion API (axum server)
- Provider adapters: Anthropic, Google, OpenAI, local (Ollama/vLLM)
- Add LLM config tables to SpacetimeDB: `llm_providers`, `routing_rules`, `rate_limits`
- Router subscribes to config -- changes propagate without restart
- Task-based routing (reason->Opus, extract->Haiku, code->Sonnet)
- Request logging back to SpacetimeDB (`llm_requests`)
- Implement `aule_request` tool in agent runtime
- Budget tables (`budget_allocations`, `budget_usage`) and budget-aware degradation
- Scheduled reducers: `update_provider_health`, `check_rate_limits`

**Deliverable:** Running router that agents call instead of hitting providers directly. Routing decisions visible in SpacetimeDB.

### Phase 6 -- Isolation & Deployment

**Goal:** Move agent execution from local processes to isolated containers.

- Docker images: base, builder, research, data, ops
- Shell safety hardening: working directory confinement, destructive command blocklist, resource limits
- K8s deployment: agent pods, SpacetimeDB, router
- Credential management: K8s Secrets mounted per task, destroyed after
- Access control: network policies blocking internal services
- Runtime health: `check_runtime_health` scheduled reducer, stale heartbeat detection
- Task expiry: `expire_tasks` scheduled reducer

**Deliverable:** Agents running in isolated K8s pods, picking up tasks, executing safely, reporting back.

### Phase 7 -- Provenance & Supervision

**Goal:** Make the system observable and self-monitoring.

- Provenance tables (`provenance_nodes`, `provenance_edges`) and reducers
- Instrument agent runtime to record reasoning chains: input -> LLM call -> tool use -> output
- Provenance visualization in frontend
- Supervisor agent: subscribes to broad queries, watches for stuck/spinning agents, intervenes via reducers
- Tool registry: `tool_registry`, `project_tools` tables, capability discovery
- Agent self-registration as tools

**Deliverable:** Full audit trail of agent reasoning. Supervisor catching problems. Tools discoverable.

### Phase 8 -- Platform Maturity

**Goal:** Fill in the remaining coordination features.

- Memory layers: working / short-term / long-term with `decay_memory` scheduled reducer
- Approval queue: `approval_queue` table, `aule_suggest` tool, approval UI in frontend
- Events: `events` table, pub/sub via subscriptions, `cleanup_events` TTL
- Budget replenishment: `replenish_budgets` scheduled reducer
- Anomaly detection: `detect_anomalies` scheduled reducer
- Agent versioning workflows: canary deploys, A/B testing, version comparison in frontend
- Browser access layers: Playwright CLI, browser sidecar, browser pool
- Caching in router: exact-match and semantic

**Deliverable:** Full platform as described in the architecture docs.

## Application Phases

### Phase 9 -- First Application: Agentic Kanban

**Goal:** Validate the platform with a real multi-agent application.

- Define Kanban-specific agent types and versions
- Deploy multi-agent workspace
- Iterate on platform based on real usage

**Deliverable:** Working agentic Kanban with genuine value.

### Phase 10+ -- Expand

- Factory Digital Twin
- Supply Chain Mesh
- Agent Marketplace
- Testing framework
- Failure recovery
- Advanced cost controls
- Multi-tenancy
- Replay / simulation / counterfactual analysis
