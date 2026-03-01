//! Aule Rust Test Client -- Phase 1: Minimal Coordination Layer
//!
//! Exercises the full Phase 1 flow:
//! 1. Connect to SpacetimeDB
//! 2. Subscribe to all tables
//! 3. Interactive commands to test the coordination layer

use aule_spacetimedb_client::*;
use spacetimedb_sdk::{DbContext, Error, Identity, Table, TableWithPrimaryKey, credentials};

const HOST: &str = "http://localhost:3000";
const DB_NAME: &str = "aule";

fn main() {
    env_logger::init();

    println!("=== Aule Phase 1 -- Test Client ===");
    println!("Connecting to {HOST} / {DB_NAME}...");

    let ctx = connect_to_db();
    register_callbacks(&ctx);
    subscribe_to_tables(&ctx);

    ctx.run_threaded();
    user_input_loop(&ctx);
}

// ---------------------------------------------------------------------------
// Connection
// ---------------------------------------------------------------------------

fn creds_store() -> credentials::File {
    credentials::File::new("aule-phase1")
}

fn connect_to_db() -> DbConnection {
    DbConnection::builder()
        .on_connect(on_connected)
        .on_connect_error(on_connect_error)
        .on_disconnect(on_disconnected)
        .with_token(creds_store().load().expect("Error loading credentials"))
        .with_database_name(DB_NAME)
        .with_uri(HOST)
        .build()
        .expect("Failed to connect")
}

fn on_connected(_ctx: &DbConnection, identity: Identity, token: &str) {
    if let Err(e) = creds_store().save(token) {
        eprintln!("Failed to save credentials: {e:?}");
    }
    println!("[connected] identity = {}", identity.to_hex());
}

fn on_connect_error(_ctx: &ErrorContext, err: Error) {
    eprintln!("[connect error] {err:?}");
    std::process::exit(1);
}

fn on_disconnected(_ctx: &ErrorContext, err: Option<Error>) {
    if let Some(err) = err {
        eprintln!("[disconnected] {err}");
        std::process::exit(1);
    } else {
        println!("[disconnected]");
        std::process::exit(0);
    }
}

// ---------------------------------------------------------------------------
// Callbacks
// ---------------------------------------------------------------------------

fn register_callbacks(ctx: &DbConnection) {
    // Runtime events
    ctx.db.agent_runtime().on_insert(|_ctx, rt| {
        println!("[runtime registered] {} (status={:?})", rt.name, rt.status);
    });
    ctx.db.agent_runtime().on_update(|_ctx, old, new| {
        if old.status != new.status {
            println!(
                "[runtime {}] {:?} -> {:?}",
                new.name, old.status, new.status
            );
        }
    });
    ctx.db.agent_runtime().on_delete(|_ctx, rt| {
        println!("[runtime deregistered] {}", rt.name);
    });

    // Task events
    ctx.db.agent_task().on_insert(|_ctx, task| {
        println!("[task created] #{}: {}", task.id, task.title);
    });
    ctx.db.agent_task().on_update(|_ctx, old, new| {
        if old.status != new.status {
            println!("[task #{}] {:?} -> {:?}", new.id, old.status, new.status);
        }
    });

    // Agent type events
    ctx.db.agent_type().on_insert(|_ctx, at| {
        println!("[agent type created] #{}: {}", at.id, at.name);
    });

    // Version events
    ctx.db.agent_type_version().on_insert(|_ctx, v| {
        println!(
            "[version created] #{}: type={} v{}",
            v.id, v.agent_type_id, v.version
        );
    });
    ctx.db.agent_type_version().on_update(|_ctx, old, new| {
        if old.status != new.status {
            println!("[version #{}] {:?} -> {:?}", new.id, old.status, new.status);
        }
    });

    // Observation events
    ctx.db.observation().on_insert(|_ctx, obs| {
        println!(
            "[observation] task=#{} ({:?}): {}",
            obs.task_id, obs.kind, obs.content
        );
    });
}

// ---------------------------------------------------------------------------
// Subscriptions
// ---------------------------------------------------------------------------

fn subscribe_to_tables(ctx: &DbConnection) {
    ctx.subscription_builder()
        .on_applied(on_sub_applied)
        .on_error(on_sub_error)
        .subscribe([
            "SELECT * FROM agent_runtime",
            "SELECT * FROM agent_task",
            "SELECT * FROM agent_type",
            "SELECT * FROM agent_type_version",
            "SELECT * FROM observation",
        ]);
}

fn on_sub_applied(ctx: &SubscriptionEventContext) {
    // Show current state
    println!();
    println!("--- Current State ---");

    let types: Vec<_> = ctx.db.agent_type().iter().collect();
    println!("Agent types: {}", types.len());
    for t in &types {
        println!("  #{}: {} - {}", t.id, t.name, t.description);
    }

    let versions: Vec<_> = ctx.db.agent_type_version().iter().collect();
    println!("Versions: {}", versions.len());
    for v in &versions {
        println!(
            "  #{}: type={} v{} ({:?})",
            v.id, v.agent_type_id, v.version, v.status
        );
    }

    let runtimes: Vec<_> = ctx.db.agent_runtime().iter().collect();
    println!("Runtimes: {}", runtimes.len());
    for r in &runtimes {
        println!("  {} - status={:?}", r.name, r.status);
    }

    let tasks: Vec<_> = ctx.db.agent_task().iter().collect();
    println!("Tasks: {}", tasks.len());
    for t in &tasks {
        println!(
            "  #{}: {} ({:?}) assigned={:?}",
            t.id,
            t.title,
            t.status,
            t.assigned_runtime
                .as_ref()
                .map(|i| i.to_hex().to_string()[..16].to_string())
        );
    }

    let observations: Vec<_> = ctx.db.observation().iter().collect();
    println!("Observations: {}", observations.len());

    println!("--- End State ---");
    println!();
    print_help();
}

fn on_sub_error(_ctx: &ErrorContext, err: Error) {
    eprintln!("[subscription error] {err}");
    std::process::exit(1);
}

// ---------------------------------------------------------------------------
// User input
// ---------------------------------------------------------------------------

fn print_help() {
    println!("Commands:");
    println!("  /type <name> <description>        - Create an agent type");
    println!("  /version <type_id> <ver> <prompt>  - Create a type version");
    println!("  /activate <version_id>             - Activate a version");
    println!("  /register <name>                   - Register as a runtime");
    println!("  /deregister                        - Deregister this runtime");
    println!("  /heartbeat                         - Send heartbeat");
    println!("  /task <type_id> <title> -- <desc>  - Create a task");
    println!("  /assign <task_id> <runtime_name>   - Assign task to runtime");
    println!("  /start <task_id>                   - Start assigned task");
    println!("  /observe <task_id> <kind> <text>   - Post observation");
    println!("  /complete <task_id> <result>       - Complete task");
    println!("  /fail <task_id> <error>            - Fail task");
    println!("  /status                            - Show current state");
    println!("  /quit                              - Disconnect");
}

fn user_input_loop(ctx: &DbConnection) {
    for line in std::io::stdin().lines() {
        let Ok(line) = line else { break };
        let line = line.trim().to_string();
        if line.is_empty() {
            continue;
        }

        if let Err(e) = handle_command(ctx, &line) {
            eprintln!("[error] {e}");
        }
    }
}

fn handle_command(ctx: &DbConnection, line: &str) -> Result<(), String> {
    let parts: Vec<&str> = line.splitn(2, ' ').collect();
    let cmd = parts[0];
    let args = parts.get(1).unwrap_or(&"");

    match cmd {
        "/quit" => {
            let _ = ctx.disconnect();
            std::process::exit(0);
        }
        "/help" => {
            print_help();
        }
        "/type" => {
            let parts: Vec<&str> = args.splitn(2, ' ').collect();
            if parts.len() < 2 {
                return Err("Usage: /type <name> <description>".into());
            }
            ctx.reducers
                .create_agent_type(parts[0].to_string(), parts[1].to_string())
                .map_err(|e| format!("{e}"))?;
        }
        "/version" => {
            let parts: Vec<&str> = args.splitn(3, ' ').collect();
            if parts.len() < 3 {
                return Err("Usage: /version <type_id> <ver> <system_prompt>".into());
            }
            let type_id: u64 = parts[0].parse().map_err(|_| "Invalid type_id")?;
            ctx.reducers
                .create_agent_type_version(type_id, parts[1].to_string(), parts[2].to_string())
                .map_err(|e| format!("{e}"))?;
        }
        "/activate" => {
            let version_id: u64 = args.parse().map_err(|_| "Invalid version_id")?;
            ctx.reducers
                .activate_agent_type_version(version_id)
                .map_err(|e| format!("{e}"))?;
        }
        "/register" => {
            if args.trim().is_empty() {
                return Err("Usage: /register <name>".into());
            }
            ctx.reducers
                .register_runtime(args.trim().to_string())
                .map_err(|e| format!("{e}"))?;
        }
        "/deregister" => {
            ctx.reducers
                .deregister_runtime()
                .map_err(|e| format!("{e}"))?;
        }
        "/heartbeat" => {
            ctx.reducers.heartbeat().map_err(|e| format!("{e}"))?;
        }
        "/task" => {
            // Parse: <type_id> <title> -- <description>
            let parts: Vec<&str> = args.splitn(2, ' ').collect();
            if parts.len() < 2 {
                return Err("Usage: /task <type_id> <title> -- <description>".into());
            }
            let type_id: u64 = parts[0].parse().map_err(|_| "Invalid type_id")?;
            let rest = parts[1];
            let (title, desc) = if let Some(idx) = rest.find(" -- ") {
                (&rest[..idx], &rest[idx + 4..])
            } else {
                (rest, "")
            };
            ctx.reducers
                .create_task(type_id, title.to_string(), desc.to_string())
                .map_err(|e| format!("{e}"))?;
        }
        "/assign" => {
            let parts: Vec<&str> = args.splitn(2, ' ').collect();
            if parts.len() < 2 {
                return Err("Usage: /assign <task_id> <runtime_name>".into());
            }
            let task_id: u64 = parts[0].parse().map_err(|_| "Invalid task_id")?;
            let runtime_name = parts[1].trim();
            let runtime = ctx
                .db
                .agent_runtime()
                .iter()
                .find(|r| r.name == runtime_name)
                .ok_or_else(|| format!("No runtime found with name '{runtime_name}'"))?;
            ctx.reducers
                .assign_task(task_id, runtime.identity)
                .map_err(|e| format!("{e}"))?;
        }
        "/start" => {
            let task_id: u64 = args.parse().map_err(|_| "Invalid task_id")?;
            ctx.reducers
                .start_task(task_id)
                .map_err(|e| format!("{e}"))?;
        }
        "/observe" => {
            let parts: Vec<&str> = args.splitn(3, ' ').collect();
            if parts.len() < 3 {
                return Err(
                    "Usage: /observe <task_id> <progress|finding|error|result> <text>".into(),
                );
            }
            let task_id: u64 = parts[0].parse().map_err(|_| "Invalid task_id")?;
            let kind = match parts[1] {
                "progress" => ObservationKind::Progress,
                "finding" => ObservationKind::Finding,
                "error" => ObservationKind::Error,
                "result" => ObservationKind::Result,
                _ => return Err("Kind must be: progress, finding, error, result".into()),
            };
            ctx.reducers
                .post_observation(task_id, kind, parts[2].to_string())
                .map_err(|e| format!("{e}"))?;
        }
        "/complete" => {
            let parts: Vec<&str> = args.splitn(2, ' ').collect();
            if parts.len() < 2 {
                return Err("Usage: /complete <task_id> <result>".into());
            }
            let task_id: u64 = parts[0].parse().map_err(|_| "Invalid task_id")?;
            ctx.reducers
                .complete_task(task_id, parts[1].to_string())
                .map_err(|e| format!("{e}"))?;
        }
        "/fail" => {
            let parts: Vec<&str> = args.splitn(2, ' ').collect();
            if parts.len() < 2 {
                return Err("Usage: /fail <task_id> <error>".into());
            }
            let task_id: u64 = parts[0].parse().map_err(|_| "Invalid task_id")?;
            ctx.reducers
                .fail_task(task_id, parts[1].to_string())
                .map_err(|e| format!("{e}"))?;
        }
        "/status" => {
            println!("--- Runtimes ---");
            for r in ctx.db.agent_runtime().iter() {
                println!(
                    "  {} ({:?}) last_hb={}",
                    r.name,
                    r.status,
                    r.last_heartbeat.to_micros_since_unix_epoch()
                );
            }
            println!("--- Tasks ---");
            for t in ctx.db.agent_task().iter() {
                println!("  #{}: {} ({:?})", t.id, t.title, t.status);
            }
        }
        _ => {
            return Err(format!("Unknown command: {cmd}. Type /help for commands."));
        }
    }

    Ok(())
}
