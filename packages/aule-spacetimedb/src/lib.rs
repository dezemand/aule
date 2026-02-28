//! Aule SpacetimeDB Module -- Phase 1: Minimal Coordination Layer
//!
//! Tables and reducers for single-agent task coordination:
//! - Agent runtimes register, receive tasks, report back
//! - Agent types and versioned system prompts
//! - Task lifecycle: create -> assign -> start -> complete/fail
//! - Observations: agents post findings, humans read them

pub mod reducers;
pub mod tables;

use spacetimedb::{reducer, ReducerContext};

use tables::{agent_runtime, AgentRuntime, RuntimeStatus};

// ---------------------------------------------------------------------------
// Lifecycle hooks
// ---------------------------------------------------------------------------

#[reducer(init)]
pub fn init(_ctx: &ReducerContext) {
    log::info!("Aule module initialized (Phase 1)");
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
