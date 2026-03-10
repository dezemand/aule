import { Box, Code, Group, Paper, Text } from "@mantine/core";

import { Markdown } from "@/shared/components/Markdown/Markdown";

import { ShellOutputDetails } from "./ShellOutputDetails";

import type { TimelineItem } from "../utils/buildTimeline";

type TimelineEntryProps = {
  item: TimelineItem;
};

export function TimelineEntry({ item }: TimelineEntryProps) {
  if (item.kind === "assistant") {
    return (
      <Box pos="relative">
        <Box
          pos="absolute"
          left={-25}
          top={10}
          w={10}
          h={10}
          style={{
            borderRadius: "50%",
            border: "1px solid var(--mantine-color-violet-7)",
            background: "rgba(137, 87, 229, 0.15)",
          }}
        />
        <Paper withBorder radius="lg" p="md">
          <Group justify="space-between" mb="xs">
            <Text fz={11} fw={500} tt="uppercase" lts={0.5} c="violet.4">
              Assistant
            </Text>
            <Text fz={11} c="dark.2">
              {item.timestamp}
            </Text>
          </Group>
          <Markdown>{item.content}</Markdown>
        </Paper>
      </Box>
    );
  }

  return (
    <Box pos="relative">
      <Box
        pos="absolute"
        left={-25}
        top={8}
        w={10}
        h={10}
        style={{
          borderRadius: "50%",
          border: "1px solid var(--mantine-color-dark-3)",
          background: "rgba(134, 142, 150, 0.15)",
        }}
      />
      <Paper withBorder radius="md" px="sm" py="xs" bg="dark.7">
        <Group justify="space-between" gap="sm">
          <Text fz={13} c="gray.4">
            {item.title}
          </Text>
          <Text fz={11} c="dark.2" style={{ flexShrink: 0 }}>
            {item.timestamp}
          </Text>
        </Group>
        {item.action && (
          <Code
            block
            mt={4}
            fz="xs"
            c="dimmed"
            style={{
              maxHeight: 120,
              overflow: "auto",
              whiteSpace: "pre",
            }}
          >
            {item.action}
          </Code>
        )}
        {item.result && (
          <Text fz="xs" c="dimmed" mt={4}>
            {item.result}
          </Text>
        )}
        {item.shellOutput && <ShellOutputDetails output={item.shellOutput} />}
      </Paper>
    </Box>
  );
}
