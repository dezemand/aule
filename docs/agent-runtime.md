# Agent Runtime

Each agent is a K8s pod with an isolated workspace, curated CLI set, and connections to SpacetimeDB and the LLM Router.

## Platform Tools

Six tools available in the LLM context. Everything else goes through `shell`.

| Tool | Purpose |
|------|---------|
| `aule_observe` | Post an observation to SpacetimeDB |
| `aule_suggest` | Propose an action for approval |
| `aule_memory` | Read/write agent memory layers |
| `aule_status` | Report progress, blockers, state |
| `aule_request` | Request LLM completion via router |
| `shell` | Execute any shell command |

## Standard CLIs (base image)

Available to all agent types:

- **File ops:** `ls`, `cat`, `find`, `tree`, `cp`, `mv`, `mkdir`, `diff`, `tar`
- **Text:** `grep`, `rg`, `sed`, `awk`, `jq`, `yq`, `xsv`
- **Network:** `curl`, `wget`
- **System:** `env`, `which`, `date`, `timeout`, `tee`
- **Data:** `sqlite3`, `duckdb`

## Capability-Specific CLIs

| Agent Type | Additional CLIs |
|------------|----------------|
| Builder | `git`, `cargo`, `rustc`, `rustfmt`, `clippy`, `spacetime`, `make`, `node`, `npm` |
| Research | `playwright`, `trafilatura`, `python3`, `pandoc` |
| Data | `python3`, `duckdb`, `pandas`, `pyarrow` |
| Ops | `spacetime`, `git`, `kubectl` (external clusters), `terraform` |
| External | `psql`, `mysql`, `redis-cli`, `docker`, `aws`, `gcloud` |

## Access Rules

- **BLOCKED:** Aule's own K8s cluster, Aule's SpacetimeDB (direct bypass)
- **ALLOWED:** External systems the agent is tasked to work with
- Credentials scoped per task, mounted from K8s Secrets, destroyed with the pod

## Browser Access

Layered approach based on agent needs:

| Level | Available to | Mechanism | Handles |
|-------|-------------|-----------|---------|
| `curl` / `wget` | All agents | Base image CLI | APIs, static pages |
| Playwright CLI | Research agents | In agent image | JS-rendered content, screenshots |
| Browser sidecar | Interactive agents | Sidecar container | Multi-step web automation |
| Browser pool | At scale | Cluster service | Shared browser instances |

## Shell Safety

- All commands wrapped in `timeout`
- Working directory confined to `/workspace/work/`
- Output truncated at 50KB
- Every invocation logged (command, exit code, output size, duration)
- Destructive command blocklist
- Container-level isolation (non-root, no host mounts, resource limits)

## Runtime Startup Flow

```
1. Pod starts with agent type version's image
2. Agent process reads its agent_type_version_id from env
3. Connects to SpacetimeDB
4. Fetches its system prompt and config from agent_type_versions table
5. Calls register_runtime reducer -> gets AgentRuntime row
6. Subscribes to task assignments
7. Enters idle loop, heartbeating
8. Task arrives -> start_task -> credentials mounted -> execute -> complete_task -> idle
```

## Agent Process Loop (during task)

```
1. Receive task assignment with description, permissions, budget
2. Fetch system prompt from SpacetimeDB (version-specific)
3. Plan (LLM call via router with context + memory)
4. Execute (shell commands, file manipulation, tool use)
5. Report (observations, status, memory updates -> SpacetimeDB)
6. Loop 3-5 until task complete or budget exhausted
7. Complete task, return to idle
```

## Source Layout

```
packages/aule-runtime/
└── src/
    ├── main.rs                lifecycle: register, subscribe, idle loop
    ├── task.rs                task pickup, execution loop, completion
    ├── executor.rs            shell command execution with safety
    ├── llm_client.rs          talks to router's standard API
    ├── spacetimedb_client.rs  subscriptions, memory ops, observations
    ├── reasoning.rs           prompt construction, tool marshaling
    └── tools/
        ├── observe.rs
        ├── suggest.rs
        ├── memory.rs
        ├── status.rs
        ├── request.rs
        └── shell.rs
```
