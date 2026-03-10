import { Box, Group, Text } from "@mantine/core";
import { useSpacetimeDB } from "spacetimedb/react";

export function ConnectionBadge() {
  const { isActive, connectionError } = useSpacetimeDB();
  const errorMessage = connectionError?.message;

  if (errorMessage) {
    return (
      <Group gap={6}>
        <Box
          w={8}
          h={8}
          style={{
            borderRadius: "50%",
            background: "var(--mantine-color-red-6)",
          }}
        />
        <Text size="xs" c="red.4">
          {errorMessage}
        </Text>
      </Group>
    );
  }

  if (!isActive) {
    return (
      <Group gap={6}>
        <Box
          w={8}
          h={8}
          style={{
            borderRadius: "50%",
            background: "var(--mantine-color-dark-4)",
          }}
        />
        <Text size="xs" c="dimmed">
          Disconnected
        </Text>
      </Group>
    );
  }

  return (
    <Group gap={6}>
      <Box
        w={8}
        h={8}
        style={{
          borderRadius: "50%",
          background: "var(--mantine-color-green-6)",
        }}
      />
      <Text size="xs" c="green.4">
        Connected
      </Text>
    </Group>
  );
}
