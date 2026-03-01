//! All SpacetimeDB table definitions for Aule Phase 1.
//!
//! Tables are organized by domain:
//! - Identity: agent_runtimes, agent_tasks
//! - Agent Types: agent_types, agent_type_versions
//! - Runtime metadata: runtime_profiles, runtime_platform_info, runtime_resource_sample
//! - Observations: observations, runtime_events

use spacetimedb::{table, Identity, ScheduleAt, SpacetimeType, Timestamp};

use crate::reducers::{
    runtime_events::prune_runtime_events, runtime_metadata::prune_runtime_resource_samples,
};

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
    pub status: RuntimeStatus,
    pub last_heartbeat: Timestamp,
    pub registered_at: Timestamp,
}

/// Stable runtime metadata that changes infrequently.
#[table(accessor = runtime_profile, public)]
pub struct RuntimeProfile {
    #[primary_key]
    pub runtime_identity: Identity,
    /// Unique ID for this process instance.
    pub runtime_instance_id: String,
    pub runtime_name: String,
    pub runtime_version: String,
    pub git_sha: Option<String>,
    pub hostname: Option<String>,
    pub os: String,
    pub arch: String,
    pub started_at: Timestamp,
    pub updated_at: Timestamp,
}

/// Platform/deployment metadata for a runtime.
///
/// This is intentionally environment-agnostic: local and Docker runtimes can
/// populate generic fields while K8s runtimes additionally populate pod fields.
#[table(accessor = runtime_platform_info, public)]
pub struct RuntimePlatformInfo {
    #[primary_key]
    pub runtime_identity: Identity,
    pub environment: RuntimeEnvironment,
    pub process_id: Option<u32>,
    pub container_id: Option<String>,
    pub image: Option<String>,
    pub image_digest: Option<String>,
    pub cluster: Option<String>,
    pub namespace: Option<String>,
    pub pod_name: Option<String>,
    pub pod_uid: Option<String>,
    pub node_name: Option<String>,
    pub workload_kind: Option<String>,
    pub workload_name: Option<String>,
    pub container_name: Option<String>,
    pub restart_count: Option<u32>,
    pub updated_at: Timestamp,
}

/// High-frequency resource usage samples for a runtime.
///
/// Rows are pruned by the scheduled `prune_runtime_resource_samples` reducer.
#[table(accessor = runtime_resource_sample, public)]
pub struct RuntimeResourceSample {
    #[auto_inc]
    #[primary_key]
    pub id: u64,
    pub runtime_identity: Identity,
    pub sampled_at: Timestamp,
    pub cpu_millicores: Option<u32>,
    pub memory_rss_bytes: u64,
    pub memory_working_set_bytes: Option<u64>,
    pub threads: Option<u32>,
    pub open_fds: Option<u32>,
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

#[derive(SpacetimeType, Clone, Debug, PartialEq)]
pub enum RuntimeEnvironment {
    Local,
    Docker,
    K8S,
    Unknown,
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

/// Internal runtime logs for task execution.
///
/// Unlike observations, these are optimized for verbose streaming/debug output
/// and are intended for log viewers. Rows are pruned by the scheduled
/// `prune_runtime_events` reducer.
#[table(accessor = runtime_event, public)]
pub struct RuntimeEvent {
    #[primary_key]
    pub id: String,
    /// The task this event relates to.
    pub task_id: u64,
    /// The runtime that emitted this event.
    pub runtime_identity: Identity,
    /// Reasoning-loop turn number (1-based).
    pub turn: u32,
    pub event_type: RuntimeEventType,
    /// Event payload. For streamed events this content may grow over time.
    pub content: String,
    pub created_at: Timestamp,
    pub updated_at: Timestamp,
}

#[derive(SpacetimeType, Clone, Debug, PartialEq)]
pub enum RuntimeEventType {
    /// LLM assistant response content (streamed and updated in-place).
    LlmResponse,
    /// Final tool call chosen for this turn.
    ToolCall,
    /// Tool execution result payload.
    ToolResult,
    /// Full shell stdout/stderr payload.
    ShellOutput,
}

/// Scheduler row for periodic runtime resource sample pruning.
#[table(
    accessor = runtime_resource_sample_prune_schedule,
    scheduled(prune_runtime_resource_samples)
)]
pub struct RuntimeResourceSamplePruneSchedule {
    #[primary_key]
    #[auto_inc]
    pub scheduled_id: u64,
    pub scheduled_at: ScheduleAt,
    pub retention_seconds: u64,
    pub prune_batch_size: u32,
}

/// Scheduler row for periodic runtime event pruning.
#[table(accessor = runtime_event_prune_schedule, scheduled(prune_runtime_events))]
pub struct RuntimeEventPruneSchedule {
    #[primary_key]
    #[auto_inc]
    pub scheduled_id: u64,
    pub scheduled_at: ScheduleAt,
    pub retention_seconds: u64,
    pub prune_batch_size: u32,
}
