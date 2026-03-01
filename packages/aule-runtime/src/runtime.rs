use std::{
    collections::HashSet,
    sync::atomic::{AtomicU64, Ordering},
    sync::mpsc::{self, Receiver, Sender},
    thread,
    time::{Duration, Instant},
    time::{SystemTime, UNIX_EPOCH},
};

use anyhow::{Context, Result, bail};
use aule_spacetimedb_client::*;
use log::{error, info, warn};
use spacetimedb_sdk::{DbContext, Error, Identity, Table, TableWithPrimaryKey, credentials};

use crate::{
    config::RuntimeConfig,
    llm_client::{AgentAction, Conversation, ObserveKind, OpenAiClient, StreamEvent},
    shell,
};

static RUNTIME_EVENT_SEQ: AtomicU64 = AtomicU64::new(1);

struct PreparedTask {
    task: AgentTask,
    system_prompt: String,
}

enum TaskExecutionReport {
    Succeeded { task_id: u64 },
    Failed { task_id: u64, error: String },
}

pub fn run(config: RuntimeConfig) -> Result<()> {
    info!(
        "Starting runtime '{}' (agent_type_id={}, agent_version={})",
        config.runtime_name, config.agent_type_id, config.agent_version
    );

    let (task_tx, task_rx) = mpsc::channel::<u64>();
    let ctx = connect_to_db(&config)?;
    register_task_callbacks(&ctx, task_tx.clone());
    subscribe_to_tables(&ctx, task_tx)?;
    let _network_thread = ctx.run_threaded();

    event_loop(&ctx, &config, task_rx)
}

fn creds_store() -> credentials::File {
    credentials::File::new("aule-runtime")
}

fn connect_to_db(config: &RuntimeConfig) -> Result<DbConnection> {
    let token = creds_store().load().ok();

    let mut builder = DbConnection::builder()
        .on_connect(on_connected)
        .on_connect_error(on_connect_error)
        .on_disconnect(on_disconnected)
        .with_database_name(&config.spacetimedb_db_name)
        .with_uri(&config.spacetimedb_uri);

    if let Some(token) = token {
        builder = builder.with_token(token);
    }

    builder.build().context("Failed to connect to SpacetimeDB")
}

fn on_connected(_ctx: &DbConnection, identity: Identity, token: &str) {
    if let Err(err) = creds_store().save(token) {
        warn!("Failed to save SpacetimeDB credentials: {err:?}");
    }
    info!("Connected as identity {}", identity.to_hex());
}

fn on_connect_error(_ctx: &ErrorContext, err: Error) {
    error!("Connection error: {err:?}");
}

fn on_disconnected(_ctx: &ErrorContext, err: Option<Error>) {
    if let Some(err) = err {
        error!("Disconnected with error: {err:?}");
    } else {
        info!("Disconnected");
    }
}

fn register_task_callbacks(ctx: &DbConnection, task_tx: Sender<u64>) {
    let inserted_tx = task_tx.clone();
    ctx.db.agent_task().on_insert(move |_ctx, task| {
        if task.status == TaskStatus::Assigned {
            let _ = inserted_tx.send(task.id);
        }
    });

    ctx.db.agent_task().on_update(move |_ctx, old, new| {
        if old.status != new.status && new.status == TaskStatus::Assigned {
            let _ = task_tx.send(new.id);
        }
    });
}

fn subscribe_to_tables(ctx: &DbConnection, task_tx: Sender<u64>) -> Result<()> {
    ctx.subscription_builder()
        .on_applied(move |sub_ctx| {
            info!("Subscriptions applied");
            for task in sub_ctx.db.agent_task().iter() {
                if task.status == TaskStatus::Assigned {
                    let _ = task_tx.send(task.id);
                }
            }
        })
        .on_error(|_ctx, err| {
            error!("Subscription error: {err}");
        })
        .subscribe([
            "SELECT * FROM agent_task",
            "SELECT * FROM agent_runtime",
            "SELECT * FROM agent_type_version",
        ]);

    Ok(())
}

fn event_loop(ctx: &DbConnection, config: &RuntimeConfig, task_rx: Receiver<u64>) -> Result<()> {
    let (task_result_tx, task_result_rx) = mpsc::channel::<TaskExecutionReport>();
    let mut in_flight_tasks = HashSet::<u64>::new();
    let mut registered = false;
    let mut last_heartbeat = Instant::now();

    loop {
        while let Ok(report) = task_result_rx.try_recv() {
            match report {
                TaskExecutionReport::Succeeded { task_id } => {
                    in_flight_tasks.remove(&task_id);
                    info!("Task {task_id} worker finished successfully");
                }
                TaskExecutionReport::Failed { task_id, error } => {
                    in_flight_tasks.remove(&task_id);
                    error!("Task {task_id} execution failed: {error}");
                    let _ = ctx
                        .reducers
                        .fail_task(task_id, format!("runtime error: {error}"));
                }
            }
        }

        if !registered && ctx.try_identity().is_some() {
            match ctx
                .reducers
                .register_runtime(config.runtime_name.clone(), config.agent_type_id)
            {
                Ok(_) => {
                    info!("Runtime registered as '{}'.", config.runtime_name);
                    registered = true;
                }
                Err(err) => {
                    let msg = err.to_string();
                    if msg.contains("already registered") {
                        info!("Runtime already registered, continuing.");
                        registered = true;
                    } else {
                        warn!("register_runtime failed (will retry): {msg}");
                    }
                }
            }
        }

        if last_heartbeat.elapsed() >= config.heartbeat_interval {
            if let Err(err) = ctx.reducers.heartbeat() {
                warn!("heartbeat failed: {err}");
            }
            last_heartbeat = Instant::now();
        }

        match task_rx.recv_timeout(Duration::from_millis(250)) {
            Ok(task_id) => {
                if in_flight_tasks.contains(&task_id) {
                    continue;
                }

                let prepared = match prepare_task_execution(ctx, config, task_id) {
                    Ok(Some(prepared)) => prepared,
                    Ok(None) => continue,
                    Err(err) => {
                        error!("Task {task_id} could not be prepared: {err:#}");
                        let _ = ctx
                            .reducers
                            .fail_task(task_id, format!("runtime error: {err:#}"));
                        continue;
                    }
                };

                in_flight_tasks.insert(task_id);
                spawn_task_worker(config.clone(), prepared, task_result_tx.clone());
            }
            Err(mpsc::RecvTimeoutError::Timeout) => {
                while let Ok(report) = task_result_rx.try_recv() {
                    match report {
                        TaskExecutionReport::Succeeded { task_id } => {
                            in_flight_tasks.remove(&task_id);
                            info!("Task {task_id} worker finished successfully");
                        }
                        TaskExecutionReport::Failed { task_id, error } => {
                            in_flight_tasks.remove(&task_id);
                            error!("Task {task_id} execution failed: {error}");
                            let _ = ctx
                                .reducers
                                .fail_task(task_id, format!("runtime error: {error}"));
                        }
                    }
                }
            }
            Err(mpsc::RecvTimeoutError::Disconnected) => {
                bail!("Task queue disconnected");
            }
        }
    }
}

fn prepare_task_execution(
    ctx: &DbConnection,
    config: &RuntimeConfig,
    task_id: u64,
) -> Result<Option<PreparedTask>> {
    let identity = ctx
        .try_identity()
        .context("No runtime identity available yet")?;

    let task = ctx
        .db
        .agent_task()
        .id()
        .find(&task_id)
        .ok_or_else(|| anyhow::anyhow!("Task {task_id} not found in cache"))?;

    if task.assigned_runtime != Some(identity) || task.status != TaskStatus::Assigned {
        return Ok(None);
    }

    let system_prompt = resolve_system_prompt(ctx, task.agent_type_id, &config.agent_version)?;

    Ok(Some(PreparedTask {
        task,
        system_prompt,
    }))
}

fn spawn_task_worker(
    config: RuntimeConfig,
    prepared: PreparedTask,
    result_tx: Sender<TaskExecutionReport>,
) {
    thread::spawn(move || {
        let task_id = prepared.task.id;
        let report = match run_task_worker(config, prepared) {
            Ok(_) => TaskExecutionReport::Succeeded { task_id },
            Err(err) => TaskExecutionReport::Failed {
                task_id,
                error: format!("{err:#}"),
            },
        };

        if result_tx.send(report).is_err() {
            error!("Task worker could not report completion for task {task_id}");
        }
    });
}

fn run_task_worker(config: RuntimeConfig, prepared: PreparedTask) -> Result<()> {
    let ctx = connect_to_db(&config)?;
    let _network_thread = ctx.run_threaded();

    let wait_start = Instant::now();
    while ctx.try_identity().is_none() {
        if wait_start.elapsed() > Duration::from_secs(5) {
            bail!("Worker connection did not obtain identity within 5 seconds");
        }
        thread::sleep(Duration::from_millis(25));
    }

    let tokio_rt = tokio::runtime::Builder::new_current_thread()
        .enable_all()
        .build()
        .context("Failed to create tokio runtime in worker")?;
    let llm = OpenAiClient::new(
        config.openai_api_key.clone(),
        config.openai_model.clone(),
        tokio_rt,
    )?;

    execute_prepared_task(&ctx, &config, &llm, &prepared.task, &prepared.system_prompt)
}

fn execute_prepared_task(
    ctx: &DbConnection,
    config: &RuntimeConfig,
    llm: &OpenAiClient,
    task: &AgentTask,
    system_prompt: &str,
) -> Result<()> {
    let task_id = task.id;

    ctx.reducers
        .start_task(task_id)
        .with_context(|| format!("Failed to start task {task_id}"))?;
    info!("Started task #{task_id}: {}", task.title);

    post_observation(
        ctx,
        task_id,
        ObservationKind::Progress,
        format!("Picked up task '{}'.", task.title),
    );

    run_reasoning_loop(ctx, config, llm, task, system_prompt)
}

fn resolve_system_prompt(ctx: &DbConnection, agent_type_id: u64, version: &str) -> Result<String> {
    let selected = ctx
        .db
        .agent_type_version()
        .iter()
        .find(|v| {
            v.agent_type_id == agent_type_id
                && v.version == version
                && v.status == VersionStatus::Active
        })
        .with_context(|| {
            format!(
                "No active agent_type_version for type_id={agent_type_id} and version='{version}'"
            )
        })?;

    Ok(selected.system_prompt)
}

fn run_reasoning_loop(
    ctx: &DbConnection,
    config: &RuntimeConfig,
    llm: &OpenAiClient,
    task: &AgentTask,
    system_prompt: &str,
) -> Result<()> {
    let initial_user_prompt = format!(
        "Task #{id}\nTitle: {title}\nDescription:\n{description}\n\n\
         Use tools to work the task incrementally.\n\
         Call aule_finish when done or aule_fail when blocked.",
        id = task.id,
        title = task.title,
        description = task.description,
    );
    let mut conversation = Conversation::new(system_prompt, &initial_user_prompt);

    for turn in 1..=config.max_steps_per_task {
        let llm_response_event_id = next_runtime_event_id(task.id, turn, "llm-response");
        let mut llm_response_content = String::new();

        create_runtime_event(
            ctx,
            llm_response_event_id.clone(),
            task.id,
            turn,
            RuntimeEventType::LlmResponse,
            String::new(),
        );

        // Stream the LLM response and keep a growing runtime_event row.
        let decision = llm
            .stream_tool_decision(&conversation, |event| match event {
                StreamEvent::TextDelta(text) => {
                    llm_response_content.push_str(&text);
                    update_runtime_event(
                        ctx,
                        llm_response_event_id.clone(),
                        llm_response_content.clone(),
                    );
                }
                StreamEvent::ToolArgsDelta {
                    tool_name,
                    args_delta,
                } => {
                    llm_response_content
                        .push_str(&format!("\n[tool_args:{tool_name}] {args_delta}"));
                    update_runtime_event(
                        ctx,
                        llm_response_event_id.clone(),
                        llm_response_content.clone(),
                    );
                }
                StreamEvent::Done => {}
            })
            .with_context(|| format!("LLM streaming failed at turn {turn}"))?;

        conversation.push_assistant_message(decision.assistant_message);

        create_runtime_event(
            ctx,
            next_runtime_event_id(task.id, turn, "tool-call"),
            task.id,
            turn,
            RuntimeEventType::ToolCall,
            format!(
                "tool_call_id={}\naction={:?}\nextra_tool_call_ids={:?}",
                decision.tool_call_id, decision.action, decision.extra_tool_call_ids
            ),
        );

        // If the model returned multiple tool calls despite
        // parallel_tool_calls: false, provide placeholder results for the
        // extra ones so the conversation stays valid.
        for extra_id in &decision.extra_tool_call_ids {
            let placeholder = "Skipped: only one tool call is executed per turn.".to_string();
            conversation.push_tool_result(extra_id, placeholder.clone());
            create_runtime_event(
                ctx,
                next_runtime_event_id(task.id, turn, "tool-result-extra"),
                task.id,
                turn,
                RuntimeEventType::ToolResult,
                format!("tool_call_id={extra_id}\nresult={placeholder}"),
            );
        }

        match decision.action {
            AgentAction::Shell { command } => {
                info!("task #{} turn {} sh: {}", task.id, turn, command);

                post_observation(
                    ctx,
                    task.id,
                    ObservationKind::Progress,
                    format!("Running command: `{command}`"),
                );

                let result = shell::run_shell(
                    &command,
                    config.shell_timeout,
                    config.shell_output_limit_bytes,
                )?;

                let outcome = format!(
                    "exit_code={:?} timed_out={} duration_ms={}\nstdout:\n{}\nstderr:\n{}",
                    result.exit_code,
                    result.timed_out,
                    result.duration_ms,
                    result.stdout,
                    result.stderr,
                );

                create_runtime_event(
                    ctx,
                    next_runtime_event_id(task.id, turn, "shell-output"),
                    task.id,
                    turn,
                    RuntimeEventType::ShellOutput,
                    format!("command: {command}\n\n{outcome}"),
                );

                let (kind, summary) = if result.timed_out {
                    (
                        ObservationKind::Error,
                        format!(
                            "Command timed out after {}ms: `{command}`",
                            result.duration_ms
                        ),
                    )
                } else if result.exit_code.unwrap_or(1) != 0 {
                    (
                        ObservationKind::Error,
                        format!(
                            "Command failed (exit_code={:?}, {}ms): `{command}`",
                            result.exit_code, result.duration_ms
                        ),
                    )
                } else {
                    (
                        ObservationKind::Finding,
                        format!("Command succeeded in {}ms: `{command}`", result.duration_ms),
                    )
                };
                post_observation(ctx, task.id, kind, summary);

                create_runtime_event(
                    ctx,
                    next_runtime_event_id(task.id, turn, "tool-result"),
                    task.id,
                    turn,
                    RuntimeEventType::ToolResult,
                    format!(
                        "tool_call_id={}\nresult=exit_code={:?} timed_out={} duration_ms={}",
                        decision.tool_call_id,
                        result.exit_code,
                        result.timed_out,
                        result.duration_ms
                    ),
                );

                // Feed full output back to conversation (not summarized)
                conversation.push_tool_result(&decision.tool_call_id, outcome);
            }
            AgentAction::Observe { kind, content } => {
                let obs_kind = observe_kind_to_observation_kind(&kind);
                post_observation(ctx, task.id, obs_kind, content);
                let tool_result = format!("observation posted ({})", kind.as_str());
                conversation.push_tool_result(&decision.tool_call_id, tool_result.clone());
                create_runtime_event(
                    ctx,
                    next_runtime_event_id(task.id, turn, "tool-result"),
                    task.id,
                    turn,
                    RuntimeEventType::ToolResult,
                    format!(
                        "tool_call_id={}\nresult={tool_result}",
                        decision.tool_call_id
                    ),
                );
            }
            AgentAction::Status { content } => {
                post_observation(
                    ctx,
                    task.id,
                    ObservationKind::Progress,
                    format!("status: {content}"),
                );
                let tool_result = "status recorded".to_string();
                conversation.push_tool_result(&decision.tool_call_id, tool_result.clone());
                create_runtime_event(
                    ctx,
                    next_runtime_event_id(task.id, turn, "tool-result"),
                    task.id,
                    turn,
                    RuntimeEventType::ToolResult,
                    format!(
                        "tool_call_id={}\nresult={tool_result}",
                        decision.tool_call_id
                    ),
                );
            }
            AgentAction::Finish { result } => {
                post_observation(ctx, task.id, ObservationKind::Result, result.clone());
                create_runtime_event(
                    ctx,
                    next_runtime_event_id(task.id, turn, "tool-result"),
                    task.id,
                    turn,
                    RuntimeEventType::ToolResult,
                    format!(
                        "tool_call_id={}\nresult=task completed",
                        decision.tool_call_id
                    ),
                );
                ctx.reducers
                    .complete_task(task.id, result)
                    .with_context(|| format!("Failed to complete task {}", task.id))?;
                info!("Completed task #{}", task.id);
                return Ok(());
            }
            AgentAction::Fail { error } => {
                post_observation(ctx, task.id, ObservationKind::Error, error.clone());
                create_runtime_event(
                    ctx,
                    next_runtime_event_id(task.id, turn, "tool-result"),
                    task.id,
                    turn,
                    RuntimeEventType::ToolResult,
                    format!("tool_call_id={}\nresult=task failed", decision.tool_call_id),
                );
                ctx.reducers
                    .fail_task(task.id, error)
                    .with_context(|| format!("Failed to mark task {} as failed", task.id))?;
                info!("Marked task #{} as failed", task.id);
                return Ok(());
            }
        }
    }

    let msg = format!(
        "Reached max steps ({}) without finish/fail action",
        config.max_steps_per_task
    );
    post_observation(ctx, task.id, ObservationKind::Error, msg.clone());
    ctx.reducers
        .fail_task(task.id, msg)
        .with_context(|| format!("Failed to fail task {} after max steps", task.id))?;
    Ok(())
}

fn post_observation(ctx: &DbConnection, task_id: u64, kind: ObservationKind, content: String) {
    let content_len = content.len();
    if let Err(err) = ctx.reducers.post_observation(task_id, kind, content) {
        warn!("post_observation failed for task {task_id}: {err} (content_len={content_len})");
    }
}

fn create_runtime_event(
    ctx: &DbConnection,
    id: String,
    task_id: u64,
    turn: u32,
    event_type: RuntimeEventType,
    content: String,
) {
    if let Err(err) =
        ctx.reducers
            .create_runtime_event(id.clone(), task_id, turn, event_type, content)
    {
        warn!("create_runtime_event failed for task {task_id} ({id}): {err}");
    }
}

fn update_runtime_event(ctx: &DbConnection, id: String, content: String) {
    let content_len = content.len();
    if let Err(err) = ctx.reducers.update_runtime_event(id.clone(), content) {
        warn!("update_runtime_event failed for {id}: {err} (content_len={content_len})");
    }
}

fn next_runtime_event_id(task_id: u64, turn: u32, label: &str) -> String {
    let seq = RUNTIME_EVENT_SEQ.fetch_add(1, Ordering::Relaxed);
    let ts_ms = SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .map(|d| d.as_millis())
        .unwrap_or(0);
    format!("task-{task_id}-turn-{turn}-{label}-{ts_ms}-{seq}")
}

fn observe_kind_to_observation_kind(kind: &ObserveKind) -> ObservationKind {
    match kind {
        ObserveKind::Progress => ObservationKind::Progress,
        ObserveKind::Finding => ObservationKind::Finding,
        ObserveKind::Error => ObservationKind::Error,
        ObserveKind::Result => ObservationKind::Result,
    }
}
