//! Identity reducers: runtime registration, deregistration, heartbeat.

use spacetimedb::{reducer, ReducerContext, Table};

use crate::tables::{
    agent_runtime, agent_task, AgentRuntime, AgentTask, RuntimeStatus, TaskStatus,
};

/// Register a new agent runtime. Called by an agent process after connecting.
#[reducer]
pub fn register_runtime(ctx: &ReducerContext, name: String) -> Result<(), String> {
    let sender = ctx.sender();

    // Check not already registered
    if ctx.db.agent_runtime().identity().find(sender).is_some() {
        return Err("Runtime already registered".to_string());
    }

    let name = name.trim().to_string();
    if name.is_empty() {
        return Err("Runtime name must not be empty".to_string());
    }

    // Name must be unique
    if ctx.db.agent_runtime().name().find(&name).is_some() {
        return Err(format!("Runtime name '{name}' is already taken"));
    }

    ctx.db.agent_runtime().insert(AgentRuntime {
        identity: sender,
        name,
        status: RuntimeStatus::Idle,
        last_heartbeat: ctx.timestamp,
        registered_at: ctx.timestamp,
    });

    log::info!("Runtime registered: {:?}", sender);
    Ok(())
}

/// Deregister an agent runtime. Called by the agent process before disconnecting.
/// Any assigned (but not yet running) tasks are unassigned.
#[reducer]
pub fn deregister_runtime(ctx: &ReducerContext) -> Result<(), String> {
    let sender = ctx.sender();

    let runtime = ctx
        .db
        .agent_runtime()
        .identity()
        .find(sender)
        .ok_or("Runtime not registered")?;

    // Reject deregistration if any tasks are still running
    let has_running = ctx
        .db
        .agent_task()
        .iter()
        .any(|t| t.assigned_runtime == Some(sender) && t.status == TaskStatus::Running);
    if has_running {
        return Err(
            "Cannot deregister: runtime has running tasks. Complete or fail them first."
                .to_string(),
        );
    }

    // Unassign any tasks that are assigned but not yet running
    for task in ctx.db.agent_task().iter() {
        if task.assigned_runtime == Some(sender) && task.status == TaskStatus::Assigned {
            ctx.db.agent_task().id().update(AgentTask {
                assigned_runtime: None,
                status: TaskStatus::Pending,
                ..task
            });
        }
    }

    ctx.db.agent_runtime().identity().delete(runtime.identity);
    log::info!("Runtime deregistered: {:?}", sender);
    Ok(())
}

/// Heartbeat from an agent runtime. Keeps the runtime marked as alive.
#[reducer]
pub fn heartbeat(ctx: &ReducerContext) -> Result<(), String> {
    let sender = ctx.sender();

    let runtime = ctx
        .db
        .agent_runtime()
        .identity()
        .find(sender)
        .ok_or("Runtime not registered")?;

    ctx.db.agent_runtime().identity().update(AgentRuntime {
        last_heartbeat: ctx.timestamp,
        ..runtime
    });

    Ok(())
}
