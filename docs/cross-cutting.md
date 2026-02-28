# Cross-Cutting Concerns

Systems that span multiple components.

## Provenance

Every agent output is traceable through a DAG: inputs -> LLM calls -> tool executions -> outputs. Stored in SpacetimeDB as `provenance_nodes` and `provenance_edges`.

- Humans subscribe and watch reasoning chains build in real-time
- Scheduled reducers audit for orphaned conclusions, circular reasoning, and over-reliance on single sources
- Full audit trail via SpacetimeDB transaction log

## Agent Supervision

Supervisor agents subscribe to broad SpacetimeDB queries watching for:

- Stuck runtimes (stale heartbeats)
- Spinning agents (activity without progress)
- Conflicting outputs
- Cascade failures
- Resource imbalance

Supervisors are regular agents with broader subscriptions. Interventions are standard reducer calls. No special machinery -- just wider visibility.

## Events

- Insert-only `events` table
- Agents subscribe to event types
- SpacetimeDB handles fan-out
- Events and state changes combine atomically in single reducers
- Scheduled reducer handles TTL cleanup

## Approval Queue

Human-in-the-loop gates for agent-proposed actions.

- Agents write approval requests via `aule_suggest`
- Humans subscribe to the approval queue, review, approve/reject via reducers
- Approval + action execution in one atomic transaction
- Scheduled reducer handles timeouts (auto-escalate or auto-reject stale requests)

## Tool Registry

Dynamic capability discovery and registration.

- Tools register in SpacetimeDB with capability tags, endpoints, schemas, health metrics
- Agents query for capabilities they need
- New tools visible instantly via subscriptions
- Agents can register themselves as tools (composability)
- Scheduled reducers health-check and update reliability scores

## Human Interface

### Dashboard

Real-time agent monitoring via SpacetimeDB subscriptions. Shows agent status, memory, budgets, observations, and provenance chains as they build.

### Intervention Console

Pause, inspect, modify, and override agent behavior. Direct reducer calls to change agent state.

### Approval Queue UI

Review and act on pending agent requests. See context, reasoning chain, proposed action. Approve/reject with notes.

### Conversation Mode

Structured dialogue with specific agents. Not chat -- scoped interaction with context.

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
