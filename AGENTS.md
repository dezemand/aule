# Aulë — Agent Instructions

## Important

- When implementing any feature, make sure to validate the plan against the existing documentation and architecture in `docs/`. The documentation is the north star; if the implementation deviates, update the docs.
- When updating documentation or adding new documentation, make sure to update all relevant documents in `docs/` to keep them in sync.
- When working on GH issues, use a feature branch like `feature/<issue-number>-<short-description>` and open a PR when ready for review.

## Repository Layout

- `packages/` — Rust workspace crates. Each sub-directory is a separate crate.
- `app/` — Front-end application built with Bun and TypeScript. The `bun` skill is loaded automatically when working in this directory.
- `docs/` — Project documentation. `docs/architecture.md` is the north star architecture (target, not current state).
- `Justfile` — preferred task runner for local dev workflows.

## Rust

- Workspace root is `Cargo.toml` at the repo root. All crates live under `packages/`.
- `aule-spacetimedb` is excluded from the default workspace — build it with `spacetime build`, not `cargo build`.
- Use `cargo build`, `cargo test`, and `cargo clippy` from the repo root to operate on the full workspace.
- Follow standard Rust conventions: `rustfmt` for formatting, `clippy` for lints.
- Prefer returning `Result` over panicking.

## Front-end (app/)

- Use Bun, not Node.js.
- Run `bun install` from `app/` before doing anything else.
- Run `bun test` for tests.

## SpacetimeDB and Just

- `just` loads `.env` from the repo root.
- Keep `.env` private; use `.env.template` as the committed template.
- Configure local DB via `SPACETIMEDB_URI` and `SPACETIMEDB_DB_NAME`.
- Use `just publish` for normal publish, and `just publish -- --delete-data` to publish with `--delete-data`.
- Use `just generate` to regenerate TypeScript and Rust client bindings after module changes.
