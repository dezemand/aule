---
name: frontend
description: Frontend architecture, conventions, and templates for the app/ directory. Use when creating or editing React components, pages, hooks, routes, or any TypeScript file under app/src/.
user-invocable: false
---

The full development guide is at `docs/frontend.md`. This skill contains the rules and templates you must follow when writing frontend code.

## Project Structure

```
app/src/
├── main.tsx                     # Bootstrap: providers, router, mount
├── index.css                    # Global styles (resets, typography, markdown)
├── global.d.ts                  # Ambient type declarations (CSS modules)
├── config/                      # App-wide configuration
│   ├── theme.ts                 #   Mantine theme object
│   ├── routes.tsx               #   Root route, section layouts, RouteContext type
│   └── spacetime.ts             #   SpacetimeDB connection builder
├── lib/                         # Infrastructure libraries
│   └── subscriptions/           #   SpacetimeDB subscription system
│       ├── subscription.ts      #     SubscriptionDef type
│       ├── subscriptionManager.ts #   Singleton manager (ensure/retain/release)
│       ├── tableEventBus.ts     #     Microtask-coalesced event bus
│       └── hooks/
│           ├── useSpacetimeConnection.ts  # Zustand store + subscriptionManager singleton
│           ├── useSubscription.ts         # Subscription lifecycle hook
│           └── useQuery.ts               # Scoped reactive query hook
├── shared/
│   ├── components/              # Reusable UI (AppShell/, TopBar/, Markdown/, ConnectionBadge/)
│   └── utils/                   # Pure utilities (statusColors.ts, formatting.ts, promise.ts)
├── features/                    # Feature modules — one per domain
│   ├── dashboard/
│   ├── tasks/
│   ├── agents/
│   └── conversation/
└── module_bindings/             # AUTO-GENERATED — never edit
```

### Feature Module Layout

```
features/<name>/
├── <Name>Page.tsx          # Page component + co-located route definition
├── components/             # Feature-specific components
├── hooks/                  # Feature-specific hooks (optional)
└── utils/                  # Feature-specific utilities (optional)
```

### Where Code Goes

- Page + route definition: `features/<name>/<Name>Page.tsx`
- Feature-specific component: `features/<name>/components/`
- Reusable component (2+ features): `shared/components/`
- SpacetimeDB subscription infrastructure: `lib/subscriptions/`
- Feature-specific hook: `features/<name>/hooks/`
- Utility function: `shared/utils/` or `features/<name>/utils/`
- Config (theme, root route, section layouts): `config/`

**Promotion rule:** Start in the feature. Move to `shared/` only when a second feature needs it.

## Path Aliases

Use `@/` alias (mapped to `src/`) for imports across features or from shared code. Use relative imports (`./`) only within the same feature.

```tsx
// ✅ Alias for cross-feature imports
import { useSubscription } from "@/lib/subscriptions/hooks/useSubscription";
import { useQuery } from "@/lib/subscriptions/hooks/useQuery";
import { taskStatusColor } from "@/shared/utils/statusColors";
import { tables } from "@/module_bindings";

// ✅ Relative for same-feature imports
import { TaskCard } from "./components/TaskCard";

// ❌ Never relative across features
import { useSubscription } from "../../lib/subscriptions/hooks/useSubscription";
```

## Conventions

- **Named exports only.** No default exports anywhere.
- **No barrel files.** Don't create `index.ts` re-export files. Import directly from the source file.
- **Component names match file names.** `TaskCard.tsx` exports `TaskCard`.
- **Props as `type`, not `interface`.** Named `<ComponentName>Props`, co-located in the same file.
- **File naming:** Components=PascalCase, hooks=camelCase with `use` prefix, utils=camelCase, tests=`*.test.tsx`/`*.test.ts`.
- **Import order:** (1) React/external, (2) `@/config/`, `@/lib/`, `@/shared/`, (3) `@/features/`, (4) feature-internal `./`, (5) `@/module_bindings`, (6) types.
- **Styling preference:** Mantine props first, CSS Modules second, global CSS never (except `index.css`), inline styles never (except dynamic values).
- **Minimal comments.** Only comment when code genuinely isn't obvious. Don't use comments as section dividers — if a file needs sections, split it into separate files. Don't narrate the obvious.

## Routing

Routes are **co-located** with their page component and parent to **section layout routes**. The root route and section layouts live in `config/routes.tsx`. The route tree is assembled in `main.tsx`.

Each nav section (Dashboard, Tasks, Agents) has a layout route. Page routes parent to their section layout with relative paths.

```tsx
// features/tasks/TaskDetailsPage.tsx
import { createRoute } from "@tanstack/react-router";
import { tasksLayout } from "@/config/routes";

export const taskDetailsRoute = createRoute({
  getParentRoute: () => tasksLayout,
  path: "/$taskId",            // relative to /tasks
  component: TaskDetailsPage,
});

function TaskDetailsPage() {
  const { taskId } = taskDetailsRoute.useParams();
  // ...
}
```

```tsx
// main.tsx — assembles the nested tree
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

Dependency flow: `config/routes` <- `features/*` <- `main.tsx`. No circular imports.

## UX Principles

1. **Skeletons, not spinners.** Use Mantine `Skeleton` to show content shape while loading. Never render "Loading..." text.
2. **Distinct empty states.** "No data yet" (guide user to action) vs "No results for filter" (suggest clearing filter).
3. **Don't block on subscriptions.** Show skeleton placeholders for data areas while SpacetimeDB subscription is pending.
4. **No flash on real-time updates.** SpacetimeDB pushes frequent updates. Avoid layout jumps or full re-renders for single-row changes.
5. **Live relative timestamps.** "2 minutes ago" ticks on a shared interval (~30s).
6. **New items don't displace.** Show "N new items" indicator instead of pushing content the user is reading.
7. **Progressive disclosure.** Summary first, details on demand.
8. **Status at a glance.** Color-coded badges: green=done, yellow=in-progress, red=error, gray=inactive.
9. **Inline forms, not modals.** Expandable inline forms preserve context.
10. **Optimistic reducer feedback.** Disable the trigger immediately after calling a reducer. Subscription confirms the change.
11. **Confirm destructive actions inline.** Not a modal.
12. **Breadcrumbs for nested pages.** Any route deeper than one level.
13. **Two-column sidebar.** Icon rail for sections, second column for sub-page links. Top bar above main content only (no full-width header).
14. **URL = view state.** Filters, tabs, sort in search params.

## SpacetimeDB Patterns

SpacetimeDB subscription infrastructure lives in `lib/subscriptions/`. It provides three layers:

1. **`SubscriptionManager`** — singleton that manages subscription lifecycle with retain/release and grace TTL. Called from route loaders to preload data.
2. **`useSubscription(key, def)`** — React hook that retains a subscription for the component's lifetime and returns `{ subscribed, error }`.
3. **`useQuery(key, selector)`** — React hook that reads from the SpacetimeDB client cache and re-renders when the subscription's tables change.

**Reading data:**
```tsx
import { useSubscription } from "@/lib/subscriptions/hooks/useSubscription";
import { useQuery } from "@/lib/subscriptions/hooks/useQuery";
import { tables } from "@/module_bindings";
import type { SubscriptionDef } from "@/lib/subscriptions/subscription";

const SUBSCRIPTION_KEY = "tasks";

const TASKS_SUBSCRIPTION: SubscriptionDef = {
  query: [tables.agent_task],
  tables: ["agent_task"],
};

// In route loader:
export const tasksRoute = createRoute({
  loader: async ({ context: { spacetime } }) => {
    await spacetime.subscriptionManager.ensure(SUBSCRIPTION_KEY, TASKS_SUBSCRIPTION);
  },
  // ...
});

// In component:
function TasksPage() {
  const sub = useSubscription(SUBSCRIPTION_KEY, TASKS_SUBSCRIPTION);
  const tasks = useQuery(SUBSCRIPTION_KEY, (db) => Array.from(db.agent_task.iter()));
  // ...
}
```

**Writing data:**
```tsx
import { useSpacetimeDB } from "spacetimedb/react";
import type { DbConnection } from "@/module_bindings";

const { getConnection } = useSpacetimeDB();
const conn = getConnection() as DbConnection | null;
conn?.reducers.createTask({ agentTypeId, title, description });
```

- Never edit `module_bindings/`. Regenerate with `just generate`.
- Rust `Option<T>` = `T | undefined` in TypeScript (NOT `T | null`).

## Templates

### Feature Page with Route

Pages parent to a section layout route, not `rootRoute` directly. The loader calls `ensure()` to preload subscriptions.

```tsx
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
        <Text c="dimmed">No items yet.</Text>
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

### Feature Page with URL Params

```tsx
import { createRoute } from "@tanstack/react-router";
import { exampleLayout } from "@/config/routes";

export const exampleDetailsRoute = createRoute({
  getParentRoute: () => exampleLayout,
  path: "/$exampleId",              // relative to section prefix
  component: ExampleDetailsPage,
});

function ExampleDetailsPage() {
  const { exampleId } = exampleDetailsRoute.useParams();
  // ...
}
```

### Feature Component

```tsx
import { Badge, Group, Paper, Text } from "@mantine/core";

import type { AgentTask } from "@/module_bindings/types";

type TaskCardProps = {
  task: AgentTask;
  onSelect?: (taskId: bigint) => void;
};

export function TaskCard({ task, onSelect }: TaskCardProps) {
  return (
    <Paper withBorder radius="md" p="md" onClick={() => onSelect?.(task.id)}>
      <Group justify="space-between">
        <Text fw={500}>{task.title}</Text>
      </Group>
    </Paper>
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
});
```

## Testing

- Tests live next to the file they test: `TaskCard.test.tsx` beside `TaskCard.tsx`.
- Integration tests go in `src/__tests__/`.
- Use `bun test` to run. Import from `bun:test`.
- Describe blocks match the module name. Test names describe behavior.
