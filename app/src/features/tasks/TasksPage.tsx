import { createRoute } from "@tanstack/react-router";
import { useState } from "react";
import { Button, Group, Skeleton, Stack, Text, Title } from "@mantine/core";
import { useSpacetimeDB } from "spacetimedb/react";

import { tasksLayout } from "@/config/routes";
import { useSubscription } from "@/lib/subscriptions/hooks/useSubscription";
import { useQuery } from "@/lib/subscriptions/hooks/useQuery";

import { CreateTaskForm } from "./components/CreateTaskForm";
import { TaskCard } from "./components/TaskCard";

import { tables } from "@/module_bindings";
import type { DbConnection } from "@/module_bindings";
import type { SubscriptionDef } from "@/lib/subscriptions/subscription";

const SUBSCRIPTION_KEY = "tasks";

const TASKS_SUBSCRIPTION: SubscriptionDef = {
  query: [
    tables.agent_task,
    tables.agent_type,
    tables.observation,
    tables.runtime_event,
    tables.agent_runtime,
  ],
  tables: [
    "agent_task",
    "agent_type",
    "observation",
    "runtime_event",
    "agent_runtime",
  ],
};

type StatusFilter = "all" | "active" | "completed" | "failed";

export const tasksRoute = createRoute({
  preload: true,
  getParentRoute: () => tasksLayout,
  path: "/",
  component: TasksPage,
  loader: ({ context: { spacetime } }) => {
    spacetime.subscriptionManager.ensure(SUBSCRIPTION_KEY, TASKS_SUBSCRIPTION);
  },
});

function TasksPage() {
  const { getConnection } = useSpacetimeDB();
  const conn = getConnection() as DbConnection | null;

  const sub = useSubscription(SUBSCRIPTION_KEY, TASKS_SUBSCRIPTION);
  const tasks = useQuery(SUBSCRIPTION_KEY, (db) =>
    Array.from(db.agent_task.iter()),
  );
  const agentTypes = useQuery(SUBSCRIPTION_KEY, (db) =>
    Array.from(db.agent_type.iter()),
  );
  const observations = useQuery(SUBSCRIPTION_KEY, (db) =>
    Array.from(db.observation.iter()),
  );
  const runtimeEvents = useQuery(SUBSCRIPTION_KEY, (db) =>
    Array.from(db.runtime_event.iter()),
  );
  const runtimes = useQuery(SUBSCRIPTION_KEY, (db) =>
    Array.from(db.agent_runtime.iter()),
  );

  const [filter, setFilter] = useState<StatusFilter>("all");
  const [showCreate, setShowCreate] = useState(false);

  if (!sub.subscribed) {
    return (
      <Stack gap="lg">
        <Group justify="space-between">
          <Title order={3}>Tasks</Title>
        </Group>
        <Stack gap="sm">
          {Array.from({ length: 3 }).map((_, i) => (
            <Skeleton key={i} height={100} radius="md" />
          ))}
        </Stack>
      </Stack>
    );
  }

  const allTasks = tasks ?? [];
  const filtered = allTasks.filter((task) => {
    const tag = task.status.tag;
    switch (filter) {
      case "active":
        return tag === "Pending" || tag === "Assigned" || tag === "Running";
      case "completed":
        return tag === "Completed";
      case "failed":
        return tag === "Failed";
      default:
        return true;
    }
  });

  const sorted = [...filtered].sort(
    (a, b) => b.createdAt.toDate().getTime() - a.createdAt.toDate().getTime(),
  );

  const agentTypeMap = new Map(
    (agentTypes ?? []).map((agentType) => [
      Number(agentType.id),
      agentType.name,
    ]),
  );
  const runtimeNameByIdentity = new Map(
    (runtimes ?? []).map((runtime) => [
      runtime.identity.toHexString(),
      runtime.name,
    ]),
  );
  const maxTurnByTask = new Map<string, number>();
  for (const event of runtimeEvents ?? []) {
    const key = event.taskId.toString();
    maxTurnByTask.set(
      key,
      Math.max(maxTurnByTask.get(key) ?? 0, Number(event.turn)),
    );
  }

  function getTaskObservations(taskId: bigint) {
    return (observations ?? [])
      .filter((observation) => observation.taskId === taskId)
      .sort(
        (a, b) =>
          a.createdAt.toDate().getTime() - b.createdAt.toDate().getTime(),
      );
  }

  return (
    <Stack gap="lg">
      <Group justify="space-between">
        <Title order={3}>Tasks</Title>
        <Button size="sm" onClick={() => setShowCreate(!showCreate)}>
          {showCreate ? "Cancel" : "New Task"}
        </Button>
      </Group>

      {showCreate && conn && (
        <CreateTaskForm
          conn={conn}
          agentTypes={agentTypes ?? []}
          onCreated={() => setShowCreate(false)}
        />
      )}

      <Group gap="xs">
        {(["all", "active", "completed", "failed"] as StatusFilter[]).map(
          (nextFilter) => (
            <Button
              key={nextFilter}
              size="xs"
              variant={filter === nextFilter ? "filled" : "subtle"}
              color="gray"
              tt="capitalize"
              onClick={() => setFilter(nextFilter)}
            >
              {nextFilter}
            </Button>
          ),
        )}
      </Group>

      {sorted.length === 0 ? (
        <Text size="sm" c="dimmed">
          No tasks found.
        </Text>
      ) : (
        <Stack gap="sm">
          {sorted.map((task) => (
            <TaskCard
              key={Number(task.id)}
              task={task}
              agentTypeName={
                agentTypeMap.get(Number(task.agentTypeId)) ?? "Unknown"
              }
              runtimeName={
                task.assignedRuntime
                  ? (runtimeNameByIdentity.get(
                      task.assignedRuntime.toHexString(),
                    ) ?? "Unknown runtime")
                  : "Unassigned"
              }
              latestTurn={maxTurnByTask.get(task.id.toString()) ?? 0}
              observations={getTaskObservations(task.id)}
              runtimes={runtimes ?? []}
              conn={conn}
              now={Date.now()}
            />
          ))}
        </Stack>
      )}
    </Stack>
  );
}
