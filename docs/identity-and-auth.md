# Identity and Auth

Three identity levels with distinct scopes and lifetimes.

## Identity Levels

### User

A human interacting through a frontend app. Persistent identity. Can view dashboards, approve/reject agent proposals, intervene in agent work, and manage configurations. Doesn't execute tasks -- directs and supervises.

### Agent Runtime

A deployed K8s pod. It exists whether or not it's working on a task. Registers itself in SpacetimeDB on startup and maintains a heartbeat. Has capabilities determined by its image (what CLIs and tools are available) and its agent type version (what system prompt and config it runs).

Think of it as an employee at their desk -- present, skilled, but not yet assigned work.

### Agent Task

When a runtime picks up a task, a task identity is created. This is the scoped, temporary, permissioned identity. It grants access to specific projects, credentials, external systems, and budgets -- whatever the task requires. When the task completes, the task identity is revoked. The runtime returns to idle.

**Permissions live on the task, not the runtime.** The same builder runtime can work on the factory-twin project (accessing production sensor data) or the supply-chain project (accessing supplier APIs). What changes is the task assignment and its permissions, not the runtime itself.

## Auth Flow

```
1. Runtime pod starts
2. Connects to SpacetimeDB
3. Calls register_runtime reducer -> gets an AgentRuntime row
4. Subscribes to task assignments for itself
5. Task appears -> runtime calls start_task reducer
6. start_task checks: does this runtime's type have required capabilities?
7. If yes: task status -> "running", runtime status -> "working"
8. During task: every reducer call validated against task permissions
9. Task completes -> complete_task reducer -> runtime status -> "idle"
10. Credentials are ephemeral -- mounted for the task, removed after
```

## Tables

### User

```rust
#[spacetimedb::table(name = users, public)]
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
#[spacetimedb::table(name = agent_runtimes, public)]
pub struct AgentRuntime {
    #[primary_key]
    pub id: u64,
    pub identity: Identity,        // SpacetimeDB connection identity
    pub agent_type_id: u64,        // -> agent_types
    pub agent_version_id: u64,     // -> agent_type_versions
    pub instance_name: String,     // "builder-01"
    pub status: String,            // "starting", "idle", "working", "draining", "offline"
    pub current_task_id: Option<u64>,
    pub registered_at: u64,
    pub last_heartbeat: u64,
}
```

### AgentTask

```rust
#[spacetimedb::table(name = agent_tasks, public)]
pub struct AgentTask {
    #[primary_key]
    pub id: u64,
    pub runtime_id: u64,           // -> agent_runtimes
    pub project_id: String,
    pub task_description: String,
    pub permissions: String,       // JSON: what this task is allowed to do
    pub credential_refs: String,   // JSON: K8s secret names to mount
    pub budget_cents: f64,
    pub budget_spent: f64,
    pub status: String,            // "assigned", "running", "completed", "failed", "cancelled"
    pub assigned_at: u64,
    pub expires_at: u64,
    pub completed_at: Option<u64>,
}
```

## Permission Checking

Every reducer validates the caller's identity against task permissions:

```rust
#[spacetimedb::reducer]
pub fn post_observation(ctx: &ReducerContext, content: String, project_id: String) {
    let runtime = get_runtime_by_identity(ctx, ctx.sender)?;
    let task = get_active_task(ctx, runtime.id)?;

    // Task must be for this project
    assert!(task.project_id == project_id);

    // Task must have observe permission
    assert!(task_has_permission(&task, "observe"));

    // Proceed...
}
```

The key insight: the runtime is a generic worker. The task is the permissioned context. This means the same pod can serve different projects with different access levels, scoped entirely by task assignment.
