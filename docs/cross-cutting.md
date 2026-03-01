# Cross-Cutting Concerns

Systems that span multiple components.

## Provenance

Every agent output is traceable through a DAG: inputs → LLM calls → tool executions → outputs. Stored in SpacetimeDB as `provenance_nodes` and `provenance_edges`.

- Humans subscribe and watch reasoning chains build in real-time
- Scheduled reducers audit for orphaned conclusions, circular reasoning, and over-reliance on single sources
- Full audit trail via SpacetimeDB transaction log

Node types: `llm_call`, `tool_use`, `observation`, `memory_read`, `human_input`.
Edge relations: `informed_by`, `produced`, `triggered`, `used`.

## Agent Supervision

Supervisor agents subscribe to broad SpacetimeDB queries watching for:

- Stuck runtimes (stale heartbeats)
- Spinning agents (activity without progress)
- Conflicting outputs
- Cascade failures
- Resource imbalance

Supervisors are regular agents with broader subscriptions. Interventions are standard reducer calls. No special machinery — just wider visibility.

## Events

- Insert-only `events` table scoped by workspace
- Event types: `task.created`, `agent.launched`, `observation.posted`, etc.
- Agents subscribe to event types via SpacetimeDB subscriptions
- SpacetimeDB handles fan-out
- Events and state changes combine atomically in single reducers
- Scheduled reducer (`cleanup_events`) handles TTL cleanup

## Approval Queue

Human-in-the-loop gates for agent-proposed actions.

- Agents write approval requests via `aule_suggest`
- Approvals include `action_type`, `payload`, and `reasoning` — the agent explains why
- Humans subscribe to the approval queue, review, approve/reject via reducers
- Approval + action execution in one atomic transaction
- Scheduled reducer (`expire_approvals`) handles timeouts — auto-escalate or auto-reject stale requests
- Approvals have an explicit `expires_at` timestamp

## Messages

Direct communication between users and agents, scoped to a task.

- `messages` table with `sender_type` ("user" or "agent") and `sender_id`
- Not a general chat — structured interaction tied to specific work
- Agents can ask clarifying questions; users can provide additional direction
- Visible in the task detail view in the dashboard

## Tool Registry

Dynamic capability discovery and registration.

- Tools register in SpacetimeDB with capability tags, endpoints, input schemas, health metrics
- Tool types: `cli`, `api`, `mcp`, `agent`
- Workspace-scoped tools (null workspace_id = global)
- Agents query for capabilities they need
- New tools visible instantly via subscriptions
- Agents can register themselves as tools (composability)
- Scheduled reducers health-check and update `reliability_score`

## Human Interface

### Dashboard

Real-time agent monitoring via SpacetimeDB subscriptions. Shows workspaces, agent status, task plans/steps, budgets, observations, and memory.

### Task Management

Create tasks with context, assign to agents, track attempts and plans in real-time. Live TaskStep progress view as agents work.

### Observation Feed

Stream of agent findings, progress, errors, and questions. Filterable by workspace, agent, and task. Acknowledge/dismiss observations.

### Intervention Console

Pause, inspect, modify, and override agent behavior. Direct reducer calls to change agent state.

### Approval Queue UI

Review and act on pending agent requests. See context, reasoning, proposed action. Approve/reject with notes.

### Conversation Mode

Structured dialogue with specific agents via the Messages table. Scoped to a task — not a general chatbot.

### Time Travel

History exploration via SpacetimeDB transaction log. Replay past states, understand how decisions were made.

### Version Management

Create, promote, deprecate, and compare agent type versions. View performance metrics per version.

## Skill / Knowledge Management

Shared curated knowledge separate from per-agent memory:

- Domain knowledge bases
- Learned patterns promoted from agent experience
- Reusable reasoning templates

Mounted into agent pods or queryable from SpacetimeDB.
