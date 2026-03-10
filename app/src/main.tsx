import { StrictMode, useMemo } from "react";
import { createRoot } from "react-dom/client";
import { RouterProvider, createRouter } from "@tanstack/react-router";
import { MantineProvider } from "@mantine/core";
import "@mantine/core/styles.css";
import "./index.css";
import { theme } from "@/config/theme";
import {
  rootRoute,
  dashboardLayout,
  tasksLayout,
  agentsLayout,
  type RouteContext,
} from "@/config/routes";
import { indexRoute } from "@/features/dashboard/DashboardPage";
import { tasksRoute } from "@/features/tasks/TasksPage";
import { taskDetailsRoute } from "@/features/tasks/TaskDetailsPage";
import { agentTypesRoute } from "@/features/agents/AgentTypesPage";
import { DbConnection } from "./module_bindings";
import { SpacetimeDBProvider, useSpacetimeDB } from "spacetimedb/react";
import { getConnectionBuilder } from "./config/spacetime";
import {
  useSpacetimeConnection,
  subscriptionManager,
} from "./lib/subscriptions/hooks/useSpacetimeConnection";

const routeTree = rootRoute.addChildren([
  dashboardLayout.addChildren([indexRoute]),
  tasksLayout.addChildren([tasksRoute, taskDetailsRoute]),
  agentsLayout.addChildren([agentTypesRoute]),
]);

const router = createRouter({
  routeTree,
  defaultPreload: "viewport",
  defaultPreloadStaleTime: Infinity,
  context: {
    spacetime: {
      connection: null,
      getConnection: () => useSpacetimeConnection.getState().connectionPromise,
      subscriptionManager,
    },
  },
});

declare module "@tanstack/react-router" {
  interface Register {
    router: typeof router;
  }
}

function Router() {
  const spacetime = useSpacetimeDB();
  const { connectionPromise } = useSpacetimeConnection();
  const context = {
    spacetime: {
      connection: spacetime.getConnection() as DbConnection | null,
      getConnection: () => connectionPromise,
      subscriptionManager,
    },
  } satisfies RouteContext;
  return <RouterProvider router={router} context={context} />;
}

function App() {
  const { setConnectionError, setConnection, disconnect } =
    useSpacetimeConnection();

  const connectionBuilder = useMemo(
    () =>
      getConnectionBuilder({
        onConnect: (conn, _identity, _token) => {
          setConnection(conn);
        },
        onConnectError: (err) => {
          setConnectionError(err);
        },
        onDisconnect: () => {
          disconnect();
        },
      }),
    [setConnectionError, setConnection, disconnect],
  );

  return (
    <SpacetimeDBProvider connectionBuilder={connectionBuilder}>
      <MantineProvider theme={theme} defaultColorScheme="dark">
        <Router />
      </MantineProvider>
    </SpacetimeDBProvider>
  );
}

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <App />
  </StrictMode>,
);
