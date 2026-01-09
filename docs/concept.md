# Concept: Project Management Agents

This document defines the core concept, domain model, execution approach, and tool contracts for a system that runs specialised agents to complete work tasks (not only software development). The intent is to enable autonomous, safe, auditable progress (including overnight runs) across mixed work types: research, design, architecture, implementation, documentation, and management updates.

The system is designed for day-to-day use by developers/architects working in innovation, alongside engineers, project managers, product owners, managers, and other stakeholders.


## 1) Goals

### Primary goals
- Represent work as **Tasks** with clear directives and concrete outputs.
- Run **Agent Instances** in isolation (Kubernetes), across multiple nodes.
- Support multiple work domains: research, ideation/exploration, design, architecture, development, documentation, integration.
- Support multiple repositories/sources of truth per project (e.g. Git repo + wiki subset + internal pages).
- Produce **real artefacts**: markdown docs, design notes, code changes, decision records, briefings.
- Ensure outputs are **reviewable, reproducible, and auditable**.

### Non-goals (initially)
- Full “Kubernetes-native CRD” orchestration (too heavy for MVP).
- Full knowledge graph implementation (can come later).
- Full MCP (HTTP) external tool ecosystem (phase 2).
- Perfect autonomy: the system must support “Needs input” and reviewer gates.


## 2) Principles

### Everything is a task with an output contract
Every task must specify:
- Directive (what to do)
- Deliverables (what artefact(s) to produce)
- Definition of Done (DoD)
- Inputs (what sources/attachments to use)
- Constraints (audience, scope, time budget, etc.)
- Tool scopes (what the agent may use)

### Separation of duties
- Orchestrator (executor/coordinator) schedules and runs tasks.
- Worker agents produce outputs.
- Reviewer agents verify quality (DoD/evidence/policy) and create fix tasks if needed.
- Integrator agents land changes into shared systems (merge, publish).

### Repos vs knowledge
- Repos and wiki are sources of truth for artefacts.
- A future knowledge layer can index meaning and provenance. Early MVP can use “evidence references” without a full graph.

### Safe by default
- No silent edits to shared artefacts.
- Work happens in per-task branches and drafts; publication/merge requires review.


## 3) Domain model overview

We do **not** store a “Board” entity. The UI builds a board dynamically by grouping tasks by:
- `TaskType` (what kind of work)
- `TaskStage` (type-specific workflow step)
- `TaskStatus` (execution state)

### 3.1 TaskType
`TaskType` defines “what kind of task this is” and its stage workflow.

Examples:
- `exploration`
- `research`
- `architecture`
- `development`
- `documentation`
- `integration`

Each TaskType defines:
- An ordered list of `TaskStage` definitions (workflow)
- Optional default DoD templates and deliverable templates
- Optional baseline tool allowances (further constrained per-task)

### 3.2 TaskStage
`TaskStage` defines “which step this task is currently in” for a given TaskType.

Examples:
- Development stages: `plan -> implement -> review -> merge`
- Architecture stages: `draft -> technical_review -> publish`
- Exploration stages: `diverge -> converge -> propose`

Each TaskStage defines:
- Stage identity (key + order)
- Optional entry checks
- Exit checks / stage-specific DoD
- Which agent types are eligible to execute this stage
- Optional allowed tool grants for that stage (baseline)

### 3.3 TaskStatus (generic execution state)
Task status is orthogonal to stage and indicates runtime state:
- `ready`
- `running`
- `blocked`
- `done`
- `failed`
- `cancelled`

A task has:
- `type = development`
- `stage = review`
- `status = running`

### 3.4 Agent types and selection
Agent selection is based on `(TaskType, TaskStage)`.

An AgentType is eligible if:
- It supports the task’s `TaskType`
- It supports the task’s current `TaskStage`
- It satisfies project policies (tool scopes, risk, permissions)


## 4) Agent roles (catalogue)

These roles are capability bundles mapped to real work. Start small (7–9 roles), expand later.

### Planning & coordination
- **Planner**: turns goals into structured tasks + DoD.
- **Coordinator**: keeps flow, manages WIP, unblocks and orchestrates runs.

### Research & discovery
- **Researcher**: gathers and compares information, produces a cited research memo.
- **Explorer**: early ideation, hypothesis generation (explicitly non-authoritative).

### Design & architecture
- **Architect**: produces architecture doc, trade-offs, risks, open questions.
- **Designer (solution/process)**: process and workflow design, diagrams and rationale.

### Execution
- **Implementer**: makes concrete changes to artefacts (docs/code) on a per-task branch.
- **Integrator**: publishes/merges approved work into shared systems.

### Quality & governance
- **Reviewer**: verifies DoD, evidence, consistency. Approves or creates fix tasks.
- **Risk & Compliance Checker**: flags policy, organisational, legal or operational risks.

### Product & management support
- **Product Analyst**: frames work in product terms, requirements, impact.
- **Manager Briefing Agent**: produces concise run/week briefings.

### Human interaction
- **Facilitator**: asks structured questions when info is missing.
- **Moderator**: runs discussions and frames decision candidates.


## 5) Execution architecture (MVP)

### 5.1 Control-plane (source of truth)
- **Project OS API** (Go): manages projects, tasks, types/stages, agents, runs, artefacts, policies.
- **Postgres DB**: source of truth, including:
  - tasks (type/stage/status/spec/result)
  - task attempts (job name, timestamps, exit codes)
  - events (for WS ordering/replay)

### 5.2 Realtime UI communication
Primary UI communication is via **WebSocket**, used as a single channel for:
- Commands (client -> server RPC style)
- Events (server -> client pushes)

Keep REST only for:
- auth/bootstrap flows (OIDC redirect patterns)
- large artefact upload/download (pre-signed URLs)
- health/debug endpoints (optional)

WS message envelope supports:
- request/response correlation (`id` + `reply_to`)
- idempotency (`idempotency_key`)
- ordering/replay (`seq`)
- correlation IDs (project/task/run/agent/attempt)

### 5.3 Executor (DB -> Kubernetes sync)
We do not use CRDs/controllers in MVP. Instead:
- DB is source of truth.
- Executor reconciles DB tasks into Kubernetes Jobs.

Executor responsibilities:
- Claim ready tasks using DB lease/lock (`claimed_by`, `lease_until`, `attempt_id`)
- Create K8s Job using the correct agent image (Go or Python runtime)
- Track Job metadata in DB (job/pod name)
- Finalise task based on agent callback (primary) + K8s status (fallback)
- Emit events to WS stream

### 5.4 Agent runtimes
Two runtime families:
- **Go agents**: fast, deterministic, orchestration/integration, structured transformations.
- **Python agents**: heavier data/document parsing, analysis, advanced libraries (not MVP).

All agents implement the same Task protocol (spec -> result) and run as isolated containers.


## 6) Agent callback mechanism (reporting)

### Primary: HTTP callback to API
Agent starts with the following environment variables:
- `API_BASE_URL`: base URL of the Project OS API
- `TASK_ID`: current task identifier
- `AUTH_TOKEN`: bearer token for authentication

Agent runner does:
1. `GET /tasks/{id}` -- fetches the full task description, including what agent type to run.
2. Executes
3. Per step reporting back to API:
   - `POST /tasks/{id}/progress` -- intermediate progress updates (optional)
   - `POST /tasks/{id}/result` -- final result with artefacts, status
4. Exit

Robustness:
- API accepts results only for current `attempt_id`
- idempotency per attempt prevents duplicates

### Fallback: executor reconciles K8s job status
Executor periodically checks job/pod state:
- If job finished but no result callback arrived, mark as failed and attach diagnostics.


## 7) Workspace and agent context

Agent instances start with:
- A local filesystem workspace
- Scoped tools
- Mounted read-only inputs (repo/wiki snapshots, reference packs)
- Writable output dirs for artefacts that will later be published via tools

### 7.1 Filesystem layout (recommended)
- `/workspace/` (writable, ephemeral)
- `/workspace/git-repo-name/` (agent worktree / branch working directory)
- `/inputs/` (read-only mounted inputs)
- `/cache/` (optional local caches)

Rule:
- **Readable inputs are mounted**; **publication happens via tools** (Git push/merge, wiki write), not by mutating shared mounts.

### 7.2 Git repo performance and concurrency
Git repository is cloned into a main filesystem. That is mounted to the agent's filesystem as near read-only. From that, a writable working tree is created for the agent to work in. When the agent is done, it can commit and push changes via the Git tool. To ensure that there's no race conditions, we lock the main repository during the clone and push operations with a simple lock-file.

Preferred pattern:
- Maintain a **bare repo mirror cache** (shared or node-local).
- For each agent task: create an isolated **worktree** from the mirror into `/workspace/repo`.
- Each task uses its own branch, e.g. `agent-task/<task-id>`.

This allows multiple agents to work on the same repo concurrently on different branches safely.

Do not share a single working tree between pods.


## 8) WebSocket message envelope

Envelope fields:
- `id` (required) unique message identifier
- `type` (required) the message's type
- `reply_to` (optional) message identifier that the message is replying to
- `idempotency_key` (optional) for cmd retries
- `seq` (optional) monotonically increasing per project (for replay and ordering)
- `payload` (optional) is type-specific JSON


## 9) Tools (initial set + recommended extensions)

Tools are explicitly scoped, auditable operations. Each call returns a receipt.

### 9.1 Current Task
- Details: get current task info
- Checklist
  - AddItem: Adds an item to the checklist
  - MarkComplete: Marks a checklist item complete
- Comment
  - Add: Adds a comment to the task
  - List: Lists comments on the task (both human and agent)

### 9.1 Filesystem tools
- Glob: file pattern matching
- Grep: content pattern matching
- File: Exists/Delete/Copy/Move
- MkDir: make directory
- ReadFile: read entire file
- ReadLines: read lines (start - stop)
- ReadMarkdownStructure: returns markdown as tree
- ReadMarkdownPart: returns named part of markdown
- OverwriteFile: writes the entire file
- ApplyPatch: applies a unified diff patch
- Command: bash shell command execution

### 9.2 Git tools
- Status: get current repo status
- Add: stage files
- Remove: unstage/delete files
- Commit: create a commit
- DiffFile: get diff of a file
- Show: get file content at specific revision
- Log: get commit history

### 9.3 Wiki tools
- Search: full-text search
- GetPage: retrieve page content
- WritePage: update/create page
- ListPages: list all pages
- GetPageHistory: get revision history
- GetPageLinks: get pages linked from a page
- GetBacklinks: get pages linking to a page

### 9.4 Web tools
- Search: web search (e.g. via search API)
- Fetch: retrieve web page content

### 9.5 Task tools
- Search: search tasks by criteria
- Create: create new task
- List: list tasks
- Update: update task fields
- Complete: mark task as complete

### 9.6 Project tools
- GetInfo
- ListAttachments
- GetAttachment

### 9.7 User Interaction tools
- MultipleChoice: present multiple choice question
- OpenQuestion: present open question
- Confirm: present yes/no confirmation


## 10) Suggested TaskType workflows (initial defaults)

These can be configured in a registry.

### Development
Stages:
- `plan -> implement -> review -> merge`

### Architecture
Stages:
- `draft -> technical_review -> publish`

### Research
Stages:
- `collect -> synthesise -> review -> publish`

### Exploration
Stages:
- `diverge -> converge -> propose`

### Documentation
Stages:
- `draft -> review -> publish`

### Integration
Stages:
- `prepare -> validate -> apply`


## 11) MVP milestones (recommended path)

1. End-to-end “happy path”:
   - create task -> executor runs job -> agent posts result -> UI shows done.
2. Add WebSocket live updates + event replay.
3. Add second runtime (Python) via agent type image selection.
4. Add reviewer stage as a normal task.
5. Optimise:
   - image pre-pull
   - repo cache + worktrees
   - optional warm worker pods for low-latency runs


## 12) Non-functional requirements (baseline)

- Auditability: tool receipts + event log.
- Safety: least privilege on tools; no shared working trees; no direct merges without review.
- Idempotency: commands and result callbacks are retry-safe.
- Observability: trace IDs across executor, agent, tool calls, and WS events.

## Appendix: Key design choices (summary)

- No stored board entity; UI projects tasks into stage columns.
- DB is source of truth; executor syncs to K8s Jobs.
- WS-first communication for UI; REST only for auth/bootstrap + large artefacts.
- Two agent runtimes (Go/Python) behind one spec/result protocol.
- Repo concurrency via bare mirror cache + per-task worktree branches.
- Primary reporting via agent callback; K8s status reconciliation as fallback.
- Tools are explicit, scoped, auditable, with receipts.
