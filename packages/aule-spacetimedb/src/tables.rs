//! All SpacetimeDB table definitions for Aule Phase 1.
//!
//! Tables are organized by domain:
//! - Identity: agent_runtimes, agent_tasks
//! - Agent Types: agent_types, agent_type_versions
//! - Observations: observations

use spacetimedb::{table, Identity, SpacetimeType, Timestamp};

// ---------------------------------------------------------------------------
// Identity domain
// ---------------------------------------------------------------------------

/// A connected agent runtime. Runtimes register when they come online
/// and deregister (or get marked offline) when they disconnect.
#[table(accessor = agent_runtime, public)]
pub struct AgentRuntime {
    #[primary_key]
    pub identity: Identity,
    /// Human-readable name for this runtime instance (e.g. "builder-01").
    #[unique]
    pub name: String,
    /// The agent type this runtime is configured to run.
    pub agent_type_id: u64,
    pub status: RuntimeStatus,
    pub last_heartbeat: Timestamp,
    pub registered_at: Timestamp,
}

#[derive(SpacetimeType, Clone, Debug, PartialEq)]
pub enum RuntimeStatus {
    /// Online and ready to accept tasks.
    Idle,
    /// Currently executing a task.
    Busy,
    /// Gracefully shutting down, will not accept new tasks.
    Draining,
    /// Disconnected or unresponsive.
    Offline,
}

/// A task that can be assigned to an agent runtime.
#[table(accessor = agent_task, public)]
pub struct AgentTask {
    #[auto_inc]
    #[primary_key]
    pub id: u64,
    /// The agent type required to handle this task.
    pub agent_type_id: u64,
    /// Which runtime is assigned to this task (if any).
    pub assigned_runtime: Option<Identity>,
    pub status: TaskStatus,
    /// Short human-readable description.
    pub title: String,
    /// Full task description / instructions for the agent.
    pub description: String,
    /// Who created the task.
    pub created_by: Identity,
    pub created_at: Timestamp,
    pub started_at: Option<Timestamp>,
    pub completed_at: Option<Timestamp>,
    /// Result summary on completion, or error message on failure.
    pub result: Option<String>,
}

#[derive(SpacetimeType, Clone, Debug, PartialEq)]
pub enum TaskStatus {
    /// Created but not yet assigned to a runtime.
    Pending,
    /// Assigned to a runtime, waiting for it to start.
    Assigned,
    /// Runtime is actively working on it.
    Running,
    /// Completed successfully.
    Completed,
    /// Failed with an error.
    Failed,
    /// Cancelled before completion.
    Cancelled,
}

// ---------------------------------------------------------------------------
// Agent type domain
// ---------------------------------------------------------------------------

/// An agent type defines a category of agent (e.g. "builder", "researcher").
#[table(accessor = agent_type, public)]
pub struct AgentType {
    #[auto_inc]
    #[primary_key]
    pub id: u64,
    #[unique]
    pub name: String,
    pub description: String,
    pub created_by: Identity,
    pub created_at: Timestamp,
}

/// A versioned release of an agent type. The system prompt lives here.
#[table(accessor = agent_type_version, public)]
pub struct AgentTypeVersion {
    #[auto_inc]
    #[primary_key]
    pub id: u64,
    /// Which agent type this version belongs to.
    pub agent_type_id: u64,
    /// Semantic version string (e.g. "1.0.0").
    pub version: String,
    /// The system prompt that defines this agent's behavior.
    pub system_prompt: String,
    pub status: VersionStatus,
    pub created_by: Identity,
    pub created_at: Timestamp,
}

#[derive(SpacetimeType, Clone, Debug, PartialEq)]
pub enum VersionStatus {
    /// Being developed, not yet ready.
    Draft,
    /// Under testing.
    Testing,
    /// Live and available for use.
    Active,
    /// Superseded by a newer version, still functional.
    Deprecated,
    /// No longer usable.
    Retired,
}

// ---------------------------------------------------------------------------
// Observations domain
// ---------------------------------------------------------------------------

/// An observation posted by an agent runtime during task execution.
/// Humans subscribe to this table to see what agents are doing.
#[table(accessor = observation, public)]
pub struct Observation {
    #[auto_inc]
    #[primary_key]
    pub id: u64,
    /// The task this observation relates to.
    pub task_id: u64,
    /// The runtime that posted this observation.
    pub runtime_identity: Identity,
    pub kind: ObservationKind,
    /// The observation content.
    pub content: String,
    pub created_at: Timestamp,
}

#[derive(SpacetimeType, Clone, Debug, PartialEq)]
pub enum ObservationKind {
    /// General progress update.
    Progress,
    /// A finding or discovery.
    Finding,
    /// An error encountered during execution.
    Error,
    /// The final result of the task.
    Result,
}
