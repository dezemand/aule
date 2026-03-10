import { Badge, Group, Paper, Stack, Text } from "@mantine/core";

import { observationKindColor } from "@/shared/utils/statusColors";

import type { Observation } from "@/module_bindings/types";

type RecentObservationsProps = {
  observations: Observation[];
};

export function RecentObservations({ observations }: RecentObservationsProps) {
  if (observations.length === 0) {
    return (
      <Text size="sm" c="dimmed">
        No observations yet.
      </Text>
    );
  }

  return (
    <Stack gap="xs">
      {observations.map((o) => (
        <Paper key={Number(o.id)} withBorder radius="md" px="md" py="sm">
          <Group gap="xs" mb={4}>
            <Badge
              size="xs"
              variant="light"
              color={observationKindColor(o.kind.tag)}
            >
              {o.kind.tag}
            </Badge>
            <Text size="xs" c="dimmed">
              Task #{Number(o.taskId)}
            </Text>
            <Text size="xs" c="dimmed">
              {o.createdAt.toDate().toLocaleString()}
            </Text>
          </Group>
          <Text size="sm" c="gray.4">
            {o.content}
          </Text>
        </Paper>
      ))}
    </Stack>
  );
}
