# LLM Router

A native Rust HTTP service (axum). Routes LLM requests from agents to the best provider based on task type, quality requirements, budget constraints, and provider health.

## Standard API

```
POST /v1/completion
{
    "task_type": "reason" | "generate" | "extract" | "classify" | "code" | "summarize" | "embed",
    "messages": [...],
    "constraints": {
        "max_tokens": 4096,
        "max_latency_ms": 5000,
        "max_cost_cents": 10,
        "min_quality": "high" | "medium" | "low",
        "structured_output": { "format": "json", "schema": {...} }
    },
    "context": {
        "agent_id": "...",
        "task_id": "...",
        "project_id": "...",
        "budget_remaining_cents": 450,
        "priority": "normal"
    },
    "tools": [...]
}
```

## Config from SpacetimeDB

The router connects as a SpacetimeDB client and subscribes to `llm_providers`, `routing_rules`, `rate_limits`. Config changes propagate instantly -- no restarts. The router writes `llm_requests` entries back via reducer calls, creating a closed feedback loop.

## Capabilities

- **Task-based routing** -- reason->Opus, extract->Haiku, code->Sonnet
- **Budget-aware degradation** -- low budget -> cheaper model automatically
- **Latency-aware routing** -- slow provider -> skip
- **Fallback chains** -- primary -> secondary -> retry queue
- **Context-window routing** -- large context -> model that supports it
- **Exact-match and semantic caching**
- **Tool format normalization** across providers
- **Provider health tracking** with auto-disable on failure spikes
- **Secrets management** -- from K8s Secrets / Vault (never in SpacetimeDB)

## Routing Logic

The router evaluates requests against routing rules from SpacetimeDB:

1. Match task type to eligible providers/models
2. Filter by constraints (latency, cost, quality)
3. Filter by health (exclude unhealthy providers)
4. Filter by rate limits (exclude throttled providers)
5. Check budget (degrade to cheaper model if needed)
6. Select best match from remaining candidates
7. Execute with fallback chain on failure

## Provider Adapters

| Provider | Module | Notes |
|----------|--------|-------|
| Anthropic | `providers/anthropic.rs` | Claude models |
| Google | `providers/google.rs` | Gemini models |
| OpenAI | `providers/openai.rs` | GPT models |
| Local | `providers/local.rs` | vLLM / Ollama for self-hosted models |

Each adapter normalizes the provider's API to the router's internal format, including tool/function calling conventions.

## Caching

Two layers:

- **Exact match** -- identical request (messages + constraints) returns cached response
- **Semantic** -- similar requests (embedding similarity above threshold) can reuse responses

Cache entries have TTL and are invalidated on routing rule changes.

## Feedback Loop

```
Agent -> Router -> Provider -> Response
                |
                v
        SpacetimeDB (llm_requests table)
                |
                v
        Scheduled reducers aggregate:
        - provider_health (latency p50/p95, error rate)
        - rate_limits (current RPM/TPM)
        - detect_anomalies (error spikes)
                |
                v
        Router subscriptions pick up changes
        -> routing decisions adapt in real-time
```

## Source Layout

```
packages/aule-router/
└── src/
    ├── server.rs              axum HTTP server
    ├── router.rs              routing logic, rule engine
    ├── providers/
    │   ├── anthropic.rs
    │   ├── google.rs
    │   ├── openai.rs
    │   └── local.rs           vLLM / Ollama
    ├── cache.rs               exact match + semantic caching
    ├── budget.rs              budget checking via SpacetimeDB
    ├── spacetimedb_client.rs  subscription management, config sync
    └── telemetry.rs           request logging, metrics
```
