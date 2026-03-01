# Running Aulë

How to run every component locally. You need four things in separate terminals: SpacetimeDB, the WASM module published to it, the agent runtime, and optionally the frontend and/or test client.

## Prerequisites

Make sure you have everything installed (see [README](../README.md)):

- Rust (stable), Bun, just, SpacetimeDB CLI, Docker

Then run initial setup once:

```sh
cp .env.template .env    # create your local env file
just setup               # install deps, generate bindings, build workspace
```

---

## 1. SpacetimeDB

Start the local SpacetimeDB instance via Docker Compose:

```sh
just db
```

This runs `clockworklabs/spacetime:v2.0.2` on port 3000, with data persisted to `.local/spacetimedb_data/`.

Other commands:

```sh
just db-logs     # tail SpacetimeDB container logs
just db-stop     # stop the container
```

## 2. Publish the module

Build the WASM module and publish it to the local SpacetimeDB instance:

```sh
just publish
```

To wipe all data and start fresh:

```sh
just publish -- --delete-data
```

After publishing, regenerate the TypeScript and Rust client bindings:

```sh
just generate
```

## 3. Agent Runtime

The runtime is a Rust process that connects to SpacetimeDB, registers itself, picks up tasks, reasons via an LLM, executes shell commands, and reports observations.

### Configuration

Add these to your `.env` (alongside the existing `SPACETIMEDB_URI` and `SPACETIMEDB_DB_NAME`):

```env
# Required
AULE_AGENT_VERSION=0.1.0
OPENAI_API_KEY=<your-api-key>

# Optional (shown with defaults)
AULE_RUNTIME_NAME=runtime-01
OPENAI_MODEL=gpt-4.1-mini
```

The runtime reads `.env` indirectly — it uses `std::env`, so either source the file or export the vars. The simplest approach:

```sh
set -a && source .env && set +a
```

### Tuning

```env
AULE_HEARTBEAT_SECONDS=10           # heartbeat interval
AULE_RESOURCE_SAMPLE_SECONDS=30     # resource telemetry interval
AULE_SHELL_TIMEOUT_SECONDS=30       # max shell command duration
AULE_SHELL_OUTPUT_LIMIT_BYTES=50000 # max shell output captured
AULE_MAX_STEPS_PER_TASK=24          # max reasoning steps per task
```

### K8s metadata (optional, auto-detected in pods)

These are only relevant when running inside Kubernetes. The runtime auto-detects the K8s environment via `KUBERNETES_SERVICE_HOST`.

```env
AULE_RUNTIME_IMAGE=...
AULE_RUNTIME_IMAGE_DIGEST=...
AULE_K8S_CLUSTER=...
AULE_GIT_SHA=...
```

### Run

```sh
cargo run -p aule-runtime
```

The runtime will connect to SpacetimeDB, register, and idle waiting for task assignments.

## 4. Frontend

The web dashboard connects to SpacetimeDB via WebSocket and shows real-time agent/task/observation state.

```sh
just dev
```

This starts the Bun dev server (with HMR) from `app/`. Open the URL shown in the terminal.

Pages:

- `/` — Dashboard overview (runtimes, tasks)
- `/tasks` — Task list
- `/tasks/:id` — Task details with observation feed
- `/agent-types` — Agent type and version management

## 5. Test Client (optional)

An interactive CLI for manually creating agent types, versions, tasks, and assigning them to runtimes. Useful for testing without the frontend.

```sh
cargo run -p aule-client
```

> **Note:** The client currently has the SpacetimeDB host and database name hardcoded to `http://localhost:3000` and `aule`. It does not read `.env`.

### Commands

```text
/type <name> <description>                    Create an agent type
/version <type_id> <ver> <system_prompt>      Create a type version
/activate <version_id>                        Activate a version
/task <type_id> <title> -- <description>      Create a task
/assign <task_id> <runtime_name>              Assign task to a runtime
/observe <task_id> <kind> <text>              Post an observation
/complete <task_id> <result>                  Complete a task
/fail <task_id> <error>                       Fail a task
/register <name>                              Register as a runtime (for testing)
/deregister                                   Deregister this client as runtime
/heartbeat                                    Send heartbeat
/status                                       Show current state
/help                                         Show all commands
/quit                                         Disconnect
```

### End-to-end example

With SpacetimeDB running, the module published, and the runtime started:

```text
/type builder "General coding agent"
/version 1 0.1.0 "You are a practical coding agent."
/activate 1
/task 1 "Inspect repo" -- "Run ls and summarize the top-level layout"
/assign 1 runtime-01
```

The runtime picks up the task, reasons about it, executes shell commands, posts observations, and completes or fails the task. Watch observations stream in on both the client and the frontend.

---

## Querying the database

```sh
spacetime sql --server http://localhost:3000 aule "SELECT * FROM agent_runtime"
spacetime sql --server http://localhost:3000 aule "SELECT * FROM agent_task"
spacetime sql --server http://localhost:3000 aule "SELECT * FROM agent_type"
spacetime sql --server http://localhost:3000 aule "SELECT * FROM observation"
```

## Viewing module logs

```sh
spacetime logs --server http://localhost:3000 aule
```

## Notes

- The runtime selects its system prompt by `AULE_AGENT_VERSION` and requires that version to be `Active`.
- `aule_status` is currently represented as progress observations.
- Shell commands run via `sh -lc` with timeout and output truncation.
