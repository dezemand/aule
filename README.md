# Aule

An agentic platform where humans and AI agents collaborate in shared workspaces. Agents are first-class participants -- they have memory, budgets, tools, and identities. They coordinate through real-time shared state powered by SpacetimeDB, execute work in isolated K8s pods, and reason through a multi-model LLM router.

Named after the Vala of craftsmanship in Tolkien's legendarium -- the smith who forged things into being.

## Documentation

- [Architecture Overview](docs/architecture.md) -- system diagram, components, project structure
- [Identity & Auth](docs/identity-and-auth.md) -- user/runtime/task identity model, permission checking
- [Agent Versioning](docs/agent-versioning.md) -- agent types, version lifecycle, upgrade strategies
- [SpacetimeDB Module](docs/spacetimedb-module.md) -- tables, reducers, scheduled reducers
- [LLM Router](docs/llm-router.md) -- multi-model routing, caching, feedback loop
- [Agent Runtime](docs/agent-runtime.md) -- platform tools, shell safety, startup/execution flow
- [Phase 2 Runbook](docs/phase-2-runbook.md) -- local runtime setup and end-to-end test steps
- [Cross-Cutting Concerns](docs/cross-cutting.md) -- provenance, supervision, events, approvals, tools, UI
- [Roadmap](docs/roadmap.md) -- phased delivery plan

## Project Structure

```
packages/       Rust workspace crates
  aule-core/    common structs, agent protocol, API types
app/            Front-end application (Bun/TypeScript)
docs/           Documentation
```

## Prerequisites

- [Rust](https://rustup.rs/) (stable)
- [Bun](https://bun.sh/)

## Getting Started

### Rust

```sh
cargo build
cargo test
```

### Front-end

```sh
cd app
bun install
bun run index.ts
```
