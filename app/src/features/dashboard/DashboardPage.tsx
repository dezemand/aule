import { createRoute } from "@tanstack/react-router";
import { Center, SimpleGrid, Skeleton, Stack, Text, Title } from "@mantine/core";

import { dashboardLayout } from "@/config/routes";
import { useSubscription } from "@/lib/subscriptions/hooks/useSubscription";
import { useQuery } from "@/lib/subscriptions/hooks/useQuery";

import { RecentObservations } from "./components/RecentObservations";
import { RuntimesTable } from "./components/RuntimesTable";
import { StatCard } from "./components/StatCard";

import { tables } from "@/module_bindings";
import type { SubscriptionDef } from "@/lib/subscriptions/subscription";

const SUBSCRIPTION_KEY = "dashboard";

const DASHBOARD_SUBSCRIPTION: SubscriptionDef = {
  query: [
    tables.agent_runtime,
    tables.agent_task,
    tables.observation,
    tables.agent_type,
  ],
  tables: ["agent_runtime", "agent_task", "observation", "agent_type"],
};

export const indexRoute = createRoute({
  preload: true,
  getParentRoute: () => dashboardLayout,
  path: "/",
  component: DashboardPage,
  loader: ({ context: { spacetime } }) => {
    spacetime.subscriptionManager.ensure(
      SUBSCRIPTION_KEY,
      DASHBOARD_SUBSCRIPTION,
    );
  },
});

function DashboardPage() {
  const sub = useSubscription(SUBSCRIPTION_KEY, DASHBOARD_SUBSCRIPTION);
  const runtimes = useQuery(SUBSCRIPTION_KEY, (db) =>
    Array.from(db.agent_runtime.iter()),
  );
  const tasks = useQuery(SUBSCRIPTION_KEY, (db) =>
    Array.from(db.agent_task.iter()),
  );
  const observations = useQuery(SUBSCRIPTION_KEY, (db) =>
    Array.from(db.observation.iter()),
  );
  const agentTypes = useQuery(SUBSCRIPTION_KEY, (db) =>
    Array.from(db.agent_type.iter()),
  );

  if (sub.error) {
    return (
      <Center h="100%">
        <Text c="red.4">Subscription error: {sub.error}</Text>
      </Center>
    );
  }

  if (!sub.subscribed) {
    return (
      <Stack gap="lg">
        <Title order={3}>Dashboard</Title>
        <SimpleGrid cols={{ base: 2, sm: 4 }}>
          {Array.from({ length: 4 }).map((_, i) => (
            <Skeleton key={i} height={80} radius="md" />
          ))}
        </SimpleGrid>
        <Skeleton height={120} radius="md" />
        <Skeleton height={160} radius="md" />
      </Stack>
    );
  }

  const allRuntimes = runtimes ?? [];
  const allTasks = tasks ?? [];
  const allObservations = observations ?? [];
  const allAgentTypes = agentTypes ?? [];

  const onlineRuntimes = allRuntimes.filter((r) => r.status.tag !== "Offline");
  const activeTasks = allTasks.filter(
    (t) =>
      t.status.tag !== "Completed" &&
      t.status.tag !== "Failed" &&
      t.status.tag !== "Cancelled",
  );
  const recentObs = [...allObservations]
    .sort(
      (a, b) =>
        b.createdAt.toDate().getTime() - a.createdAt.toDate().getTime(),
    )
    .slice(0, 10);

  return (
    <Stack gap="lg">
      <Title order={3}>Dashboard</Title>

      <SimpleGrid cols={{ base: 2, sm: 4 }}>
        <StatCard title="Agent Types" value={allAgentTypes.length} />
        <StatCard
          title="Runtimes"
          value={onlineRuntimes.length}
          sub={`${allRuntimes.length} total`}
        />
        <StatCard
          title="Active Tasks"
          value={activeTasks.length}
          sub={`${allTasks.length} total`}
        />
        <StatCard title="Observations" value={allObservations.length} />
      </SimpleGrid>

      <section>
        <Text size="xs" fw={500} tt="uppercase" lts={0.5} c="dimmed" mb="sm">
          Runtimes
        </Text>
        <RuntimesTable runtimes={allRuntimes} />
      </section>

      <section>
        <Text size="xs" fw={500} tt="uppercase" lts={0.5} c="dimmed" mb="sm">
          Recent Observations
        </Text>
        <RecentObservations observations={recentObs} />
      </section>
    </Stack>
  );
}
