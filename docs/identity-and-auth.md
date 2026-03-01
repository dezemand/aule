# Identity & Auth

Four identity levels with distinct scopes and lifetimes.

## Identity Levels

### User

A human interacting through the frontend. Persistent identity. Can view dashboards, create tasks, approve/reject agent proposals, intervene in agent work, and manage configurations. Doesn't execute tasks — directs and supervises.

### AgentRuntime

A running K8s pod. The compute substrate. Registers itself in SpacetimeDB on startup and maintains a heartbeat. Has capabilities determined by its container image (what CLIs and programs are installed). Does NOT have an agent type — it's a generic machine that can host any agent whose requirements it satisfies.

Think of it as a workbench — equipped with tools, ready for someone to use it.

### Agent

An instance of an AgentTypeVersion, launched on a runtime. The actual worker. One agent per runtime at a time. Created when work needs to happen, stopped when done.

Think of it as the craftsman sitting at the workbench — they have a specific skill set (from their AgentTypeVersion) and work with whatever tools the workbench provides (from the runtime's capabilities).

### Task

When an agent picks up a task, task-scoped permissions and credentials activate. Credentials are mounted from K8s Secrets, scoped per task, and destroyed on completion.

**Permissions live on the task, not the agent or runtime.** The same builder agent can work on workspace A (accessing production sensor data) or workspace B (accessing supplier APIs). What changes is the task assignment and its permissions.

## Auth Flow

```text
1. Runtime pod starts
2. Connects to SpacetimeDB
3. Calls register_runtime reducer → gets an AgentRuntime row (status: "available")
4. Idles in warm pool, heartbeating
5. Agent launch requested for this runtime
6. Agent row created (status: "starting"), runtime status → "occupied"
7. Runtime fetches system prompt from agent_type_versions table
8. Agent status → "idle", subscribes to task assignments
9. Task arrives → agent starts attempt
10. During task: every reducer call validated against task permissions
11. Task completes → agent returns to idle or stops
12. On stop: scratch dir cleaned, credentials revoked, runtime → "available"
```

## Tables

### User

```rust
pub struct User {
    #[primary_key]
    pub id: u64,
    pub identity: Identity,        // SpacetimeDB connection identity
    pub name: String,
    pub role: String,              // "admin", "operator", "viewer"
    pub created_at: u64,
}
```

### AgentRuntime

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

### Agent

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

### Task

```rust
pub struct Task {
    #[primary_key]
    pub id: u64,
    pub workspace_id: u64,             // → workspaces
    pub parent_task_id: Option<u64>,   // → tasks (for decomposition)
    pub title: String,
    pub description: String,
    pub trust_level: String,           // "autonomous", "supervised", "approval_required"
    pub priority: String,              // "low", "normal", "high", "urgent"
    pub status: String,                // "created", "queued", "assigned", "running", "completed", "failed", "cancelled"
    pub max_attempts: u32,
    pub budget_cents: Option<i64>,
    pub created_by: u64,               // → users
    pub created_at: u64,
    pub completed_at: Option<u64>,
}
```

## Capability Check at Launch

When launching an agent, the system verifies the runtime can support the agent type:

```rust
fn can_run(runtime: &AgentRuntime, agent_type: &AgentType) -> bool {
    let runtime_caps: HashSet<String> = parse_json(&runtime.capabilities);
    let required_caps: HashSet<String> = parse_json(&agent_type.required_capabilities);
    required_caps.is_subset(&runtime_caps)
}
```

## Permission Checking

Every reducer validates the caller's identity against task permissions:

```rust
#[spacetimedb::reducer]
pub fn post_observation(ctx: &ReducerContext, content: String, workspace_id: u64) {
    let runtime = get_runtime_by_identity(ctx, ctx.sender())?;
    let agent = get_agent_for_runtime(ctx, runtime.id)?;
    let task = get_active_task(ctx, agent.current_task_id)?;

    // Agent must belong to this workspace
    if agent.workspace_id != workspace_id {
        return Err("workspace mismatch".into());
    }

    // Task must have observe permission
    if !task_has_permission(&task, "observe") {
        return Err("missing observe permission".into());
    }

    // Proceed...
}
```

The key insight: the runtime is a generic worker. The agent gives it a personality. The task gives it permissions. This means the same pod can serve different agent types across its lifetime. Runtimes are bound to a single workspace via their PVC mount — workspace isolation is enforced at the K8s level.
