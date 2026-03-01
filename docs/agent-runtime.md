# Agent Runtime

Each agent runs on a K8s pod (AgentRuntime) with a shared workspace filesystem, curated CLI set, and connections to SpacetimeDB and the LLM Router. Runtimes are generic compute — they host agents, which are instances of AgentTypeVersions.

## Runtime vs Agent

- **AgentRuntime** — a warm K8s pod in a workspace. Has capabilities from its container image. Persists across agent launches. Managed as a pool by the K8s operator.
- **Agent** — an instance of an AgentTypeVersion, launched on a runtime. Defines behavior (system prompt, config). One agent per runtime at a time.

## Platform Tools

Six tools available in the LLM context (~1,800 tokens total). Everything else goes through `shell`.

| Tool | Purpose |
|------|---------|
| `aule_observe` | Post an observation to SpacetimeDB |
| `aule_suggest` | Propose an action for approval |
| `aule_memory` | Read/write agent memory layers |
| `aule_status` | Report progress, update plan steps |
| `aule_request` | Request LLM completion via router |
| `shell` | Execute any shell command |

The agent discovers available CLIs by exploring its environment (`which`, `ls /usr/local/bin/`, `--help`).

## Standard CLIs (base image)

Available to all agent types:

- **File ops:** `ls`, `cat`, `find`, `tree`, `cp`, `mv`, `mkdir`, `diff`, `tar`
- **Text:** `grep`, `rg`, `sed`, `awk`, `jq`, `yq`, `xsv`
- **Network:** `curl`, `wget`
- **System:** `env`, `which`, `date`, `timeout`, `tee`
- **Data:** `sqlite3`, `duckdb`

## Capability-Specific CLIs

| Image | Additional CLIs |
|-------|----------------|
| `aule-builder` | `git`, `cargo`, `rustc`, `rustfmt`, `clippy`, `spacetime`, `make`, `node`, `npm` |
| `aule-research` | `playwright`, `trafilatura`, `python3`, `pandoc` |
| `aule-data` | `python3`, `duckdb`, `pandas`, `pyarrow` |
| `aule-ops` | `spacetime`, `git`, `kubectl`, `terraform` |

## Access Rules

- **BLOCKED:** Aulë's own K8s cluster, Aulë's SpacetimeDB (direct bypass)
- **ALLOWED:** External systems the agent is tasked to work with
- Credentials scoped per task, mounted from K8s Secrets, revoked on agent stop

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
- Working directory confined to `/workspace/`
- Output truncated at configurable limit (default 50KB)
- Every invocation logged (command, exit code, output size, duration)
- Destructive command blocklist (pattern-matched)
- Container-level isolation (non-root, no host mounts, resource limits)

## Runtime Startup Flow

```text
1. Operator creates warm pod in workspace with PVC mounted
2. Runtime process connects to SpacetimeDB
3. Calls register_runtime reducer → gets AgentRuntime row (status: "available")
4. Enters idle loop, heartbeating
5. Waits for agent launch assignment
```

## Agent Launch Flow

```text
1. Launch request: agent_type_version_id + workspace_id
2. System resolves required_capabilities from agent_type
3. Finds available runtime in workspace with matching capabilities
4. Creates Agent row (status: "starting"), runtime status → "occupied"
5. Runtime fetches system prompt from agent_type_versions table
6. Agent status → "idle"
7. Agent subscribes to task assignments, waits for work
```

## Agent Process Loop (during task)

```text
1. Receive task with description, context, budget
2. Load relevant memory from workspace
3. If retrying: load previous attempts' observations and failure reasons
4. Plan: LLM call → produce TaskPlan with TaskSteps
5. Execute steps:
   a. Update current step status → "in_progress"
   b. Reason (LLM call via router)
   c. Act (shell commands, tool use)
   d. Observe (post observations to SpacetimeDB)
   e. Update step status → "completed" or "failed"
   f. May add new steps to plan during execution
6. On completion: post TaskResult, attempt status → "completed"
7. On failure: record failure_reason, attempt status → "failed"
8. Return to idle, wait for next task
```

## Agent Stop Flow

```text
1. Finish or abort current work
2. Agent status → "stopped"
3. Operator cleans scratch dir (/workspace/scratch/agent-{id}/)
4. Credentials revoked
5. Runtime status → "available", returned to warm pool
```

## Source Layout

```text
packages/aule-runtime/
└── src/
    ├── main.rs                lifecycle: boot, register, idle loop
    ├── agent.rs               agent launch, system prompt fetch, config
    ├── task.rs                task pickup, attempt loop, plan management
    ├── executor.rs            shell command execution with safety
    ├── llm_client.rs          talks to router's standard API
    ├── spacetimedb_client.rs  subscriptions, state updates
    ├── reasoning.rs           prompt construction, tool marshaling
    └── tools/
        ├── observe.rs
        ├── suggest.rs
        ├── memory.rs
        ├── status.rs
        ├── request.rs
        └── shell.rs
```
