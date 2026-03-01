//! Runtime event reducers: verbose runtime logs for task execution.

use spacetimedb::{ReducerContext, Table, reducer};

use crate::tables::{RuntimeEvent, RuntimeEventType, TaskStatus, agent_task, runtime_event};

#[reducer]
pub fn create_runtime_event(
    ctx: &ReducerContext,
    id: String,
    task_id: u64,
    turn: u32,
    event_type: RuntimeEventType,
    content: String,
) -> Result<(), String> {
    let sender = ctx.sender();
    let id = id.trim().to_string();
    if id.is_empty() {
        return Err("Runtime event id must not be empty".to_string());
    }

    if ctx.db.runtime_event().id().find(id.clone()).is_some() {
        return Err(format!("Runtime event {id} already exists"));
    }

    let event_id = id.clone();

    ensure_runtime_can_write_task_logs(ctx, task_id, sender)?;

    ctx.db.runtime_event().insert(RuntimeEvent {
        id,
        task_id,
        runtime_identity: sender,
        turn,
        event_type: event_type.clone(),
        content,
        created_at: ctx.timestamp,
        updated_at: ctx.timestamp,
    });

    log::info!(
        "Runtime event created: id={}, task_id={}, turn={}, event_type={:?}",
        event_id,
        task_id,
        turn,
        event_type
    );

    Ok(())
}

#[reducer]
pub fn update_runtime_event(
    ctx: &ReducerContext,
    id: String,
    content: String,
) -> Result<(), String> {
    let sender = ctx.sender();

    let event = ctx
        .db
        .runtime_event()
        .id()
        .find(id.clone())
        .ok_or(format!("Runtime event {id} not found"))?;

    if event.runtime_identity != sender {
        return Err("Only the event owner can update runtime events".to_string());
    }

    ensure_runtime_can_write_task_logs(ctx, event.task_id, sender)?;

    ctx.db.runtime_event().id().update(RuntimeEvent {
        content,
        updated_at: ctx.timestamp,
        ..event.clone()
    });

    log::info!(
        "Runtime event updated: id={}, task_id={}, event_type={:?}",
        event.id,
        event.task_id,
        event.event_type
    );

    Ok(())
}

fn ensure_runtime_can_write_task_logs(
    ctx: &ReducerContext,
    task_id: u64,
    sender: spacetimedb::Identity,
) -> Result<(), String> {
    let task = ctx
        .db
        .agent_task()
        .id()
        .find(task_id)
        .ok_or(format!("Task {task_id} not found"))?;

    if task.assigned_runtime != Some(sender) {
        return Err("Only the assigned runtime can write task logs".to_string());
    }

    if task.status != TaskStatus::Running {
        return Err(format!(
            "Task is {:?}, can only write runtime logs for Running tasks",
            task.status
        ));
    }

    Ok(())
}
