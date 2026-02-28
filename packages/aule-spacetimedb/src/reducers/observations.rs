//! Observation reducers: agents post observations, humans read them.

use spacetimedb::{reducer, ReducerContext, Table};

use crate::tables::{agent_task, observation, Observation, ObservationKind, TaskStatus};

/// Post an observation about a task. Only the runtime assigned to the task
/// can post observations for it.
#[reducer]
pub fn post_observation(
    ctx: &ReducerContext,
    task_id: u64,
    kind: ObservationKind,
    content: String,
) -> Result<(), String> {
    let sender = ctx.sender();

    let content = content.trim().to_string();
    if content.is_empty() {
        return Err("Observation content must not be empty".to_string());
    }

    // Verify the task exists and the sender is the assigned runtime
    let task = ctx
        .db
        .agent_task()
        .id()
        .find(task_id)
        .ok_or(format!("Task {task_id} not found"))?;

    if task.assigned_runtime != Some(sender) {
        return Err("Only the assigned runtime can post observations".to_string());
    }

    if task.status != TaskStatus::Running {
        return Err(format!(
            "Task is {:?}, can only post observations for Running tasks",
            task.status
        ));
    }

    ctx.db.observation().insert(Observation {
        id: 0,
        task_id,
        runtime_identity: sender,
        kind,
        content,
        created_at: ctx.timestamp,
    });

    Ok(())
}
