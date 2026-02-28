# Aule — project task runner
# Run `just` to see all available recipes.

set dotenv-load := false

# Default recipe: list all available recipes
default:
    @just --list

# --- SpacetimeDB instance ---

# Start SpacetimeDB via Docker Compose
db:
    docker compose up -d

# Stop SpacetimeDB
db-stop:
    docker compose down

# View SpacetimeDB logs
db-logs:
    docker compose logs -f spacetimedb

# --- SpacetimeDB module ---

# Build the SpacetimeDB WASM module
build-module:
    spacetime build -p packages/aule-spacetimedb

# Generate TypeScript and Rust client bindings from the module
generate: build-module
    spacetime generate --lang typescript --out-dir app/src/module_bindings --module-path packages/aule-spacetimedb
    spacetime generate --lang rust --out-dir packages/aule-spacetimedb-client/src/module_bindings --module-path packages/aule-spacetimedb

# Publish the module to a local SpacetimeDB instance
publish: build-module
    spacetime publish --server http://localhost:3000 aule -p packages/aule-spacetimedb

# --- Rust workspace ---

# Build all workspace crates
build:
    cargo build

# Run all workspace tests
test:
    cargo test

# Run clippy on the workspace
lint:
    cargo clippy -- -D warnings

# --- Frontend ---

# Install frontend dependencies
install:
    cd app && bun install

# Start the frontend dev server
dev:
    cd app && bun run dev

# Type-check the frontend
typecheck:
    cd app && bunx tsc --noEmit

# --- Full setup ---

# Bootstrap everything: install deps, generate bindings, build workspace
setup: install generate build
    @echo "Setup complete. Run 'just dev' to start the dashboard."
