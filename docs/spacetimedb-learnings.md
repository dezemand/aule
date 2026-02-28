# SpacetimeDB Learnings

Patterns, constraints, and gotchas discovered during Phase 0-1 work with SpacetimeDB 2.0.2.

## Module Patterns (Rust WASM)

### Table macro: `accessor` not `name`

SpacetimeDB v2 changed the `#[table]` macro syntax. The `name` parameter now expects a string literal and is the table name stored in the database. Use `accessor` for the Rust method name:

```rust
// v2 syntax
#[table(accessor = user, public)]
pub struct User { ... }

// Accessing: ctx.db.user().insert(...)
```

The quickstart docs (which target v1) still show `name = user` but v2 requires `accessor = user`.

### Table accessor traits must be imported in multi-file modules

The `#[table]` macro generates a trait with the same name as the accessor. When tables are defined in a separate `tables.rs` module and used from reducer files, you must import these traits explicitly:

```rust
// In reducers/identity.rs
use crate::tables::{agent_runtime, agent_task, agent_type};  // trait imports
use crate::tables::{AgentRuntime, AgentTask};                 // struct imports
```

Without the trait import, `ctx.db.agent_runtime()` won't resolve. The compiler error message is helpful and suggests the right import.

### ReducerContext fields are methods

In v2, `ctx.sender` is private. Use `ctx.sender()` instead:

```rust
let sender = ctx.sender();  // not ctx.sender
```

`ctx.timestamp` remains a public field.

### Timestamp API

```rust
// Correct in v2
timestamp.to_micros_since_unix_epoch()

// Not available (v1 name)
// timestamp.to_micros_since_epoch()
```

### Scheduled reducers use a schedule table

v2 requires a dedicated table for scheduling. The table must have:
- A `#[primary_key]` `#[auto_inc]` id field
- A `ScheduleAt` field

```rust
#[table(accessor = cleanup_schedule, scheduled(cleanup_old_messages))]
pub struct CleanupSchedule {
    #[primary_key]
    #[auto_inc]
    scheduled_id: u64,
    scheduled_at: ScheduleAt,
}

#[reducer]
pub fn cleanup_old_messages(ctx: &ReducerContext, _schedule: CleanupSchedule) {
    // The second arg is the schedule row that triggered this call
}
```

To start a repeating schedule, insert a row with a `TimeDuration`:

```rust
let five_minutes = TimeDuration::from_micros(5 * 60 * 1_000_000);
ctx.db.cleanup_schedule().insert(CleanupSchedule {
    scheduled_id: 0,
    scheduled_at: five_minutes.into(),
});
```

### Auto-inc columns

Pass `0` as the sentinel value when inserting; SpacetimeDB replaces it with a unique auto-incrementing value.

### Module cannot be built with `cargo build`

The WASM module links against SpacetimeDB host functions (`bytes_sink_write`, `datastore_insert_bsatn`, etc.) that don't exist in native builds. Always build with:

```sh
spacetime build -p packages/aule-spacetimedb
```

The workspace `Cargo.toml` should exclude WASM module crates:

```toml
[workspace]
exclude = ["packages/aule-spacetimedb"]
```

## Rust Client SDK Patterns

### Builder method: `with_database_name`

v2 renamed `with_module_name` to `with_database_name`:

```rust
DbConnection::builder()
    .with_database_name("aule")
    .with_uri("http://localhost:3000")
    .build()
```

### Status enum

Reducer status in v2:

```rust
enum Status {
    Committed,
    Err(String),      // was Failed(String) in v1
    Panic(InternalError),
}
```

### No per-reducer `on_` callbacks

v2 removed `on_set_name`, `on_send_message` etc. from `RemoteReducers`. Instead use `_then` variants to get a callback for a specific invocation:

```rust
ctx.reducers.set_name_then("Alice".into(), |ctx, result| {
    match result {
        Ok(Ok(())) => println!("Name set"),
        Ok(Err(msg)) => println!("Reducer error: {msg}"),
        Err(e) => println!("Internal error: {e}"),
    }
});
```

Or use the table `on_insert`/`on_update`/`on_delete` callbacks to react to state changes.

### Unique column `find` takes a reference

```rust
ctx.db.stats().id().find(&0)  // not find(0)
```

### Credential storage

The SDK provides `credentials::File` for persisting tokens across sessions:

```rust
let store = credentials::File::new("my-app");
store.save(token)?;
let token = store.load()?;
```

## TypeScript Client SDK Patterns (Bun)

### Builder method: `withDatabaseName`

Same rename as Rust:

```ts
DbConnection.builder()
  .withUri("ws://localhost:3000")
  .withDatabaseName("aule")
  .build();
```

### Reducer calls take objects

Reducers accept an object matching the argument schema, not positional args:

```ts
conn.reducers.sendMessage({ text: "Hello" });   // not sendMessage("Hello")
conn.reducers.setName({ name: "Alice" });        // not setName("Alice")
```

### Option types map to `T | undefined`

Rust `Option<String>` becomes `string | undefined` in TypeScript (not `string | null`).

### Event tag checking

```ts
ctx.db.message.onInsert((ctx, message) => {
    if (ctx.event.tag === "Reducer") {
        // This was inserted by a reducer, not initial subscription data
    }
});
```

### Bun compatibility

The `spacetimedb` npm package (v2.0.2) works with Bun out of the box. No polyfills needed. WebSocket support is built into Bun.

## Frontend: Bun/TypeScript for now, Leptos later

The current frontend uses the SpacetimeDB TypeScript SDK with Bun. This is a temporary choice. The plan is to migrate to **Leptos** (Rust WASM frontend) once SpacetimeDB ships browser WASM support for the Rust client SDK (PR in progress upstream). This aligns with the "Rust everywhere" principle and eliminates the codegen step for client bindings.

The Bun client serves as a working proof-of-concept and reference implementation for the migration.

## WASM Constraints (Server Module)

- No filesystem access (`std::fs` will panic at runtime)
- No network access (`std::net` will panic at runtime)
- No threads (`std::thread` will panic at runtime)
- The `log` crate works -- SpacetimeDB installs a logger automatically
- `rand` works if enabled via SpacetimeDB's feature flags (deterministic seeding per reducer call)
- Any Rust crate that compiles to `wasm32-unknown-unknown` can be used

## Development Workflow

### Starting a local instance

```sh
spacetime start
```

This runs SpacetimeDB locally on port 3000.

### Publishing a module

```sh
spacetime build -p packages/aule-spacetimedb
spacetime publish --server local -p packages/aule-spacetimedb aule
```

### Generating client bindings

```sh
# Rust
spacetime generate --lang rust --out-dir packages/aule-spacetimedb-client/src/module_bindings -p packages/aule-spacetimedb

# TypeScript
spacetime generate --lang typescript --out-dir app/src/module_bindings -p packages/aule-spacetimedb
```

### Running clients

```sh
# Rust client
cargo run -p aule-client

# Bun client
cd app && bun index.ts
```

### Querying the database

```sh
spacetime sql --server local aule "SELECT * FROM agent_runtime"
spacetime sql --server local aule "SELECT * FROM agent_task"
spacetime sql --server local aule "SELECT * FROM agent_type"
```

### Viewing logs

```sh
spacetime logs --server local aule
```
