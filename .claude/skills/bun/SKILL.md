---
name: bun
description: Bun runtime conventions and APIs for the app/ front-end. Use when working on files in app/, writing TypeScript for the front-end, or setting up Bun servers, tests, or builds.
user-invocable: false
---

Default to using Bun instead of Node.js.

- Use `bun <file>` instead of `node <file>` or `ts-node <file>`
- Use `bun test` instead of `jest` or `vitest`
- Use `bun build <file.html|file.ts|file.css>` instead of `webpack` or `esbuild`
- Use `bun install` instead of `npm install` or `yarn install` or `pnpm install`
- Use `bun run <script>` instead of `npm run <script>` or `yarn run <script>` or `pnpm run <script>`
- Use `bunx <package> <command>` instead of `npx <package> <command>`
- Bun automatically loads .env, so don't use dotenv.

## APIs

- `bun:sqlite` for SQLite. Don't use `better-sqlite3`.
- `Bun.redis` for Redis. Don't use `ioredis`.
- `Bun.sql` for Postgres. Don't use `pg` or `postgres.js`.
- `WebSocket` is built-in. Don't use `ws`.
- Prefer `Bun.file` over `node:fs`'s readFile/writeFile
- Bun.$`ls` instead of execa.

## Testing

Use `bun test` to run tests.

```ts
import { test, expect } from "bun:test";

test("hello world", () => {
  expect(1).toBe(1);
});
```

## Frontend

The `app/` frontend uses **Vite** as dev server and bundler (not Bun.serve). Bun is used as the package manager and test runner only.

```bash
cd app
bun install       # Install dependencies
bun run dev       # Start Vite dev server with HMR
bun run build     # Type-check + production build
bun run typecheck # Type-check only
bun test          # Run tests
```
