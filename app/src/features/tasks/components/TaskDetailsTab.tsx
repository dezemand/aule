import { Box, Grid, Paper, Stack, Text } from "@mantine/core";

import { Markdown } from "@/shared/components/Markdown/Markdown";

import { TaskPropertiesSidebar } from "./TaskPropertiesSidebar";

import type { DbConnection } from "@/module_bindings";
import type { AgentTask, AgentTypeVersion, AgentRuntime } from "@/module_bindings/types";

type TaskDetailsTabProps = {
  task: AgentTask;
  activeVersion: AgentTypeVersion | undefined;
  agentTypeName: string;
  assignedRuntimeName: string | null;
  latestTurn: number;
  elapsed: string | null;
  runtimes: AgentRuntime[];
  conn: DbConnection | null;
};

export function TaskDetailsTab({
  task,
  activeVersion,
  agentTypeName,
  assignedRuntimeName,
  latestTurn,
  elapsed,
  runtimes,
  conn,
}: TaskDetailsTabProps) {
  return (
    <Grid gutter="lg">
      <Grid.Col span={{ base: 12, lg: 8 }}>
        <Stack gap="lg">
          <section>
            <Text
              size="xs"
              fw={500}
              tt="uppercase"
              lts={0.5}
              c="dimmed"
              mb="sm"
            >
              Description
            </Text>
            <Paper withBorder radius="md" p="lg">
              <Markdown>
                {task.description || "No description provided."}
              </Markdown>
            </Paper>
          </section>

          {task.result && (
            <section>
              <Text
                size="xs"
                fw={500}
                tt="uppercase"
                lts={0.5}
                c="dimmed"
                mb="sm"
              >
                Result
              </Text>
              <Paper
                withBorder
                radius="md"
                p="lg"
                style={{
                  borderColor:
                    task.status.tag === "Failed"
                      ? "var(--mantine-color-red-9)"
                      : "var(--mantine-color-green-9)",
                  background:
                    task.status.tag === "Failed"
                      ? "rgba(100, 0, 0, 0.08)"
                      : "rgba(0, 100, 0, 0.08)",
                }}
              >
                <Markdown>{task.result}</Markdown>
              </Paper>
            </section>
          )}

          {activeVersion && (
            <section>
              <Text
                size="xs"
                fw={500}
                tt="uppercase"
                lts={0.5}
                c="dimmed"
                mb="sm"
              >
                System prompt
              </Text>
              <Paper withBorder radius="md" style={{ overflow: "hidden" }}>
                <details>
                  <summary
                    style={{
                      cursor: "pointer",
                      padding: "0.75rem 1.25rem",
                      fontSize: "0.75rem",
                      color: "var(--mantine-color-dimmed)",
                    }}
                  >
                    v{activeVersion.version} — click to expand
                  </summary>
                  <Box
                    px="lg"
                    py="md"
                    style={{
                      borderTop: "1px solid var(--mantine-color-dark-4)",
                    }}
                  >
                    <Markdown>{activeVersion.systemPrompt}</Markdown>
                  </Box>
                </details>
              </Paper>
            </section>
          )}
        </Stack>
      </Grid.Col>

      <Grid.Col span={{ base: 12, lg: 4 }}>
        <TaskPropertiesSidebar
          task={task}
          agentTypeName={agentTypeName}
          assignedRuntimeName={assignedRuntimeName}
          latestTurn={latestTurn}
          elapsed={elapsed}
          runtimes={runtimes}
          conn={conn}
        />
      </Grid.Col>
    </Grid>
  );
}
