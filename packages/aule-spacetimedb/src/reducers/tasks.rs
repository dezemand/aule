//! Task lifecycle reducers: create, assign, start, complete, fail.

use spacetimedb::{reducer, ReducerContext, Table};

use crate::tables::{
    agent_runtime, agent_task, agent_type, AgentRuntime, AgentTask, RuntimeStatus, TaskStatus,
};

/// Create a new task. Anyone can create tasks.
#[reducer]
pub fn create_task(
    ctx: &ReducerContext,
    agent_type_id: u64,
    title: String,
    description: String,
) -> Result<(), String> {
    let title = title.trim().to_string();
    if title.is_empty() {
        return Err("Task title must not be empty".to_string());
    }

    // Verify agent type exists
    if ctx.db.agent_type().id().find(agent_type_id).is_none() {
        return Err(format!("Agent type {agent_type_id} not found"));
    }

    ctx.db.agent_task().insert(AgentTask {
        id: 0,
        agent_type_id,
        assigned_runtime: None,
        status: TaskStatus::Pending,
        title: title.clone(),
        description,
        created_by: ctx.sender(),
        created_at: ctx.timestamp,
        started_at: None,
        completed_at: None,
        result: None,
    });

    log::info!("Task created: {title}");
    Ok(())
}

/// Assign a pending task to an idle runtime.
/// The runtime must be registered and idle, and must match the task's agent type.
#[reducer]
pub fn assign_task(
    ctx: &ReducerContext,
    task_id: u64,
    runtime_identity: spacetimedb::Identity,
) -> Result<(), String> {
    let task = ctx
        .db
        .agent_task()
        .id()
        .find(task_id)
        .ok_or(format!("Task {task_id} not found"))?;

    if task.status != TaskStatus::Pending {
        return Err(format!(
            "Task is {:?}, can only assign Pending tasks",
            task.status
        ));
    }

    let runtime = ctx
        .db
        .agent_runtime()
        .identity()
        .find(runtime_identity)
        .ok_or("Runtime not found")?;

    if runtime.status != RuntimeStatus::Idle {
        return Err(format!(
            "Runtime is {:?}, can only assign to Idle runtimes",
            runtime.status
        ));
    }

    if runtime.agent_type_id != task.agent_type_id {
        return Err(format!(
            "Runtime agent type {} does not match task agent type {}",
            runtime.agent_type_id, task.agent_type_id
        ));
    }

    // Update task
    ctx.db.agent_task().id().update(AgentTask {
        assigned_runtime: Some(runtime_identity),
        status: TaskStatus::Assigned,
        ..task
    });

    // Mark runtime as busy
    ctx.db.agent_runtime().identity().update(AgentRuntime {
        status: RuntimeStatus::Busy,
        ..runtime
    });

    log::info!("Task {task_id} assigned to runtime {:?}", runtime_identity);
    Ok(())
}

/// Called by the assigned runtime to indicate it has started working on the task.
#[reducer]
pub fn start_task(ctx: &ReducerContext, task_id: u64) -> Result<(), String> {
    let sender = ctx.sender();
    let task = ctx
        .db
        .agent_task()
        .id()
        .find(task_id)
        .ok_or(format!("Task {task_id} not found"))?;

    if task.status != TaskStatus::Assigned {
        return Err(format!(
            "Task is {:?}, can only start Assigned tasks",
            task.status
        ));
    }

    if task.assigned_runtime != Some(sender) {
        return Err("Only the assigned runtime can start a task".to_string());
    }

    ctx.db.agent_task().id().update(AgentTask {
        status: TaskStatus::Running,
        started_at: Some(ctx.timestamp),
        ..task
    });

    log::info!("Task {task_id} started by runtime {:?}", sender);
    Ok(())
}

/// Called by the assigned runtime when the task is completed successfully.
#[reducer]
pub fn complete_task(ctx: &ReducerContext, task_id: u64, result: String) -> Result<(), String> {
    let sender = ctx.sender();
    let task = ctx
        .db
        .agent_task()
        .id()
        .find(task_id)
        .ok_or(format!("Task {task_id} not found"))?;

    if task.status != TaskStatus::Running {
        return Err(format!(
            "Task is {:?}, can only complete Running tasks",
            task.status
        ));
    }

    if task.assigned_runtime != Some(sender) {
        return Err("Only the assigned runtime can complete a task".to_string());
    }

    // Update task
    ctx.db.agent_task().id().update(AgentTask {
        status: TaskStatus::Completed,
        completed_at: Some(ctx.timestamp),
        result: Some(result),
        ..task
    });

    // Mark runtime as idle again
    if let Some(runtime) = ctx.db.agent_runtime().identity().find(sender) {
        ctx.db.agent_runtime().identity().update(AgentRuntime {
            status: RuntimeStatus::Idle,
            ..runtime
        });
    }

    log::info!("Task {task_id} completed by runtime {:?}", sender);
    Ok(())
}

/// Called by the assigned runtime when the task fails.
#[reducer]
pub fn fail_task(ctx: &ReducerContext, task_id: u64, error: String) -> Result<(), String> {
    let sender = ctx.sender();
    let task = ctx
        .db
        .agent_task()
        .id()
        .find(task_id)
        .ok_or(format!("Task {task_id} not found"))?;

    if task.status != TaskStatus::Running && task.status != TaskStatus::Assigned {
        return Err(format!(
            "Task is {:?}, can only fail Running or Assigned tasks",
            task.status
        ));
    }

    if task.assigned_runtime != Some(sender) {
        return Err("Only the assigned runtime can fail a task".to_string());
    }

    // Update task
    ctx.db.agent_task().id().update(AgentTask {
        status: TaskStatus::Failed,
        completed_at: Some(ctx.timestamp),
        result: Some(error),
        ..task
    });

    // Mark runtime as idle again
    if let Some(runtime) = ctx.db.agent_runtime().identity().find(sender) {
        ctx.db.agent_runtime().identity().update(AgentRuntime {
            status: RuntimeStatus::Idle,
            ..runtime
        });
    }

    log::info!("Task {task_id} failed by runtime {:?}", sender);
    Ok(())
}

/// Admin: cancel any non-terminal task. Resets the assigned runtime to Idle
/// if one was assigned. Useful for recovering from stuck tasks during
/// development.
#[reducer]
pub fn cancel_task(ctx: &ReducerContext, task_id: u64) -> Result<(), String> {
    let task = ctx
        .db
        .agent_task()
        .id()
        .find(task_id)
        .ok_or(format!("Task {task_id} not found"))?;

    if task.status == TaskStatus::Completed
        || task.status == TaskStatus::Failed
        || task.status == TaskStatus::Cancelled
    {
        return Err(format!("Task is already {:?}, cannot cancel", task.status));
    }

    // Free the assigned runtime if there is one
    if let Some(runtime_identity) = task.assigned_runtime {
        if let Some(runtime) = ctx.db.agent_runtime().identity().find(runtime_identity) {
            if runtime.status == RuntimeStatus::Busy {
                ctx.db.agent_runtime().identity().update(AgentRuntime {
                    status: RuntimeStatus::Idle,
                    ..runtime
                });
            }
        }
    }

    ctx.db.agent_task().id().update(AgentTask {
        status: TaskStatus::Cancelled,
        completed_at: Some(ctx.timestamp),
        result: Some("Cancelled by admin".to_string()),
        ..task
    });

    log::info!("Task {task_id} cancelled by {:?}", ctx.sender());
    Ok(())
}
