import { createRoute } from "@tanstack/react-router";
import { useState } from "react";
import { useSpacetimeDB } from "spacetimedb/react";
import { Box, Button, Group, Skeleton, Stack, Text, Title } from "@mantine/core";

import { agentsLayout } from "@/config/routes";
import { useSubscription } from "@/lib/subscriptions/hooks/useSubscription";
import { useQuery } from "@/lib/subscriptions/hooks/useQuery";

import { AgentTypeCard } from "./components/AgentTypeCard";
import { CreateAgentTypeForm } from "./components/CreateAgentTypeForm";

import { tables } from "@/module_bindings";
import type { DbConnection } from "@/module_bindings";
import type { SubscriptionDef } from "@/lib/subscriptions/subscription";

const SUBSCRIPTION_KEY = "agent-types";

const AGENT_TYPES_SUBSCRIPTION: SubscriptionDef = {
  query: [tables.agent_type, tables.agent_type_version, tables.agent_runtime],
  tables: ["agent_type", "agent_type_version", "agent_runtime"],
};

export const agentTypesRoute = createRoute({
  preload: true,
  getParentRoute: () => agentsLayout,
  path: "/",
  component: AgentTypesPage,
  loader: ({ context: { spacetime } }) => {
    spacetime.subscriptionManager.ensure(
      SUBSCRIPTION_KEY,
      AGENT_TYPES_SUBSCRIPTION,
    );
  },
});

function AgentTypesPage() {
  const { getConnection } = useSpacetimeDB();
  const conn = getConnection() as DbConnection | null;

  const sub = useSubscription(SUBSCRIPTION_KEY, AGENT_TYPES_SUBSCRIPTION);
  const agentTypes = useQuery(SUBSCRIPTION_KEY, (db) =>
    Array.from(db.agent_type.iter()),
  );
  const versions = useQuery(SUBSCRIPTION_KEY, (db) =>
    Array.from(db.agent_type_version.iter()),
  );
  const runtimes = useQuery(SUBSCRIPTION_KEY, (db) =>
    Array.from(db.agent_runtime.iter()),
  );

  const [showCreateType, setShowCreateType] = useState(false);
  const [showCreateVersion, setShowCreateVersion] = useState<bigint | null>(
    null,
  );

  if (!sub.subscribed) {
    return (
      <Stack gap="lg">
        <Group justify="space-between">
          <Title order={3}>Agent Types</Title>
        </Group>
        <Stack gap="md">
          {Array.from({ length: 2 }).map((_, i) => (
            <Skeleton key={i} height={120} radius="md" />
          ))}
        </Stack>
      </Stack>
    );
  }

  const allTypes = agentTypes ?? [];
  const allVersions = versions ?? [];
  const allRuntimes = runtimes ?? [];
  const onlineRuntimeCount = allRuntimes.filter(
    (r) => r.status.tag !== "Offline",
  ).length;

  function versionsForType(typeId: bigint) {
    return allVersions
      .filter((v) => v.agentTypeId === typeId)
      .sort(
        (a, b) =>
          b.createdAt.toDate().getTime() - a.createdAt.toDate().getTime(),
      );
  }

  return (
    <Stack gap="lg">
      <Group justify="space-between">
        <Title order={3}>Agent Types</Title>
        <Group gap="sm">
          <Text size="xs" c="dimmed">
            {onlineRuntimeCount} runtime
            {onlineRuntimeCount !== 1 ? "s" : ""} online
          </Text>
          <Button size="sm" onClick={() => setShowCreateType(!showCreateType)}>
            {showCreateType ? "Cancel" : "New Agent Type"}
          </Button>
        </Group>
      </Group>

      {showCreateType && conn && (
        <CreateAgentTypeForm
          conn={conn}
          onCreated={() => setShowCreateType(false)}
        />
      )}

      {allTypes.length === 0 ? (
        <Text size="sm" c="dimmed">
          No agent types defined yet.
        </Text>
      ) : (
        <Stack gap="md">
          {allTypes.map((at) => (
            <AgentTypeCard
              key={Number(at.id)}
              agentType={at}
              versions={versionsForType(at.id)}
              showCreateVersion={showCreateVersion === at.id}
              onToggleCreateVersion={() =>
                setShowCreateVersion(
                  showCreateVersion === at.id ? null : at.id,
                )
              }
              conn={conn}
            />
          ))}
        </Stack>
      )}
    </Stack>
  );
}
