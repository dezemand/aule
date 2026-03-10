import {
  Badge,
  Box,
  Group,
  Paper,
  Stack,
  Text,
  UnstyledButton,
} from "@mantine/core";

import { versionStatusColor } from "@/shared/utils/statusColors";

import { CreateVersionForm } from "./CreateVersionForm";

import type { DbConnection } from "@/module_bindings";
import type { AgentType, AgentTypeVersion } from "@/module_bindings/types";

type AgentTypeCardProps = {
  agentType: AgentType;
  versions: AgentTypeVersion[];
  showCreateVersion: boolean;
  onToggleCreateVersion: () => void;
  conn: DbConnection | null;
};

export function AgentTypeCard({
  agentType,
  versions,
  showCreateVersion,
  onToggleCreateVersion,
  conn,
}: AgentTypeCardProps) {
  return (
    <Paper withBorder radius="md" p="md">
      <Group justify="space-between" align="flex-start">
        <div>
          <Text fw={500}>{agentType.name}</Text>
          <Text size="sm" c="dimmed" mt={2}>
            {agentType.description}
          </Text>
        </div>
        <Text size="xs" c="dimmed">
          {versions.length} version{versions.length !== 1 ? "s" : ""}
        </Text>
      </Group>

      <Box
        mt="sm"
        pt="sm"
        style={{ borderTop: "1px solid var(--mantine-color-dark-4)" }}
      >
        <Group justify="space-between" mb="xs">
          <Text size="xs" fw={500} tt="uppercase" lts={0.5} c="dimmed">
            Versions
          </Text>
          <UnstyledButton fz="xs" c="blue.4" onClick={onToggleCreateVersion}>
            {showCreateVersion ? "Cancel" : "+ Add Version"}
          </UnstyledButton>
        </Group>

        {showCreateVersion && conn && (
          <CreateVersionForm
            conn={conn}
            agentTypeId={agentType.id}
            onCreated={onToggleCreateVersion}
          />
        )}

        {versions.length === 0 ? (
          <Text size="xs" c="dimmed">
            No versions yet.
          </Text>
        ) : (
          <Stack gap={6}>
            {versions.map((v) => (
              <Paper key={Number(v.id)} radius="sm" px="sm" py="xs" bg="dark.6">
                <Group justify="space-between">
                  <Group gap="xs">
                    <Text ff="monospace" size="sm" c="gray.4">
                      {v.version}
                    </Text>
                    <Badge
                      size="xs"
                      variant="light"
                      color={versionStatusColor(v.status.tag)}
                    >
                      {v.status.tag}
                    </Badge>
                  </Group>
                  <Group gap="xs">
                    {v.status.tag === "Draft" && conn && (
                      <UnstyledButton
                        fz="xs"
                        c="blue.4"
                        onClick={() =>
                          conn.reducers.activateAgentTypeVersion({
                            versionId: v.id,
                          })
                        }
                      >
                        Activate
                      </UnstyledButton>
                    )}
                    <Text size="xs" c="dimmed">
                      {v.createdAt.toDate().toLocaleDateString()}
                    </Text>
                  </Group>
                </Group>
              </Paper>
            ))}
          </Stack>
        )}
      </Box>
    </Paper>
  );
}
