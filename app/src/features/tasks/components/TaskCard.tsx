import { Link } from "@tanstack/react-router";
import { Badge, Box, Group, Paper, Progress, Text } from "@mantine/core";

import { Markdown } from "@/shared/components/Markdown/Markdown";
import { formatDuration } from "@/shared/utils/formatting";
import { taskStatusColor } from "@/shared/utils/statusColors";

import { ObservationsList } from "./ObservationsList";
import { TaskAssignmentControl } from "./TaskAssignmentControl";

import type { DbConnection } from "@/module_bindings";
import type { AgentTask, Observation, AgentRuntime } from "@/module_bindings/types";

type TaskCardProps = {
  task: AgentTask;
  agentTypeName: string;
  runtimeName: string;
  latestTurn: number;
  observations: Observation[];
  runtimes: AgentRuntime[];
  conn: DbConnection | null;
  now: number;
};

export function TaskCard({
  task,
  agentTypeName,
  runtimeName,
  latestTurn,
  observations,
  runtimes,
  conn,
  now,
}: TaskCardProps) {
  const elapsed = task.startedAt
    ? formatDuration(now - task.startedAt.toDate().getTime())
    : null;

  return (
    <Paper withBorder radius="md" p="md">
      <Group justify="space-between" align="flex-start" gap="md">
        <Box style={{ minWidth: 0, flex: 1 }}>
          <Group gap="xs">
            <Text fw={500}>{task.title}</Text>
            <Badge
              size="sm"
              variant="light"
              color={taskStatusColor(task.status.tag)}
            >
              {task.status.tag}
            </Badge>
          </Group>
          <Box mt={4}>
            <Markdown>{task.description || "(no description)"}</Markdown>
          </Box>
        </Box>
        <Text size="xs" c="dimmed">
          #{Number(task.id)}
        </Text>
      </Group>

      <Group gap="md" mt="sm" wrap="wrap">
        <Text size="xs" c="dimmed">
          Type: {agentTypeName}
        </Text>
        <Text size="xs" c="dimmed">
          Created: {task.createdAt.toDate().toLocaleString()}
        </Text>
        <Text size="xs" c="dimmed">
          Runtime: {runtimeName}
        </Text>
        {latestTurn > 0 && (
          <Text size="xs" c="dimmed">
            Turn: {latestTurn}
          </Text>
        )}
        {task.status.tag === "Running" && elapsed && (
          <Text size="xs" c="yellow.3">
            Running for {elapsed}
          </Text>
        )}
      </Group>

      {task.status.tag === "Running" && (
        <Progress
          value={Math.min((latestTurn / 24) * 100, 100)}
          size={6}
          radius="xl"
          color="yellow"
          mt="xs"
        />
      )}

      {task.result && (
        <Paper
          mt="sm"
          p="sm"
          radius="sm"
          style={{
            border: "1px solid var(--mantine-color-green-9)",
            background: "rgba(0, 100, 0, 0.08)",
          }}
        >
          <Text size="xs" fw={500} tt="uppercase" lts={0.5} c="green.3">
            Result
          </Text>
          <Box mt={4}>
            <Markdown>{task.result}</Markdown>
          </Box>
        </Paper>
      )}

      <Box mt="sm">
        <TaskAssignmentControl
          taskId={task.id}
          taskStatus={task.status.tag}
          assignedRuntime={task.assignedRuntime ?? null}
          runtimes={runtimes}
          conn={conn}
          compact
        />
      </Box>

      <Box mt="sm">
        <Text size="xs">
          <Link
            to="/tasks/$taskId"
            params={{ taskId: task.id.toString() }}
            style={{ color: "var(--mantine-color-blue-4)" }}
          >
            View details →
          </Link>
        </Text>
      </Box>

      {observations.length > 0 && (
        <Box
          mt="sm"
          pt="sm"
          style={{ borderTop: "1px solid var(--mantine-color-dark-4)" }}
        >
          <Text
            size="xs"
            fw={500}
            tt="uppercase"
            lts={0.5}
            c="dimmed"
            mb="xs"
          >
            Observations
          </Text>
          <ObservationsList observations={observations} compact />
        </Box>
      )}
    </Paper>
  );
}
