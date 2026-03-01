use std::{
    collections::HashSet,
    env, fs,
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
    llm_client::{AgentAction, Conversation, ObserveKind, OpenAiClient, StreamEvent, ToolCall},
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

struct RuntimeProfilePayload {
    runtime_instance_id: String,
    runtime_name: String,
    runtime_version: String,
    git_sha: Option<String>,
    hostname: Option<String>,
    os: String,
    arch: String,
}

struct RuntimePlatformInfoPayload {
    environment: RuntimeEnvironment,
    process_id: Option<u32>,
    container_id: Option<String>,
    image: Option<String>,
    image_digest: Option<String>,
    cluster: Option<String>,
    namespace: Option<String>,
    pod_name: Option<String>,
    pod_uid: Option<String>,
    node_name: Option<String>,
    workload_kind: Option<String>,
    workload_name: Option<String>,
    container_name: Option<String>,
    restart_count: Option<u32>,
}

struct RuntimeResourceSamplePayload {
    cpu_millicores: Option<u32>,
    memory_rss_bytes: u64,
    memory_working_set_bytes: Option<u64>,
    threads: Option<u32>,
    open_fds: Option<u32>,
}

struct IndexedToolCall {
    index: usize,
    tool_call_id: String,
    action: AgentAction,
}

pub fn run(config: RuntimeConfig) -> Result<()> {
    info!(
        "Starting runtime '{}' (agent_version={})",
        config.runtime_name, config.agent_version
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
    let mut last_resource_sample = Instant::now();
    let start_millis = SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .map(|d| d.as_millis())
        .unwrap_or(0);
    let runtime_instance_id = format!("{}-{}", std::process::id(), start_millis);

    loop {
        drain_task_reports(ctx, &mut in_flight_tasks, &task_result_rx);

        if !registered && ctx.try_identity().is_some() {
            match ctx.reducers.register_runtime(config.runtime_name.clone()) {
                Ok(_) => {
                    info!("Runtime registered as '{}'.", config.runtime_name);
                    registered = true;
                    upsert_runtime_metadata(ctx, config, &runtime_instance_id);
                }
                Err(err) => {
                    let msg = err.to_string();
                    if msg.contains("already registered") {
                        info!("Runtime already registered, continuing.");
                        registered = true;
                        upsert_runtime_metadata(ctx, config, &runtime_instance_id);
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

        if registered && last_resource_sample.elapsed() >= config.resource_sample_interval {
            insert_runtime_resource_sample(ctx, collect_runtime_resource_sample());
            last_resource_sample = Instant::now();
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
                drain_task_reports(ctx, &mut in_flight_tasks, &task_result_rx);
            }
            Err(mpsc::RecvTimeoutError::Disconnected) => {
                bail!("Task queue disconnected");
            }
        }
    }
}

fn drain_task_reports(
    ctx: &DbConnection,
    in_flight_tasks: &mut HashSet<u64>,
    task_result_rx: &Receiver<TaskExecutionReport>,
) {
    while let Ok(report) = task_result_rx.try_recv() {
        process_task_execution_report(ctx, in_flight_tasks, report);
    }
}

fn process_task_execution_report(
    ctx: &DbConnection,
    in_flight_tasks: &mut HashSet<u64>,
    report: TaskExecutionReport,
) {
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

fn upsert_runtime_metadata(ctx: &DbConnection, config: &RuntimeConfig, runtime_instance_id: &str) {
    let profile = collect_runtime_profile(config, runtime_instance_id);
    if let Err(err) = ctx.reducers.upsert_runtime_profile(
        profile.runtime_instance_id,
        profile.runtime_name,
        profile.runtime_version,
        profile.git_sha,
        profile.hostname,
        profile.os,
        profile.arch,
    ) {
        warn!("upsert_runtime_profile failed: {err}");
    }

    let platform = collect_runtime_platform_info();
    if let Err(err) = ctx.reducers.upsert_runtime_platform_info(
        platform.environment,
        platform.process_id,
        platform.container_id,
        platform.image,
        platform.image_digest,
        platform.cluster,
        platform.namespace,
        platform.pod_name,
        platform.pod_uid,
        platform.node_name,
        platform.workload_kind,
        platform.workload_name,
        platform.container_name,
        platform.restart_count,
    ) {
        warn!("upsert_runtime_platform_info failed: {err}");
    }
}

fn insert_runtime_resource_sample(ctx: &DbConnection, sample: RuntimeResourceSamplePayload) {
    if let Err(err) = ctx.reducers.insert_runtime_resource_sample(
        sample.cpu_millicores,
        sample.memory_rss_bytes,
        sample.memory_working_set_bytes,
        sample.threads,
        sample.open_fds,
    ) {
        warn!("insert_runtime_resource_sample failed: {err}");
    }
}

fn collect_runtime_profile(
    config: &RuntimeConfig,
    runtime_instance_id: &str,
) -> RuntimeProfilePayload {
    RuntimeProfilePayload {
        runtime_instance_id: runtime_instance_id.to_string(),
        runtime_name: config.runtime_name.clone(),
        runtime_version: env!("CARGO_PKG_VERSION").to_string(),
        git_sha: env::var("AULE_GIT_SHA").ok().map(|v| v.trim().to_string()),
        hostname: read_hostname(),
        os: env::consts::OS.to_string(),
        arch: env::consts::ARCH.to_string(),
    }
}

fn collect_runtime_platform_info() -> RuntimePlatformInfoPayload {
    let environment = detect_runtime_environment();
    RuntimePlatformInfoPayload {
        environment,
        process_id: Some(std::process::id()),
        container_id: read_container_id(),
        image: env_opt("AULE_RUNTIME_IMAGE"),
        image_digest: env_opt("AULE_RUNTIME_IMAGE_DIGEST"),
        cluster: env_opt("AULE_K8S_CLUSTER"),
        namespace: env_opt("POD_NAMESPACE").or_else(|| env_opt("K8S_NAMESPACE")),
        pod_name: env_opt("POD_NAME"),
        pod_uid: env_opt("POD_UID"),
        node_name: env_opt("NODE_NAME"),
        workload_kind: env_opt("AULE_WORKLOAD_KIND"),
        workload_name: env_opt("AULE_WORKLOAD_NAME"),
        container_name: env_opt("CONTAINER_NAME"),
        restart_count: env_opt("RESTART_COUNT").and_then(|v| v.parse::<u32>().ok()),
    }
}

fn collect_runtime_resource_sample() -> RuntimeResourceSamplePayload {
    RuntimeResourceSamplePayload {
        cpu_millicores: None,
        memory_rss_bytes: read_memory_rss_bytes().unwrap_or(0),
        memory_working_set_bytes: None,
        threads: read_thread_count(),
        open_fds: read_open_fd_count(),
    }
}

fn detect_runtime_environment() -> RuntimeEnvironment {
    if env::var("KUBERNETES_SERVICE_HOST").is_ok()
        || env::var("POD_NAME").is_ok()
        || env::var("POD_NAMESPACE").is_ok()
    {
        return RuntimeEnvironment::K8S;
    }

    if fs::metadata("/.dockerenv").is_ok() {
        return RuntimeEnvironment::Docker;
    }

    if let Ok(cgroup) = fs::read_to_string("/proc/self/cgroup") {
        if cgroup.contains("docker") || cgroup.contains("containerd") || cgroup.contains("kubepods")
        {
            return RuntimeEnvironment::Docker;
        }
    }

    RuntimeEnvironment::Local
}

fn read_hostname() -> Option<String> {
    if let Some(hostname) = env_opt("HOSTNAME") {
        return Some(hostname);
    }

    #[cfg(unix)]
    {
        let mut buf = [0u8; 256];
        let rc = unsafe { libc::gethostname(buf.as_mut_ptr().cast(), buf.len()) };
        if rc == 0 {
            let len = buf.iter().position(|&b| b == 0).unwrap_or(buf.len());
            let hostname = String::from_utf8_lossy(&buf[..len]).trim().to_string();
            if !hostname.is_empty() {
                return Some(hostname);
            }
        }
    }

    None
}

fn read_container_id() -> Option<String> {
    if let Ok(cgroup) = fs::read_to_string("/proc/self/cgroup") {
        for line in cgroup.lines() {
            if let Some(candidate) = line
                .split('/')
                .next_back()
                .map(str::trim)
                .filter(|s| s.len() >= 12)
            {
                if candidate.chars().all(|c| c.is_ascii_hexdigit()) {
                    return Some(candidate.to_string());
                }
            }
        }
    }
    None
}

fn env_opt(key: &str) -> Option<String> {
    env::var(key)
        .ok()
        .map(|v| v.trim().to_string())
        .filter(|v| !v.is_empty())
}

fn read_memory_rss_bytes() -> Option<u64> {
    #[cfg(target_os = "linux")]
    {
        let status = fs::read_to_string("/proc/self/status").ok()?;
        for line in status.lines() {
            if let Some(rest) = line.strip_prefix("VmRSS:") {
                let kb = rest
                    .split_whitespace()
                    .next()
                    .and_then(|v| v.parse::<u64>().ok())?;
                return Some(kb.saturating_mul(1024));
            }
        }
        None
    }

    #[cfg(target_os = "macos")]
    {
        let mut usage = std::mem::MaybeUninit::<libc::rusage>::uninit();
        let rc = unsafe { libc::getrusage(libc::RUSAGE_SELF, usage.as_mut_ptr()) };
        if rc == 0 {
            let usage = unsafe { usage.assume_init() };
            let rss = usage.ru_maxrss;
            if rss > 0 {
                // On macOS, ru_maxrss is already reported in bytes.
                return Some(rss as u64);
            }
        }
        None
    }

    #[cfg(all(not(target_os = "linux"), not(target_os = "macos")))]
    {
        let mut usage = std::mem::MaybeUninit::<libc::rusage>::uninit();
        let rc = unsafe { libc::getrusage(libc::RUSAGE_SELF, usage.as_mut_ptr()) };
        if rc == 0 {
            let usage = unsafe { usage.assume_init() };
            let rss = usage.ru_maxrss;
            if rss > 0 {
                return Some((rss as u64).saturating_mul(1024));
            }
        }
        None
    }
}

fn read_thread_count() -> Option<u32> {
    #[cfg(target_os = "linux")]
    {
        let status = fs::read_to_string("/proc/self/status").ok()?;
        for line in status.lines() {
            if let Some(rest) = line.strip_prefix("Threads:") {
                return rest.trim().parse::<u32>().ok();
            }
        }
        None
    }

    #[cfg(target_os = "macos")]
    {
        let pid = std::process::id().to_string();
        let output = std::process::Command::new("ps")
            .args(["-o", "thcount=", "-p", pid.as_str()])
            .output()
            .ok()?;
        if !output.status.success() {
            return None;
        }

        let stdout = String::from_utf8(output.stdout).ok()?;
        stdout
            .lines()
            .find_map(|line| line.trim().parse::<u32>().ok())
    }

    #[cfg(not(any(target_os = "linux", target_os = "macos")))]
    {
        None
    }
}

fn read_open_fd_count() -> Option<u32> {
    for dir in ["/proc/self/fd", "/dev/fd"] {
        if let Ok(entries) = fs::read_dir(dir) {
            let count = entries.count();
            return u32::try_from(count).ok();
        }
    }
    None
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
    // Each run_task_worker thread opens its own connection via connect_to_db so
    // streaming callbacks and reducer calls stay isolated per task. The
    // trade-off is extra connection overhead under high concurrency.
    let ctx = connect_to_db(&config)?;
    let _network_thread = ctx.run_threaded();

    let (identity_timeout, identity_poll_interval) = worker_identity_wait_config();
    let wait_start = Instant::now();
    while ctx.try_identity().is_none() {
        if wait_start.elapsed() > identity_timeout {
            bail!(
                "Worker connection did not obtain identity within {}ms",
                identity_timeout.as_millis()
            );
        }
        thread::sleep(identity_poll_interval);
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

fn worker_identity_wait_config() -> (Duration, Duration) {
    let timeout_ms = env::var("AULE_WORKER_IDENTITY_TIMEOUT_MS")
        .ok()
        .and_then(|v| v.trim().parse::<u64>().ok())
        .filter(|v| *v > 0)
        .unwrap_or(5_000);
    let poll_ms = env::var("AULE_WORKER_IDENTITY_POLL_MS")
        .ok()
        .and_then(|v| v.trim().parse::<u64>().ok())
        .filter(|v| *v > 0)
        .unwrap_or(25);
    (
        Duration::from_millis(timeout_ms),
        Duration::from_millis(poll_ms),
    )
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
        let decision_set = llm
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

        conversation.push_assistant_message(decision_set.assistant_message);

        let mut executable_calls = Vec::new();
        let mut terminal_calls = Vec::new();

        for (
            index,
            ToolCall {
                tool_call_id,
                action,
            },
        ) in decision_set.tool_calls.into_iter().enumerate()
        {
            create_runtime_event(
                ctx,
                next_runtime_event_id(task.id, turn, &format!("tool-call-{index}")),
                task.id,
                turn,
                RuntimeEventType::ToolCall,
                format!("tool_call_id={tool_call_id}\naction={action:?}"),
            );

            let call = IndexedToolCall {
                index,
                tool_call_id,
                action,
            };

            if matches!(
                &call.action,
                AgentAction::Finish { .. } | AgentAction::Fail { .. }
            ) {
                terminal_calls.push(call);
            } else {
                executable_calls.push(call);
            }
        }

        let mut tool_results =
            execute_non_terminal_tool_calls(ctx, config, task.id, turn, executable_calls);
        tool_results.sort_by_key(|(index, _, _)| *index);

        for (_, tool_call_id, tool_result) in tool_results {
            conversation.push_tool_result(&tool_call_id, tool_result);
        }

        let selected_terminal_index = terminal_calls
            .iter()
            .find(|call| matches!(&call.action, AgentAction::Finish { .. }))
            .map(|call| call.index)
            .or_else(|| {
                terminal_calls
                    .iter()
                    .find(|call| matches!(&call.action, AgentAction::Fail { .. }))
                    .map(|call| call.index)
            });

        for call in &terminal_calls {
            if Some(call.index) != selected_terminal_index {
                let skipped = "Skipped: another terminal action was prioritized.".to_string();
                conversation.push_tool_result(&call.tool_call_id, skipped.clone());
                create_runtime_event(
                    ctx,
                    next_runtime_event_id(task.id, turn, &format!("tool-{}-result", call.index)),
                    task.id,
                    turn,
                    RuntimeEventType::ToolResult,
                    format!("tool_call_id={}\nresult={skipped}", call.tool_call_id),
                );
            }
        }

        if let Some(terminal_index) = selected_terminal_index {
            let Some(terminal_call) = terminal_calls
                .into_iter()
                .find(|call| call.index == terminal_index)
            else {
                warn!(
                    "Selected terminal index {} missing for task {} turn {}",
                    terminal_index, task.id, turn
                );
                continue;
            };

            match terminal_call.action {
                AgentAction::Finish { result } => {
                    post_observation(ctx, task.id, ObservationKind::Result, result.clone());
                    let tool_result = "task completed".to_string();
                    conversation.push_tool_result(&terminal_call.tool_call_id, tool_result.clone());
                    create_runtime_event(
                        ctx,
                        next_runtime_event_id(
                            task.id,
                            turn,
                            &format!("tool-{terminal_index}-result"),
                        ),
                        task.id,
                        turn,
                        RuntimeEventType::ToolResult,
                        format!(
                            "tool_call_id={}\nresult={tool_result}",
                            terminal_call.tool_call_id
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
                    let tool_result = "task failed".to_string();
                    conversation.push_tool_result(&terminal_call.tool_call_id, tool_result.clone());
                    create_runtime_event(
                        ctx,
                        next_runtime_event_id(
                            task.id,
                            turn,
                            &format!("tool-{terminal_index}-result"),
                        ),
                        task.id,
                        turn,
                        RuntimeEventType::ToolResult,
                        format!(
                            "tool_call_id={}\nresult={tool_result}",
                            terminal_call.tool_call_id
                        ),
                    );
                    ctx.reducers
                        .fail_task(task.id, error)
                        .with_context(|| format!("Failed to mark task {} as failed", task.id))?;
                    info!("Marked task #{} as failed", task.id);
                    return Ok(());
                }
                _ => {
                    warn!(
                        "Unexpected non-terminal action selected as terminal in task {} turn {}",
                        task.id, turn
                    );
                }
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

fn execute_non_terminal_tool_calls(
    ctx: &DbConnection,
    config: &RuntimeConfig,
    task_id: u64,
    turn: u32,
    executable_calls: Vec<IndexedToolCall>,
) -> Vec<(usize, String, String)> {
    let mut tool_results = Vec::new();
    let shell_timeout = config.shell_timeout;
    let shell_output_limit_bytes = config.shell_output_limit_bytes;

    thread::scope(|scope| {
        let mut shell_jobs = Vec::new();

        for call in executable_calls {
            let IndexedToolCall {
                index,
                tool_call_id,
                action,
            } = call;

            match action {
                AgentAction::Shell { command } => {
                    info!("task #{} turn {} sh[{}]: {}", task_id, turn, index, command);

                    post_observation(
                        ctx,
                        task_id,
                        ObservationKind::Progress,
                        format!("Running command: `{command}`"),
                    );

                    let command_for_thread = command.clone();
                    let handle = scope.spawn(move || {
                        let run_result = shell::run_shell(
                            &command_for_thread,
                            shell_timeout,
                            shell_output_limit_bytes,
                        );
                        (command_for_thread, run_result)
                    });

                    shell_jobs.push((index, tool_call_id, command, handle));
                }
                AgentAction::Observe { kind, content } => {
                    let obs_kind = observe_kind_to_observation_kind(&kind);
                    post_observation(ctx, task_id, obs_kind, content);
                    let tool_result = format!("observation posted ({})", kind.as_str());
                    create_runtime_event(
                        ctx,
                        next_runtime_event_id(task_id, turn, &format!("tool-{index}-result")),
                        task_id,
                        turn,
                        RuntimeEventType::ToolResult,
                        format!("tool_call_id={tool_call_id}\nresult={tool_result}"),
                    );
                    tool_results.push((index, tool_call_id, tool_result));
                }
                AgentAction::Status { content } => {
                    post_observation(
                        ctx,
                        task_id,
                        ObservationKind::Progress,
                        format!("status: {content}"),
                    );
                    let tool_result = "status recorded".to_string();
                    create_runtime_event(
                        ctx,
                        next_runtime_event_id(task_id, turn, &format!("tool-{index}-result")),
                        task_id,
                        turn,
                        RuntimeEventType::ToolResult,
                        format!("tool_call_id={tool_call_id}\nresult={tool_result}"),
                    );
                    tool_results.push((index, tool_call_id, tool_result));
                }
                AgentAction::Finish { .. } | AgentAction::Fail { .. } => {
                    warn!(
                        "execute_non_terminal_tool_calls received terminal action at index {}",
                        index
                    );
                }
            }
        }

        for (index, tool_call_id, command, handle) in shell_jobs {
            let (returned_command, shell_result) = match handle.join() {
                Ok(v) => v,
                Err(_) => {
                    let panic_result = "shell execution panicked".to_string();
                    create_runtime_event(
                        ctx,
                        next_runtime_event_id(task_id, turn, &format!("tool-{index}-shell-output")),
                        task_id,
                        turn,
                        RuntimeEventType::ShellOutput,
                        format!("command: {command}\n\nerror: {panic_result}"),
                    );
                    post_observation(
                        ctx,
                        task_id,
                        ObservationKind::Error,
                        format!("Command execution panicked: `{command}`"),
                    );
                    create_runtime_event(
                        ctx,
                        next_runtime_event_id(task_id, turn, &format!("tool-{index}-result")),
                        task_id,
                        turn,
                        RuntimeEventType::ToolResult,
                        format!("tool_call_id={tool_call_id}\nresult={panic_result}"),
                    );
                    tool_results.push((index, tool_call_id, panic_result));
                    continue;
                }
            };

            match shell_result {
                Ok(result) => {
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
                        next_runtime_event_id(task_id, turn, &format!("tool-{index}-shell-output")),
                        task_id,
                        turn,
                        RuntimeEventType::ShellOutput,
                        format!("command: {returned_command}\n\n{outcome}"),
                    );

                    let (kind, summary) = if result.timed_out {
                        (
                            ObservationKind::Error,
                            format!(
                                "Command timed out after {}ms: `{returned_command}`",
                                result.duration_ms
                            ),
                        )
                    } else if result.exit_code.unwrap_or(1) != 0 {
                        (
                            ObservationKind::Error,
                            format!(
                                "Command failed (exit_code={:?}, {}ms): `{returned_command}`",
                                result.exit_code, result.duration_ms
                            ),
                        )
                    } else {
                        (
                            ObservationKind::Finding,
                            format!(
                                "Command succeeded in {}ms: `{returned_command}`",
                                result.duration_ms
                            ),
                        )
                    };
                    post_observation(ctx, task_id, kind, summary);

                    create_runtime_event(
                        ctx,
                        next_runtime_event_id(task_id, turn, &format!("tool-{index}-result")),
                        task_id,
                        turn,
                        RuntimeEventType::ToolResult,
                        format!(
                            "tool_call_id={tool_call_id}\nresult=exit_code={:?} timed_out={} duration_ms={}",
                            result.exit_code, result.timed_out, result.duration_ms
                        ),
                    );

                    tool_results.push((index, tool_call_id, outcome));
                }
                Err(err) => {
                    let err_text = format!("shell execution error: {err:#}");
                    create_runtime_event(
                        ctx,
                        next_runtime_event_id(task_id, turn, &format!("tool-{index}-shell-output")),
                        task_id,
                        turn,
                        RuntimeEventType::ShellOutput,
                        format!("command: {returned_command}\n\nerror: {err_text}"),
                    );
                    post_observation(
                        ctx,
                        task_id,
                        ObservationKind::Error,
                        format!("Failed to run command `{returned_command}`: {err:#}"),
                    );
                    create_runtime_event(
                        ctx,
                        next_runtime_event_id(task_id, turn, &format!("tool-{index}-result")),
                        task_id,
                        turn,
                        RuntimeEventType::ToolResult,
                        format!("tool_call_id={tool_call_id}\nresult={err_text}"),
                    );
                    tool_results.push((index, tool_call_id, err_text));
                }
            }
        }
    });

    tool_results
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
    if turn == 0 {
        warn!("create_runtime_event called with invalid turn 0 for task {task_id}");
        return;
    }

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
