# Frontend Development Guide

## Overview

The Aulë frontend is a single-page application built with **Vite**, **React 19**, **Mantine v8**, and **TanStack Router**. It connects to SpacetimeDB over WebSocket for all data — there is no REST or GraphQL API. The SpacetimeDB client-side replica acts as the state store, and subscriptions provide real-time updates.

> **Note:** The long-term plan is to migrate to Leptos (Rust WASM) once SpacetimeDB ships browser WASM support for the Rust client SDK. This codebase should still be well-structured — clean patterns are easier to port, and the migration timeline is uncertain.

## Quick Start

```bash
cd app
bun install
bun run dev        # Dev server at http://localhost:5173
bun run typecheck  # Type-check without emitting
bun test           # Run tests
```

Requires a running SpacetimeDB instance (see [running.md](./running.md)).

## Tech Stack

| Concern         | Choice                                        |
|-----------------|-----------------------------------------------|
| Dev server      | Vite (bundler, HMR, CSS modules)              |
| Runtime         | Bun (package manager, test runner)             |
| Framework       | React 19 (StrictMode)                         |
| UI Library      | Mantine v8 (dark theme default)               |
| Routing         | TanStack React Router v1 (code-based)         |
| Styling         | Mantine component props + CSS Modules         |
| State           | SpacetimeDB client-side replica + Zustand       |
| Data layer      | SpacetimeDB WebSocket subscriptions           |
| Code generation | SpacetimeDB CLI (`just generate`)             |
| Icons           | Tabler Icons for React                        |
| Markdown        | Streamdown (streaming renderer)               |
| Testing         | Bun test runner (`bun test`)                  |

---

## Project Structure

```
app/
├── index.html                          # SPA HTML shell (Vite entry point)
├── vite.config.ts                      # Vite config (React plugin, @/ alias)
├── package.json
├── tsconfig.json
└── src/
    ├── main.tsx                         # App bootstrap: providers → router → mount
    ├── index.css                        # Global styles (resets, typography, markdown)
    ├── global.d.ts                      # Ambient type declarations (CSS modules, etc.)
    │
    ├── module_bindings/                 # ⚠️  AUTO-GENERATED — never edit manually
    │                                    #    Regenerate with: just generate
    │
    ├── config/                          # App-wide configuration
    │   ├── theme.ts                     #   Mantine theme object
    │   └── routes.ts                    #   Root route + RouteContext type
    │
    ├── lib/                             # Infrastructure libraries
    │   └── subscriptions/               #   SpacetimeDB subscription system
    │       ├── subscription.ts          #     SubscriptionDef type
    │       ├── subscriptionManager.ts   #     Singleton manager (ensure/retain/release)
    │       ├── tableEventBus.ts         #     Microtask-coalesced event bus
    │       └── hooks/
    │           ├── useSpacetimeConnection.ts  # Zustand store + subscriptionManager singleton
    │           ├── useSubscription.ts         # Subscription lifecycle hook
    │           └── useQuery.ts               # Scoped reactive query hook
    │
    ├── shared/                          # Cross-cutting reusable code
    │   ├── components/                  #   UI components used across features
    │   │   ├── AppShell/                #     Layout: icon rail + sub-nav + Outlet
    │   │   ├── TopBar/                  #     Top bar above main content (connection badge, future search)
    │   │   ├── Markdown/
    │   │   └── ConnectionBadge/
    │   └── utils/                       #   Pure utility functions
    │       └── statusColors.ts
    │
    └── features/                        # Feature modules (one per domain)
        ├── dashboard/
        ├── tasks/
        ├── agents/
        ├── conversation/
        ├── runtimes/                    # (planned)
        ├── observations/                # (planned)
        ├── approvals/                   # (planned)
        └── provenance/                  # (planned)
```

### Path Aliases

The project uses a `@/` path alias mapped to `src/` in `tsconfig.json`. Always use alias imports instead of relative paths when importing across features or from shared code.

```tsx
// ✅ Good — alias import
import { useSubscription } from "@/lib/subscriptions/hooks/useSubscription";
import { taskStatusColor } from "@/shared/utils/statusColors";

// ❌ Bad — relative path across feature boundaries
import { useSubscription } from "../../lib/subscriptions/hooks/useSubscription";
import { taskStatusColor } from "../../../shared/utils/statusColors";
```

Use relative imports only for files within the same feature (e.g., `./components/TaskCard`).

---

## Feature Modules

Each feature in `features/` follows this internal layout:

```
features/<name>/
├── <Name>Page.tsx              # Route-level page component + route definition
├── components/                 # Components specific to this feature
│   ├── SomeComponent.tsx
│   └── ComplexComponent/       # Multi-file components get a subfolder
│       ├── ComplexComponent.tsx
│       └── SubPart.tsx
├── hooks/                      # Hooks specific to this feature (optional)
│   └── useSomething.ts
└── utils/                      # Utilities specific to this feature (optional)
    └── helpers.ts
```

The SpacetimeDB subscription infrastructure lives in `lib/subscriptions/` rather than a feature folder — it's a library, not a feature with pages.

### Rules

- **No barrel files (`index.ts`).** Import directly from the source file. This keeps the dependency graph explicit and avoids the indirection of re-exports.
- A feature folder is created when a domain has at least one page or a substantial component set.
- `hooks/` and `utils/` inside a feature are optional — create them only when a feature has logic that doesn't belong in the component file.

### Where code goes

| Code type                                    | Location                          | Example                                            |
|----------------------------------------------|-----------------------------------|----------------------------------------------------|
| Page component (maps to a route)             | `features/<name>/<Name>Page.tsx`  | `features/tasks/TasksPage.tsx`                     |
| Route definition for that page               | Same file as the page component   | `export const tasksRoute = createRoute({...})`     |
| Feature-specific component                   | `features/<name>/components/`     | `features/tasks/components/TaskCard.tsx`            |
| Feature-specific hook                        | `features/<name>/hooks/`          | `features/tasks/hooks/useTaskSubscription.ts`      |
| Feature-specific utility                     | `features/<name>/utils/`          | `features/tasks/utils/taskHelpers.ts`              |
| SpacetimeDB subscription infrastructure      | `lib/subscriptions/`              | `lib/subscriptions/hooks/useSubscription.ts`       |
| Reusable UI component (used by 2+ features)  | `shared/components/`              | `shared/components/Markdown/Markdown.tsx`           |
| Cross-cutting utility function               | `shared/utils/`                   | `shared/utils/statusColors.ts`                     |
| Theme, root route, or global config          | `config/`                         | `config/theme.ts`                                  |

**Promotion rule:** A component starts in its feature. If a second feature needs it, move it to `shared/components/`. Don't prematurely abstract.

---

## Routing

Routes use TanStack Router's code-based API with **co-located route definitions** and **section layout routes** for navigation grouping.

### Architecture

The route tree uses a three-level hierarchy:

```
rootRoute                                    →  AppShell (sidebar + TopBar + Outlet)
├── dashboardLayout (pathless, id-only)      →  Section: Dashboard
│   └── indexRoute        "/"                →  DashboardPage
├── tasksLayout           "/tasks"           →  Section: Tasks
│   ├── tasksRoute        "/"                →  TasksPage      (resolves to /tasks)
│   └── taskDetailsRoute  "/$taskId"         →  TaskDetailsPage (resolves to /tasks/$taskId)
└── agentsLayout          "/agent-types"     →  Section: Agent Types
    └── agentTypesRoute   "/"                →  AgentTypesPage  (resolves to /agent-types)
```

**Section layout routes** are defined in `config/routes.tsx`. They carry no component — they exist to group pages under a navigation section. The AppShell uses the current URL to determine which section is active and which sub-page links to show.

### Pattern

**`config/routes.tsx`** exports the root route, section layouts, and `RouteContext` type:

```tsx
// config/routes.tsx
import { createRootRouteWithContext, createRoute, Outlet } from "@tanstack/react-router";
import { AppShell } from "@/shared/components/AppShell/AppShell";

export const rootRoute = createRootRouteWithContext<RouteContext>()({
  component: () => (
    <AppShell>
      <Outlet />
    </AppShell>
  ),
});

// Section layout routes — one per nav section
export const dashboardLayout = createRoute({
  getParentRoute: () => rootRoute,
  id: "dashboard",  // pathless layout for the root "/"
});

export const tasksLayout = createRoute({
  getParentRoute: () => rootRoute,
  path: "/tasks",
});

export const agentsLayout = createRoute({
  getParentRoute: () => rootRoute,
  path: "/agent-types",
});
```

**Each feature page** co-locates its route definition and component, parenting to the section layout:

```tsx
// features/tasks/TaskDetailsPage.tsx
import { createRoute } from "@tanstack/react-router";
import { tasksLayout } from "@/config/routes";

export const taskDetailsRoute = createRoute({
  getParentRoute: () => tasksLayout,
  path: "/$taskId",          // relative to /tasks
  component: TaskDetailsPage,
});

function TaskDetailsPage() {
  const { taskId } = taskDetailsRoute.useParams();
  // ...
}
```

**`main.tsx`** imports all routes and assembles the nested tree:

```tsx
// main.tsx
import { rootRoute, dashboardLayout, tasksLayout, agentsLayout } from "@/config/routes";
import { indexRoute } from "@/features/dashboard/DashboardPage";
import { tasksRoute } from "@/features/tasks/TasksPage";
import { taskDetailsRoute } from "@/features/tasks/TaskDetailsPage";
import { agentTypesRoute } from "@/features/agents/AgentTypesPage";

const routeTree = rootRoute.addChildren([
  dashboardLayout.addChildren([indexRoute]),
  tasksLayout.addChildren([tasksRoute, taskDetailsRoute]),
  agentsLayout.addChildren([agentTypesRoute]),
]);
```

Dependency flow is one-way: `config/routes` ← `features/*` ← `main.tsx`. No circular imports.

### Adding a new route

1. Identify which section (layout route) the page belongs to. If none fits, create a new section layout in `config/routes.tsx` and add a corresponding entry in `NAV_SECTIONS` in `AppShell.tsx`.
2. Create the page component in the appropriate feature folder.
3. Define the route with `createRoute()` in the same file, importing the **section layout** (not `rootRoute`) as the parent. Use a path relative to the section prefix.
4. Import the route in `main.tsx` and add it as a child of the section layout's `.addChildren()`.
5. If the page should appear in the sidebar sub-nav, add it to the section's `subPages` array in `AppShell.tsx`.

### Adding a new section

1. Create a new layout route in `config/routes.tsx` with `createRoute({ getParentRoute: () => rootRoute, path: "/new-section" })`.
2. Add a `NavSection` entry to `NAV_SECTIONS` in `AppShell.tsx` with icon, label, basePath, and subPages.
3. Create page routes that parent to the new layout.
4. Wire everything in `main.tsx`: `newLayout.addChildren([...])`.
5. Export the layout from `config/routes.tsx`.

---

## UX Principles

### Loading & Empty States

1. **Skeletons over spinners.** Never show "Loading..." text or a bare spinner. Use Mantine's `Skeleton` component to show the shape of incoming content. This reduces perceived latency and prevents layout shift.

2. **Distinct empty states.** Differentiate between "no data yet" (first use) and "no results match your filter". Empty states should guide the user toward action (e.g., "No tasks yet — create one to get started").

3. **Subscription-aware loading.** The app has two loading phases: connecting to SpacetimeDB, and waiting for subscription data. Show a subtle connection indicator in the AppShell header, but don't block the entire page while subscribing — show skeletons for the data areas instead.

### Real-Time Behavior

4. **Don't flash on updates.** SpacetimeDB pushes updates frequently (heartbeats, events, observations). UI updates should be smooth, not jarring. Avoid full re-renders or layout jumps when a single row changes.

5. **Relative timestamps update live.** "2 minutes ago" should tick automatically. Use a single shared interval (e.g., every 30s) rather than per-component timers.

6. **New items appear, don't displace.** When new observations or events arrive while the user is reading, don't push content down. Use a "N new items" indicator the user can click to scroll up, or append at the natural position without disrupting the scroll position.

### Data Density & Readability

7. **Progressive disclosure.** Show summary first, details on demand. Task cards show status + title; click to expand or navigate to full details. Conversation events are collapsed by default with a one-line summary.

8. **Status at a glance.** Use color-coded badges consistently across the app for task status, runtime status, observation kind, and version status. The color vocabulary should be learnable: green = good/done, yellow = in progress, red = error, gray = inactive.

### Forms & Actions

9. **Inline forms over modals.** Prefer inline/expandable forms (e.g., "New Task" expanding a form in-place) over modal dialogs. Modals interrupt context; inline forms keep the user oriented.

10. **Optimistic feedback on reducer calls.** SpacetimeDB reducer calls are fire-and-forget. After calling a reducer, immediately show a pending/disabled state on the trigger element. The subscription will confirm the change. If it doesn't arrive within a timeout, show a subtle error.

11. **Destructive actions require confirmation.** Cancel task, deregister runtime, etc. should require a confirmation step (e.g., hold-to-confirm or an inline "Are you sure?" prompt). Not a modal.

### Navigation & Layout

12. **Breadcrumbs for depth.** Any page deeper than one level (e.g., `/tasks/$taskId`) shows breadcrumbs for orientation and quick navigation back.

13. **Two-column sidebar.** The left icon rail shows section icons (Dashboard, Tasks, Agents, etc.). The second column shows sub-page links for the active section. No full-width header — the top bar sits above the main content area only, containing the connection badge (and eventually search).

14. **URL is the source of truth for view state.** Filters, active tabs, and sort order should be reflected in URL search params so views are shareable and bookmarkable.

---

## Conventions

### File Naming

| Type          | Convention                  | Example                           |
|---------------|-----------------------------|-----------------------------------|
| Component     | PascalCase                  | `TaskCard.tsx`                    |
| Hook          | camelCase with `use` prefix | `useTaskSubscription.ts`          |
| Utility       | camelCase                   | `statusColors.ts`                |
| Test          | Same name + `.test` suffix  | `TaskCard.test.tsx`              |
| CSS Module    | Same name + `.module.css`   | `AppShell.module.css`            |

**No barrel files.** Don't create `index.ts` re-export files. Import directly from the source file.

### Comments

- **Minimal comments.** Only add comments when the code genuinely isn't obvious. If you feel the need to comment a section, consider whether the code should be clearer or split into a separate file instead.
- **Don't use comments as section dividers.** No `// --- Section ---` or `// ========` banners. If a file has distinct sections that need labeling, it's too big — split it up.
- **Don't narrate the obvious.** Comments like `// Create the router` above `createRouter()` add noise.

```tsx
// ❌ Bad — narrating the obvious, section dividers
// --- Hooks ---
const { ctx } = useSpacetime(); // get the spacetime context

// --- Render ---
return <Title>Dashboard</Title>;

// ✅ Good — explaining a non-obvious decision
// SpacetimeDB timestamps are microseconds since epoch, not milliseconds
const ms = Number(row.createdAt) / 1000;
```

### Exports

- **Named exports only** — no default exports, anywhere.
- Component name must match the file name.

```tsx
// ✅ Good — features/tasks/components/TaskCard.tsx
export function TaskCard({ task }: TaskCardProps) { ... }

// ❌ Bad — default export
export default function TaskCard({ task }: TaskCardProps) { ... }
```

### Props

- Define props as a `type` (not `interface`) named `<ComponentName>Props`.
- Co-locate props with the component in the same file.

```tsx
type TaskCardProps = {
  task: AgentTask;
  onSelect?: (taskId: bigint) => void;
};

export function TaskCard({ task, onSelect }: TaskCardProps) { ... }
```

### Styling

Order of preference:

1. **Mantine component props** for spacing, colors, typography, layout.
2. **CSS Modules** when Mantine props aren't enough (complex layouts, animations).
3. **Global CSS** only for base resets and markdown styles in `index.css`.
4. **Never use inline `style` objects** unless for truly dynamic values (e.g., calculated widths).

### Import Grouping

Group imports in this order, separated by blank lines:

1. React / external libraries
2. Config, lib, and shared code (`@/config/`, `@/lib/`, `@/shared/`)
3. Other features (`@/features/`)
4. Feature-internal code (`./components/`, `./hooks/`)
5. Module bindings (`@/module_bindings/`)
6. Types (if separate)

```tsx
import { useCallback, useState } from "react";
import { createRoute } from "@tanstack/react-router";
import { Badge, Stack, Title } from "@mantine/core";

import { rootRoute } from "@/config/routes";
import { useSubscription } from "@/lib/subscriptions/hooks/useSubscription";
import { useQuery } from "@/lib/subscriptions/hooks/useQuery";
import { taskStatusColor } from "@/shared/utils/statusColors";

import { TaskCard } from "./components/TaskCard";

import { tables } from "@/module_bindings";
```

Use `@/` alias imports for anything outside the current feature. Use relative imports (`./`) only within the same feature.

### SpacetimeDB Patterns

The SpacetimeDB subscription infrastructure lives in `lib/subscriptions/`. It provides three layers:

1. **`SubscriptionManager`** — singleton that manages subscription lifecycle with retain/release and grace TTL. Called from route loaders to preload data before components mount.
2. **`useSubscription`** — React hook that retains a subscription for the component's lifetime and returns `{ subscribed, error }`.
3. **`useQuery`** — React hook that reads from the SpacetimeDB client cache and re-renders when the subscription's tables change.

The router context exposes the `subscriptionManager` so loaders can call `ensure()` before the page component mounts.

### Reading data

Define a `SubscriptionDef` at module scope (stable reference). Call `ensure()` in the loader, then `useSubscription` + `useQuery` in the component.

```tsx
import { useSubscription } from "@/lib/subscriptions/hooks/useSubscription";
import { useQuery } from "@/lib/subscriptions/hooks/useQuery";
import { tables } from "@/module_bindings";
import type { SubscriptionDef } from "@/lib/subscriptions/subscription";

const SUBSCRIPTION_KEY = "tasks";

const TASKS_SUBSCRIPTION: SubscriptionDef = {
  query: [tables.agent_task, tables.observation],
  tables: ["agent_task", "observation"],
};

export const tasksRoute = createRoute({
  loader: async ({ context: { spacetime } }) => {
    await spacetime.subscriptionManager.ensure(SUBSCRIPTION_KEY, TASKS_SUBSCRIPTION);
  },
  // ...
});

function TasksPage() {
  const sub = useSubscription(SUBSCRIPTION_KEY, TASKS_SUBSCRIPTION);
  const tasks = useQuery(SUBSCRIPTION_KEY, (db) => Array.from(db.agent_task.iter()));
  // ...
}
```

The `query` field takes `tables.X` query builder references (the SDK v2 type-safe API). The `tables` field lists the string table names for wiring `onInsert`/`onUpdate`/`onDelete` change notifications.

### Writing data

Use `useSpacetimeDB()` from the SDK to get the connection for reducer calls.

```tsx
import { useSpacetimeDB } from "spacetimedb/react";
import type { DbConnection } from "@/module_bindings";

const { getConnection } = useSpacetimeDB();
const conn = getConnection() as DbConnection | null;
conn?.reducers.createTask({ agentTypeId, title, description });
```

### Subscription keys

Use stable, descriptive string keys. For routes without dynamic parameters, a simple name is fine (`"dashboard"`, `"tasks"`). For parameterized routes (e.g., `/tasks/$taskId`), consider including the param in the key: `\`task-details:${taskId}\``.

### Module bindings

Never edit `module_bindings/`. Regenerate with `just generate` after any SpacetimeDB module changes.

**Rust `Option<T>`** maps to `T | undefined` in TypeScript (NOT `T | null`).

---

## Templates

Copy-paste starters for common file types. Replace placeholder names and customize to your needs.

### Feature Page with Route

Pages parent to a **section layout route** (not `rootRoute`). The path is relative to the section's prefix.

```tsx
// features/<name>/<Name>Page.tsx
import { createRoute } from "@tanstack/react-router";
import { Skeleton, Stack, Text, Title } from "@mantine/core";

import { exampleLayout } from "@/config/routes"; // section layout
import { useSubscription } from "@/lib/subscriptions/hooks/useSubscription";
import { useQuery } from "@/lib/subscriptions/hooks/useQuery";

import { tables } from "@/module_bindings";
import type { SubscriptionDef } from "@/lib/subscriptions/subscription";

const SUBSCRIPTION_KEY = "example";

const EXAMPLE_SUBSCRIPTION: SubscriptionDef = {
  query: [tables.agent_task],
  tables: ["agent_task"],
};

export const exampleRoute = createRoute({
  preload: true,
  getParentRoute: () => exampleLayout, // parent is the section layout
  path: "/",                           // index route within the section
  component: ExamplePage,
  loader: async ({ context: { spacetime } }) => {
    await spacetime.subscriptionManager.ensure(SUBSCRIPTION_KEY, EXAMPLE_SUBSCRIPTION);
  },
});

function ExamplePage() {
  const sub = useSubscription(SUBSCRIPTION_KEY, EXAMPLE_SUBSCRIPTION);
  const tasks = useQuery(SUBSCRIPTION_KEY, (db) => Array.from(db.agent_task.iter()));

  if (!sub.subscribed) {
    return (
      <Stack gap="sm">
        <Skeleton height={60} radius="md" />
        <Skeleton height={60} radius="md" />
        <Skeleton height={60} radius="md" />
      </Stack>
    );
  }

  return (
    <Stack gap="lg">
      <Title order={3}>Example</Title>
      {(tasks ?? []).length === 0 ? (
        <Text c="dimmed">No items yet — create one to get started.</Text>
      ) : (
        <Stack gap="sm">
          {(tasks ?? []).map((task) => (
            <Text key={Number(task.id)}>{task.title}</Text>
          ))}
        </Stack>
      )}
    </Stack>
  );
}
```

### Feature Page with Route + URL Params

```tsx
// features/<name>/<Name>DetailsPage.tsx
import { createRoute, Link } from "@tanstack/react-router";
import { Anchor, Breadcrumbs, Skeleton, Stack, Text, Title } from "@mantine/core";

import { exampleLayout } from "@/config/routes"; // same section layout as list page
import { useSubscription } from "@/lib/subscriptions/hooks/useSubscription";
import { useQuery } from "@/lib/subscriptions/hooks/useQuery";

import { tables } from "@/module_bindings";
import type { SubscriptionDef } from "@/lib/subscriptions/subscription";

const SUBSCRIPTION_KEY = "example-details";

const EXAMPLE_DETAILS_SUBSCRIPTION: SubscriptionDef = {
  query: [tables.agent_task],
  tables: ["agent_task"],
};

export const exampleDetailsRoute = createRoute({
  preload: true,
  getParentRoute: () => exampleLayout,
  path: "/$exampleId",              // relative to section prefix
  component: ExampleDetailsPage,
  loader: async ({ context: { spacetime } }) => {
    await spacetime.subscriptionManager.ensure(SUBSCRIPTION_KEY, EXAMPLE_DETAILS_SUBSCRIPTION);
  },
});

function ExampleDetailsPage() {
  const { exampleId } = exampleDetailsRoute.useParams();

  const sub = useSubscription(SUBSCRIPTION_KEY, EXAMPLE_DETAILS_SUBSCRIPTION);
  const item = useQuery(SUBSCRIPTION_KEY, (db) => {
    for (const row of db.agent_task.iter()) {
      if (Number(row.id) === Number(exampleId)) return row;
    }
    return undefined;
  });

  if (!sub.subscribed) {
    return (
      <Stack gap="sm">
        <Skeleton height={30} width="40%" />
        <Skeleton height={120} radius="md" />
      </Stack>
    );
  }

  return (
    <Stack gap="lg">
      <Breadcrumbs>
        <Anchor component={Link} to="/" size="sm">Dashboard</Anchor>
        <Anchor component={Link} to="/example" size="sm">Example</Anchor>
        <Text size="sm">#{exampleId}</Text>
      </Breadcrumbs>

      {!item ? (
        <Text c="dimmed">Item not found.</Text>
      ) : (
        <Title order={3}>{item.title}</Title>
      )}
    </Stack>
  );
}
```

### Feature Component

```tsx
// features/<name>/components/<ComponentName>.tsx
import { Badge, Group, Paper, Text } from "@mantine/core";

import { taskStatusColor } from "@/shared/utils/statusColors";

import type { AgentTask } from "@/module_bindings";

type TaskCardProps = {
  task: AgentTask;
  onSelect?: (taskId: bigint) => void;
};

export function TaskCard({ task, onSelect }: TaskCardProps) {
  return (
    <Paper
      withBorder
      radius="md"
      p="md"
      onClick={() => onSelect?.(task.id)}
      style={{ cursor: onSelect ? "pointer" : undefined }}
    >
      <Group justify="space-between">
        <Text fw={500}>{task.title}</Text>
        <Badge size="sm" variant="light" color={taskStatusColor(task.status.tag)}>
          {task.status.tag}
        </Badge>
      </Group>
      {task.description && (
        <Text size="sm" c="dimmed" mt="xs" lineClamp={2}>
          {task.description}
        </Text>
      )}
    </Paper>
  );
}
```

### Feature Subscription Hook

```tsx
// features/<name>/hooks/use<Name>Subscription.ts
import { useSubscription } from "@/lib/subscriptions/hooks/useSubscription";
import { useQuery } from "@/lib/subscriptions/hooks/useQuery";

import { tables } from "@/module_bindings";
import type { SubscriptionDef } from "@/lib/subscriptions/subscription";

const SUBSCRIPTION_KEY = "tasks";

const TASKS_SUBSCRIPTION: SubscriptionDef = {
  query: [tables.agent_task, tables.observation],
  tables: ["agent_task", "observation"],
};

export function useTaskList() {
  const sub = useSubscription(SUBSCRIPTION_KEY, TASKS_SUBSCRIPTION);

  const tasks = useQuery(SUBSCRIPTION_KEY, (db) =>
    Array.from(db.agent_task.iter()).sort(
      (a, b) =>
        b.createdAt.toDate().getTime() - a.createdAt.toDate().getTime(),
    ),
  );

  const observationsByTask = useQuery(SUBSCRIPTION_KEY, (db) => {
    const map = new Map<bigint, number>();
    for (const obs of db.observation.iter()) {
      map.set(obs.taskId, (map.get(obs.taskId) ?? 0) + 1);
    }
    return map;
  });

  return {
    tasks,
    observationsByTask,
    subscribed: sub.subscribed,
    error: sub.error,
  };
}
```

### Shared Component

```tsx
// shared/components/<ComponentName>/<ComponentName>.tsx
import { Badge, type MantineColor } from "@mantine/core";

type StatusBadgeProps = {
  label: string;
  color: MantineColor;
  size?: "xs" | "sm" | "md";
};

export function StatusBadge({ label, color, size = "sm" }: StatusBadgeProps) {
  return (
    <Badge size={size} variant="light" color={color}>
      {label}
    </Badge>
  );
}
```

### Utility Function

```tsx
// shared/utils/<utilName>.ts  OR  features/<name>/utils/<utilName>.ts
export function taskStatusColor(tag: string): string {
  switch (tag) {
    case "Completed": return "green";
    case "Running":   return "yellow";
    case "Assigned":  return "blue";
    case "Failed":    return "red";
    case "Cancelled": return "gray";
    case "Pending":   return "gray";
    default:          return "gray";
  }
}
```

### CSS Module

```css
.wrapper {
  position: relative;
  overflow: hidden;
}

.header {
  display: flex;
  align-items: center;
  gap: var(--mantine-spacing-sm);
}

.muted {
  color: var(--mantine-color-dimmed);
  font-size: var(--mantine-font-size-sm);
}
```

```tsx
import classes from "./Component.module.css";

export function Component() {
  return (
    <div className={classes.wrapper}>
      <div className={classes.header}>...</div>
    </div>
  );
}
```

### Test File

```tsx
import { describe, test, expect } from "bun:test";

import { taskStatusColor } from "./statusColors";

describe("taskStatusColor", () => {
  test("returns green for completed tasks", () => {
    expect(taskStatusColor("Completed")).toBe("green");
  });

  test("returns gray for unknown tags", () => {
    expect(taskStatusColor("SomethingNew")).toBe("gray");
  });
});
```

---

## Testing

Tests use the **Bun test runner** (`bun test`).

### File Locations

- **Unit / component tests** live next to the file they test:
  ```
  features/tasks/utils/taskHelpers.ts
  features/tasks/utils/taskHelpers.test.ts
  ```
- **Integration tests** live in `src/__tests__/`:
  ```
  src/__tests__/taskCreation.test.tsx
  ```

### Naming

- Test files: `<filename>.test.ts` or `<filename>.test.tsx`
- Describe blocks: match the module or component name
- Test names: describe behavior, not implementation

### What to Test

| Priority | What                                      | How                          |
|----------|-------------------------------------------|------------------------------|
| High     | Utility functions                         | Unit tests with `bun:test`   |
| High     | Data transformations / selectors          | Unit tests                   |
| Medium   | Hook logic (subscriptions, derived state) | Hook tests                   |
| Medium   | Complex component interactions            | Component tests              |
| Low      | Simple presentational components          | Covered by page-level tests  |

---

## Scripts

| Command            | Description                                           |
|--------------------|-------------------------------------------------------|
| `bun run dev`      | Start Vite dev server with HMR (port 5173)            |
| `bun run build`    | Type-check and build for production                   |
| `bun run preview`  | Preview the production build locally                  |
| `bun run typecheck`| Run TypeScript type checking (no emit)                |
| `bun test`         | Run all tests                                         |
| `just generate`    | Regenerate SpacetimeDB module bindings (from repo root) |
| `just dev`         | Same as `bun run dev` but from repo root via Justfile |
