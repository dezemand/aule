# Agent Versioning

Two concepts: **Agent Types** (what kind of agent) and **Agent Type Versions** (a specific release of that kind).

## Agent Types

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

The `required_capabilities` are checked against the runtime's `capabilities` at agent launch time. A runtime must have all required capabilities to host an agent of this type.

## Agent Type Versions

A specific release of an agent type. Contains the image tag, system prompt, configuration, and behavioral parameters. This is what defines agent behavior and what actually gets deployed.

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

## Version Lifecycle

```
draft → testing → active → deprecated → retired
```

| Status | Meaning |
|--------|---------|
| `draft` | Being developed, not deployable |
| `testing` | Can deploy to test runtimes, not production |
| `active` | Production-ready, new agents use this version |
| `deprecated` | Still works, but new agents should use a newer version |
| `retired` | No longer deployable, existing agents flagged for upgrade |

## Upgrade Strategies

### Rolling

Drain agents on old version one by one (finish current task, stop), launch new agents on available runtimes with the new version.

### Canary

Launch 1 agent on new version alongside existing. Route some tasks to it. Compare performance via SpacetimeDB metrics.

### A/B Testing

Both versions active simultaneously. Track success rates, cost efficiency, approval rates per version. Data-driven decision.

All version metadata in SpacetimeDB means the dashboard shows exactly what's running, which versions are in play, and how each performs.

## System Prompt as Versioned Data

The system prompt is the agent's behavior. Storing it in SpacetimeDB means:

- Diff prompts between versions
- Trace behavioral changes to specific edits
- Runtime fetches the prompt from SpacetimeDB on agent launch — prompt-only changes don't require pod restarts
- A/B test prompts within the same image version
