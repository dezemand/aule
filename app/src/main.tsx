import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import {
  RouterProvider,
  createRouter,
  createRootRoute,
  createRoute,
  Outlet,
} from "@tanstack/react-router";
import { SpacetimeProvider } from "./hooks/useSpacetime";
import { AppShell } from "./components/AppShell";
import { DashboardPage } from "./routes/DashboardPage";
import { TaskDetailsPage } from "./routes/TaskDetailsPage";
import { TasksPage } from "./routes/TasksPage";
import { AgentTypesPage } from "./routes/AgentTypesPage";

// --- Route tree ---

const rootRoute = createRootRoute({
  component: () => (
    <SpacetimeProvider>
      <AppShell>
        <Outlet />
      </AppShell>
    </SpacetimeProvider>
  ),
});

const indexRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/",
  component: DashboardPage,
});

const tasksRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/tasks",
  component: TasksPage,
});

const taskDetailsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/tasks/$taskId",
  component: TaskDetailsPage,
});

const agentTypesRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/agent-types",
  component: AgentTypesPage,
});

const routeTree = rootRoute.addChildren([
  indexRoute,
  tasksRoute,
  taskDetailsRoute,
  agentTypesRoute,
]);

const router = createRouter({ routeTree });

// --- Mount ---

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <RouterProvider router={router} />
  </StrictMode>
);
