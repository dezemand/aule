import { createRoute, Link } from "@tanstack/react-router";
import { useEffect, useState } from "react";
import { useSpacetimeDB } from "spacetimedb/react";
import {
  Anchor,
  Badge,
  Box,
  Breadcrumbs,
  Group,
  Progress,
  Skeleton,
  Stack,
  Tabs,
  Text,
  Title,
} from "@mantine/core";

import { tasksLayout } from "@/config/routes";
import { ConversationView } from "@/features/conversation/ConversationView";
import { useSubscription } from "@/lib/subscriptions/hooks/useSubscription";
import { useQuery } from "@/lib/subscriptions/hooks/useQuery";
import { formatDuration } from "@/shared/utils/formatting";
import { taskStatusColor } from "@/shared/utils/statusColors";

import { ObservationsList } from "./components/ObservationsList";
import { TaskDetailsTab } from "./components/TaskDetailsTab";

import { tables } from "@/module_bindings";
import type { DbConnection } from "@/module_bindings";
import type { SubscriptionDef } from "@/lib/subscriptions/subscription";

const SUBSCRIPTION_KEY = "task-details";

const TASK_DETAILS_SUBSCRIPTION: SubscriptionDef = {
  query: [
    tables.agent_task,
    tables.agent_type,
    tables.agent_type_version,
    tables.observation,
    tables.runtime_event,
    tables.agent_runtime,
  ],
  tables: [
    "agent_task",
    "agent_type",
    "agent_type_version",
    "observation",
    "runtime_event",
    "agent_runtime",
  ],
};

export const taskDetailsRoute = createRoute({
  preload: true,
  getParentRoute: () => tasksLayout,
  path: "/$taskId",
  component: TaskDetailsPage,
  loader: ({ context: { spacetime } }) => {
    spacetime.subscriptionManager.ensure(
      SUBSCRIPTION_KEY,
      TASK_DETAILS_SUBSCRIPTION,
    );
  },
});

function TaskBreadcrumbs({ taskLabel }: { taskLabel: string }) {
  return (
    <Breadcrumbs
      separator="/"
      separatorMargin={6}
      styles={{ separator: { color: "var(--mantine-color-dark-2)" } }}
    >
      <Anchor component={Link} to="/" size="xs" c="dimmed">
        Dashboard
      </Anchor>
      <Anchor component={Link} to="/tasks" size="xs" c="dimmed">
        Tasks
      </Anchor>
      <Text size="xs" c="gray.4">
        {taskLabel}
      </Text>
    </Breadcrumbs>
  );
}

function TaskDetailsPage() {
  const { taskId } = taskDetailsRoute.useParams();
  const { getConnection } = useSpacetimeDB();
  const conn = getConnection() as DbConnection | null;
  const [now, setNow] = useState(() => Date.now());

  useEffect(() => {
    const timer = setInterval(() => setNow(Date.now()), 1000);
    return () => clearInterval(timer);
  }, []);

  const sub = useSubscription(SUBSCRIPTION_KEY, TASK_DETAILS_SUBSCRIPTION);
  const tasks = useQuery(SUBSCRIPTION_KEY, (db) =>
    Array.from(db.agent_task.iter()),
  );
  const agentTypes = useQuery(SUBSCRIPTION_KEY, (db) =>
    Array.from(db.agent_type.iter()),
  );
  const agentTypeVersions = useQuery(SUBSCRIPTION_KEY, (db) =>
    Array.from(db.agent_type_version.iter()),
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

  if (!sub.subscribed) {
    return (
      <Stack gap="md">
        <Skeleton height={16} width="30%" />
        <Skeleton height={28} width="60%" />
        <Skeleton height={400} radius="md" />
      </Stack>
    );
  }

  let parsedTaskId: bigint;
  try {
    parsedTaskId = BigInt(taskId);
  } catch {
    return (
      <Stack gap="md">
        <TaskBreadcrumbs taskLabel={taskId} />
        <Text size="sm" c="red.4">
          Invalid task id: {taskId}
        </Text>
      </Stack>
    );
  }

  const task = (tasks ?? []).find((c) => c.id === parsedTaskId);
  const agentTypeMap = new Map(
    (agentTypes ?? []).map((a) => [Number(a.id), a.name]),
  );
  const runtimeNameMap = new Map(
    (runtimes ?? []).map((r) => [r.identity.toHexString(), r.name]),
  );

  if (!task) {
    return (
      <Stack gap="md">
        <TaskBreadcrumbs taskLabel={`#${taskId}`} />
        <Text size="sm" c="dimmed">
          Task #{taskId} was not found.
        </Text>
      </Stack>
    );
  }

  const taskObservations = (observations ?? [])
    .filter((o) => o.taskId === task.id)
    .sort(
      (a, b) =>
        a.createdAt.toDate().getTime() - b.createdAt.toDate().getTime(),
    );

  const taskRuntimeEvents = (runtimeEvents ?? [])
    .filter((e) => e.taskId === task.id)
    .sort(
      (a, b) =>
        a.createdAt.toDate().getTime() - b.createdAt.toDate().getTime(),
    );

  const latestTurn = taskRuntimeEvents.reduce(
    (max, e) => Math.max(max, Number(e.turn)),
    0,
  );

  const activeVersion = (agentTypeVersions ?? []).find(
    (v) => v.agentTypeId === task.agentTypeId && v.status.tag === "Active",
  );

  const isRunning = task.status.tag === "Running";
  const isTerminal =
    task.status.tag === "Completed" ||
    task.status.tag === "Failed" ||
    task.status.tag === "Cancelled";

  const elapsed = task.startedAt
    ? formatDuration(
        (isTerminal && task.completedAt
          ? task.completedAt.toDate().getTime()
          : now) - task.startedAt.toDate().getTime(),
      )
    : null;

  const assignedRuntimeName = task.assignedRuntime
    ? (runtimeNameMap.get(task.assignedRuntime.toHexString()) ?? "Unknown")
    : null;

  const agentTypeName =
    agentTypeMap.get(Number(task.agentTypeId)) ?? "Unknown";

  return (
    <Stack gap={0} h="100%">
      {isRunning && (
        <Progress
          value={Math.min((latestTurn / 24) * 100, 100)}
          size={2}
          radius={0}
          color="yellow"
        />
      )}

      <Box pb="md">
        <Stack gap="sm">
          <TaskBreadcrumbs taskLabel={`#${Number(task.id)}`} />

          <Group gap="sm">
            <Title order={4}>{task.title}</Title>
            <Badge
              size="sm"
              variant="light"
              color={taskStatusColor(task.status.tag)}
            >
              {task.status.tag}
            </Badge>
            {isRunning && elapsed && (
              <Text size="xs" c="yellow.3">
                {elapsed}
              </Text>
            )}
            <Text size="xs" c="dimmed" ml="auto">
              #{Number(task.id)}
            </Text>
          </Group>
        </Stack>
      </Box>

      <Tabs
        defaultValue="conversation"
        style={{
          flex: 1,
          minHeight: 0,
          display: "flex",
          flexDirection: "column",
        }}
      >
        <Tabs.List>
          <Tabs.Tab value="conversation">Conversation</Tabs.Tab>
          <Tabs.Tab value="details">Details</Tabs.Tab>
          <Tabs.Tab
            value="observations"
            rightSection={
              taskObservations.length > 0 ? (
                <Badge size="xs" variant="filled" color="dark.5" circle>
                  {taskObservations.length}
                </Badge>
              ) : undefined
            }
          >
            Observations
          </Tabs.Tab>
        </Tabs.List>

        <Tabs.Panel
          value="conversation"
          pt="md"
          style={{ flex: 1, minHeight: 0, overflowY: "auto" }}
        >
          <ConversationView
            events={taskRuntimeEvents}
            taskDescription={task.description}
            taskStatus={task.status.tag}
            systemPrompt={activeVersion?.systemPrompt}
          />
        </Tabs.Panel>

        <Tabs.Panel
          value="details"
          pt="md"
          style={{ flex: 1, minHeight: 0, overflowY: "auto" }}
        >
          <TaskDetailsTab
            task={task}
            activeVersion={activeVersion}
            agentTypeName={agentTypeName}
            assignedRuntimeName={assignedRuntimeName}
            latestTurn={latestTurn}
            elapsed={elapsed}
            runtimes={runtimes ?? []}
            conn={conn}
          />
        </Tabs.Panel>

        <Tabs.Panel
          value="observations"
          pt="md"
          style={{ flex: 1, minHeight: 0, overflowY: "auto" }}
        >
          <ObservationsList observations={taskObservations} />
        </Tabs.Panel>
      </Tabs>
    </Stack>
  );
}
