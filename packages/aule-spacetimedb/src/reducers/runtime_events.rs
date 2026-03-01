//! Runtime event reducers: verbose runtime logs for task execution.

use spacetimedb::{reducer, ReducerContext, Table, TimeDuration};

use crate::tables::{
    agent_task, runtime_event, runtime_event_prune_schedule, RuntimeEvent,
    RuntimeEventPruneSchedule, RuntimeEventType, TaskStatus,
};

const MAX_RUNTIME_EVENT_CONTENT_BYTES: usize = 256 * 1024;
const RUNTIME_EVENT_RETENTION_SECONDS: u64 = 24 * 60 * 60;
const RUNTIME_EVENT_PRUNE_INTERVAL_SECONDS: u64 = 5 * 60;
const RUNTIME_EVENT_PRUNE_BATCH_SIZE: usize = 500;

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
    if turn == 0 {
        return Err("Runtime event turn must be >= 1".to_string());
    }
    if content.as_bytes().len() > MAX_RUNTIME_EVENT_CONTENT_BYTES {
        return Err(format!(
            "Runtime event content exceeds {} bytes",
            MAX_RUNTIME_EVENT_CONTENT_BYTES
        ));
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

    prune_runtime_events_internal(
        ctx,
        RUNTIME_EVENT_RETENTION_SECONDS,
        RUNTIME_EVENT_PRUNE_BATCH_SIZE,
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

    if content.as_bytes().len() > MAX_RUNTIME_EVENT_CONTENT_BYTES {
        return Err(format!(
            "Runtime event content exceeds {} bytes",
            MAX_RUNTIME_EVENT_CONTENT_BYTES
        ));
    }

    let event = ctx
        .db
        .runtime_event()
        .id()
        .find(id.clone())
        .ok_or(format!("Runtime event {id} not found"))?;

    if event.turn == 0 {
        return Err("Runtime event turn must be >= 1".to_string());
    }

    if event.runtime_identity != sender {
        return Err("Only the event owner can update runtime events".to_string());
    }

    ensure_runtime_can_write_task_logs(ctx, event.task_id, sender)?;

    let event_id = event.id.clone();
    let task_id = event.task_id;
    let event_type = event.event_type.clone();

    ctx.db.runtime_event().id().update(RuntimeEvent {
        content,
        updated_at: ctx.timestamp,
        ..event
    });

    log::info!(
        "Runtime event updated: id={}, task_id={}, event_type={:?}",
        event_id,
        task_id,
        event_type
    );

    prune_runtime_events_internal(
        ctx,
        RUNTIME_EVENT_RETENTION_SECONDS,
        RUNTIME_EVENT_PRUNE_BATCH_SIZE,
    );

    Ok(())
}

#[reducer]
pub fn prune_runtime_events(
    ctx: &ReducerContext,
    schedule: RuntimeEventPruneSchedule,
) -> Result<(), String> {
    prune_runtime_events_internal(
        ctx,
        schedule.retention_seconds,
        schedule.prune_batch_size as usize,
    );
    schedule_next_runtime_event_prune(ctx, schedule.retention_seconds, schedule.prune_batch_size);

    Ok(())
}

pub(crate) fn schedule_next_runtime_event_prune(
    ctx: &ReducerContext,
    retention_seconds: u64,
    prune_batch_size: u32,
) {
    let interval = TimeDuration::from_micros(
        i64::try_from(RUNTIME_EVENT_PRUNE_INTERVAL_SECONDS)
            .unwrap_or(0)
            .saturating_mul(1_000_000),
    );

    ctx.db
        .runtime_event_prune_schedule()
        .insert(RuntimeEventPruneSchedule {
            scheduled_id: 0,
            scheduled_at: interval.into(),
            retention_seconds,
            prune_batch_size,
        });
}

fn prune_runtime_events_internal(
    ctx: &ReducerContext,
    retention_seconds: u64,
    prune_batch_size: usize,
) {
    let retention_micros = i128::from(retention_seconds).saturating_mul(1_000_000);
    let cutoff_micros =
        i128::from(ctx.timestamp.to_micros_since_unix_epoch()).saturating_sub(retention_micros);

    let stale_ids: Vec<String> = ctx
        .db
        .runtime_event()
        .iter()
        .filter_map(|event| {
            let created_at = i128::from(event.created_at.to_micros_since_unix_epoch());
            if created_at < cutoff_micros {
                Some(event.id)
            } else {
                None
            }
        })
        .take(prune_batch_size)
        .collect();

    for id in stale_ids {
        ctx.db.runtime_event().id().delete(id);
    }
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
