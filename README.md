# Aulë

A general-purpose agentic execution engine. Humans define work, agents execute it. Agents reason through LLMs, act through shell commands and CLIs, coordinate through real-time shared state in SpacetimeDB, and run in isolated K8s pods managed by a custom operator.

Named after the Vala of craftsmanship in Tolkien's legendarium — the smith who forged things into being.

## Why

This started out as a brainstorm session around trying out [SpacetimeDB](https://spacetimedb.com/) for something cool. While originally built for games, it seemed like a natural fit for coordinating agents and tasks in a more general execution engine. The idea was to have agents read from and write to shared state in SpacetimeDB, with reducers handling task lifecycle, scheduling, and routing. This would allow for more complex coordination patterns than just having agents call APIs or message queues, in a rather simple way.

## Documentation

- [Architecture (North Star)](docs/architecture.md) — target architecture, metamodel, design patterns
- [Identity & Auth](docs/identity-and-auth.md) — user/runtime/task identity model, permission checking
- [Agent Versioning](docs/agent-versioning.md) — agent types, version lifecycle, upgrade strategies
- [SpacetimeDB Module](docs/spacetimedb-module.md) — tables, reducers, scheduled reducers
- [LLM Router](docs/llm-router.md) — multi-model routing, caching, feedback loop
- [Agent Runtime](docs/agent-runtime.md) — platform tools, shell safety, startup/execution flow
- [Running](docs/running.md) — how to run every component locally
- [Cross-Cutting Concerns](docs/cross-cutting.md) — provenance, supervision, events, approvals, tools, UI
- [Roadmap](docs/roadmap.md) — phased delivery plan
- [SpacetimeDB Learnings](docs/spacetimedb-learnings.md) — practical notes from working with SpacetimeDB

## Project Structure

```
packages/                      Rust workspace crates
  aule-core/                   shared types, agent protocol, API types
  aule-spacetimedb/            SpacetimeDB module (compiles to WASM via `spacetime build`)
  aule-spacetimedb-client/     generated SpacetimeDB Rust client bindings
  aule-runtime/                agent process — task execution, shell safety, LLM calls
  aule-client/                 interactive test client for creating tasks/agents
app/                           front-end dashboard (Bun/TypeScript)
docs/                          documentation
```

## Prerequisites

- [Rust](https://rustup.rs/) (stable)
- [Bun](https://bun.sh/)
- [just](https://github.com/casey/just)
- [SpacetimeDB CLI](https://spacetimedb.com/docs)
- Docker (for the local SpacetimeDB instance)

## Environment

Copy the template and adjust values as needed:

```sh
cp .env.template .env
```

```env
SPACETIMEDB_URI=http://localhost:3000
SPACETIMEDB_DB_NAME=aule
```

`just` loads `.env` automatically.

## Getting Started

```sh
just setup          # install deps, generate bindings, build workspace
just db             # start local SpacetimeDB (Docker)
just publish        # build and publish the WASM module
just generate       # regenerate TS + Rust client bindings
just dev            # start the frontend dev server
```

Publish with data reset:

```sh
just publish --delete-data
```

### Manual commands

```sh
cargo build         # build Rust workspace
cargo test          # run Rust tests
cargo clippy        # lint

cd app && bun install && bun run dev   # frontend
```

## License

[MIT](LICENSE)
