use std::{
    sync::mpsc::{self, Receiver, Sender},
    time::{Duration, Instant},
};

use anyhow::{bail, Context, Result};
use aule_spacetimedb_client::*;
use log::{error, info, warn};
use spacetimedb_sdk::{credentials, DbContext, Error, Identity, Table, TableWithPrimaryKey};

use crate::{
    config::RuntimeConfig,
    llm_client::{AgentAction, Conversation, ObserveKind, OpenAiClient, StreamEvent},
    shell,
};

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

    // Build a single-threaded tokio runtime for async HTTP streaming
    let tokio_rt = tokio::runtime::Builder::new_current_thread()
        .enable_all()
        .build()
        .context("Failed to create tokio runtime")?;

    let llm = OpenAiClient::new(
        config.openai_api_key.clone(),
        config.openai_model.clone(),
        tokio_rt.handle().clone(),
    )?;

    event_loop(&ctx, &config, &llm, task_rx)
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

fn event_loop(
    ctx: &DbConnection,
    config: &RuntimeConfig,
    llm: &OpenAiClient,
    task_rx: Receiver<u64>,
) -> Result<()> {
    let mut registered = false;
    let mut last_heartbeat = Instant::now();

    loop {
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
                if let Err(err) = execute_task(ctx, config, llm, task_id) {
                    error!("Task {task_id} execution failed: {err:#}");
                    let _ = ctx
                        .reducers
                        .fail_task(task_id, format!("runtime error: {err:#}"));
                }
            }
            Err(mpsc::RecvTimeoutError::Timeout) => {}
            Err(mpsc::RecvTimeoutError::Disconnected) => {
                bail!("Task queue disconnected");
            }
        }
    }
}

fn execute_task(
    ctx: &DbConnection,
    config: &RuntimeConfig,
    llm: &OpenAiClient,
    task_id: u64,
) -> Result<()> {
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
        return Ok(());
    }

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

    let system_prompt = resolve_system_prompt(ctx, task.agent_type_id, &config.agent_version)?;
    run_reasoning_loop(ctx, config, llm, &task, &system_prompt)
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
        let task_id = task.id;

        // Stream the LLM response, emitting live progress observations
        let decision = llm
            .stream_tool_decision(&conversation, |event| {
                match event {
                    StreamEvent::TextDelta(text) => {
                        // Model is thinking aloud -- post as progress observation
                        post_observation(
                            ctx,
                            task_id,
                            ObservationKind::Progress,
                            format!("[turn {turn} thinking] {text}"),
                        );
                    }
                    StreamEvent::ToolArgsDelta {
                        tool_name,
                        args_delta,
                    } => {
                        // Tool arguments are being built -- post as progress
                        post_observation(
                            ctx,
                            task_id,
                            ObservationKind::Progress,
                            format!("[turn {turn} calling {tool_name}] {args_delta}"),
                        );
                    }
                    StreamEvent::Done => {}
                }
            })
            .with_context(|| format!("LLM streaming failed at turn {turn}"))?;

        conversation.push_assistant_message(decision.assistant_message);

        match decision.action {
            AgentAction::Shell { command } => {
                info!("task #{} turn {} sh: {}", task.id, turn, command);
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

                let kind = if result.timed_out || result.exit_code.unwrap_or(1) != 0 {
                    ObservationKind::Error
                } else {
                    ObservationKind::Finding
                };
                post_observation(ctx, task.id, kind, format!("sh `{command}`\n{outcome}"));

                // Feed full output back to conversation (not summarized)
                conversation.push_tool_result(&decision.tool_call_id, outcome);
            }
            AgentAction::Observe { kind, content } => {
                let obs_kind = observe_kind_to_observation_kind(&kind);
                post_observation(ctx, task.id, obs_kind, content.clone());
                conversation.push_tool_result(
                    &decision.tool_call_id,
                    format!("observation posted ({})", kind.as_str()),
                );
            }
            AgentAction::Status { content } => {
                post_observation(
                    ctx,
                    task.id,
                    ObservationKind::Progress,
                    format!("status: {content}"),
                );
                conversation
                    .push_tool_result(&decision.tool_call_id, "status recorded".to_string());
            }
            AgentAction::Finish { result } => {
                post_observation(ctx, task.id, ObservationKind::Result, result.clone());
                ctx.reducers
                    .complete_task(task.id, result)
                    .with_context(|| format!("Failed to complete task {}", task.id))?;
                info!("Completed task #{}", task.id);
                return Ok(());
            }
            AgentAction::Fail { error } => {
                post_observation(ctx, task.id, ObservationKind::Error, error.clone());
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

fn observe_kind_to_observation_kind(kind: &ObserveKind) -> ObservationKind {
    match kind {
        ObserveKind::Progress => ObservationKind::Progress,
        ObserveKind::Finding => ObservationKind::Finding,
        ObserveKind::Error => ObservationKind::Error,
        ObserveKind::Result => ObservationKind::Result,
    }
}
