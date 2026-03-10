import {
  createRootRouteWithContext,
  createRoute,
  Outlet,
} from "@tanstack/react-router";
import { AppShell } from "@/shared/components/AppShell/AppShell";
import type { DbConnection } from "@/module_bindings";
import type { SubscriptionManager } from "@/lib/subscriptions/subscriptionManager";

export type SpacetimeRouterContext = {
  getConnection: () => Promise<DbConnection>;
  connection: DbConnection | null;
  subscriptionManager: SubscriptionManager;
};

export type RouteContext = {
  spacetime: SpacetimeRouterContext;
};

export const rootRoute = createRootRouteWithContext<RouteContext>()({
  component: () => (
    <AppShell>
      <Outlet />
    </AppShell>
  ),
});

export const dashboardLayout = createRoute({
  getParentRoute: () => rootRoute,
  id: "dashboard",
});

export const tasksLayout = createRoute({
  getParentRoute: () => rootRoute,
  path: "/tasks",
});

export const agentsLayout = createRoute({
  getParentRoute: () => rootRoute,
  path: "/agent-types",
});
