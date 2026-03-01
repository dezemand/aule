# Aulë — North Star Architecture

> **This document describes the target architecture for Aulë.** It is the north star, not the current state. The existing codebase implements a subset of what is described here (see the roadmap document for current phase and progress). When working on the codebase, use this document to understand the direction and make decisions consistent with the overall design. The roadmap phases need to be revisited to align with this north star.

---

## Vision

Aulë is a general-purpose agentic execution engine. Humans define work, agents execute it. Agents reason through LLMs, act through shell commands and CLIs, coordinate through real-time shared state in SpacetimeDB, and run in isolated K8s pods managed by a custom operator.

It is not a knowledge base. It is not a chatbot. It is an execution engine where the complexity of what you can accomplish emerges from composable primitives: agents that can reason, act, observe, remember, and access external systems — orchestrated by humans who define tasks, review results, and make decisions.

The name comes from the Vala of craftsmanship in Tolkien's legendarium — the smith who forged things into being.

## Core Principles

- **Agents are processes, not magic.** An agent is a process with an identity, tools, memory, and a reasoning loop.
- **Shell-first tooling.** Agents use CLIs and shell commands. Formal LLM tools are reserved for platform interaction only (~6 tools). Everything else is `shell()`. This keeps context windows lean.
- **SpacetimeDB is the nervous system.** All coordination, configuration, observation, memory, and audit state flows through SpacetimeDB. If it needs to be shared, real-time, or auditable — it's a table.
- **Rust everywhere.** SpacetimeDB modules (WASM), LLM router, agent runtime, K8s operator — all Rust. Shared types across the stack via `aule-core`.
- **Isolation through K8s.** One agent per pod. Workspace-scoped shared filesystems. Credentials mounted per-task, revoked on completion. Network policies per-workspace.
- **Open, not prescriptive.** The platform doesn't impose workflows. A task can be "check if this API returns 200" or "design and implement a distributed scheduling system." The agent gets a description, tools, and credentials, and figures it out.

---

## Architecture Overview

```text
┌──────────────────────────────────────────────────────────────┐
│  Aulë Platform                                                │
│                                                                │
│  ┌──────────────────────────────────────────────────────────┐ │
│  │  SpacetimeDB Module (Rust → WASM)                        │ │
│  │  The nervous system: all shared state lives here          │ │
│  │  Tables / Reducers / Scheduled Reducers / Subscriptions   │ │
│  └──────────────────────────────────────────────────────────┘ │
│                                                                │
│  ┌──────────────────────────────────────────────────────────┐ │
│  │  K8s Operator (Rust, kube-rs)                             │ │
│  │  Watches SpacetimeDB state, reconciles K8s resources      │ │
│  │  Manages: workspaces, warm pod pools, agent lifecycle     │ │
│  └──────────────────────────────────────────────────────────┘ │
│                                                                │
│  ┌──────────────────────────────────────────────────────────┐ │
│  │  LLM Router (Rust, axum)                                  │ │
│  │  Multi-model routing / Budget enforcement / Caching       │ │
│  │  Config from SpacetimeDB subscriptions                    │ │
│  └──────────────────────────────────────────────────────────┘ │
│                                                                │
│  ┌──────────────────────────────────────────────────────────┐ │
│  │  Agent Runtimes (K8s pods, one agent per pod)             │ │
│  │  Warm pod pools per workspace                             │ │
│  │  Workspace PVC mounted / Shell + CLIs / Scoped creds     │ │
│  └──────────────────────────────────────────────────────────┘ │
│                                                                │
│  ┌──────────────────────────────────────────────────────────┐ │
│  │  Frontend (Bun/TypeScript)                                │ │
│  │  SpacetimeDB subscriptions for real-time UI               │ │
│  │  Dashboard / Task management / Observation feed           │ │
│  └──────────────────────────────────────────────────────────┘ │
│                                                                │
│  ┌──────────────────────────────────────────────────────────┐ │
│  │  LLM Providers                                            │ │
│  │  Claude / Gemini / OpenAI / Local (vLLM, Ollama)          │ │
│  └──────────────────────────────────────────────────────────┘ │
└──────────────────────────────────────────────────────────────┘
```

---

## Metamodel

All objects are SpacetimeDB tables unless noted otherwise. The SpacetimeDB module is the single source of truth for all platform state. The K8s operator, LLM router, agent runtimes, and frontend all connect as SpacetimeDB clients, subscribing to relevant tables and calling reducers to mutate state.

### Identity

#### User

A human operator interacting through the frontend.

```rust
pub struct User {
    #[primary_key]
    pub id: u64,
    pub identity: Identity,             // SpacetimeDB connection identity
    pub name: String,
    pub role: String,                   // "admin", "operator", "viewer"
    pub created_at: u64,
}
```

#### AgentType

A definition of a kind of agent — its name, purpose, and required capabilities.

```rust
pub struct AgentType {
    #[primary_key]
    pub id: u64,
    pub name: String,                   // "builder", "research", "generalist"
    pub description: String,
    pub required_capabilities: String,  // JSON array: ["rust", "git", "cargo"]
    pub created_at: u64,
    pub created_by: u64,                // → users
}
```

An agent type defines what capabilities a runtime must have to run agents of this type. The `required_capabilities` are checked against the runtime's `capabilities` at agent launch time.

#### AgentTypeVersion

A specific release of an agent type. Contains the system prompt, configuration, and behavioral parameters. This is what defines agent behavior.

```rust
pub struct AgentTypeVersion {
    #[primary_key]
    pub id: u64,
    pub agent_type_id: u64,            // → agent_types
    pub version: String,               // semver: "1.3.0"
    pub image_tag: String,             // "aule-builder:1.3.0" (minimum image for this version)
    pub system_prompt: String,         // the agent's core instructions
    pub config: String,                // JSON: default budget, tool prefs, behavior params
    pub release_notes: String,
    pub status: String,                // "draft", "testing", "active", "deprecated", "retired"
    pub created_at: u64,
    pub created_by: u64,               // → users
}
```

**Version lifecycle:**

```text
draft → testing → active → deprecated → retired
```

| Status | Meaning |
|--------|---------|
| `draft` | Being developed, not deployable |
| `testing` | Can deploy to test runtimes, not production |
| `active` | Production-ready, new agents use this version |
| `deprecated` | Still works, but new agents should use a newer version |
| `retired` | No longer deployable, existing agents flagged for upgrade |

The system prompt is versioned data. The runtime fetches it from SpacetimeDB on agent launch — prompt-only changes don't require pod restarts. This enables A/B testing of prompts within the same image version.

#### AgentRuntime

A running K8s pod. The compute substrate. Has capabilities determined by its container image (what CLIs and programs are installed). Does NOT have an agent type — it's a generic machine that can host any agent whose requirements it satisfies.

```rust
pub struct AgentRuntime {
    #[primary_key]
    pub id: u64,
    pub identity: Identity,             // SpacetimeDB connection identity
    pub workspace_id: u64,              // → workspaces (bound via PVC mount)
    pub instance_name: String,          // "runtime-01"
    pub image_tag: String,              // "aule-builder:latest"
    pub capabilities: String,           // JSON array: ["rust", "git", "cargo", "spacetime-cli"]
    pub status: String,                 // "available", "occupied", "draining", "offline"
    pub current_agent_id: Option<u64>,  // → agents
    pub registered_at: u64,
    pub last_heartbeat: u64,
}
```

**Runtime status values:**

| Status | Meaning |
|--------|---------|
| `available` | Warm, idle, ready to host an agent |
| `occupied` | Currently hosting an active agent |
| `draining` | Finishing current work, won't accept new agents |
| `offline` | Not responding, operator will handle |

Runtimes are managed as warm pools by the K8s operator. They start up, connect to SpacetimeDB, register, and wait. They persist across agent launches — when an agent stops, the runtime is cleaned up (scratch dir wiped, credentials removed) and returned to the pool.

#### Agent

An instance of an AgentTypeVersion, launched on a runtime. The actual worker. One agent per runtime at a time.

```rust
pub struct Agent {
    #[primary_key]
    pub id: u64,
    pub runtime_id: u64,               // → agent_runtimes (where I'm running)
    pub agent_type_version_id: u64,    // → agent_type_versions (what I am)
    pub workspace_id: u64,             // → workspaces
    pub status: String,                // "starting", "idle", "working", "stopping", "stopped"
    pub current_task_id: Option<u64>,  // → tasks
    pub launched_at: u64,
    pub stopped_at: Option<u64>,
}
```

**Agent launch flow:**

```text
1. Launch request: agent_type_version_id + workspace_id
2. System resolves required_capabilities from agent_type → agent_type_versions
3. Finds available runtime in workspace with matching capabilities
4. Creates Agent row: status "starting"
5. Runtime receives agent config via subscription
6. Runtime fetches system prompt from agent_type_versions table
7. Agent status → "idle", runtime status → "occupied"
8. Agent subscribes to task assignments, waits for work
```

**Capability check at launch:**

```rust
fn can_run(runtime: &AgentRuntime, agent_type: &AgentType) -> bool {
    let runtime_caps: HashSet<String> = parse_json(&runtime.capabilities);
    let required_caps: HashSet<String> = parse_json(&agent_type.required_capabilities);
    required_caps.is_subset(&runtime_caps)
}
```

### Workspace

A shared context boundary. Groups related work, agents, and resources. Owns a persistent filesystem (K8s PVC) that all agents in the workspace share.

```rust
pub struct Workspace {
    #[primary_key]
    pub id: u64,
    pub name: String,                   // "aule-dev", "scania-flex-prod"
    pub description: String,
    pub status: String,                 // "creating", "active", "draining", "archived"
    pub storage_size_gb: u32,           // PVC size
    pub created_by: u64,               // → users
    pub created_at: u64,
}
```

**What the K8s operator creates for a workspace:**

- PersistentVolumeClaim (ReadWriteMany) — the shared filesystem
- NetworkPolicy — egress rules for agents in this workspace
- Warm pod pool — pre-booted runtimes with the PVC mounted

**Workspace filesystem layout:**

```text
/workspace/
    ├── /repos/                ← cloned once, persists across agents
    ├── /cache/                ← build caches (target/, node_modules/)
    ├── /shared/               ← agent outputs for other agents
    └── /scratch/agent-{id}/   ← per-agent temp space, cleaned on agent stop
```

**Isolation model:**

| Boundary | Within workspace | Between workspaces |
|----------|-----------------|-------------------|
| Filesystem | Shared (by design) | Fully isolated (different PVC) |
| Processes | Isolated (separate pods) | Isolated |
| Network | Same egress rules | Different network policies |
| Credentials | Per-task (K8s secrets per pod) | Per-workspace |
| SpacetimeDB state | Agents see each other's observations | Scoped by workspace |

Agents within a workspace are collaborators. Workspaces are isolation boundaries.

#### WorkspacePool

Configuration for the warm pod pool per workspace. Defines which images to keep warm and how many.

```rust
pub struct WorkspacePool {
    #[primary_key]
    pub id: u64,
    pub workspace_id: u64,             // → workspaces
    pub image_tag: String,             // "aule-builder:v1"
    pub warm_count: u32,               // how many warm pods to maintain
    pub resources_cpu: String,         // "500m", "1000m"
    pub resources_memory: String,      // "512Mi", "2Gi"
}
```

### Work

#### Task

The unit of work. Describes what needs to happen. Doesn't prescribe how.

```rust
pub struct Task {
    #[primary_key]
    pub id: u64,
    pub workspace_id: u64,             // → workspaces
    pub parent_task_id: Option<u64>,   // → tasks (for decomposition)
    pub title: String,                 // short label
    pub description: String,           // full natural language description
    pub trust_level: String,           // "autonomous", "supervised", "approval_required"
    pub priority: String,              // "low", "normal", "high", "urgent"
    pub status: String,                // see lifecycle below
    pub max_attempts: u32,             // retry limit (default: 3)
    pub budget_cents: Option<i64>,     // max spend for this task
    pub created_by: u64,               // → users
    pub created_at: u64,
    pub completed_at: Option<u64>,
}
```

**Task status lifecycle:**

```text
created → queued → assigned → running → completed
                                      → failed (all attempts exhausted or terminal)
                                      → cancelled (by human)
```

| Status | Meaning |
|--------|---------|
| `created` | Just made, may be missing context or waiting on dependencies |
| `queued` | Ready for an agent, waiting for assignment |
| `assigned` | Matched to an agent, attempt about to start |
| `running` | Active attempt in progress |
| `completed` | At least one attempt succeeded |
| `failed` | All attempts exhausted, or terminal failure |
| `cancelled` | Human killed it |

#### TaskAttempt

A single try at completing a task. A task can have multiple attempts. When attempt N fails, attempt N+1 receives all previous observations and failure reasons as context.

```rust
pub struct TaskAttempt {
    #[primary_key]
    pub id: u64,
    pub task_id: u64,                  // → tasks
    pub attempt_number: u32,           // 1, 2, 3...
    pub agent_id: u64,                 // → agents
    pub status: String,                // "running", "completed", "failed", "cancelled"
    pub failure_reason: Option<String>,// why it failed (context for next attempt)
    pub tokens_used: u64,
    pub cost_cents: i64,
    pub started_at: u64,
    pub ended_at: Option<u64>,
}
```

#### TaskPlan

The agent's todo list for an attempt. Created at the start of work, updated in real-time. The user sees this as a live progress view in the dashboard.

```rust
pub struct TaskPlan {
    #[primary_key]
    pub id: u64,
    pub attempt_id: u64,              // → task_attempts
    pub created_at: u64,
    pub updated_at: u64,
}
```

#### TaskStep

An individual step in a plan. Updated by the agent as it works. Steps can be added mid-execution when the agent discovers additional work.

```rust
pub struct TaskStep {
    #[primary_key]
    pub id: u64,
    pub plan_id: u64,                  // → task_plans
    pub sort_order: u32,               // ordering within the plan
    pub title: String,                 // "Implement scheduled reducer"
    pub description: Option<String>,   // more detail if needed
    pub status: String,                // "pending", "in_progress", "completed", "failed", "skipped"
    pub added_during_execution: bool,  // true if agent added this step mid-work
    pub started_at: Option<u64>,
    pub completed_at: Option<u64>,
}
```

**Step status values:**

| Status | Meaning |
|--------|---------|
| `pending` | Not started yet |
| `in_progress` | Agent is currently working on this |
| `completed` | Done successfully |
| `failed` | Step failed (agent may retry, replan, or fail the attempt) |
| `skipped` | Agent determined this step is unnecessary |

#### TaskDependency

Relationships between tasks.

```rust
pub struct TaskDependency {
    #[primary_key]
    pub id: u64,
    pub task_id: u64,                  // this task...
    pub depends_on_task_id: u64,       // ...depends on this task
    pub dependency_type: String,       // "blocks", "needs_output", "informs"
}
```

| Type | Meaning |
|------|---------|
| `blocks` | Hard dependency — can't start until the other completes |
| `needs_output` | Needs the result/artifacts from the other task as input |
| `informs` | Soft — the other task's observations are useful context |

#### TaskContext

Input materials attached to a task. What the agent receives alongside the description.

```rust
pub struct TaskContext {
    #[primary_key]
    pub id: u64,
    pub task_id: u64,                  // → tasks
    pub context_type: String,          // "text", "file", "url", "task_ref", "observation_ref"
    pub content: String,               // the text, path, URL, or reference ID
    pub label: Option<String>,         // human-readable label
    pub added_by: u64,                 // → users or system
    pub added_at: u64,
}
```

#### TaskResult

The structured output of a successful attempt.

```rust
pub struct TaskResult {
    #[primary_key]
    pub id: u64,
    pub task_id: u64,                  // → tasks
    pub attempt_id: u64,              // → task_attempts
    pub summary: String,               // headline answer
    pub detail: Option<String>,        // full explanation
    pub result_type: String,           // "answer", "artifact", "recommendation", "decision_needed"
    pub artifacts: Option<String>,     // JSON: file paths, repo refs, generated documents
    pub follow_up_suggestions: Option<String>, // JSON: what agent thinks should happen next
    pub created_at: u64,
}
```

**Result types:**

| Type | Meaning | UI behavior |
|------|---------|-------------|
| `answer` | Definitive response | Display directly |
| `artifact` | Produced files/code | Link to workspace filesystem |
| `recommendation` | Options with tradeoffs | Present for user decision |
| `decision_needed` | Agent hit a fork it can't resolve | Prompt user for input |

#### TaskReview

Human feedback on a result. Closes the loop.

```rust
pub struct TaskReview {
    #[primary_key]
    pub id: u64,
    pub task_id: u64,                  // → tasks
    pub result_id: u64,               // → task_results
    pub verdict: String,               // "approved", "needs_revision", "rejected"
    pub notes: Option<String>,         // feedback for revision
    pub reviewed_by: u64,             // → users
    pub reviewed_at: u64,
}
```

If `needs_revision`, a new attempt is triggered with the review notes as additional context. The agent sees the previous result AND the feedback.

### Observations

What agents produce during work. The primary communication channel between agents and humans.

```rust
pub struct Observation {
    #[primary_key]
    pub id: u64,
    pub task_id: u64,                  // → tasks
    pub attempt_id: u64,              // → task_attempts
    pub agent_id: u64,                 // → agents
    pub workspace_id: u64,            // → workspaces (denormalized for subscription queries)
    pub observation_type: String,      // "finding", "progress", "error", "result", "question"
    pub content: String,               // the observation content
    pub acknowledged_by: Option<u64>,  // → users (when human has seen/dismissed it)
    pub created_at: u64,
}
```

**Observation types:**

| Type | Meaning |
|------|---------|
| `finding` | Something the agent discovered during work |
| `progress` | Status update on what's happening |
| `error` | Something went wrong (may or may not be fatal) |
| `result` | An intermediate result (distinct from TaskResult which is the final output) |
| `question` | Agent needs clarification from the human |

Observations are visible to all agents in the same workspace via SpacetimeDB subscriptions. This is how agents coordinate — they see each other's findings without direct messaging.

### Agent Memory

Persistent knowledge that survives across tasks and agent instances.

```rust
pub struct AgentMemory {
    #[primary_key]
    pub id: u64,
    pub workspace_id: u64,            // → workspaces
    pub agent_type_id: Option<u64>,   // → agent_types (type-scoped memory, or null for workspace-wide)
    pub layer: String,                 // "working", "short_term", "long_term"
    pub content: String,
    pub decay_rate: f64,               // 0.0 = permanent, 1.0 = highly ephemeral
    pub created_at: u64,
    pub last_accessed_at: u64,
}
```

**Memory layers:**

| Layer | Purpose | Decay |
|-------|---------|-------|
| `working` | Current task context | Ephemeral, cleared on task completion |
| `short_term` | Recent task findings | Decays over time via scheduled reducer |
| `long_term` | Explicitly promoted knowledge | Permanent or very slow decay |

Memory can be scoped to an agent type (only builder agents see builder memory) or workspace-wide (all agents see it).

### Budget

Resource allocation and tracking.

```rust
pub struct BudgetAllocation {
    #[primary_key]
    pub id: u64,
    pub workspace_id: Option<u64>,     // → workspaces (workspace-level budget)
    pub agent_type_id: Option<u64>,    // → agent_types (type-level budget)
    pub task_id: Option<u64>,          // → tasks (task-level budget)
    pub allocated_cents: i64,
    pub consumed_cents: i64,
    pub cycle: String,                 // "daily", "weekly", "monthly", "task" (one-time)
    pub replenished_at: u64,
}

pub struct BudgetUsage {
    #[primary_key]
    pub id: u64,
    pub allocation_id: u64,            // → budget_allocations
    pub agent_id: u64,                 // → agents
    pub task_id: u64,                  // → tasks
    pub cost_cents: i64,
    pub tokens: u64,
    pub provider: String,
    pub timestamp: u64,
}
```

### Events

Insert-only event log. Agents and the system publish events. Other agents and humans subscribe to event types.

```rust
pub struct Event {
    #[primary_key]
    pub id: u64,
    pub workspace_id: u64,            // → workspaces
    pub event_type: String,           // "task.created", "agent.launched", "observation.posted", etc.
    pub source_agent_id: Option<u64>, // → agents (null for system/user events)
    pub payload: String,              // JSON event data
    pub timestamp: u64,
}
```

### Approvals

Human-in-the-loop gates for high-stakes actions.

```rust
pub struct Approval {
    #[primary_key]
    pub id: u64,
    pub task_id: u64,                 // → tasks
    pub agent_id: u64,                // → agents (who's asking)
    pub action_type: String,          // what the agent wants to do
    pub payload: String,              // JSON: details of the proposed action
    pub reasoning: String,            // why the agent wants to do this
    pub status: String,               // "pending", "approved", "rejected", "expired"
    pub reviewed_by: Option<u64>,     // → users
    pub reviewed_at: Option<u64>,
    pub created_at: u64,
    pub expires_at: u64,
}
```

### Messages

Direct communication between users and agents.

```rust
pub struct Message {
    #[primary_key]
    pub id: u64,
    pub task_id: u64,                  // → tasks
    pub sender_type: String,           // "user", "agent"
    pub sender_id: u64,                // → users or → agents
    pub content: String,
    pub created_at: u64,
}
```

### Tools

Registry of available tools and capabilities.

```rust
pub struct Tool {
    #[primary_key]
    pub id: u64,
    pub name: String,
    pub tool_type: String,             // "cli", "api", "mcp", "agent"
    pub capability_tags: String,       // JSON array
    pub description: String,
    pub input_schema: Option<String>,  // JSON schema
    pub endpoint: Option<String>,      // for API/MCP tools
    pub reliability_score: f64,        // tracked over time
    pub workspace_id: Option<u64>,     // → workspaces (null = global)
    pub registered_at: u64,
}
```

### Provenance

Audit trail of agent reasoning and actions.

```rust
pub struct ProvenanceNode {
    #[primary_key]
    pub id: u64,
    pub task_id: u64,                  // → tasks
    pub attempt_id: u64,              // → task_attempts
    pub agent_id: u64,                 // → agents
    pub node_type: String,             // "llm_call", "tool_use", "observation", "memory_read", "human_input"
    pub content_hash: String,          // hash of the content for deduplication
    pub summary: String,               // human-readable summary
    pub timestamp: u64,
}

pub struct ProvenanceEdge {
    #[primary_key]
    pub id: u64,
    pub from_node_id: u64,            // → provenance_nodes
    pub to_node_id: u64,              // → provenance_nodes
    pub relation: String,              // "informed_by", "produced", "triggered", "used"
}
```

### LLM Configuration

Router configuration and telemetry, all in SpacetimeDB.

```rust
pub struct LlmProvider {
    #[primary_key]
    pub id: u64,
    pub name: String,                  // "anthropic-opus", "google-gemini-pro"
    pub provider_type: String,         // "anthropic", "google", "openai", "local"
    pub endpoint: String,
    pub model_id: String,              // "claude-sonnet-4-20250514"
    pub context_window: u64,           // max tokens
    pub cost_per_input_token: f64,
    pub cost_per_output_token: f64,
    pub enabled: bool,
}

pub struct RoutingRule {
    #[primary_key]
    pub id: u64,
    pub priority: u32,                 // lower = higher priority
    pub task_type: String,             // "reason", "code", "extract", "summarize", etc.
    pub min_quality: String,           // "high", "medium", "low"
    pub provider_id: u64,             // → llm_providers
    pub fallback_provider_id: Option<u64>, // → llm_providers
}

pub struct ProviderHealth {
    #[primary_key]
    pub id: u64,
    pub provider_id: u64,             // → llm_providers
    pub status: String,                // "healthy", "degraded", "down"
    pub avg_latency_ms: f64,
    pub error_rate: f64,
    pub last_updated: u64,
}

pub struct RateLimit {
    #[primary_key]
    pub id: u64,
    pub provider_id: u64,             // → llm_providers
    pub requests_per_minute: u32,
    pub tokens_per_minute: u64,
    pub current_rpm: u32,
    pub current_tpm: u64,
    pub last_updated: u64,
}

pub struct LlmRequest {
    #[primary_key]
    pub id: u64,
    pub agent_id: u64,                // → agents
    pub task_id: u64,                 // → tasks
    pub provider_id: u64,            // → llm_providers
    pub task_type: String,
    pub input_tokens: u64,
    pub output_tokens: u64,
    pub latency_ms: u64,
    pub cost_cents: i64,
    pub success: bool,
    pub timestamp: u64,
}
```

The router subscribes to `LlmProvider`, `RoutingRule`, `RateLimit`. Config changes propagate instantly. The router writes `LlmRequest` entries back via reducer calls. Scheduled reducers aggregate requests into `ProviderHealth`. Closed feedback loop.

---

## Relationship Map

```text
User
 ├── creates → Workspace
 ├── creates → Task
 ├── creates → AgentType → has many → AgentTypeVersion
 ├── launches → Agent (on a runtime in a workspace)
 ├── reviews → TaskResult (via TaskReview)
 ├── approves/rejects → Approval
 ├── acknowledges → Observation
 └── sends → Message

Workspace
 ├── has → PersistentVolume (shared filesystem, managed by operator)
 ├── has → WorkspacePool[] (warm pod configuration)
 ├── has → AgentRuntime[] (warm pods, managed by operator)
 ├── has → Agent[] (active workers)
 ├── has → Task[] (work items)
 ├── has → Observation[] (visible to all agents in workspace)
 ├── has → AgentMemory[] (shared and type-scoped)
 ├── has → BudgetAllocation[]
 ├── has → Event[]
 └── has → Tool[] (workspace-scoped tools)

AgentRuntime (warm pod, managed by operator)
 ├── belongs to → Workspace
 ├── has capabilities from → container image
 ├── hosts → Agent (one at a time)
 └── mounts → workspace PVC

Agent (launched on a runtime)
 ├── running on → AgentRuntime
 ├── instance of → AgentTypeVersion
 ├── belongs to → Workspace
 ├── working on → Task (optional)
 ├── posts → Observation[]
 ├── posts → Event[]
 ├── requests → Approval[]
 ├── reads/writes → AgentMemory
 └── calls → LLM Router → LlmProvider

Task
 ├── belongs to → Workspace
 ├── has parent → Task (optional, decomposition)
 ├── has children → Task[] (subtasks)
 ├── has → TaskDependency[] (blocks, needs_output, informs)
 ├── has → TaskContext[] (input materials)
 ├── has → TaskAttempt[]
 │    ├── has → TaskPlan
 │    │    └── has → TaskStep[] (the live todo list)
 │    ├── has → Observation[] (findings during this attempt)
 │    └── produces → TaskResult (on success)
 ├── has → TaskReview (human feedback)
 ├── has → Approval[] (when agent needs permission)
 ├── has → Message[] (human-agent conversation)
 └── has → BudgetAllocation (task-level budget)
```

---

## Scheduled Reducers

| Reducer | Frequency | Purpose |
|---------|-----------|---------|
| `decay_memory` | Every 10 min | Erode short-term memory based on decay rates |
| `replenish_budgets` | Daily / per-cycle | Reset or top up budgets |
| `update_provider_health` | Every 30 sec | Aggregate LLM requests into health metrics |
| `check_rate_limits` | Every 10 sec | Update current RPM/TPM |
| `detect_anomalies` | Every 1 min | Error rate spikes, budget burn anomalies |
| `expire_approvals` | Every 5 min | Auto-escalate or auto-reject stale approvals |
| `cleanup_events` | Every 1 hour | TTL on old events |
| `check_runtime_health` | Every 1 min | Flag runtimes with stale heartbeats |
| `expire_tasks` | Every 1 min | Cancel tasks past their expiry time |

---

## K8s Operator

The operator bridges SpacetimeDB state and K8s resources. Written in Rust using `kube-rs`. Subscribes to SpacetimeDB tables and reconciles K8s resources to match.

### Custom Resource Definitions

```yaml
apiVersion: aule.io/v1
kind: AuleWorkspace
metadata:
  name: aule-dev
spec:
  storage:
    size: 50Gi
    storageClass: fast-rwx
  pool:
    - image: aule-builder:v1
      warm: 2
      resources:
        cpu: "1000m"
        memory: "2Gi"
    - image: aule-research:v1
      warm: 1
      resources:
        cpu: "500m"
        memory: "1Gi"
  network:
    allowedEgress:
      - github.com
      - api.anthropic.com
      - crates.io
```

### Reconciliation

```text
SpacetimeDB state change            → Operator action
──────────────────────────────────────────────────────
Workspace created                    → Create PVC, NetworkPolicy, warm pods
Workspace archived                   → Drain agents, delete pods, retain PVC
Agent launch requested               → Find warm pod, mount creds, configure
Agent stopped                        → Clean scratch dir, revoke creds, return pod to pool
Warm pool below threshold            → Create new warm pod
Runtime heartbeat stale              → Restart pod, re-register
Workspace pool config changed        → Scale warm pods up/down
Workspace network rules changed      → Update NetworkPolicy
```

**Source of truth:** SpacetimeDB. The dashboard and API create workspaces in SpacetimeDB. The operator watches SpacetimeDB and creates K8s resources to match. CRDs are owned by the operator, not applied by humans directly.

---

## LLM Router

A native Rust HTTP service (axum). Routes LLM requests from agents to the best provider.

### Standard API

```text
POST /v1/completion
{
    "task_type": "reason" | "generate" | "extract" | "classify" | "code" | "summarize" | "embed",
    "messages": [...],
    "constraints": {
        "max_tokens": 4096,
        "max_latency_ms": 5000,
        "max_cost_cents": 10,
        "min_quality": "high" | "medium" | "low",
        "structured_output": { "format": "json", "schema": {...} }
    },
    "context": {
        "agent_id": "...",
        "task_id": "...",
        "workspace_id": "...",
        "budget_remaining_cents": 450,
        "priority": "normal"
    },
    "tools": [...]
}
```

### Capabilities

- Task-based routing (reason→Opus, extract→Haiku, code→Sonnet)
- Budget-aware degradation (low budget → cheaper model)
- Latency-aware routing with fallback chains
- Context-window routing (large context → model that supports it)
- Exact-match and semantic caching
- Tool format normalization across providers
- Provider health tracking with auto-disable on failure spikes
- Config from SpacetimeDB subscriptions (no restarts for config changes)
- Telemetry written back to SpacetimeDB (closed feedback loop)
- Secrets from K8s Secrets / Vault (never in SpacetimeDB)

---

## Agent Runtime

### Platform Tools

The agent has exactly 6 tools defined in its LLM context (~1,800 tokens total):

| Tool | Purpose |
|------|---------|
| `aule_observe` | Post an observation to SpacetimeDB |
| `aule_suggest` | Propose an action for approval |
| `aule_memory` | Read/write agent memory layers |
| `aule_status` | Report progress, update plan steps |
| `aule_request` | Request LLM completion via router |
| `shell` | Execute any shell command |

Everything else goes through `shell`. The agent discovers available CLIs by exploring its environment (`which`, `ls /usr/local/bin/`, `--help`).

### Standard CLIs (base image)

File ops: `ls`, `cat`, `find`, `tree`, `cp`, `mv`, `mkdir`, `diff`, `tar`
Text: `grep`, `rg`, `sed`, `awk`, `jq`, `yq`, `xsv`
Network: `curl`, `wget`
System: `env`, `which`, `date`, `timeout`, `tee`
Data: `sqlite3`, `duckdb`

### Capability-specific CLIs (per image)

| Image | Additional CLIs |
|-------|----------------|
| `aule-builder` | `git`, `cargo`, `rustc`, `rustfmt`, `clippy`, `spacetime`, `make`, `node`, `npm` |
| `aule-research` | `playwright`, `trafilatura`, `python3`, `pandoc` |
| `aule-data` | `python3`, `duckdb`, `pandas`, `pyarrow` |
| `aule-ops` | `spacetime`, `git`, `kubectl`, `terraform` |

### Shell Safety

- All commands wrapped in `timeout`
- Working directory confined to `/workspace/`
- Output truncated at configurable limit (default 50KB)
- Every invocation logged (command, exit code, output size, duration)
- Destructive command blocklist (pattern-matched)
- Container-level isolation (non-root, no host mounts, resource limits)

### Agent Process Loop

```text
On launch:
  1. Receive agent_type_version_id from operator
  2. Connect to SpacetimeDB
  3. Fetch system prompt from agent_type_versions table
  4. Register as Agent (status: "idle")
  5. Subscribe to task assignments

On task assignment:
  1. Receive task with description, context, budget
  2. Load relevant memory from workspace
  3. If retrying: load previous attempts' observations and failure reasons
  4. Plan: LLM call → produce TaskPlan with TaskSteps
  5. Execute steps:
     a. Update current step status → "in_progress"
     b. Reason (LLM call via router)
     c. Act (shell commands, tool use)
     d. Observe (post observations to SpacetimeDB)
     e. Update step status → "completed" or "failed"
     f. May add new steps to plan during execution
  6. On completion: post TaskResult, attempt status → "completed"
  7. On failure: record failure_reason, attempt status → "failed"
  8. Return to idle, wait for next task

On stop:
  1. Finish or abort current work
  2. Agent status → "stopped"
  3. Operator cleans scratch dir, revokes credentials
  4. Runtime status → "available"
```

---

## Project Structure

```text
aule/
├── packages/                              ← Rust workspace crates
│   ├── aule-core/                         ← shared types across all components
│   │   └── src/
│   │       ├── agent.rs                   ← Agent, AgentRuntime, AgentType, AgentTypeVersion
│   │       ├── task.rs                    ← Task, TaskAttempt, TaskPlan, TaskStep, etc.
│   │       ├── workspace.rs              ← Workspace, WorkspacePool
│   │       ├── observation.rs            ← Observation
│   │       ├── memory.rs                 ← AgentMemory, MemoryLayer
│   │       ├── budget.rs                 ← BudgetAllocation, BudgetUsage
│   │       ├── routing.rs                ← CompletionRequest, CompletionResponse, TaskType
│   │       ├── provenance.rs             ← ProvenanceNode, ProvenanceEdge
│   │       ├── approval.rs              ← Approval
│   │       ├── event.rs                 ← Event
│   │       └── tool.rs                  ← Tool
│   │
│   ├── aule-spacetimedb/                  ← coordination layer (compiles to WASM)
│   │   └── src/
│   │       ├── tables.rs                  ← all SpacetimeDB table definitions
│   │       ├── reducers/
│   │       │   ├── identity.rs            ← user reg, runtime reg, agent launch/stop
│   │       │   ├── task.rs                ← task lifecycle, attempt, plan, steps, review
│   │       │   ├── observation.rs         ← post, acknowledge
│   │       │   ├── memory.rs             ← read, write, promote, prune
│   │       │   ├── budget.rs             ← allocate, consume, check
│   │       │   ├── provenance.rs         ← record nodes, link edges
│   │       │   ├── events.rs             ← publish events
│   │       │   ├── approvals.rs          ← request, approve, reject
│   │       │   ├── tools.rs              ← register, deregister, health
│   │       │   ├── versioning.rs         ← agent type CRUD, version lifecycle
│   │       │   ├── workspace.rs          ← workspace CRUD, pool config
│   │       │   └── llm_config.rs         ← provider CRUD, routing rules, log requests
│   │       ├── scheduled.rs              ← all scheduled reducers
│   │       └── lib.rs
│   │
│   ├── aule-router/                       ← LLM routing service (native Rust binary)
│   │   └── src/
│   │       ├── server.rs                  ← axum HTTP server
│   │       ├── router.rs                 ← routing logic, rule engine
│   │       ├── providers/
│   │       │   ├── anthropic.rs
│   │       │   ├── google.rs
│   │       │   ├── openai.rs
│   │       │   └── local.rs              ← vLLM / Ollama
│   │       ├── cache.rs                  ← exact match + semantic caching
│   │       ├── budget.rs                 ← budget checking via SpacetimeDB
│   │       ├── spacetimedb_client.rs     ← subscription management, config sync
│   │       └── telemetry.rs             ← request logging, metrics
│   │
│   ├── aule-runtime/                      ← agent process (native Rust binary, in pods)
│   │   └── src/
│   │       ├── main.rs                   ← runtime lifecycle: boot, register, idle loop
│   │       ├── agent.rs                  ← agent launch, system prompt fetch, config
│   │       ├── task.rs                   ← task pickup, attempt loop, plan management
│   │       ├── executor.rs              ← shell command execution with safety
│   │       ├── llm_client.rs            ← talks to router's standard API
│   │       ├── spacetimedb_client.rs    ← subscriptions, state updates
│   │       ├── reasoning.rs             ← prompt construction, tool marshaling
│   │       └── tools/
│   │           ├── observe.rs
│   │           ├── suggest.rs
│   │           ├── memory.rs
│   │           ├── status.rs
│   │           ├── request.rs
│   │           └── shell.rs
│   │
│   └── aule-operator/                     ← K8s operator (Rust, kube-rs)
│       └── src/
│           ├── main.rs                    ← operator entrypoint
│           ├── controllers/
│           │   ├── workspace.rs           ← reconcile workspaces (PVC, NetworkPolicy, pool)
│           │   ├── runtime.rs            ← reconcile warm pod pool
│           │   └── agent.rs              ← reconcile agent launch/stop
│           ├── crds/
│           │   ├── workspace.rs          ← AuleWorkspace CRD definition
│           │   ├── runtime.rs            ← AuleRuntime CRD definition
│           │   └── agent.rs              ← AuleAgent CRD definition
│           └── spacetimedb.rs            ← subscription bridge to SpacetimeDB
│
├── app/                                   ← frontend application (Bun/TypeScript)
│   └── ...                               ← dashboard, task management, observation feed
│
├── docs/                                  ← documentation
│   └── ...
│
└── docker/                                ← agent container images
    ├── Dockerfile.base                    ← core utils, shared tools
    ├── Dockerfile.builder                 ← + cargo, git, spacetime CLI
    ├── Dockerfile.research               ← + playwright, trafilatura
    ├── Dockerfile.data                   ← + duckdb, python, pandas
    └── Dockerfile.ops                    ← + spacetime, kubectl, terraform
```

---

## Key Design Patterns

### Stigmergic Coordination

Agents coordinate through shared state (SpacetimeDB tables), not direct messaging. Agent A writes an observation → Agent B's subscription fires → Agent B reacts. Like ants communicating through pheromones.

### Transactional Everything

Every agent action is a reducer call = atomic transaction. Either fully succeeds or fully fails. No partial states, no race conditions.

### Real-time Subscriptions

Agents, the operator, the router, and humans subscribe to SQL queries over SpacetimeDB. State changes push instantly to all subscribers. No polling.

### Provenance as First-Class

Every output traceable to inputs, LLM calls, tool executions, and other agents' outputs. Provenance graph stored in SpacetimeDB, queryable and visualizable.

### SpacetimeDB as Source of Truth

All platform state lives in SpacetimeDB. The K8s operator, LLM router, and frontend are all consumers/actuators. They subscribe to state and reconcile their domain (K8s resources, routing decisions, UI) to match.

### Learn From Failure

When a task attempt fails, the next attempt receives all previous observations and failure reasons. Agents improve across retries without human re-explanation.
