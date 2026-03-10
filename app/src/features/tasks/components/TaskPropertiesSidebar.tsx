import { Badge, Group, Paper, Stack, Text } from "@mantine/core";

import { formatDuration, formatTimestamp } from "@/shared/utils/formatting";
import { taskStatusColor } from "@/shared/utils/statusColors";

import { TaskAssignmentControl } from "./TaskAssignmentControl";

import type { DbConnection } from "@/module_bindings";
import type { AgentTask, AgentRuntime } from "@/module_bindings/types";

function Property({
  label,
  children,
}: {
  label: string;
  children: React.ReactNode;
}) {
  return (
    <Group
      justify="space-between"
      align="flex-start"
      gap="md"
      py="xs"
      style={{ borderBottom: "1px solid var(--mantine-color-dark-4)" }}
    >
      <Text size="xs" c="dimmed" style={{ flexShrink: 0 }}>
        {label}
      </Text>
      <Text size="xs" ta="right">
        {children}
      </Text>
    </Group>
  );
}

type TaskPropertiesSidebarProps = {
  task: AgentTask;
  agentTypeName: string;
  assignedRuntimeName: string | null;
  latestTurn: number;
  elapsed: string | null;
  runtimes: AgentRuntime[];
  conn: DbConnection | null;
};

export function TaskPropertiesSidebar({
  task,
  agentTypeName,
  assignedRuntimeName,
  latestTurn,
  elapsed,
  runtimes,
  conn,
}: TaskPropertiesSidebarProps) {
  return (
    <Stack gap="md">
      <Paper withBorder radius="md" p="md">
        <Text
          size="xs"
          fw={500}
          tt="uppercase"
          lts={0.5}
          c="dimmed"
          mb="xs"
        >
          Properties
        </Text>
        <Stack gap={0}>
          <Property label="Status">
            <Badge
              size="xs"
              variant="light"
              color={taskStatusColor(task.status.tag)}
            >
              {task.status.tag}
            </Badge>
          </Property>
          <Property label="Agent type">{agentTypeName}</Property>
          <Property label="Runtime">
            {assignedRuntimeName ?? (
              <Text span c="dimmed" size="xs">
                Unassigned
              </Text>
            )}
          </Property>
          <Property label="Turn">
            {latestTurn > 0 ? (
              `${latestTurn} / 24`
            ) : (
              <Text span c="dimmed" size="xs">
                —
              </Text>
            )}
          </Property>
          {elapsed && <Property label="Duration">{elapsed}</Property>}
          <Property label="Created">
            {formatTimestamp(task.createdAt.toDate())}
          </Property>
          {task.startedAt && (
            <Property label="Started">
              {formatTimestamp(task.startedAt.toDate())}
            </Property>
          )}
          {task.completedAt && (
            <Property label="Completed">
              {formatTimestamp(task.completedAt.toDate())}
            </Property>
          )}
        </Stack>
      </Paper>

      {task.status.tag === "Pending" && (
        <Paper withBorder radius="md" p="md">
          <Text
            size="xs"
            fw={500}
            tt="uppercase"
            lts={0.5}
            c="dimmed"
            mb="sm"
          >
            Assignment
          </Text>
          <TaskAssignmentControl
            taskId={task.id}
            taskStatus={task.status.tag}
            assignedRuntime={task.assignedRuntime ?? null}
            runtimes={runtimes}
            conn={conn}
            compact
          />
        </Paper>
      )}
    </Stack>
  );
}
