# Phase 2 Runbook

This runbook starts the local Phase 2 loop:

- SpacetimeDB module running locally
- Rust test client to create/assign tasks
- `aule-runtime` process using OpenAI for reasoning

## 1) Start SpacetimeDB and publish module

```sh
just db
just publish
just generate
```

To publish with a clean database:

```sh
just publish -- --delete-data
```

## 2) Build runtime

```sh
cargo build -p aule-runtime
```

## 3) Start runtime

Add runtime-specific vars to your `.env` (or export in a new shell):

```sh
AULE_RUNTIME_NAME="runtime-01"
AULE_AGENT_VERSION="0.1.0"
OPENAI_API_KEY="<your-api-key>"
OPENAI_MODEL="gpt-4.1-mini"
```

`SPACETIMEDB_URI` and `SPACETIMEDB_DB_NAME` are already in `.env` from the project setup.

Optional tuning:

```sh
AULE_HEARTBEAT_SECONDS="10"
AULE_RESOURCE_SAMPLE_SECONDS="30"
AULE_SHELL_TIMEOUT_SECONDS="30"
AULE_SHELL_OUTPUT_LIMIT_BYTES="50000"
AULE_MAX_STEPS_PER_TASK="24"
```

Run:

```sh
cargo run -p aule-runtime
```

## 4) Create data and assign tasks

In another shell, run the test client:

```sh
cargo run -p aule-client
```

Inside the client:

```text
/type builder "General coding agent"
/version 1 0.1.0 "You are a practical coding agent."
/activate 1
/task 1 "Inspect repo" -- "Run ls and summarize the top-level layout"
/assign 1 runtime-01
```

The runtime should start the task, post observations, and complete or fail it.

## Notes

- Phase 2 runtime selects prompt by `AULE_AGENT_VERSION` and requires that version to be `Active`.
- `aule_status` is currently represented as progress observations.
- `sh` commands run via `sh -lc` with timeout and output truncation.
