import { Badge, Box, Group, Paper, Stack, Text } from "@mantine/core";

import { Markdown } from "@/shared/components/Markdown/Markdown";
import { formatTimestamp } from "@/shared/utils/formatting";
import { observationKindColor } from "@/shared/utils/statusColors";

import type { Observation } from "@/module_bindings/types";

type ObservationsListProps = {
  observations: Observation[];
  compact?: boolean;
};

export function ObservationsList({
  observations,
  compact = false,
}: ObservationsListProps) {
  if (observations.length === 0) {
    return (
      <Text size="sm" c="dimmed">
        No observations yet.
      </Text>
    );
  }

  return (
    <Stack gap={compact ? 6 : "xs"}>
      {observations.map((obs) => (
        <Paper
          key={Number(obs.id)}
          withBorder={!compact}
          radius={compact ? "sm" : "md"}
          px={compact ? "sm" : "md"}
          py={compact ? "xs" : "sm"}
          bg={compact ? "dark.8" : undefined}
        >
          <Group gap="xs" mb={4}>
            <Badge
              size="xs"
              variant="light"
              color={observationKindColor(obs.kind.tag)}
            >
              {obs.kind.tag}
            </Badge>
            <Text size="xs" c="dimmed" ml={compact ? "auto" : undefined}>
              {formatTimestamp(obs.createdAt.toDate())}
            </Text>
          </Group>
          <Box mt={4}>
            <Markdown>{obs.content}</Markdown>
          </Box>
        </Paper>
      ))}
    </Stack>
  );
}
