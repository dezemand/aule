//! Aule SpacetimeDB Module -- Phase 1: Minimal Coordination Layer
//!
//! Tables and reducers for single-agent task coordination:
//! - Agent runtimes register, receive tasks, report back
//! - Agent types and versioned system prompts
//! - Task lifecycle: create -> assign -> start -> complete/fail
//! - Observations: agents post findings, humans read them

pub mod reducers;
pub mod tables;

use spacetimedb::{reducer, ReducerContext, Table};

use tables::{
    agent_runtime, runtime_event_prune_schedule, runtime_resource_sample_prune_schedule,
    AgentRuntime, RuntimeStatus,
};

const DEFAULT_RUNTIME_EVENT_RETENTION_SECONDS: u64 = 24 * 60 * 60;
const DEFAULT_RUNTIME_RESOURCE_SAMPLE_RETENTION_SECONDS: u64 = 24 * 60 * 60;
const DEFAULT_PRUNE_BATCH_SIZE: u32 = 500;

// ---------------------------------------------------------------------------
// Lifecycle hooks
// ---------------------------------------------------------------------------

#[reducer(init)]
pub fn init(ctx: &ReducerContext) {
    log::info!("Aule module initialized (Phase 1)");

    if ctx
        .db
        .runtime_event_prune_schedule()
        .iter()
        .next()
        .is_none()
    {
        reducers::runtime_events::schedule_next_runtime_event_prune(
            ctx,
            DEFAULT_RUNTIME_EVENT_RETENTION_SECONDS,
            DEFAULT_PRUNE_BATCH_SIZE,
        );
    }

    if ctx
        .db
        .runtime_resource_sample_prune_schedule()
        .iter()
        .next()
        .is_none()
    {
        reducers::runtime_metadata::schedule_next_runtime_resource_sample_prune(
            ctx,
            DEFAULT_RUNTIME_RESOURCE_SAMPLE_RETENTION_SECONDS,
            DEFAULT_PRUNE_BATCH_SIZE,
        );
    }
}

#[reducer(client_connected)]
pub fn client_connected(ctx: &ReducerContext) {
    let sender = ctx.sender();
    log::info!("Client connected: {:?}", sender);

    // If this is a registered runtime reconnecting, mark it online
    if let Some(runtime) = ctx.db.agent_runtime().identity().find(sender) {
        if runtime.status == RuntimeStatus::Offline {
            ctx.db.agent_runtime().identity().update(AgentRuntime {
                status: RuntimeStatus::Idle,
                last_heartbeat: ctx.timestamp,
                ..runtime
            });
            log::info!("Runtime {:?} reconnected", sender);
        }
    }
}

#[reducer(client_disconnected)]
pub fn client_disconnected(ctx: &ReducerContext) {
    let sender = ctx.sender();
    log::info!("Client disconnected: {:?}", sender);

    // If this is a registered runtime, mark it offline
    if let Some(runtime) = ctx.db.agent_runtime().identity().find(sender) {
        ctx.db.agent_runtime().identity().update(AgentRuntime {
            status: RuntimeStatus::Offline,
            ..runtime
        });
        log::info!("Runtime {:?} marked offline", sender);
    }
}
